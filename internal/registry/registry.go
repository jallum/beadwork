// Package registry provides config-backed helpers for the host-local
// repository registry (the registry.repos key in the global config).
package registry

import (
	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/repo"
)

const key = "registry.repos"

// Auto reports whether automatic registration is enabled.
func Auto(cfg *config.Config) bool {
	return cfg.Bool("registry.auto")
}

// Paths returns the registered repo paths from config.
func Paths(cfg *config.Config) []string {
	return cfg.StringSlice(key)
}

// Repo pairs a filesystem path with the prefix read live from the repo.
type Repo struct {
	Path   string
	Prefix string
}

// Repos returns all registered repos with their prefixes. Entries that
// can't be opened or aren't initialized are silently skipped.
func Repos(cfg *config.Config) []Repo {
	var out []Repo
	for _, p := range Paths(cfg) {
		r, err := repo.FindRepoAt(p)
		if err != nil || !r.IsInitialized() {
			continue
		}
		out = append(out, Repo{Path: p, Prefix: r.Prefix})
	}
	return out
}

// Resolve finds the repo path for a given prefix. Returns ("", false)
// if no match is found.
func Resolve(cfg *config.Config, prefix string) (string, bool) {
	for _, r := range Repos(cfg) {
		if r.Prefix == prefix {
			return r.Path, true
		}
	}
	return "", false
}

// Register returns a new config with path added to the registry. If the
// path is already registered, returns cfg unchanged (same pointer).
func Register(cfg *config.Config, path string) *config.Config {
	for _, p := range Paths(cfg) {
		if p == path {
			return cfg
		}
	}
	return cfg.Set(key, append(Paths(cfg), path))
}
