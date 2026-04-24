package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

// HistoryArgs holds parsed arguments for the history command.
type HistoryArgs struct {
	ID    string
	Limit int
	JSON  bool
}

func parseHistoryArgs(raw []string) (HistoryArgs, error) {
	if len(raw) == 0 {
		return HistoryArgs{}, fmt.Errorf("usage: bw history <id> [--limit N] [--json]")
	}
	a, err := ParseArgs(raw[1:],
		[]string{"--limit"},
		[]string{"--json"},
	)
	if err != nil {
		return HistoryArgs{}, err
	}
	ha := HistoryArgs{
		ID:   raw[0],
		JSON: a.JSON(),
	}
	if a.Has("--limit") {
		ha.Limit = a.Int("--limit")
	}
	return ha, nil
}

type commitEntry struct {
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
	Author    string `json:"author"`
	Intent    string `json:"intent"`
}

func cmdHistory(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ha, err := parseHistoryArgs(args)
	if err != nil {
		return nil, err
	}

	// Resolve to full ID
	iss, err := store.Get(ha.ID)
	if err != nil {
		return nil, err
	}

	r := store.Committer.(*repo.Repo)
	commits, err := r.AllCommits()
	if err != nil {
		return nil, fmt.Errorf("reading history: %w", err)
	}

	// Filter for this issue and reverse to chronological (oldest first)
	var matched []commitEntry
	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]
		if strings.Contains(c.Message, iss.ID) {
			matched = append(matched, commitEntry{
				Hash:      c.Hash,
				Timestamp: c.Time.UTC().Format("2006-01-02 15:04"),
				Author:    c.Author,
				Intent:    c.Message,
			})
		}
	}

	// Apply limit (keep the most recent N entries)
	if ha.Limit > 0 && len(matched) > ha.Limit {
		matched = matched[len(matched)-ha.Limit:]
	}

	if ha.JSON {
		fprintJSON(w, matched)
		return nil, nil
	}

	if len(matched) == 0 {
		fmt.Fprintln(w, "no history found")
		return nil, nil
	}

	for _, e := range matched {
		fmt.Fprintf(w, "%s  %s  %s\n", e.Timestamp, e.Author, e.Intent)
	}
	return nil, nil
}
