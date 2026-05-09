package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
	"golang.org/x/term"
)

const version = "0.12.3"

// globalNoColor is set by commands (e.g. recap --no-color) to force
// non-colored output even when stdout is a TTY. Consulted at render setup.
var globalNoColor bool

// globalDryRun mirrors the --dry-run flag. For commands with NeedsStore
// it drives store.DryRun; for commands without a store (e.g. recap) it
// suppresses side-effects like advancing the registry cursor.
var globalDryRun bool

func resolveRenderMode(args []string) string {
	if mode, ok := flagValue(args, "--x-render-as"); ok && mode != "" {
		return mode
	}
	if hasFlag(args, "--x-raw") {
		return "raw"
	}
	if hasFlag(args, "--no-color") {
		return "markdown"
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
	if hasFlag(args, "--no-color") {
		globalNoColor = true
		args = removeFlag(args, "--no-color")
	}

	dryRun := hasFlag(args, "--dry-run")
	if dryRun {
		args = removeFlag(args, "--dry-run")
		globalDryRun = true
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

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		fatal(err.Error())
	}

	originalCfg := cfg

	newCfg, err := c.Run(store, args, w, cfg)
	if err != nil {
		fatal(err.Error())
	}
	if newCfg != nil {
		cfg = newCfg
	}

	if store != nil && registry.Auto(cfg) {
		r := store.Committer.(*repo.Repo)
		cfg = registry.Register(cfg, r.RepoDir())
	}

	if cfg != originalCfg {
		_ = cfg.Save()
	}
}

// bwNow returns the current time respecting BW_CLOCK.
// The returned Time preserves its original location (local time when no
// BW_CLOCK is set; whatever offset BW_CLOCK carries otherwise) so that
// day-boundary math ("today", "yesterday") uses the user's local zone.
// Callers that need UTC for storage should .UTC() themselves.
func bwNow() time.Time {
	if v := os.Getenv("BW_CLOCK"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Now()
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
