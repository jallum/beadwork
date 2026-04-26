package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
)

// resolveCFlag expands the value of -C: tries the registry first
// (prefix), falls through to a filesystem path if no match.
func resolveCFlag(cfg *config.Config, arg string) string {
	paths := registry.ResolveAll(cfg, arg)
	if len(paths) > 1 {
		fatal(fmt.Sprintf("-C %s: prefix registered for %d repositories; use -C <path> to disambiguate:\n  %s",
			arg, len(paths), strings.Join(paths, "\n  ")))
	}
	if len(paths) == 1 {
		return paths[0]
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		fatal(fmt.Sprintf("-C %s: %s", arg, err))
	}
	return abs
}

// resolveCrossRepo rewrites repoDir if the first positional argument
// references an issue in a different beadwork-enabled repository, using
// the host-local registry as a prefix → path lookup.
//
// If the prefix is unknown, we do nothing (let the downstream command
// fail with its usual "not found" error). If multiple positionals
// reference conflicting prefixes, the call fails loudly.
//
// The resolver runs AFTER -C extraction and BEFORE command dispatch, so
// an explicit -C wins.
func resolveCrossRepo(cfg *config.Config, args []string) {
	if repoDir != "" {
		return
	}

	localPrefix := ""
	if r, err := repo.FindRepoAt(""); err == nil && r.IsInitialized() {
		localPrefix = r.Prefix
	}

	known := knownPrefixes(cfg)
	if localPrefix != "" {
		known[localPrefix] = true
	}

	prefixes := extractPrefixCandidates(args, known)
	if len(prefixes) == 0 {
		return
	}

	var foreign string
	localSeen := false
	for _, p := range prefixes {
		if p == localPrefix {
			localSeen = true
			continue
		}
		if foreign == "" {
			foreign = p
			continue
		}
		if foreign != p {
			fatal(fmt.Sprintf("cross-repo references mixing prefixes %q and %q are not supported", foreign, p))
		}
	}
	if foreign == "" {
		return
	}
	if localSeen {
		fatal(fmt.Sprintf("cross-repo references mixing local prefix %q with %q are not supported", localPrefix, foreign))
	}

	paths := registry.ResolveAll(cfg, foreign)
	if len(paths) > 1 {
		fatal(fmt.Sprintf("prefix %q registered for %d repositories; use -C <path> to disambiguate:\n  %s",
			foreign, len(paths), strings.Join(paths, "\n  ")))
	}
	if len(paths) == 0 {
		return
	}
	repoDir = paths[0]
	if os.Getenv("BW_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[cross-repo] %q → %s\n", foreign, paths[0])
	}
}

// knownPrefixes returns the set of live prefixes from registered repos.
func knownPrefixes(cfg *config.Config) map[string]bool {
	known := map[string]bool{}
	for _, r := range registry.Repos(cfg) {
		if r.Prefix != "" {
			known[r.Prefix] = true
		}
	}
	return known
}

// extractPrefixCandidates returns the prefixes of ID-shaped positional
// arguments that are also in the `known` set.
func extractPrefixCandidates(args []string, known map[string]bool) []string {
	var prefixes []string
	seen := map[string]bool{}

	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			if !strings.Contains(a, "=") {
				skipNext = true
			}
			continue
		}
		idx := strings.IndexByte(a, '-')
		if idx <= 0 {
			continue
		}
		p := a[:idx]
		if !known[p] {
			continue
		}
		if !seen[p] {
			seen[p] = true
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
}

