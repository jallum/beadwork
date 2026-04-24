package main

import (
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

var registrySubcommands = map[string]struct {
	summary string
	run     func([]string, Writer) error
}{
	"list":  {"List registered repositories", cmdRegistryList},
	"prune": {"Remove stale registry entries", cmdRegistryPrune},
}

func cmdRegistry(_ *issue.Store, args []string, w Writer) error {
	if len(args) == 0 {
		return printRegistryHelp(w)
	}

	sub := args[0]
	if sub == "--help" || sub == "-h" {
		return printRegistryHelp(w)
	}

	entry, ok := registrySubcommands[sub]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown registry subcommand: %s\n", sub)
		return printRegistryHelp(w)
	}

	subArgs := args[1:]
	if hasFlag(subArgs, "--help") || hasFlag(subArgs, "-h") {
		return printRegistrySubHelp(w, sub, entry.summary)
	}

	return entry.run(subArgs, w)
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
	Path    string `json:"path"`
	Prefix  string `json:"prefix,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}

func cmdRegistryList(args []string, w Writer) error {
	a, err := ParseArgs(args, nil, []string{"--json"})
	if err != nil {
		return err
	}

	reg, err := registry.Load(registry.DefaultPath())
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	paths := reg.Paths()
	if len(paths) == 0 {
		if a.JSON() {
			fmt.Fprintln(w, "[]")
		} else {
			fmt.Fprintln(w, "no registered repositories")
		}
		return nil
	}

	var list []registryListEntry
	for _, path := range paths {
		le := registryListEntry{Path: path}
		if _, err := os.Stat(path); err != nil {
			le.Missing = true
		} else if r, err := repo.FindRepoAt(path); err == nil && r.IsInitialized() {
			le.Prefix = r.Prefix
		}
		list = append(list, le)
	}

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
		line := fmt.Sprintf("[%s] %s", prefix, le.Path)
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

	reg, err := registry.Load(registry.DefaultPath())
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	paths := reg.Paths()
	if len(paths) == 0 {
		fmt.Fprintln(w, "registry is empty, nothing to prune")
		return nil
	}

	var missing []string
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)

	if len(missing) == 0 {
		fmt.Fprintln(w, "all registered repos exist, nothing to prune")
		return nil
	}

	if len(missing) > len(paths)/2 {
		fmt.Fprintf(w, "Warning: %d of %d entries would be removed (more than half).\n",
			len(missing), len(paths))
	}

	fmt.Fprintf(w, "Found %d missing repo(s):\n", len(missing))
	w.Push(2)
	for _, p := range missing {
		fmt.Fprintln(w, p)
	}
	w.Pop()

	if !force {
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

	for _, p := range missing {
		reg.Remove(p)
	}
	if err := reg.Save(); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Fprintf(w, "pruned %d entries\n", len(missing))
	return nil
}
