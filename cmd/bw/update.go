package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

type UpdateArgs struct {
	ID          string
	Title       string
	TitleSet    bool
	Description string
	DescSet     bool
	Priority    int
	PrioritySet bool
	Assignee    string
	AssigneeSet bool
	Type        string
	TypeSet     bool
	Status      string
	StatusSet   bool
	DeferUntil  string
	DeferSet    bool
	JSON        bool
}

func parseUpdateArgs(raw []string) (UpdateArgs, error) {
	if len(raw) == 0 {
		return UpdateArgs{}, fmt.Errorf("usage: bw update <id> [flags]")
	}
	a := ParseArgs(raw[1:], "--title", "--description", "--priority", "--assignee", "--type", "--status", "--defer")
	ua := UpdateArgs{ID: raw[0], JSON: a.JSON()}

	if a.Has("--title") {
		ua.Title = a.String("--title")
		ua.TitleSet = true
	}
	if a.Has("--description") {
		ua.Description = a.String("--description")
		ua.DescSet = true
	}
	if a.Has("--priority") {
		p, _, err := a.IntErr("--priority")
		if err != nil {
			return ua, err
		}
		ua.Priority = p
		ua.PrioritySet = true
	}
	if a.Has("--assignee") {
		ua.Assignee = a.String("--assignee")
		ua.AssigneeSet = true
	}
	if a.Has("--type") {
		ua.Type = a.String("--type")
		ua.TypeSet = true
	}
	if a.Has("--status") {
		ua.Status = a.String("--status")
		ua.StatusSet = true
	}
	if a.Has("--defer") {
		ua.DeferUntil = a.String("--defer")
		ua.DeferSet = true
		if err := validateDate(ua.DeferUntil); err != nil {
			return ua, err
		}
	}
	return ua, nil
}

func cmdUpdate(args []string, w io.Writer) error {
	ua, err := parseUpdateArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	opts := issue.UpdateOpts{}
	var changes []string

	if ua.TitleSet {
		opts.Title = &ua.Title
		changes = append(changes, "title="+ua.Title)
	}
	if ua.DescSet {
		opts.Description = &ua.Description
		changes = append(changes, "description=...")
	}
	if ua.PrioritySet {
		opts.Priority = &ua.Priority
		changes = append(changes, fmt.Sprintf("priority=%d", ua.Priority))
	}
	if ua.AssigneeSet {
		opts.Assignee = &ua.Assignee
		changes = append(changes, "assignee="+ua.Assignee)
	}
	if ua.TypeSet {
		opts.Type = &ua.Type
		changes = append(changes, "type="+ua.Type)
	}
	if ua.StatusSet {
		opts.Status = &ua.Status
		changes = append(changes, "status="+ua.Status)
	}
	if ua.DeferSet {
		opts.DeferUntil = &ua.DeferUntil
		status := "deferred"
		opts.Status = &status
		changes = append(changes, "defer="+ua.DeferUntil)
	}

	iss, err := store.Update(ua.ID, opts)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("update %s %s", iss.ID, strings.Join(changes, " "))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ua.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "updated %s\n", iss.ID)
	}
	return nil
}
