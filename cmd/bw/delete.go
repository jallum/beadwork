package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
)

type DeleteArgs struct {
	ID        string
	Recursive bool
	Force     bool
	JSON      bool
}

func parseDeleteArgs(raw []string) (DeleteArgs, error) {
	if len(raw) == 0 {
		return DeleteArgs{}, fmt.Errorf("usage: bw delete <id> [--recursive] [--force] [--json]")
	}
	a, err := ParseArgs(raw[1:],
		nil,
		[]string{"--recursive", "--force", "--json"},
	)
	if err != nil {
		return DeleteArgs{}, err
	}
	return DeleteArgs{
		ID:        raw[0],
		Recursive: a.Bool("--recursive"),
		Force:     a.Bool("--force"),
		JSON:      a.JSON(),
	}, nil
}

func cmdDelete(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	da, err := parseDeleteArgs(args)
	if err != nil {
		return nil, err
	}

	if da.Recursive {
		return cmdDeleteRecursive(store, da, w)
	}

	if !da.Force {
		// Preview mode
		plan, err := store.DeletePreview(da.ID)
		if err != nil {
			return nil, err
		}
		if da.JSON {
			fprintJSON(w, plan)
			return nil, nil
		}
		fmt.Fprintln(w, sectionHeader(w, "DELETE PREVIEW"))
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Issue to delete:")
		w.Push(2)
		fmt.Fprintf(w, "%s: %s [P%d %s]\n",
			w.Style(plan.Issue.ID, Cyan), plan.Issue.Title, plan.Issue.Priority, plan.Issue.Type)
		w.Pop()
		if len(plan.Blocks) > 0 || len(plan.BlockedBy) > 0 {
			total := len(plan.Blocks) + len(plan.BlockedBy)
			fmt.Fprintf(w, "\nDependency links to remove: %d\n", total)
			w.Push(2)
			for _, id := range plan.Blocks {
				fmt.Fprintf(w, "%s blocks %s\n", w.Style(plan.Issue.ID, Cyan), w.Style(id, Cyan))
			}
			for _, id := range plan.BlockedBy {
				fmt.Fprintf(w, "%s blocked by %s\n", w.Style(plan.Issue.ID, Cyan), w.Style(id, Cyan))
			}
			w.Pop()
		}
		if len(plan.Children) > 0 {
			fmt.Fprintf(w, "\nChildren to orphan: %d\n", len(plan.Children))
			w.Push(2)
			for _, id := range plan.Children {
				fmt.Fprintf(w, "%s\n", w.Style(id, Cyan))
			}
			w.Pop()
		}
		fmt.Fprintf(w, "\nTo proceed: %s\n", w.Style(fmt.Sprintf("bw delete %s --force", plan.Issue.ID), Dim))
		return nil, nil
	}

	// Force mode — actually delete
	var iss *issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var derr error
		iss, derr = store.Delete(da.ID)
		if derr != nil {
			return "", derr
		}
		return fmt.Sprintf("delete %s", iss.ID), nil
	})
	if err != nil {
		return nil, err
	}

	if da.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "deleted %s: %s\n", w.Style(iss.ID, Cyan), iss.Title)
	}
	return nil, nil
}

// cmdDeleteRecursive deletes an issue and its entire subtree. Like the
// non-recursive path it previews by default; --force commits the removal in a
// single commit (one `delete <id>` intent line per removed issue, leaf-up).
func cmdDeleteRecursive(store *issue.Store, da DeleteArgs, w Writer) (*config.Config, error) {
	if !da.Force {
		plan, err := store.DeleteSubtreePreview(da.ID)
		if err != nil {
			return nil, err
		}
		if da.JSON {
			fprintJSON(w, plan)
			return nil, nil
		}
		fmt.Fprintln(w, sectionHeader(w, "DELETE PREVIEW"))
		fmt.Fprintf(w, "\nIssues to delete (subtree): %d\n", len(plan.Deleted))
		w.Push(2)
		for _, iss := range plan.Deleted {
			fmt.Fprintf(w, "%s: %s\n", w.Style(iss.ID, Cyan), iss.Title)
		}
		w.Pop()
		if len(plan.Unblocked) > 0 {
			fmt.Fprintf(w, "\nWould unblock: %d\n", len(plan.Unblocked))
			w.Push(2)
			for _, u := range plan.Unblocked {
				fmt.Fprintf(w, "%s: %s\n", w.Style(u.ID, Cyan), u.Title)
			}
			w.Pop()
		}
		fmt.Fprintf(w, "\nTo proceed: %s\n", w.Style(fmt.Sprintf("bw delete %s --recursive --force", da.ID), Dim))
		return nil, nil
	}

	var result *issue.SubtreeDeleteResult
	err := commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var derr error
		result, derr = store.DeleteSubtree(da.ID)
		if derr != nil {
			return "", derr
		}
		if len(result.Deleted) == 0 {
			return "", fmt.Errorf("nothing to delete: %s", da.ID)
		}
		lines := make([]string, len(result.Deleted))
		for i, iss := range result.Deleted {
			lines[i] = fmt.Sprintf("delete %s", iss.ID)
		}
		return strings.Join(lines, "\n"), nil
	})
	if err != nil {
		return nil, err
	}

	if da.JSON {
		if result.Deleted == nil {
			result.Deleted = []*issue.Issue{}
		}
		if result.Unblocked == nil {
			result.Unblocked = []*issue.Issue{}
		}
		fprintJSON(w, result)
		return nil, nil
	}

	fmt.Fprintf(w, "deleted %d issue(s) under {id:%s}:\n", len(result.Deleted), da.ID)
	w.Push(2)
	for _, iss := range result.Deleted {
		fmt.Fprintf(w, "{id:%s}: ~~%s~~\n", iss.ID, md.Escape(iss.Title))
	}
	w.Pop()
	if len(result.Unblocked) > 0 {
		fmt.Fprintln(w)
		w.Push(2)
		for _, u := range result.Unblocked {
			fmt.Fprintf(w, "unblocked {id:%s}: %s\n", u.ID, md.Escape(u.Title))
		}
		w.Pop()
		fmt.Fprintln(w)
		if len(result.Unblocked) == 1 {
			fmt.Fprintf(w, "Next: `bw start %s` to begin, or `bw ready` for all options.\n", result.Unblocked[0].ID)
		} else {
			fmt.Fprintln(w, "Next: `bw ready` to see available work.")
		}
	}
	return nil, nil
}
