package repo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/cache"
	format "github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// gitConfigFile is the config file's name inside the .git directory.
const gitConfigFile = "config"

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

// openGitRepo opens the repo at repoDir, reading git config through
// configFilteringStorer so config bw has no use for can't block the open.
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
	return git.Open(&configFilteringStorer{Storer: s, dotGit: dotGit}, wt)
}

// configFilteringStorer wraps a storage.Storer and parses .git/config itself,
// dropping entries go-git rejects but bw never reads: extensions listed in
// bypassedExtensions, and negative fetch refspecs.
//
// SetConfig is inherited from the embedded Storer unchanged — bw never writes
// git config, so there's no round-trip concern.
type configFilteringStorer struct {
	storage.Storer
	dotGit billy.Filesystem
}

// Config parses .git/config, filtering it in its raw on-disk form before
// go-git decodes it. It replaces the embedded storer's Config rather than
// post-processing its result because go-git rejects a negative refspec while
// decoding: Config returns a nil config alongside the error, leaving nothing
// to filter after the fact.
func (s *configFilteringStorer) Config() (*config.Config, error) {
	f, err := s.dotGit.Open(gitConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return config.NewConfig(), nil
		}
		return nil, err
	}
	defer f.Close()

	raw := format.New()
	if err := format.NewDecoder(f).Decode(raw); err != nil {
		return nil, err
	}

	stripBypassedExtensions(raw)
	stripNegativeFetchRefSpecs(raw)

	// go-git's decoders for the typed config are unexported, so re-encoding
	// the filtered raw config is the only way to produce a *config.Config.
	var buf bytes.Buffer
	if err := format.NewEncoder(&buf).Encode(raw); err != nil {
		return nil, err
	}
	cfg := config.NewConfig()
	if err := cfg.Unmarshal(buf.Bytes()); err != nil {
		return nil, err
	}
	return cfg, nil
}

// stripBypassedExtensions removes the extensions named in bypassedExtensions
// so go-git's extension check doesn't reject the repo.
func stripBypassedExtensions(raw *format.Config) {
	if !raw.HasSection("extensions") {
		return
	}
	section := raw.Section("extensions")
	kept := section.Options[:0]
	for _, opt := range section.Options {
		if _, skip := bypassedExtensions[strings.ToLower(opt.Key)]; skip {
			continue
		}
		kept = append(kept, opt)
	}
	section.Options = kept
}

// stripNegativeFetchRefSpecs removes negative fetch refspecs — the
// `fetch = ^refs/heads/...` form git 2.29+ uses to exclude refs from a fetch.
// go-git's refspec validator requires exactly one ":" separator, so it rejects
// these outright and every bw command in such a repo fails at open with
// "malformed refspec, separators are wrong".
//
// Dropping them costs bw nothing: it runs fetch and push by shelling out to
// git with an explicit refspec, and never fetches through go-git.
func stripNegativeFetchRefSpecs(raw *format.Config) {
	if !raw.HasSection("remote") {
		return
	}
	for _, sub := range raw.Section("remote").Subsections {
		kept := sub.Options[:0]
		for _, opt := range sub.Options {
			if opt.IsKey("fetch") && strings.HasPrefix(opt.Value, "^") {
				continue
			}
			kept = append(kept, opt)
		}
		sub.Options = kept
	}
}
