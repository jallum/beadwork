package main

import (
	"fmt"
)

// Flag describes a single command-line flag.
type Flag struct {
	Long  string // e.g. "--priority"
	Short string // e.g. "-p" (optional)
	Value string // metavar for help, e.g. "N" — empty means boolean
	Help  string // e.g. "Priority (0-4 or P0-P4, 0=highest)"
}

// Positional describes a positional argument.
type Positional struct {
	Name     string // e.g. "<id>", "<title...>"
	Required bool
	Help     string
}

// Example describes a usage example shown in per-command help.
type Example struct {
	Cmd  string // e.g. "bw graph --all"
	Help string // e.g. "Show all open issues"
}

// Command describes a CLI subcommand.
type Command struct {
	Name        string
	Aliases     []string // alternative names (e.g., "view" for "show")
	Summary     string   // one-line description for top-level usage
	Description string   // multi-line, shown in per-command help (falls back to Summary)
	Positionals []Positional
	Flags       []Flag
	Examples    []Example
	Run         func(args []string, w Writer) error
}

// valueFlags returns the long names of flags that take a value (non-boolean).
func (c *Command) valueFlags() []string {
	var vf []string
	for _, f := range c.Flags {
		if f.Value != "" {
			vf = append(vf, f.Long)
		}
	}
	return vf
}


// expandAliases replaces short flags with their long equivalents.
func expandAliases(raw []string, flags []Flag) []string {
	shorts := make(map[string]string, len(flags))
	for _, f := range flags {
		if f.Short != "" {
			shorts[f.Short] = f.Long
		}
	}
	result := make([]string, len(raw))
	for i, tok := range raw {
		if long, ok := shorts[tok]; ok {
			result[i] = long
		} else {
			result[i] = tok
		}
	}
	return result
}

// commands defines all CLI subcommands.
var commands = []Command{
	{
		Name:        "create",
		Summary:     "Create an issue",
		Description: "Create a new issue. Multiple words are joined into the title.",
		Positionals: []Positional{
			{Name: "<title>", Required: true, Help: "Issue title (multiple words joined)"},
		},
		Flags: []Flag{
			{Long: "--priority", Short: "-p", Value: "N", Help: "Priority (0-4 or P0-P4, 0=highest)"},
			{Long: "--type", Short: "-t", Value: "TYPE", Help: "Issue type (task, bug, etc.)"},
			{Long: "--assignee", Short: "-a", Value: "WHO", Help: "Assignee"},
			{Long: "--description", Short: "-d", Value: "TEXT", Help: "Description"},
			{Long: "--defer", Value: "DATE", Help: "Defer until date (YYYY-MM-DD)"},
			{Long: "--parent", Value: "ID", Help: "Parent issue ID"},
			{Long: "--json", Help: "Output as JSON"},
			{Long: "--silent", Help: "Output bare issue ID only"},
		},
		Examples: []Example{
			{Cmd: `bw create "Fix login bug" --priority 1 --type bug`},
			{Cmd: `bw create "New feature" -a alice`},
			{Cmd: `bw create "Q3 planning" --defer 2027-07-01`},
			{Cmd: `bw create "Fix bug" --silent`, Help: "Output bare ID for scripting"},
		},
		Run: cmdCreate,
	},
	{
		Name:        "show",
		Aliases:     []string{"view"},
		Summary:     "Show issue details",
		Description: "Display full details for an issue including status, priority, labels, and dependency context.\nThe BLOCKED BY section shows actionable tips — the leaf issues that need work to unblock this one.\nThe BLOCKS section shows what completing this issue would eventually unblock.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID (can specify multiple)"},
		},
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
			{Long: "--only", Value: "SECTIONS", Help: "Show only named sections (comma-separated: summary,description,blockedby,blocks,comments,labels,parent)"},
		},
		Examples: []Example{
			{Cmd: "bw show bw-a3f8"},
			{Cmd: "bw show bw-a3f8 bw-b2c1"},
			{Cmd: "bw show bw-a3f8 --only summary", Help: "Compact one-line summary"},
			{Cmd: "bw show bw-a3f8 --only blockedby,blocks", Help: "Dependency context only"},
		},
		Run: cmdShow,
	},
	{
		Name:        "list",
		Summary:     "List issues",
		Description: "List issues matching filters. Defaults to open and in-progress issues, limit 10.",
		Flags: []Flag{
			{Long: "--status", Short: "-s", Value: "STATUS", Help: "Filter by status"},
			{Long: "--assignee", Short: "-a", Value: "WHO", Help: "Filter by assignee"},
			{Long: "--priority", Short: "-p", Value: "N", Help: "Priority (0-4 or P0-P4, 0=highest)"},
			{Long: "--type", Short: "-t", Value: "TYPE", Help: "Filter by type"},
			{Long: "--label", Value: "LABEL", Help: "Filter by label"},
			{Long: "--grep", Short: "-g", Value: "TEXT", Help: "Search title and description"},
			{Long: "--limit", Value: "N", Help: "Max results (default 10)"},
			{Long: "--all", Help: "Show all issues (no status/limit filter)"},
			{Long: "--deferred", Help: "Show only deferred issues"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw list --assignee alice"},
			{Cmd: "bw list --all --type bug"},
			{Cmd: "bw list --status closed --limit 5"},
			{Cmd: "bw list --deferred"},
		},
		Run: cmdList,
	},
	{
		Name:        "update",
		Summary:     "Update an issue",
		Description: "Update fields on an existing issue. Only specified fields change.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
		},
		Flags: []Flag{
			{Long: "--title", Value: "TEXT", Help: "New title"},
			{Long: "--description", Short: "-d", Value: "TEXT", Help: "New description"},
			{Long: "--priority", Short: "-p", Value: "N", Help: "Priority (0-4 or P0-P4, 0=highest)"},
			{Long: "--assignee", Short: "-a", Value: "WHO", Help: "New assignee"},
			{Long: "--type", Short: "-t", Value: "TYPE", Help: "New type"},
			{Long: "--status", Short: "-s", Value: "STATUS", Help: "New status"},
			{Long: "--defer", Value: "DATE", Help: "Defer until date (YYYY-MM-DD)"},
			{Long: "--parent", Value: "ID", Help: "Parent issue ID (empty to clear)"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw update bw-a3f8 --priority 1 --assignee bob"},
			{Cmd: "bw update bw-a3f8 --status in_progress"},
			{Cmd: "bw update bw-a3f8 --defer 2027-06-01"},
		},
		Run: cmdUpdate,
	},
	{
		Name:        "close",
		Summary:     "Close an issue",
		Description: "Close an issue. Optionally provide a reason.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
		},
		Flags: []Flag{
			{Long: "--reason", Value: "REASON", Help: "Closing reason"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw close bw-a3f8"},
			{Cmd: "bw close bw-a3f8 --reason duplicate"},
		},
		Run: cmdClose,
	},
	{
		Name:    "comments",
		Summary: "View or add comments on an issue",
		Description: "List comments or add a new comment.\nSubcommands: add.",
		Positionals: []Positional{
			{Name: "[add] <id> [text]", Required: true, Help: "Issue ID (or 'add <id> <text>' to add)"},
		},
		Flags: []Flag{
			{Long: "--author", Short: "-a", Value: "NAME", Help: "Comment author (add mode)"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: `bw comments bw-a3f8`, Help: "List comments"},
			{Cmd: `bw comments add bw-a3f8 "Fixed in latest deploy"`, Help: "Add a comment"},
		},
		Run: cmdComments,
	},
	{
		Name:    "reopen",
		Summary: "Reopen a closed issue",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
		},
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Run: cmdReopen,
	},
	{
		Name:        "delete",
		Summary:     "Delete an issue",
		Description: "Permanently delete an issue and clean up references.\nWithout --force, shows a preview of what would be affected.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
		},
		Flags: []Flag{
			{Long: "--force", Help: "Actually delete (default: preview only)"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw delete bw-a3f8", Help: "Preview deletion"},
			{Cmd: "bw delete bw-a3f8 --force", Help: "Delete permanently"},
		},
		Run: cmdDelete,
	},
	{
		Name:        "label",
		Summary:     "Add/remove labels",
		Description: "Add or remove labels on an issue. Prefix with + to add, - to remove.\nBare names (without prefix) are added.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
			{Name: "+label [-label]...", Required: true, Help: "Labels to add (+) or remove (-)"},
		},
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw label bw-a3f8 +bug +urgent"},
			{Cmd: "bw label bw-a3f8 -wontfix"},
		},
		Run: cmdLabel,
	},
	{
		Name:        "dep",
		Summary:     "Manage dependencies",
		Description: "Add or remove dependency links between issues.\nSubcommands: add, remove.",
		Positionals: []Positional{
			{Name: "add|remove", Required: true, Help: "Subcommand"},
			{Name: "<id> blocks <id>", Required: true, Help: "Blocker and blocked issue IDs"},
		},
		Examples: []Example{
			{Cmd: "bw dep add bw-1234 blocks bw-5678"},
			{Cmd: "bw dep remove bw-1234 blocks bw-5678"},
		},
		Run: cmdDep,
	},
	{
		Name:    "ready",
		Summary: "List unblocked issues",
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Run: cmdReady,
	},
	{
		Name:    "blocked",
		Summary: "List blocked issues",
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Run: cmdBlocked,
	},
	{
		Name:        "defer",
		Summary:     "Defer an issue until a date",
		Description: "Set an issue's status to deferred with a target date.\nDeferred issues are hidden from ready.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
			{Name: "<date>", Required: true, Help: "Date (YYYY-MM-DD)"},
		},
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw defer bw-a3f8 2027-06-01"},
		},
		Run: cmdDefer,
	},
	{
		Name:        "undefer",
		Summary:     "Restore a deferred issue to open",
		Description: "Restore a deferred issue to open status and clear its defer date.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
		},
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw undefer bw-a3f8"},
		},
		Run: cmdUndefer,
	},
	{
		Name:        "history",
		Summary:     "Show issue history",
		Description: "Show the git commit history for a specific issue.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID"},
		},
		Flags: []Flag{
			{Long: "--limit", Value: "N", Help: "Max entries to show"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw history bw-a3f8"},
			{Cmd: "bw history bw-a3f8 --limit 5"},
		},
		Run: cmdHistory,
	},
	{
		Name:        "graph",
		Summary:     "Dependency graph",
		Description: "Display the dependency graph for an issue or all open issues.\nWith --all, shows all open issues grouped by connected component.",
		Positionals: []Positional{
			{Name: "<id>", Help: "Root issue ID (or use --all)"},
		},
		Flags: []Flag{
			{Long: "--all", Help: "Show all open issues"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: "bw graph bw-a3f8"},
			{Cmd: "bw graph --all"},
		},
		Run: cmdGraph,
	},
	{
		Name:        "sync",
		Summary:     "Fetch, rebase/replay, push",
		Description: "Fetch from remote, rebase local commits, and push.\nUses intent replay to resolve conflicts automatically.",
		Run:         cmdSync,
	},
	{
		Name:        "export",
		Summary:     "Export issues as JSONL",
		Description: "Export issues as JSONL (one JSON object per line).",
		Flags: []Flag{
			{Long: "--status", Short: "-s", Value: "STATUS", Help: "Filter by status"},
		},
		Examples: []Example{
			{Cmd: "bw export --status open"},
		},
		Run: cmdExport,
	},
	{
		Name:        "import",
		Summary:     "Import issues from JSONL",
		Description: "Import issues from a JSONL file. Detects ID collisions and wires dependencies.",
		Positionals: []Positional{
			{Name: "<file>", Required: true, Help: "JSONL file path (use - for stdin)"},
		},
		Flags: []Flag{
			{Long: "--dry-run", Help: "Preview without importing"},
		},
		Examples: []Example{
			{Cmd: "bw import issues.jsonl"},
			{Cmd: "bw import issues.jsonl --dry-run"},
			{Cmd: "bw import - < issues.jsonl"},
		},
		Run: cmdImport,
	},
	{
		Name:        "init",
		Summary:     "Initialize beadwork",
		Description: "Initialize beadwork in the current git repository.\nCreates an orphan branch for issue storage.",
		Flags: []Flag{
			{Long: "--prefix", Value: "PREFIX", Help: "Issue ID prefix"},
			{Long: "--force", Help: "Force reinitialize"},
		},
		Examples: []Example{
			{Cmd: "bw init --prefix myproj"},
			{Cmd: "bw init --force"},
		},
		Run: cmdInit,
	},
	{
		Name:        "config",
		Summary:     "View/set config options",
		Description: "View or modify configuration. Subcommands: get, set, list.",
		Positionals: []Positional{
			{Name: "get|set|list", Required: true, Help: "Subcommand"},
		},
		Examples: []Example{
			{Cmd: "bw config set default.priority 2"},
			{Cmd: "bw config get default.priority"},
			{Cmd: "bw config list"},
		},
		Run: cmdConfig,
	},
	{
		Name:        "upgrade",
		Summary:     "Upgrade binary or repo schema",
		Description: "Upgrade the bw binary from GitHub releases, or migrate the repo schema.\nSubcommands: repo (migrate schema). Default: binary upgrade.",
		Flags: []Flag{
			{Long: "--check", Help: "Check only, don't install (binary mode)"},
			{Long: "--yes", Help: "Skip confirmation prompt (binary mode)"},
		},
		Examples: []Example{
			{Cmd: "bw upgrade"},
			{Cmd: "bw upgrade --check"},
			{Cmd: "bw upgrade repo"},
		},
		Run: cmdUpgrade,
	},
	{
		Name:    "onboard",
		Summary: "Print AGENTS.md snippet",
		Run:     wrapNoArgs(cmdOnboard),
	},
	{
		Name:    "prime",
		Summary: "Print workflow context for agents",
		Run:     wrapNoArgs(cmdPrime),
	},
}

// wrapNoArgs adapts a func(Writer) error to the standard command signature.
func wrapNoArgs(fn func(w Writer) error) func([]string, Writer) error {
	return func(_ []string, w Writer) error {
		return fn(w)
	}
}

// commandMap provides O(1) lookup by name.
var commandMap map[string]*Command

func init() {
	commandMap = make(map[string]*Command, len(commands))
	for i := range commands {
		commandMap[commands[i].Name] = &commands[i]
		for _, alias := range commands[i].Aliases {
			commandMap[alias] = &commands[i]
		}
	}
}

// commandGroups defines the display order for usage output.
var commandGroups = []struct {
	name string
	cmds []string
}{
	{"Working With Issues", []string{"create", "show", "list", "update", "close", "reopen", "delete", "comments", "label", "defer", "undefer", "history"}},
	{"Finding Work", []string{"ready", "blocked"}},
	{"Dependencies", []string{"dep", "graph"}},
	{"Sync & Data", []string{"sync", "export", "import"}},
	{"Setup & Config", []string{"init", "config", "upgrade", "onboard", "prime"}},
}

func printUsage(w Writer) {
	fmt.Fprintln(w, "bw — lightweight issue tracking with first-class dependency support")
	fmt.Fprintf(w, "\n%s\n", w.Style("Usage:", Cyan))
	w.Push(2)
	fmt.Fprintln(w, "bw <command> [args]")
	fmt.Fprintln(w, "bw <command> --help")
	w.Pop()

	for _, g := range commandGroups {
		fmt.Fprintf(w, "\n%s\n", w.Style(g.name+":", Cyan))
		w.Push(2)
		for _, name := range g.cmds {
			c := commandMap[name]
			if c == nil {
				continue
			}
			usage := name
			for _, p := range c.Positionals {
				usage += " " + p.Name
			}
			if len(c.Flags) > 0 {
				usage += " [flags]"
			}
			fmt.Fprintf(w, "%-28s %s\n", usage, c.Summary)
		}
		w.Pop()
	}

	fmt.Fprintln(w, "\nUse \"bw <command> --help\" for more information about a command.")
}

func printCommandHelp(w Writer, c *Command) {
	// Description (or Summary fallback)
	desc := c.Description
	if desc == "" {
		desc = c.Summary
	}
	fmt.Fprintf(w, "%s\n", desc)

	// Usage line
	usage := "bw " + c.Name
	for _, p := range c.Positionals {
		usage += " " + p.Name
	}
	if len(c.Flags) > 0 {
		usage += " [flags]"
	}
	fmt.Fprintf(w, "\n%s\n", w.Style("Usage:", Cyan))
	w.Push(2)
	fmt.Fprintln(w, usage)
	w.Pop()

	if len(c.Positionals) > 0 {
		fmt.Fprintf(w, "\n%s\n", w.Style("Arguments:", Cyan))
		w.Push(2)
		for _, p := range c.Positionals {
			fmt.Fprintf(w, "%-24s %s\n", p.Name, p.Help)
		}
		w.Pop()
	}

	if len(c.Flags) > 0 {
		fmt.Fprintf(w, "\n%s\n", w.Style("Flags:", Cyan))
		w.Push(2)
		for _, f := range c.Flags {
			flag := f.Long
			if f.Short != "" {
				flag = f.Short + ", " + f.Long
			}
			if f.Value != "" {
				flag += " " + f.Value
			}
			fmt.Fprintf(w, "%-28s %s\n", flag, f.Help)
		}
		w.Pop()
	}

	if len(c.Examples) > 0 {
		fmt.Fprintf(w, "\n%s\n", w.Style("Examples:", Cyan))
		w.Push(2)
		for _, ex := range c.Examples {
			fmt.Fprintln(w, ex.Cmd)
			if ex.Help != "" {
				w.Push(4)
				fmt.Fprintln(w, ex.Help)
				w.Pop()
			}
		}
		w.Pop()
	}
}
