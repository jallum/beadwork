// Package registry tracks which repositories on this host use beadwork.
// The registry is a plain text file (one absolute path per line) at
// ~/.bw by default, overridden by $BEADWORK_HOME.
package registry

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Registry holds the in-memory set of registered repo paths.
type Registry struct {
	paths map[string]bool
	file  string
	mu    sync.Mutex
}

// Load reads the registry from file. If the file does not exist, returns
// an empty registry.
func Load(file string) (*Registry, error) {
	r := &Registry{
		paths: make(map[string]bool),
		file:  file,
	}

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			r.paths[line] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	return r, nil
}

// Save atomically writes the registry to disk.
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	dir := filepath.Dir(r.file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	paths := r.sortedPaths()
	var b strings.Builder
	for _, p := range paths {
		b.WriteString(p)
		b.WriteByte('\n')
	}

	tmp := r.file + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("write temp registry: %w", err)
	}
	if err := os.Rename(tmp, r.file); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename registry: %w", err)
	}
	return nil
}

// Add registers a repo path and saves atomically.
func (r *Registry) Add(repoPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paths[repoPath] {
		return nil
	}
	r.paths[repoPath] = true
	return r.saveLocked()
}

// Remove deletes a repo path and saves atomically.
func (r *Registry) Remove(repoPath string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.paths[repoPath] {
		return false
	}
	delete(r.paths, repoPath)
	return true
}

// Paths returns a sorted snapshot of all registered paths.
func (r *Registry) Paths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sortedPaths()
}

func (r *Registry) sortedPaths() []string {
	paths := make([]string, 0, len(r.paths))
	for p := range r.paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
