package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

func getRepo() (*repo.Repo, error) {
	return repo.FindRepo()
}

func getInitializedRepo() (*repo.Repo, *issue.Store, error) {
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
	if val, ok := r.GetConfig("id.retries"); ok {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			store.IDRetries = n
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
	"-l": "--labels",
	"-g": "--grep",
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

// formatDeps returns a compact inline dependency string for list output.
// Example: " [blocks: bw-abc, bw-def] [blocked by: bw-xyz]"
// Returns "" when there are no dependencies.
// Ticket IDs are styled Red; labels and brackets are styled Dim.
func formatDeps(w Writer, iss *issue.Issue) string {
	var parts []string
	if len(iss.Blocks) > 0 {
		ids := make([]string, len(iss.Blocks))
		for i, id := range iss.Blocks {
			ids[i] = w.Style(id, Red)
		}
		parts = append(parts, w.Style("[blocks: ", Dim)+strings.Join(ids, w.Style(", ", Dim))+w.Style("]", Dim))
	}
	if len(iss.BlockedBy) > 0 {
		ids := make([]string, len(iss.BlockedBy))
		for i, id := range iss.BlockedBy {
			ids[i] = w.Style(id, Red)
		}
		parts = append(parts, w.Style("[blocked by: ", Dim)+strings.Join(ids, w.Style(", ", Dim))+w.Style("]", Dim))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

func fprintJSON(w Writer, v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(w, string(data))
}

func fprintIssue(w Writer, iss *issue.Issue) {
	// Header: ○ bw-f0ae [BUG] · Title   [● P1 · OPEN]
	typeTag := ""
	if iss.Type != "" && iss.Type != "task" {
		typeTag = " [" + strings.ToUpper(iss.Type) + "]"
	}
	ps := PriorityStyle(iss.Priority)
	fmt.Fprintf(w, "%s %s%s · %s   [%s %s · %s]\n",
		issue.StatusIcon(iss.Status),
		w.Style(iss.ID, Cyan),
		typeTag,
		iss.Title,
		w.Style("●", ps),
		w.Style(fmt.Sprintf("P%d", iss.Priority), ps),
		strings.ToUpper(iss.Status),
	)

	// Metadata line: Assignee · Type
	assignee := iss.Assignee
	if assignee == "" {
		assignee = "—"
	}
	fmt.Fprintf(w, "Assignee: %s · Type: %s\n", assignee, iss.Type)

	// Date line: Created · Updated · Deferred
	dateParts := []string{"Created: " + trimDate(iss.Created)}
	if iss.UpdatedAt != "" {
		dateParts = append(dateParts, "Updated: "+trimDate(iss.UpdatedAt))
	}
	if iss.DeferUntil != "" {
		dateParts = append(dateParts, "Deferred: "+iss.DeferUntil)
	}
	fmt.Fprintln(w, strings.Join(dateParts, " · "))

	// Close reason
	if iss.CloseReason != "" {
		fmt.Fprintf(w, "Close reason: %s\n", iss.CloseReason)
	}

	// Optional metadata
	if len(iss.Labels) > 0 {
		fmt.Fprintf(w, "Labels: %s\n", strings.Join(iss.Labels, ", "))
	}

	if iss.Parent != "" {
		fmt.Fprintf(w, "Parent: %s\n", iss.Parent)
	}

	// Description
	if iss.Description != "" {
		fmt.Fprintf(w, "\n%s\n\n", w.Style("DESCRIPTION", Bold))
		for _, line := range strings.Split(iss.Description, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

}

// fprintComments renders the COMMENTS section for an issue.
func fprintComments(w Writer, iss *issue.Issue) {
	if len(iss.Comments) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, w.Style("COMMENTS", Bold))
		for _, c := range iss.Comments {
			ts := trimDate(c.Timestamp)
			if c.Author != "" {
				fmt.Fprintf(w, "  %s %s\n", w.Style(ts, Dim), w.Style(c.Author, Bold))
			} else {
				fmt.Fprintf(w, "  %s\n", w.Style(ts, Dim))
			}
			fmt.Fprintf(w, "    %s\n", c.Text)
		}
	}
}

// fprintDeps renders rich dependency sections using store lookups.
func fprintDeps(w Writer, iss *issue.Issue, store *issue.Store) {
	if len(iss.Blocks) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, w.Style("BLOCKS", Bold))
		for _, id := range iss.Blocks {
			dep, err := store.Get(id)
			if err != nil {
				fmt.Fprintf(w, "  ← %s\n", id)
				continue
			}
			ps := PriorityStyle(dep.Priority)
			fmt.Fprintf(w, "  ← %s %s: %s  [%s %s]\n",
				issue.StatusIcon(dep.Status), dep.ID, dep.Title,
				w.Style("●", ps), w.Style(fmt.Sprintf("P%d", dep.Priority), ps))
		}
	}
	if len(iss.BlockedBy) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, w.Style("DEPENDS ON", Bold))
		for _, id := range iss.BlockedBy {
			dep, err := store.Get(id)
			if err != nil {
				fmt.Fprintf(w, "  → %s\n", id)
				continue
			}
			ps := PriorityStyle(dep.Priority)
			fmt.Fprintf(w, "  → %s %s: %s  [%s %s]\n",
				issue.StatusIcon(dep.Status), dep.ID, dep.Title,
				w.Style("●", ps), w.Style(fmt.Sprintf("P%d", dep.Priority), ps))
		}
	}
}

// styleMD adds ANSI color to markdown text without altering it.
// Headings → Bold+Cyan, bullets → Cyan, **bold** → Bold, `code` → Dim+Yellow.
func styleMD(w Writer, text string) string {
	var out strings.Builder
	inFence := false
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			out.WriteByte('\n')
		}
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		switch {
		case inFence && strings.HasPrefix(strings.TrimSpace(line), "#"):
			out.WriteString(w.Style(line, Green))
		case inFence:
			out.WriteString(line)
		case strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## "):
			out.WriteString(w.Style(line, Bold, Cyan))
		case strings.HasPrefix(line, "- "):
			out.WriteString(w.Style("- ", Cyan) + styleInline(w, line[2:]))
		default:
			out.WriteString(styleInline(w, line))
		}
	}
	return out.String()
}

// styleInline colors **bold** and `code` spans in place.
func styleInline(w Writer, line string) string {
	var out strings.Builder
	i := 0
	for i < len(line) {
		if i+1 < len(line) && line[i] == '*' && line[i+1] == '*' {
			end := strings.Index(line[i+2:], "**")
			if end >= 0 {
				out.WriteString(w.Style(line[i:i+2+end+2], Bold))
				i += 2 + end + 2
				continue
			}
		}
		if line[i] == '`' {
			end := strings.IndexByte(line[i+1:], '`')
			if end >= 0 {
				out.WriteString(w.Style(line[i:i+1+end+1], Dim, Yellow))
				i += 1 + end + 1
				continue
			}
		}
		out.WriteByte(line[i])
		i++
	}
	return out.String()
}

func trimDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// relativeTime returns a human-readable relative time like "2h ago".
func relativeTime(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return relativeTimeSince(t, time.Now())
}

// relativeTimeSince computes relative time between t and now.
func relativeTimeSince(t, now time.Time) string {
	d := now.Sub(t)
	if d < time.Minute {
		return "just now"
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		mo := int(d.Hours() / 24 / 30)
		if mo < 1 {
			mo = 1
		}
		return fmt.Sprintf("%dmo ago", mo)
	}
}
