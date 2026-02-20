package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

func getRepo() (*repo.Repo, error) {
	return repo.FindRepo()
}

func getInitialized() (*repo.Repo, *issue.Store, error) {
	r, err := getRepo()
	if err != nil {
		return nil, nil, err
	}
	if !r.IsInitialized() {
		return nil, nil, fmt.Errorf("beadwork not initialized. Run: bw init")
	}
	if v := r.Version(); v > repo.CurrentVersion {
		return nil, nil, fmt.Errorf("repo version %d is newer than this binary supports (max %d); run: bw upgrade", v, repo.CurrentVersion)
	} else if v < repo.CurrentVersion {
		return nil, nil, fmt.Errorf("repo version %d needs upgrade (current %d); run: bw upgrade repo", v, repo.CurrentVersion)
	}
	store := issue.NewStore(r.TreeFS(), r.Prefix)
	if val, ok := r.GetConfig("default.priority"); ok {
		if p, err := strconv.Atoi(val); err == nil && p >= 0 {
			store.DefaultPriority = &p
		}
	}
	return r, store, nil
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// aliases maps short flags to their long forms.
var aliases = map[string]string{
	"-p": "--priority",
	"-t": "--type",
	"-a": "--assignee",
	"-d": "--description",
	"-s": "--status",
}

// Args holds parsed command-line arguments separated into boolean flags,
// key-value flags, and positional arguments.
type Args struct {
	bools map[string]bool
	flags map[string]string
	pos   []string
}

// ParseArgs separates raw args into booleans, key-value pairs, and positionals.
// valueFlags lists flags that consume the next token as a value (e.g. "--status").
// boolFlags lists boolean flags (e.g. "--json", "--all").
// Any "--" prefixed token not in valueFlags or boolFlags returns an error.
func ParseArgs(raw []string, valueFlags []string, boolFlags []string) (Args, error) {
	vf := make(map[string]bool, len(valueFlags))
	for _, f := range valueFlags {
		vf[f] = true
	}
	bf := make(map[string]bool, len(boolFlags))
	for _, f := range boolFlags {
		bf[f] = true
	}

	a := Args{
		bools: make(map[string]bool),
		flags: make(map[string]string),
	}

	for i := 0; i < len(raw); i++ {
		tok := raw[i]
		if long, ok := aliases[tok]; ok {
			tok = long
		}

		if !strings.HasPrefix(tok, "--") {
			a.pos = append(a.pos, raw[i])
			continue
		}

		if vf[tok] {
			if i+1 < len(raw) {
				a.flags[tok] = raw[i+1]
				i++
			}
		} else if bf[tok] {
			a.bools[tok] = true
		} else {
			return a, fmt.Errorf("unknown flag: %s", tok)
		}
	}
	return a, nil
}

// Bool returns true if the named boolean flag was present.
func (a Args) Bool(name string) bool { return a.bools[name] }

// JSON is shorthand for Bool("--json").
func (a Args) JSON() bool { return a.bools["--json"] }

// String returns the value of a key-value flag, or "" if absent.
func (a Args) String(name string) string { return a.flags[name] }

// Int returns the parsed int value of a flag, or 0 if absent/invalid.
func (a Args) Int(name string) int {
	v, _ := strconv.Atoi(a.flags[name])
	return v
}

// IntErr returns the parsed int, whether the flag was set, and any parse error.
func (a Args) IntErr(name string) (int, bool, error) {
	v, ok := a.flags[name]
	if !ok {
		return 0, false, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, true, fmt.Errorf("invalid %s: %s", name, v)
	}
	return n, true, nil
}

// Has returns true if a key-value flag was provided.
func (a Args) Has(name string) bool {
	_, ok := a.flags[name]
	return ok
}

// Pos returns all positional arguments.
func (a Args) Pos() []string { return a.pos }

// PosFirst returns the first positional argument, or "" if none.
func (a Args) PosFirst() string {
	if len(a.pos) > 0 {
		return a.pos[0]
	}
	return ""
}

// PosJoined returns all positional args joined with spaces.
func (a Args) PosJoined() string { return strings.Join(a.pos, " ") }

// parsePriority parses a priority value from a string.
// Accepts numeric "0"-"4" or prefixed "P0"-"P4" (case-insensitive).
// Returns the parsed priority and an error if the value is invalid.
func parsePriority(s string) (int, error) {
	v := s
	if len(v) > 0 && (v[0] == 'P' || v[0] == 'p') {
		v = v[1:]
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 4 {
		return 0, fmt.Errorf("invalid priority %q: expected 0-4 or P0-P4", s)
	}
	return n, nil
}

func fprintJSON(w io.Writer, v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(w, string(data))
}

func fprintIssue(w io.Writer, iss *issue.Issue) {
	// Header: ○ bw-f0ae · Title   [● P3 · OPEN]
	fmt.Fprintf(w, "%s %s · %s   [%s P%d · %s]\n",
		issue.StatusIcon(iss.Status),
		iss.ID,
		iss.Title,
		issue.PriorityDot(iss.Priority),
		iss.Priority,
		strings.ToUpper(iss.Status),
	)

	// Metadata line: Assignee · Type
	assignee := iss.Assignee
	if assignee == "" {
		assignee = "—"
	}
	fmt.Fprintf(w, "Assignee: %s · Type: %s\n", assignee, iss.Type)

	// Created date (trim to date only)
	created := iss.Created
	if len(created) >= 10 {
		created = created[:10]
	}
	fmt.Fprintf(w, "Created: %s\n", created)

	if iss.DeferUntil != "" {
		fmt.Fprintf(w, "Deferred: %s\n", iss.DeferUntil)
	}

	// Optional metadata
	if len(iss.Labels) > 0 {
		fmt.Fprintf(w, "Labels: %s\n", strings.Join(iss.Labels, ", "))
	}

	var deps []string
	if len(iss.Blocks) > 0 {
		deps = append(deps, "Blocks: "+strings.Join(iss.Blocks, ", "))
	}
	if len(iss.BlockedBy) > 0 {
		deps = append(deps, "Blocked by: "+strings.Join(iss.BlockedBy, ", "))
	}
	if len(deps) > 0 {
		fmt.Fprintln(w, strings.Join(deps, " · "))
	}
	if iss.Parent != "" {
		fmt.Fprintf(w, "Parent: %s\n", iss.Parent)
	}

	// Description
	if iss.Description != "" {
		fmt.Fprintf(w, "\nDESCRIPTION\n\n")
		for _, line := range strings.Split(iss.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}
}
