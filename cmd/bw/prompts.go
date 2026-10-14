package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jallum/beadwork/internal/repo"
)

// loadPrompt selects a configured file or the embedded default. Relative paths
// belong to the current checkout, which may differ from r.RepoDir() in worktrees.
func loadPrompt(r *repo.Repo, name, fallback string) (string, string, error) {
	key := "prompts." + name
	path, _ := r.GetConfig(key)
	if path == "" {
		return fallback, name, nil
	}
	if !filepath.IsAbs(path) {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = r.CWD
		out, err := cmd.Output()
		if err != nil {
			return "", "", fmt.Errorf("resolve %s checkout root: %w", key, err)
		}
		path = filepath.Join(strings.TrimSpace(string(out)), path)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s (%s): %w", key, path, err)
	}
	return string(src), key + " (" + path + ")", nil
}
