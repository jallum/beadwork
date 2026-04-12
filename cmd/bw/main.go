package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jallum/beadwork/internal/config"
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

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		fatal(err.Error())
	}

	newCfg, err := c.Run(store, args, w, cfg)
	if err != nil {
		fatal(err.Error())
	}
	if newCfg != nil {
		if err := newCfg.Save(); err != nil {
			fatal(err.Error())
		}
	}

	touchRegistry(bwNow())
}

var registryOnce sync.Once

func touchRegistry(now time.Time) {
	r, err := repo.FindRepoAt(repoDir)
	if err != nil {
		return
	}
	if !r.IsInitialized() {
		return
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
