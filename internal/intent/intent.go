package intent

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

// repoFrom extracts the *repo.Repo from a store's Committer.
// Used by replay functions that need repo-level operations (e.g. SetConfig).
func repoFrom(store *issue.Store) *repo.Repo {
	return store.Committer.(*repo.Repo)
}

// Replay executes a list of intent strings against the current state.
// Each intent is a structured commit message like "create bw-a1b2 p1 task \"title\"".
// Returns a list of errors for intents that failed (non-fatal).
func Replay(store *issue.Store, intents []string) []error {
	var errors []error
	for _, raw := range intents {
		if err := replayOne(store, raw); err != nil {
			errors = append(errors, fmt.Errorf("replay %q: %w", raw, err))
		}
	}
	return errors
}

func replayOne(store *issue.Store, raw string) error {
	parts := parseIntent(raw)
	if len(parts) == 0 {
		return nil // skip empty or unparseable
	}

	verb := parts[0]
	switch verb {
	case "create":
		return replayCreate(store, parts[1:], raw)
	case "close":
		return replayClose(store, parts[1:], raw)
	case "reopen":
		return replayReopen(store, parts[1:], raw)
	case "update":
		return replayUpdate(store, parts[1:], raw)
	case "link":
		return replayLink(store, parts[1:], raw)
	case "unlink":
		return replayUnlink(store, parts[1:], raw)
	case "label":
		return replayLabel(store, parts[1:], raw)
	case "delete":
		return replayDelete(store, parts[1:], raw)
	case "config":
		return replayConfig(store, parts[1:], raw)
	case "comment":
		return replayComment(store, parts[1:], raw)
	case "init":
		return nil // skip init intents
	default:
		return nil // unknown intent, skip
	}
}

func replayCreate(store *issue.Store, parts []string, raw string) error {
	// create <id> p<n> <type> "<title>"
	if len(parts) < 4 {
		return fmt.Errorf("malformed create intent")
	}

	var priority *int
	if strings.HasPrefix(parts[1], "p") {
		var p int
		fmt.Sscanf(parts[1], "p%d", &p)
		priority = &p
	}
	issueType := parts[2]

	// Extract quoted title
	title := extractQuoted(raw)
	if title == "" {
		title = strings.Join(parts[3:], " ")
	}

	opts := issue.CreateOpts{
		Priority: priority,
		Type:     issueType,
	}
	_, err := store.Create(title, opts)
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayClose(store *issue.Store, parts []string, raw string) error {
	if len(parts) < 1 {
		return fmt.Errorf("malformed close intent")
	}
	_, err := store.Close(parts[0], "")
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayReopen(store *issue.Store, parts []string, raw string) error {
	if len(parts) < 1 {
		return fmt.Errorf("malformed reopen intent")
	}
	_, err := store.Reopen(parts[0])
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayUpdate(store *issue.Store, parts []string, raw string) error {
	// update <id> key=value key=value ...
	if len(parts) < 2 {
		return fmt.Errorf("malformed update intent")
	}
	id := parts[0]
	opts := issue.UpdateOpts{}

	for _, kv := range parts[1:] {
		eqIdx := strings.Index(kv, "=")
		if eqIdx == -1 {
			continue
		}
		key := kv[:eqIdx]
		val := kv[eqIdx+1:]

		switch key {
		case "status":
			opts.Status = &val
		case "assignee":
			opts.Assignee = &val
		case "priority":
			var p int
			fmt.Sscanf(val, "%d", &p)
			opts.Priority = &p
		case "type":
			opts.Type = &val
		case "title":
			opts.Title = &val
		case "parent":
			opts.Parent = &val
		}
	}

	_, err := store.Update(id, opts)
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayLink(store *issue.Store, parts []string, raw string) error {
	// link <id1> blocks <id2>
	if len(parts) < 3 || parts[1] != "blocks" {
		return fmt.Errorf("malformed link intent")
	}
	if err := store.Link(parts[0], parts[2]); err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayUnlink(store *issue.Store, parts []string, raw string) error {
	// unlink <id1> blocks <id2>
	if len(parts) < 3 || parts[1] != "blocks" {
		return fmt.Errorf("malformed unlink intent")
	}
	if err := store.Unlink(parts[0], parts[2]); err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayLabel(store *issue.Store, parts []string, raw string) error {
	// label <id> +bug +frontend -wontfix
	if len(parts) < 2 {
		return fmt.Errorf("malformed label intent")
	}
	id := parts[0]
	var add, remove []string
	for _, arg := range parts[1:] {
		if strings.HasPrefix(arg, "+") {
			add = append(add, strings.TrimPrefix(arg, "+"))
		} else if strings.HasPrefix(arg, "-") {
			remove = append(remove, strings.TrimPrefix(arg, "-"))
		}
	}
	_, err := store.Label(id, add, remove)
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayDelete(store *issue.Store, parts []string, raw string) error {
	if len(parts) < 1 {
		return fmt.Errorf("malformed delete intent")
	}
	_, err := store.Delete(parts[0])
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayConfig(store *issue.Store, parts []string, raw string) error {
	// config key=value
	if len(parts) < 1 {
		return fmt.Errorf("malformed config intent")
	}
	kv := parts[0]
	eqIdx := strings.Index(kv, "=")
	if eqIdx == -1 {
		return fmt.Errorf("malformed config intent: missing '='")
	}
	key := kv[:eqIdx]
	value := kv[eqIdx+1:]
	r := repoFrom(store)
	if err := r.SetConfig(key, value); err != nil {
		return err
	}
	return store.Commit(raw)
}

func replayComment(store *issue.Store, parts []string, raw string) error {
	if len(parts) < 1 {
		return fmt.Errorf("malformed comment intent")
	}
	text := extractQuoted(raw)
	if text == "" && len(parts) > 1 {
		text = strings.Join(parts[1:], " ")
	}
	_, err := store.Comment(parts[0], text, "")
	if err != nil {
		return err
	}
	return store.Commit(raw)
}

// parseIntent splits an intent string respecting quoted strings.
func parseIntent(raw string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for _, ch := range raw {
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == ' ' && !inQuote {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// extractQuoted extracts the first quoted string from a raw intent.
func extractQuoted(raw string) string {
	start := strings.Index(raw, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(raw[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return raw[start+1 : start+1+end]
}
