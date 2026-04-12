package main

import (
	"github.com/jallum/beadwork/internal/config"

	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
	"golang.org/x/term"
)

// registrySubcommands holds the dispatch table for `bw registry <sub>`.
var registrySubcommands = map[string]struct {
	summary string
	run     func([]string, Writer) error
}{
	"list":  {"List registered repositories", cmdRegistryList},
	"prune": {"Remove stale registry entries", cmdRegistryPrune},
}

func cmdRegistry(_ *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	if len(args) == 0 {
		return nil, printRegistryHelp(w)
	}

	sub := args[0]
	if sub == "--help" || sub == "-h" {
		return nil, printRegistryHelp(w)
	}

	entry, ok := registrySubcommands[sub]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown registry subcommand: %s\n", sub)
		return nil, printRegistryHelp(w)
	}

	subArgs := args[1:]
	if hasFlag(subArgs, "--help") || hasFlag(subArgs, "-h") {
		return nil, printRegistrySubHelp(w, sub, entry.summary)
	}

	return nil, entry.run(subArgs, w)
}

func printRegistryHelp(w Writer) error {
	fmt.Fprintln(w, "Manage the beadwork repository registry.")
	fmt.Fprintf(w, "\n%s\n", w.Style("Usage:", Cyan))
	w.Push(2)
	fmt.Fprintln(w, "bw registry <subcommand> [flags]")
	w.Pop()
	fmt.Fprintf(w, "\n%s\n", w.Style("Subcommands:", Cyan))
	w.Push(2)
	fmt.Fprintf(w, "%-20s %s\n", "list", "List registered repositories")
	fmt.Fprintf(w, "%-20s %s\n", "prune", "Remove stale registry entries")
	w.Pop()
	return nil
}

func printRegistrySubHelp(w Writer, name, summary string) error {
	fmt.Fprintln(w, summary)
	fmt.Fprintf(w, "\n%s\n", w.Style("Usage:", Cyan))
	w.Push(2)
	fmt.Fprintf(w, "bw registry %s [flags]\n", name)
	w.Pop()
	return nil
}

type registryListEntry struct {
	Path       string `json:"path"`
	Prefix     string `json:"prefix,omitempty"`
	LastSeenAt string `json:"last_seen_at"`
	Missing    bool   `json:"missing,omitempty"`
}

func cmdRegistryList(args []string, w Writer) error {
	a, err := ParseArgs(args, nil, []string{"--json"})
	if err != nil {
		return err
	}

	dir := registry.DefaultDir()
	reg, err := registry.Load(dir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	entries := reg.Entries()
	if len(entries) == 0 {
		if a.JSON() {
			fmt.Fprintln(w, "[]")
		} else {
			fmt.Fprintln(w, "no registered repositories")
		}
		return nil
	}

	// Build sorted list with live prefix read and missing detection.
	var list []registryListEntry
	for path, e := range entries {
		le := registryListEntry{
			Path:       path,
			LastSeenAt: e.LastSeenAt,
			Prefix:     e.Prefix,
		}
		// Check if the repo still exists and try to read its prefix.
		if _, err := os.Stat(path); err != nil {
			le.Missing = true
		} else if le.Prefix == "" {
			if r, err := repo.FindRepoAt(path); err == nil && r.IsInitialized() {
				le.Prefix = r.Prefix
			}
		}
		list = append(list, le)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Path < list[j].Path
	})

	if a.JSON() {
		data, _ := json.MarshalIndent(list, "", "  ")
		fmt.Fprintln(w, string(data))
		return nil
	}

	for _, le := range list {
		prefix := le.Prefix
		if prefix == "" {
			prefix = "?"
		}
		age := relativeTime(le.LastSeenAt)
		line := fmt.Sprintf("[%s] %s  (%s)", prefix, le.Path, age)
		if le.Missing {
			line += "  " + w.Style("MISSING", Red)
		}
		fmt.Fprintln(w, line)
	}
	return nil
}

func cmdRegistryPrune(args []string, w Writer) error {
	a, err := ParseArgs(args, nil, []string{"--yes", "-y"})
	if err != nil {
		return err
	}

	force := a.Bool("--yes") || a.Bool("-y")

	dir := registry.DefaultDir()
	reg, err := registry.Load(dir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	entries := reg.Entries()
	if len(entries) == 0 {
		fmt.Fprintln(w, "registry is empty, nothing to prune")
		return nil
	}

	// Find missing entries.
	var missing []string
	for path := range entries {
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)

	if len(missing) == 0 {
		fmt.Fprintln(w, "all registered repos exist, nothing to prune")
		return nil
	}

	// Half-removal warning.
	if len(missing) > len(entries)/2 {
		fmt.Fprintf(w, "Warning: %d of %d entries would be removed (more than half).\n",
			len(missing), len(entries))
	}

	fmt.Fprintf(w, "Found %d missing repo(s):\n", len(missing))
	w.Push(2)
	for _, p := range missing {
		fmt.Fprintln(w, p)
	}
	w.Pop()

	if !force {
		// Check if stdin is a TTY for interactive confirmation.
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("non-interactive: pass --yes to confirm")
		}
		fmt.Fprint(w, "\nRemove these entries? [y/N] ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(w, "aborted")
			return nil
		}
	}

	missingSet := make(map[string]bool, len(missing))
	for _, p := range missing {
		missingSet[p] = true
	}
	removed := reg.Prune(func(path string, _ registry.Entry) bool {
		return missingSet[path]
	})

	if err := reg.Save(); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Fprintf(w, "pruned %d entries\n", len(removed))

	return nil
}
