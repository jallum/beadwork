package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
)

type ArchiveArgs struct {
	IDs    []string
	Before string
	Close  bool
	Detach bool
	Force  bool
	JSON   bool
}

func parseArchiveArgs(raw []string) (ArchiveArgs, error) {
	a, err := ParseArgs(raw,
		[]string{"--before"},
		[]string{"--close", "--detach", "--force", "-y", "--json"},
	)
	if err != nil {
		return ArchiveArgs{}, err
	}
	args := ArchiveArgs{
		IDs:    a.Pos(),
		Before: a.String("--before"),
		Close:  a.Bool("--close"),
		Detach: a.Bool("--detach"),
		Force:  a.Bool("--force") || a.Bool("-y"),
		JSON:   a.JSON(),
	}
	if args.Before != "" && len(args.IDs) > 0 {
		return args, fmt.Errorf("cannot combine --before with explicit issue IDs")
	}
	if args.Before == "" && len(args.IDs) == 0 {
		return args, fmt.Errorf("usage: bw archive <id>... | bw archive --before <date> [--force]")
	}
	return args, nil
}

func cmdArchive(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	aa, err := parseArchiveArgs(args)
	if err != nil {
		return nil, err
	}
	if aa.Before != "" {
		return nil, runArchiveBefore(store, aa, w)
	}
	return nil, runArchiveIDs(store, aa, w)
}

// runArchiveIDs archives explicitly named issues. Named issues are an explicit
// instruction, so this executes directly (no preview).
func runArchiveIDs(store *issue.Store, aa ArchiveArgs, w Writer) error {
	opts := issue.ArchiveOpts{Close: aa.Close, Detach: aa.Detach}

	var archived []*issue.Issue
	err := commitWithRetry(store, commitMaxRetries, func() (string, error) {
		archived = archived[:0]
		var lines []string
		for _, id := range aa.IDs {
			iss, e := store.Archive(id, opts)
			if e != nil {
				return "", e
			}
			archived = append(archived, iss)
			lines = append(lines, archiveIntentLine(iss.ID, aa))
		}
		return strings.Join(lines, "\n"), nil
	})
	if err != nil {
		return enrichArchiveError(err)
	}

	if aa.JSON {
		fprintJSON(w, archived)
		return nil
	}
	for _, iss := range archived {
		fmt.Fprintf(w, "archived {id:%s}: %s\n", iss.ID, md.Escape(iss.Title))
	}
	return nil
}

// runArchiveBefore sweeps closed issues whose closed_at precedes a cutoff.
// It previews by default; --force commits. Issues with open work still attached
// are skipped unless --detach is given.
func runArchiveBefore(store *issue.Store, aa ArchiveArgs, w Writer) error {
	resolved, err := resolveDateBeforeNow(aa.Before, store.Now())
	if err != nil {
		return err
	}
	cutoff, err := parseCutoffTime(resolved)
	if err != nil {
		return err
	}

	candidates, err := store.ClosedBefore(cutoff)
	if err != nil {
		return err
	}

	// Partition into cleanly-archivable vs. blocked-by-open-work.
	type skip struct {
		iss  *issue.Issue
		deps []string
		kids []string
	}
	var ready []*issue.Issue
	var skipped []skip
	for _, iss := range candidates {
		deps, kids, e := store.ArchiveCheck(iss.ID)
		if e != nil {
			continue
		}
		if (len(deps) > 0 || len(kids) > 0) && !aa.Detach {
			skipped = append(skipped, skip{iss, deps, kids})
			continue
		}
		ready = append(ready, iss)
	}

	if len(candidates) == 0 {
		fmt.Fprintf(w, "no closed issues found before %s\n", resolved)
		return nil
	}

	// Preview mode (default): report, mutate nothing.
	if !aa.Force {
		fmt.Fprintf(w, "Would archive %d issue(s) closed before %s:\n", len(ready), resolved)
		w.Push(2)
		for _, iss := range ready {
			fmt.Fprintf(w, "{id:%s}: %s\n", iss.ID, md.Escape(iss.Title))
		}
		w.Pop()
		if len(skipped) > 0 {
			fmt.Fprintf(w, "\nSkipped %d with open work attached (use --detach to include):\n", len(skipped))
			w.Push(2)
			for _, s := range skipped {
				fmt.Fprintf(w, "{id:%s}: %s — %s\n", s.iss.ID, md.Escape(s.iss.Title), attachmentSummary(s.deps, s.kids))
			}
			w.Pop()
		}
		fmt.Fprintln(w, "\nRe-run with --force to archive.")
		return nil
	}

	if len(ready) == 0 {
		fmt.Fprintln(w, "nothing to archive")
		if len(skipped) > 0 {
			fmt.Fprintf(w, "(%d skipped; use --detach to include them)\n", len(skipped))
		}
		return nil
	}

	opts := issue.ArchiveOpts{Detach: aa.Detach}
	readyIDs := make([]string, len(ready))
	for i, iss := range ready {
		readyIDs[i] = iss.ID
	}

	var archived []*issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		archived = archived[:0]
		var lines []string
		for _, id := range readyIDs {
			iss, e := store.Archive(id, opts)
			if e != nil {
				return "", e
			}
			archived = append(archived, iss)
			lines = append(lines, archiveIntentLine(iss.ID, aa))
		}
		return strings.Join(lines, "\n"), nil
	})
	if err != nil {
		return enrichArchiveError(err)
	}

	if aa.JSON {
		fprintJSON(w, archived)
		return nil
	}
	fmt.Fprintf(w, "archived %d issue(s) closed before %s\n", len(archived), resolved)
	w.Push(2)
	for _, iss := range archived {
		fmt.Fprintf(w, "{id:%s}: %s\n", iss.ID, md.Escape(iss.Title))
	}
	w.Pop()
	if len(skipped) > 0 {
		fmt.Fprintf(w, "(%d skipped; use --detach to include them)\n", len(skipped))
	}
	return nil
}

// archiveIntentLine builds the replayable intent for one archived issue,
// echoing the flags that governed the operation.
func archiveIntentLine(id string, aa ArchiveArgs) string {
	line := "archive " + id
	if aa.Close {
		line += " --close"
	}
	if aa.Detach {
		line += " --detach"
	}
	return line
}

func attachmentSummary(deps, kids []string) string {
	var parts []string
	if len(deps) > 0 {
		parts = append(parts, "open dependents: "+strings.Join(deps, ", "))
	}
	if len(kids) > 0 {
		parts = append(parts, "open children: "+strings.Join(kids, ", "))
	}
	return strings.Join(parts, "; ")
}

// enrichArchiveError adds an actionable hint to the not-closed case. The
// blocked-by-open-work case already carries a --detach hint in its message.
func enrichArchiveError(err error) error {
	var nce *issue.NotClosedError
	if errors.As(err, &nce) {
		return fmt.Errorf("%w; pass --close to close it first, or close it with `bw close %s`", err, nce.ID)
	}
	return err
}

// parseCutoffTime turns a resolved date string (YYYY-MM-DD or RFC3339, as
// produced by resolveDateBeforeNow) into a comparable instant. Date-only
// cutoffs are interpreted as midnight UTC.
func parseCutoffTime(resolved string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, resolved); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", resolved); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("could not parse cutoff %q", resolved)
}
