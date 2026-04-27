package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

type UpdateArgs struct {
	ID          string
	Title       string
	TitleSet    bool
	Description string
	DescSet     bool
	Priority    *int
	Assignee    string
	AssigneeSet bool
	Type        string
	TypeSet     bool
	Status      string
	StatusSet   bool
	DeferUntil  string
	DeferSet    bool
	Due         string
	DueSet      bool
	Parent      string
	ParentSet   bool
	JSON        bool
}

func parseUpdateArgs(raw []string) (UpdateArgs, error) {
	if len(raw) == 0 {
		return UpdateArgs{}, fmt.Errorf("usage: bw update <id> [flags]")
	}
	a, err := ParseArgs(raw[1:],
		[]string{"--title", "--description", "--priority", "--assignee", "--type", "--status", "--defer", "--due", "--parent"},
		[]string{"--json"},
	)
	if err != nil {
		return UpdateArgs{}, err
	}
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
		p, err := parsePriority(a.String("--priority"))
		if err != nil {
			return ua, err
		}
		ua.Priority = &p
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
	}
	if a.Has("--due") {
		ua.Due = a.String("--due")
		ua.DueSet = true
	}
	if a.Has("--parent") {
		ua.Parent = a.String("--parent")
		ua.ParentSet = true
	}
	return ua, nil
}

func cmdUpdate(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ua, err := parseUpdateArgs(args)
	if err != nil {
		return nil, err
	}

	now := store.Now()
	if ua.DeferSet && ua.DeferUntil != "" {
		resolved, err := resolveDate(ua.DeferUntil, now)
		if err != nil {
			return nil, err
		}
		ua.DeferUntil = resolved
	}
	if ua.DueSet && ua.Due != "" {
		resolved, err := resolveDate(ua.Due, now)
		if err != nil {
			return nil, err
		}
		ua.Due = resolved
	}

	opts := issue.UpdateOpts{}
	var changes []string

	if ua.TitleSet {
		opts.Title = &ua.Title
		changes = append(changes, fmt.Sprintf("title=%q", ua.Title))
	}
	if ua.DescSet {
		opts.Description = &ua.Description
		changes = append(changes, fmt.Sprintf("description=%q", ua.Description))
	}
	if ua.Priority != nil {
		opts.Priority = ua.Priority
		changes = append(changes, fmt.Sprintf("priority=%d", *ua.Priority))
	}
	if ua.AssigneeSet {
		opts.Assignee = &ua.Assignee
		changes = append(changes, fmt.Sprintf("assignee=%q", ua.Assignee))
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
	if ua.DueSet {
		opts.Due = &ua.Due
		changes = append(changes, "due="+ua.Due)
	}
	if ua.ParentSet {
		opts.Parent = &ua.Parent
		changes = append(changes, "parent="+ua.Parent)
	}

	iss, err := store.Update(ua.ID, opts)
	if err != nil {
		return nil, err
	}

	intent := fmt.Sprintf("update %s %s", iss.ID, strings.Join(changes, " "))
	if err := store.Commit(intent); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	if ua.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "updated %s\n", iss.ID)
	}
	return nil, nil
}
