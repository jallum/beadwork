package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
	"golang.org/x/term"
)

const version = "0.12.3"

func resolveRenderMode(args []string) string {
	if mode, ok := flagValue(args, "--x-render-as"); ok && mode != "" {
		return mode
	}
	if hasFlag(args, "--x-raw") {
		return "raw"
	}
	if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
		return "tty"
	}
	return "markdown"
}

func main() {
	var w Writer
	switch resolveRenderMode(os.Args) {
	case "tty":
		width := 80
		if term.IsTerminal(int(os.Stdout.Fd())) {
			width, _, _ = term.GetSize(int(os.Stdout.Fd()))
		}
		w = ColorWriter(os.Stdout, width)
	case "raw":
		w = RawWriter(os.Stdout)
	default:
		w = PlainWriter(os.Stdout)
	}

	allArgs := os.Args[1:]
	allArgs = extractDirFlag(allArgs)

	if len(allArgs) < 1 {
		printUsage(w)
		os.Exit(1)
	}

	cmd := allArgs[0]
	args := allArgs[1:]

	args = removeFlag(args, "--x-raw")
	args, _ = removeFlagValue(args, "--x-render-as")

	dryRun := hasFlag(args, "--dry-run")
	if dryRun {
		args = removeFlag(args, "--dry-run")
	}

	switch cmd {
	case "--version", "-v":
		fmt.Fprintln(w, "bw "+version)
		return
	case "--help", "-h":
		printUsage(w)
		return
	}

	c, ok := commandMap[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage(w)
		os.Exit(1)
	}

	if hasFlag(args, "--help") || hasFlag(args, "-h") {
		printCommandHelp(w, c)
		return
	}

	var store *issue.Store
	if c.NeedsStore {
		var err error
		store, err = getInitializedStore()
		if err != nil {
			fatal(err.Error())
		}
		store.DryRun = dryRun
		maybeCheckForUpgrade(store, w)
	}

	if err := c.Run(store, args, w); err != nil {
		fatal(err.Error())
	}

	// Auto-register the repo after a successful command. Runs AFTER the
	// command so `bw init` gets captured (IsInitialized flips during init).
	// Gated on IsInitialized so non-beadwork git repos don't clutter the
	// registry.
	touchRegistry(bwNow())
}

// touchRegistry registers the current repo in the host-local registry.
// It resolves the repo via FindRepoAt (not NeedsStore), resolves worktrees
// to their main repo, and silently ignores errors. Runs on every command
// so the registry stays up-to-date even for read-only operations.
var registryOnce sync.Once

func touchRegistry(now time.Time) {
	r, err := repo.FindRepoAt(repoDir)
	if err != nil {
		return // not in a git repo — nothing to register
	}
	if !r.IsInitialized() {
		return // git repo but beadwork never ran `bw init` here
	}
	repoPath, err := registry.CanonicalRepoPath(r.RepoDir())
	if err != nil {
		return
	}
	dir := registry.DefaultDir()
	reg, err := registry.Load(dir)
	if err != nil {
		registryOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "warning: could not load registry: %v\n", err)
		})
		return
	}
	if err := reg.TouchAndSave(repoPath, now); err != nil {
		registryOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "warning: could not save registry: %v\n", err)
		})
	}
}

// bwNow returns the current time respecting BW_CLOCK.
func bwNow() time.Time {
	if v := os.Getenv("BW_CLOCK"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

// extractDirFlag removes all -C <dir> pairs from args and sets repoDir.
func extractDirFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-C" {
			if i+1 >= len(args) {
				fatal("-C requires an argument")
			}
			abs, err := filepath.Abs(args[i+1])
			if err != nil {
				fatal(fmt.Sprintf("-C %s: %s", args[i+1], err))
			}
			repoDir = abs
			i++ // skip value
		} else {
			out = append(out, args[i])
		}
	}
	return out
}
