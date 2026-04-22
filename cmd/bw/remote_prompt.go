package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// promptForRemoteWithReader presents a numbered menu of candidate
// remotes and reads a selection from the given reader. Empty input
// defaults to the first candidate. Retries up to 3 times on invalid
// input before giving up. Shared between the sync and init prompts —
// each command passes its own stdin variable (syncStdin or initStdin)
// so the two prompts don't share input state.
func promptForRemoteWithReader(candidates []string, w Writer, source io.Reader) (string, error) {
	reader := bufio.NewReader(source)
	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintln(w, "multiple remotes — pick one for bw to sync with:")
		for i, name := range candidates {
			fmt.Fprintf(w, "  %d) %s\n", i+1, name)
		}
		fmt.Fprint(w, "select [1]: ")

		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("read input: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			line = "1"
		}
		n, convErr := strconv.Atoi(line)
		if convErr != nil || n < 1 || n > len(candidates) {
			fmt.Fprintf(w, "invalid selection %q; expected a number between 1 and %d\n", line, len(candidates))
			continue
		}
		return candidates[n-1], nil
	}
	return "", fmt.Errorf("too many invalid selections")
}

// gitConfigSet writes a git config key to the repo at repoDir.
func gitConfigSet(repoDir, key, value string) error {
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config %s %s: %s: %w", key, value, strings.TrimSpace(string(out)), err)
	}
	return nil
}
