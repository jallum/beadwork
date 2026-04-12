package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/recap"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
)

// storeLookup adapts an *issue.Store to the recap.IssueLookup interface.
type storeLookup struct {
	store *issue.Store
}

func (s *storeLookup) Title(id string) string {
	if s.store == nil {
		return ""
	}
	iss, err := s.store.Get(id)
	if err != nil {
		return ""
	}
	return iss.Title
}

type recapArgs struct {
	Tokens []string
	Since  string
	JSON   bool
	ASCII  bool
	DryRun bool
	All    bool
}

func parseRecapArgs(raw []string) (recapArgs, error) {
	a, err := ParseArgs(raw,
		[]string{"--since"},
		[]string{"--json", "--ascii", "--dry-run", "--all"},
	)
	if err != nil {
		return recapArgs{}, err
	}
	return recapArgs{
		Tokens: a.Pos(),
		Since:  a.String("--since"),
		JSON:   a.Bool("--json"),
		ASCII:  a.Bool("--ascii") || globalNoColor,
		DryRun: a.Bool("--dry-run") || globalDryRun,
		All:    a.Bool("--all"),
	}, nil
}

// cmdRecap is the entry point. NeedsStore is false; we open our own store.
func cmdRecap(_ *issue.Store, args []string, w Writer) error {
	ra, err := parseRecapArgs(args)
	if err != nil {
		return err
	}

	if ra.All {
		return cmdRecapAll(ra, w)
	}

	return runRecapSingle(ra, w, repoDir)
}

// runRecapSingle runs recap against a single repository rooted at dir.
// If dir is "", the current working directory is used (via FindRepoAt).
func runRecapSingle(ra recapArgs, w Writer, dir string) error {
	r, err := repo.FindRepoAt(dir)
	if err != nil {
		return fmt.Errorf("not a git repository (run bw init first)")
	}
	if !r.IsInitialized() {
		return fmt.Errorf("beadwork not initialized. Run: bw init")
	}

	store := issue.NewStore(r.TreeFS(), r.Prefix)
	store.Committer = r

	repoPath, err := registry.CanonicalRepoPath(r.RepoDir())
	if err != nil {
		repoPath = r.RepoDir()
	}

	regDir := registry.DefaultDir()
	reg, regErr := registry.Load(regDir)

	// Resolve the window.
	now := bwNow()
	var window recap.Window

	if ra.Since != "" || len(ra.Tokens) > 0 {
		window, err = recap.ParseWindow(ra.Tokens, ra.Since, now)
		if err != nil {
			return err
		}
	} else {
		// No explicit window: use cursor-based incremental recap.
		// First-recap (no cursor) = 24h backfill per D1.
		var cursor string
		if regErr == nil {
			if e, ok := reg.Entries()[repoPath]; ok {
				cursor = e.Cursor
			}
		}
		if cursor == "" {
			window = recap.Window{
				Start: now.Add(-24 * time.Hour),
				End:   now,
				Label: "last 24h (first recap)",
			}
		} else {
			// Use the commits since the cursor.
			window = recap.Window{
				Start: time.Unix(0, 0),
				End:   now,
				Label: "since last recap",
			}
		}
	}

	// Gather commits.
	commits, err := r.AllCommits()
	if err != nil {
		return fmt.Errorf("read commits: %w", err)
	}

	// Build recap.
	rcp := recap.Build(commits, window, &storeLookup{store: store})

	// Render.
	if err := renderRecap(w, rcp, ra); err != nil {
		return err
	}

	// Advance the cursor unless --dry-run.
	if !ra.DryRun && regErr == nil && len(commits) > 0 {
		newCursor := commits[0].Hash
		// Silent failure: advancing cursor is best-effort.
		_ = reg.AdvanceCursorAndSave(repoPath, newCursor)
	}

	return nil
}

// cmdRecapAll is the cross-repo fan-out (S7).
func cmdRecapAll(ra recapArgs, w Writer) error {
	// -C + --all is inconsistent — warn.
	if repoDir != "" {
		fmt.Fprintln(os.Stderr, "warning: -C is ignored with --all")
	}

	regDir := registry.DefaultDir()
	reg, err := registry.Load(regDir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	entries := reg.Entries()
	if len(entries) == 0 {
		fmt.Fprintln(w, "no registered repositories")
		return nil
	}

	// Iterate repos in sorted order for deterministic output.
	var paths []string
	for p := range entries {
		paths = append(paths, p)
	}
	sortStrings(paths)

	now := bwNow()
	var all []repoRecap

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", p, err)
			continue
		}
		r, err := repo.FindRepoAt(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", p, err)
			continue
		}
		if !r.IsInitialized() {
			fmt.Fprintf(os.Stderr, "skipping %s: beadwork not initialized\n", p)
			continue
		}

		store := issue.NewStore(r.TreeFS(), r.Prefix)
		store.Committer = r

		// Resolve window per-repo (cursor-based unless user overrode).
		var window recap.Window
		if ra.Since != "" || len(ra.Tokens) > 0 {
			window, err = recap.ParseWindow(ra.Tokens, ra.Since, now)
			if err != nil {
				return err
			}
		} else {
			cursor := entries[p].Cursor
			if cursor == "" {
				window = recap.Window{
					Start: now.Add(-24 * time.Hour),
					End:   now,
					Label: "last 24h (first recap)",
				}
			} else {
				window = recap.Window{
					Start: time.Unix(0, 0),
					End:   now,
					Label: "since last recap",
				}
			}
		}

		commits, err := r.AllCommits()
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: read commits: %v\n", p, err)
			continue
		}

		rcp := recap.Build(commits, window, &storeLookup{store: store})
		all = append(all, repoRecap{Path: p, Recap: rcp})

		// Advance cursor per repo unless dry-run.
		if !ra.DryRun && len(commits) > 0 {
			_ = reg.AdvanceCursorAndSave(p, commits[0].Hash)
		}
	}

	return renderCrossRepo(w, all, ra)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
