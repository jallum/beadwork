package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
)

// resolveCFlag expands the value of -C: tries the registry first
// (prefix or alias), falls through to a filesystem path if no match.
// Prefixes are [a-zA-Z0-9][a-zA-Z0-9_-]* ≤ 16 chars, so they never
// collide with path-shaped strings (/, ~, ., embedded /).
//
// Collisions (multiple repos registered under the same prefix) fail
// loudly with the same error shape as the cross-repo resolver.
func resolveCFlag(arg string) string {
	if reg, err := registry.Load(registry.DefaultDir()); err == nil {
		paths := reg.LookupPrefix(arg)
		switch len(paths) {
		case 1:
			return paths[0]
		case 0:
			// fall through to path interpretation
		default:
			var b strings.Builder
			fmt.Fprintf(&b, "-C %q is registered for %d repositories:\n", arg, len(paths))
			for _, p := range paths {
				fmt.Fprintf(&b, "  %s\n", p)
			}
			fmt.Fprintf(&b, "use -C with the full path to disambiguate")
			fatal(b.String())
		}
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
// Behavior:
//   - If the first positional matches <prefix>-<suffix> and <prefix> is
//     registered for a different repo than the current one, repoDir is
//     rewritten to that repo's canonical path.
//   - If the prefix is unknown, we do nothing (let the downstream command
//     fail with its usual "not found" error).
//   - If multiple positionals reference conflicting prefixes (e.g. a dep
//     command linking across repos), the call fails loudly — cross-repo
//     linking would produce non-round-trippable data.
//
// The resolver runs AFTER -C extraction and BEFORE command dispatch, so
// an explicit -C wins. The resolver never rewrites repoDir when -C was
// provided: the user was explicit.
func resolveCrossRepo(args []string) {
	if repoDir != "" {
		return // user was explicit with -C
	}

	// Determine the current repo's prefix (if any) so we can filter it out.
	localPrefix := ""
	if r, err := repo.FindRepoAt(""); err == nil && r.IsInitialized() {
		localPrefix = r.Prefix
	}

	// Load the registry. If it's not available, do nothing — commands still
	// run against the current repo.
	reg, err := registry.Load(registry.DefaultDir())
	if err != nil {
		return
	}

	// Build the set of known prefixes. A token only counts as a potential
	// cross-repo ID if its prefix is registered AND at most 16 chars — that
	// keeps date-shaped args (2026-06-01) and similar noise out.
	known := map[string]bool{}
	for _, e := range reg.Entries() {
		if e.Prefix != "" {
			known[e.Prefix] = true
		}
	}
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

	paths := reg.LookupPrefix(foreign)
	switch len(paths) {
	case 0:
		return // unknown prefix — let the command fail with a useful error
	case 1:
		repoDir = paths[0]
		if os.Getenv("BW_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "[cross-repo] %q → %s\n", foreign, paths[0])
		}
	default:
		// Prefix collision: multiple repos registered under the same prefix.
		// Refuse to guess; give the user a clear listing and the -C escape.
		var b strings.Builder
		fmt.Fprintf(&b, "prefix %q is registered for %d repositories:\n", foreign, len(paths))
		for _, p := range paths {
			fmt.Fprintf(&b, "  %s\n", p)
		}
		fmt.Fprintf(&b, "use -C <path> to disambiguate")
		fatal(b.String())
	}
}

// extractPrefixCandidates returns the prefixes of ID-shaped positional
// arguments that are also in the `known` set (registered prefixes). This
// keeps date-shaped args and other hyphenated text from being mistaken
// for issue IDs. Value-flag arguments (e.g. --assignee bob) are skipped
// best-effort: if a flag lacks "=" and its next token doesn't start with
// "-", that token is treated as a value, not a positional.
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

// isPrefixCandidate reports whether s looks like a beadwork prefix token.
// Matches repo.ValidatePrefix: alphanumerics + _/- allowed, length ≤ 16.
func isPrefixCandidate(s string) bool {
	if s == "" || len(s) > 16 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '_' || c == '-':
		default:
			return false
		}
	}
	return true
}
