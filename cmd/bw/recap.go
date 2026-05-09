package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/recap"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/treefs"
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
	Tokens  []string
	Since   string
	JSON    bool
	ASCII   bool
	DryRun  bool
	All     bool
	Verbose bool
}

func parseRecapArgs(raw []string) (recapArgs, error) {
	expanded := make([]string, len(raw))
	for i, tok := range raw {
		if tok == "-v" {
			expanded[i] = "--verbose"
		} else {
			expanded[i] = tok
		}
	}
	a, err := ParseArgs(expanded,
		[]string{"--since"},
		[]string{"--json", "--ascii", "--dry-run", "--all", "--verbose"},
	)
	if err != nil {
		return recapArgs{}, err
	}
	return recapArgs{
		Tokens:  a.Pos(),
		Since:   a.String("--since"),
		JSON:    a.Bool("--json"),
		ASCII:   a.Bool("--ascii") || globalNoColor,
		DryRun:  a.Bool("--dry-run") || globalDryRun,
		All:     a.Bool("--all"),
		Verbose: a.Bool("--verbose"),
	}, nil
}

func cmdRecap(_ *issue.Store, args []string, w Writer, cfg *config.Config) (*config.Config, error) {
	ra, err := parseRecapArgs(args)
	if err != nil {
		return nil, err
	}

	if ra.All {
		return nil, cmdRecapAll(ra, w, cfg)
	}

	return nil, runRecapSingle(ra, w, repoDir)
}

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

	now := bwNow()
	window, commits, err := resolveRecapWindow(ra, r, now)
	if err != nil {
		return err
	}

	rcp := recap.Build(commits, window, &storeLookup{store: store})

	if err := renderRecap(w, rcp, ra); err != nil {
		return err
	}

	if !ra.DryRun {
		explicit := ra.Since != "" || len(ra.Tokens) > 0
		cursor := r.RecapCursor()
		if explicit && cursor != "" {
			gap, cursorKnown := countGap(commits, cursor, window.Start)
			if cursorKnown && gap > 0 {
				fmt.Fprint(os.Stderr, gapNoticeLine("", gap))
				return nil
			}
		}
		switch {
		case len(commits) > 0:
			_ = r.SetRecapCursor(commits[0].Hash)
		case cursor != "":
			_ = r.TouchRecapCursor()
		}
	}

	return nil
}

// resolveRecapWindow determines the time window and commit set for a recap.
// Three cases:
//  1. Explicit window (--since or positional tokens) → filter AllCommits by time.
//  2. No cursor yet (first recap) → last 24h backfill.
//  3. Cursor present → use CommitsSince(cursor).
func resolveRecapWindow(ra recapArgs, r *repo.Repo, now time.Time) (recap.Window, []treefs.CommitInfo, error) {
	explicit := ra.Since != "" || len(ra.Tokens) > 0
	cursor := r.RecapCursor()

	switch {
	case explicit:
		window, err := recap.ParseWindow(ra.Tokens, ra.Since, now)
		if err != nil {
			return recap.Window{}, nil, err
		}
		commits, err := r.AllCommits()
		if err != nil {
			return recap.Window{}, nil, fmt.Errorf("read commits: %w", err)
		}
		return window, commits, nil

	case cursor == "":
		window := recap.Window{
			Start: now.Add(-24 * time.Hour),
			End:   now,
			Label: "last 24h (first recap)",
		}
		commits, err := r.AllCommits()
		if err != nil {
			return recap.Window{}, nil, fmt.Errorf("read commits: %w", err)
		}
		return window, commits, nil

	default:
		commits, err := r.TreeFS().CommitsSince(cursor)
		if err != nil {
			return recap.Window{}, nil, fmt.Errorf("read commits: %w", err)
		}
		start := now
		if len(commits) > 0 {
			start = commits[len(commits)-1].Time
		}
		label := "since last recap"
		if lastAt := r.LastRecapAt(); !lastAt.IsZero() {
			label = fmt.Sprintf("since last recap (%s)", relativeTimeSince(lastAt, now))
		}
		return recap.Window{Start: start, End: now, Label: label}, commits, nil
	}
}

func gapNoticeLine(prefix string, gap int) string {
	noun := "commits"
	if gap == 1 {
		noun = "commit"
	}
	lead := ""
	if prefix != "" {
		lead = prefix + ": "
	}
	return fmt.Sprintf("%s%d %s older than this window and newer than your last recap were not shown. Run 'bw recap' to see them.\n",
		lead, gap, noun)
}

func countGap(commits []treefs.CommitInfo, cursor string, windowStart time.Time) (int, bool) {
	var cursorTime time.Time
	found := false
	for _, c := range commits {
		if c.Hash == cursor {
			cursorTime = c.Time
			found = true
			break
		}
	}
	if !found {
		return 0, false
	}
	gap := 0
	for _, c := range commits {
		if c.Time.After(cursorTime) && c.Time.Before(windowStart) {
			gap++
		}
	}
	return gap, true
}

func cmdRecapAll(ra recapArgs, w Writer, cfg *config.Config) error {
	if repoDir != "" {
		fmt.Fprintln(os.Stderr, "warning: -C is ignored with --all")
	}

	paths := registry.Paths(cfg)
	if len(paths) == 0 {
		fmt.Fprintln(w, "no registered repositories")
		return nil
	}

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

		window, commits, err := resolveRecapWindow(ra, r, now)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", p, err)
			continue
		}

		rcp := recap.Build(commits, window, &storeLookup{store: store})
		all = append(all, repoRecap{Path: p, Recap: rcp})

		if !ra.DryRun {
			explicit := ra.Since != "" || len(ra.Tokens) > 0
			cursor := r.RecapCursor()
			if explicit && cursor != "" {
				gap, cursorKnown := countGap(commits, cursor, window.Start)
				if cursorKnown && gap > 0 {
					fmt.Fprint(os.Stderr, gapNoticeLine(p, gap))
					continue
				}
			}
			switch {
			case len(commits) > 0:
				_ = r.SetRecapCursor(commits[0].Hash)
			case cursor != "":
				_ = r.TouchRecapCursor()
			}
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	return renderCrossRepo(w, all, ra)
}
