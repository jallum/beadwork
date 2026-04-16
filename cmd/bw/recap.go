package main

import (
	"fmt"
	"os"
	"time"

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
	// Expand -v to --verbose before generic parsing.
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

	// Resolve the window + commits. Three cases:
	//   1. User gave an explicit window → filter AllCommits by time.
	//   2. No cursor yet (first recap)  → last 24h backfill.
	//   3. Cursor present                → use CommitsSince(cursor) so the
	//      rendered set is EXACTLY the commits newer than the last recap,
	//      independent of any wall-clock window.
	now := bwNow()
	var window recap.Window
	var commits []treefs.CommitInfo

	explicit := ra.Since != "" || len(ra.Tokens) > 0
	var cursor string
	if regErr == nil {
		if e, ok := reg.Entries()[repoPath]; ok {
			cursor = e.Cursor
		}
	}

	switch {
	case explicit:
		window, err = recap.ParseWindow(ra.Tokens, ra.Since, now)
		if err != nil {
			return err
		}
		commits, err = r.AllCommits()
		if err != nil {
			return fmt.Errorf("read commits: %w", err)
		}

	case cursor == "":
		// First recap on this repo — 24h backfill.
		window = recap.Window{
			Start: now.Add(-24 * time.Hour),
			End:   now,
			Label: "last 24h (first recap)",
		}
		commits, err = r.AllCommits()
		if err != nil {
			return fmt.Errorf("read commits: %w", err)
		}

	default:
		// Cursor-driven incremental recap.
		commits, err = r.TreeFS().CommitsSince(cursor)
		if err != nil {
			return fmt.Errorf("read commits: %w", err)
		}
		// Window bounds: earliest commit → now. If CommitsSince returned
		// nothing (cursor == HEAD), Start stays at now and Build shows empty.
		start := now
		if len(commits) > 0 {
			start = commits[len(commits)-1].Time
		}
		// Label reflects when the user LAST RAN recap (not the cursor
		// commit time) so repeated runs update monotonically even when
		// there's nothing new.
		label := "since last recap"
		if regErr == nil {
			if e, ok := reg.Entries()[repoPath]; ok && e.LastRecapAt != "" {
				if t, perr := time.Parse(time.RFC3339, e.LastRecapAt); perr == nil {
					label = fmt.Sprintf("since last recap (%s)", relativeTimeSince(t, now))
				}
			}
		}
		window = recap.Window{Start: start, End: now, Label: label}
	}

	// Build recap.
	rcp := recap.Build(commits, window, &storeLookup{store: store})

	// Render.
	if err := renderRecap(w, rcp, ra); err != nil {
		return err
	}

	// Cursor advance and last_recap_at stamping.
	//
	// Normally we stamp last_recap_at on every non-dry-run recap and advance
	// the cursor to HEAD when there are new commits. Exception: an explicit
	// window that starts AFTER the current cursor (a "gap") would strand
	// commits between cursor_time and window.Start — they'd never render and
	// would be marked seen. See ADR
	// recap-explicit-window-conditional-advance: on a gapped explicit run we
	// stamp neither field and print a stderr notice.
	if !ra.DryRun && regErr == nil {
		if explicit && cursor != "" {
			gap, cursorKnown := countGap(commits, cursor, window.Start)
			if cursorKnown && gap > 0 {
				fmt.Fprint(os.Stderr, gapNoticeLine("", gap))
				return nil
			}
		}
		newCursor := ""
		if len(commits) > 0 {
			newCursor = commits[0].Hash
		}
		// Silent failure: updating registry is best-effort.
		_ = reg.StampRecapAndSave(repoPath, newCursor, now)
	}

	return nil
}

// gapNoticeLine builds the stderr line printed on a gapped explicit recap.
// If prefix is non-empty (the --all fan-out case), it's rendered as "<prefix>: ".
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

// countGap returns the number of commits in `commits` that are newer than the
// commit identified by `cursor` and strictly older than `windowStart`. It also
// reports whether the cursor hash was found in `commits` at all. If the cursor
// isn't found (e.g. registry is stale or commits has been trimmed), the caller
// should fall back to the normal stamping path.
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

		// Resolve window + commits per-repo. Same three cases as single-repo.
		explicit := ra.Since != "" || len(ra.Tokens) > 0
		cursor := entries[p].Cursor

		var window recap.Window
		var commits []treefs.CommitInfo
		switch {
		case explicit:
			window, err = recap.ParseWindow(ra.Tokens, ra.Since, now)
			if err != nil {
				return err
			}
			commits, err = r.AllCommits()
		case cursor == "":
			window = recap.Window{
				Start: now.Add(-24 * time.Hour),
				End:   now,
				Label: "last 24h (first recap)",
			}
			commits, err = r.AllCommits()
		default:
			commits, err = r.TreeFS().CommitsSince(cursor)
			start := now
			if err == nil && len(commits) > 0 {
				start = commits[len(commits)-1].Time
			}
			label := "since last recap"
			if lr := entries[p].LastRecapAt; lr != "" {
				if t, perr := time.Parse(time.RFC3339, lr); perr == nil {
					label = fmt.Sprintf("since last recap (%s)", relativeTimeSince(t, now))
				}
			}
			window = recap.Window{Start: start, End: now, Label: label}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: read commits: %v\n", p, err)
			continue
		}

		rcp := recap.Build(commits, window, &storeLookup{store: store})
		all = append(all, repoRecap{Path: p, Recap: rcp})

		// Stamp + optionally advance cursor per repo unless dry-run. See
		// ADR recap-explicit-window-conditional-advance: a gapped explicit
		// run stamps neither field and emits a per-repo stderr notice.
		if !ra.DryRun {
			if explicit && cursor != "" {
				gap, cursorKnown := countGap(commits, cursor, window.Start)
				if cursorKnown && gap > 0 {
					fmt.Fprint(os.Stderr, gapNoticeLine(p, gap))
					continue
				}
			}
			newCursor := ""
			if len(commits) > 0 {
				newCursor = commits[0].Hash
			}
			_ = reg.StampRecapAndSave(p, newCursor, now)
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
