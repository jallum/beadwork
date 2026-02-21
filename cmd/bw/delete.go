package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

type DeleteArgs struct {
	ID    string
	Force bool
	JSON  bool
}

func parseDeleteArgs(raw []string) (DeleteArgs, error) {
	if len(raw) == 0 {
		return DeleteArgs{}, fmt.Errorf("usage: bw delete <id> [--force] [--json]")
	}
	a, err := ParseArgs(raw[1:],
		nil,
		[]string{"--force", "--json"},
	)
	if err != nil {
		return DeleteArgs{}, err
	}
	return DeleteArgs{
		ID:    raw[0],
		Force: a.Bool("--force"),
		JSON:  a.JSON(),
	}, nil
}

func cmdDelete(r *repo.Repo, store *issue.Store, args []string, w Writer) error {
	da, err := parseDeleteArgs(args)
	if err != nil {
		return err
	}

	if !da.Force {
		// Preview mode
		plan, err := store.DeletePreview(da.ID)
		if err != nil {
			return err
		}
		if da.JSON {
			fprintJSON(w, plan)
			return nil
		}
		fmt.Fprintln(w, w.Style("DELETE PREVIEW", Bold))
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Issue to delete:\n  %s: %s [P%d %s]\n",
			w.Style(plan.Issue.ID, Cyan), plan.Issue.Title, plan.Issue.Priority, plan.Issue.Type)
		if len(plan.Blocks) > 0 || len(plan.BlockedBy) > 0 {
			total := len(plan.Blocks) + len(plan.BlockedBy)
			fmt.Fprintf(w, "\nDependency links to remove: %d\n", total)
			for _, id := range plan.Blocks {
				fmt.Fprintf(w, "  %s blocks %s\n", w.Style(plan.Issue.ID, Cyan), w.Style(id, Cyan))
			}
			for _, id := range plan.BlockedBy {
				fmt.Fprintf(w, "  %s blocked by %s\n", w.Style(plan.Issue.ID, Cyan), w.Style(id, Cyan))
			}
		}
		if len(plan.Children) > 0 {
			fmt.Fprintf(w, "\nChildren to orphan: %d\n", len(plan.Children))
			for _, id := range plan.Children {
				fmt.Fprintf(w, "  %s\n", w.Style(id, Cyan))
			}
		}
		fmt.Fprintf(w, "\nTo proceed: %s\n", w.Style(fmt.Sprintf("bw delete %s --force", plan.Issue.ID), Dim))
		return nil
	}

	// Force mode — actually delete
	iss, err := store.Delete(da.ID)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("delete %s", iss.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if da.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "deleted %s: %s\n", w.Style(iss.ID, Cyan), iss.Title)
	}
	return nil
}
