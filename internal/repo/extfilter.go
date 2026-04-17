package repo

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// bypassedExtensions names git repo extensions that are safe for bw to ignore.
// These don't affect the object format or anything treefs reads; bypassing
// them lets bw open repos that upstream go-git would otherwise reject.
//
// worktreeConfig is set (and core.repositoryformatversion bumped to 1) as
// soon as a user runs `git config --worktree ...` — a common pattern in
// repos that use git worktrees.
var bypassedExtensions = map[string]struct{}{
	"worktreeconfig": {},
}

// openGitRepo opens the repo at repoDir, filtering out extensions listed in
// bypassedExtensions so go-git's extension check doesn't reject the repo.
func openGitRepo(repoDir string) (*git.Repository, error) {
	wt := osfs.New(repoDir)
	dotGit := osfs.New(filepath.Join(repoDir, ".git"))

	if _, err := dotGit.Stat(""); err != nil {
		if os.IsNotExist(err) {
			return nil, git.ErrRepositoryNotExists
		}
		return nil, err
	}

	s := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())
	return git.Open(&extFilteringStorer{Storer: s}, wt)
}

// extFilteringStorer wraps a storage.Storer and strips bypassed extensions
// from the config returned by Config(). SetConfig is passed through unchanged
// — bw never writes git config, so no round-trip concern here.
type extFilteringStorer struct {
	storage.Storer
}

func (s *extFilteringStorer) Config() (*config.Config, error) {
	cfg, err := s.Storer.Config()
	if err != nil {
		return cfg, err
	}
	stripBypassedExtensions(cfg)
	return cfg, nil
}

func stripBypassedExtensions(cfg *config.Config) {
	if cfg == nil || cfg.Raw == nil || !cfg.Raw.HasSection("extensions") {
		return
	}
	section := cfg.Raw.Section("extensions")
	kept := section.Options[:0]
	for _, opt := range section.Options {
		if _, skip := bypassedExtensions[strings.ToLower(opt.Key)]; skip {
			continue
		}
		kept = append(kept, opt)
	}
	section.Options = kept
}
