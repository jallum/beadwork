package main

import (
	"fmt"
	"io"
)

// Flag describes a single command-line flag.
type Flag struct {
	Long  string // e.g. "--priority"
	Short string // e.g. "-p" (optional)
	Value string // metavar for help, e.g. "N" — empty means boolean
	Help  string // e.g. "Priority level (1-5)"
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
	Run         func(args []string, w io.Writer) error
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

// boolFlags returns the long names of boolean flags (no value).
func (c *Command) boolFlags() []string {
	var bf []string
	for _, f := range c.Flags {
		if f.Value == "" {
			bf = append(bf, f.Long)
		}
	}
	return bf
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
			{Long: "--priority", Short: "-p", Value: "N", Help: "Priority level (1-5)"},
			{Long: "--type", Short: "-t", Value: "TYPE", Help: "Issue type (task, bug, etc.)"},
			{Long: "--assignee", Short: "-a", Value: "WHO", Help: "Assignee"},
			{Long: "--description", Short: "-d", Value: "TEXT", Help: "Description"},
			{Long: "--defer", Value: "DATE", Help: "Defer until date (YYYY-MM-DD)"},
			{Long: "--json", Help: "Output as JSON"},
		},
		Examples: []Example{
			{Cmd: `bw create "Fix login bug" --priority 1 --type bug`},
			{Cmd: `bw create "New feature" -a alice`},
			{Cmd: `bw create "Q3 planning" --defer 2027-07-01`},
		},
		Run: cmdCreate,
	},
	{
		Name:        "show",
		Aliases:     []string{"view"},
		Summary:     "Show issue details",
		Description: "Display full details for an issue including status, priority, labels, and dependencies.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "Issue ID (can specify multiple)"},
		},
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
			{Long: "--short", Help: "Compact one-line output"},
		},
		Examples: []Example{
			{Cmd: "bw show bw-a3f8"},
			{Cmd: "bw show bw-a3f8 bw-b2c1"},
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
			{Long: "--priority", Short: "-p", Value: "N", Help: "Filter by priority"},
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
			{Long: "--priority", Short: "-p", Value: "N", Help: "New priority"},
			{Long: "--assignee", Short: "-a", Value: "WHO", Help: "New assignee"},
			{Long: "--type", Short: "-t", Value: "TYPE", Help: "New type"},
			{Long: "--status", Short: "-s", Value: "STATUS", Help: "New status"},
			{Long: "--defer", Value: "DATE", Help: "Defer until date (YYYY-MM-DD)"},
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
			{Name: "<file>", Required: true, Help: "JSONL file path"},
		},
		Flags: []Flag{
			{Long: "--dry-run", Help: "Preview without importing"},
		},
		Examples: []Example{
			{Cmd: "bw import issues.jsonl"},
			{Cmd: "bw import issues.jsonl --dry-run"},
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

// wrapNoArgs adapts a func(io.Writer) error to the standard command signature.
func wrapNoArgs(fn func(w io.Writer) error) func([]string, io.Writer) error {
	return func(_ []string, w io.Writer) error {
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
	{"Working With Issues", []string{"create", "show", "list", "update", "close", "reopen", "label", "defer", "undefer"}},
	{"Finding Work", []string{"ready", "blocked"}},
	{"Dependencies", []string{"dep", "graph"}},
	{"Sync & Data", []string{"sync", "export", "import"}},
	{"Setup & Config", []string{"init", "config", "upgrade", "onboard", "prime"}},
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "bw — lightweight issue tracking with first-class dependency support")
	fmt.Fprintln(w, "\nUsage:")
	fmt.Fprintln(w, "  bw <command> [args]")
	fmt.Fprintln(w, "  bw <command> --help")

	for _, g := range commandGroups {
		fmt.Fprintf(w, "\n%s:\n", g.name)
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
			fmt.Fprintf(w, "  %-28s %s\n", usage, c.Summary)
		}
	}

	fmt.Fprintln(w, "\nUse \"bw <command> --help\" for more information about a command.")
}

func printCommandHelp(w io.Writer, c *Command) {
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
	fmt.Fprintf(w, "\nUsage:\n  %s\n", usage)

	if len(c.Positionals) > 0 {
		fmt.Fprintln(w, "\nArguments:")
		for _, p := range c.Positionals {
			fmt.Fprintf(w, "  %-24s %s\n", p.Name, p.Help)
		}
	}

	if len(c.Flags) > 0 {
		fmt.Fprintln(w, "\nFlags:")
		for _, f := range c.Flags {
			flag := f.Long
			if f.Short != "" {
				flag = f.Short + ", " + f.Long
			}
			if f.Value != "" {
				flag += " " + f.Value
			}
			fmt.Fprintf(w, "  %-28s %s\n", flag, f.Help)
		}
	}

	if len(c.Examples) > 0 {
		fmt.Fprintln(w, "\nExamples:")
		for _, ex := range c.Examples {
			fmt.Fprintf(w, "  %s\n", ex.Cmd)
			if ex.Help != "" {
				fmt.Fprintf(w, "      %s\n", ex.Help)
			}
		}
	}
}
