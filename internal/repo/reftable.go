package repo

import (
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	format "github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

// refStorageExtension is the git config extension naming the ref backend.
const refStorageExtension = "refstorage"

// usesReftable reports whether the repo stores refs in the reftable backend
// (`extensions.refstorage = reftable`, git 2.45+) rather than as loose files
// under .git/refs plus packed-refs.
func usesReftable(raw *format.Config) bool {
	if !raw.HasSection("extensions") {
		return false
	}
	for _, opt := range raw.Section("extensions").Options {
		if opt.IsKey(refStorageExtension) {
			return strings.EqualFold(strings.TrimSpace(opt.Value), "reftable")
		}
	}
	return false
}

// isHeadRef reports whether name is HEAD or a per-remote HEAD, the ref names
// that are conventionally symbolic. Checking avoids spending a git process on
// a symbolic-ref probe for every ordinary branch read.
func isHeadRef(name plumbing.ReferenceName) bool {
	s := name.String()
	return s == plumbing.HEAD.String() || strings.HasSuffix(s, "/HEAD")
}

// reftableRefStorer serves the ReferenceStorer half of a storage.Storer from
// the git CLI, leaving every other method to the embedded Storer.
//
// reftable changes where refs live (.git/reftable) but not how objects are
// stored, so go-git's filesystem storage stays correct for objects, trees and
// commits — it just cannot see a single ref. Shelling out for refs is enough
// to make bw work on these repos without waiting for go-git#1827.
//
// Ref reads cost a git process each, so this is only installed when the repo
// actually uses reftable; ordinary repos keep go-git's native path.
type reftableRefStorer struct {
	storage.Storer
	repoDir string
}

// Reference resolves a single ref. HEAD — and the per-remote HEADs, which are
// symbolic too — are reported as symbolic references, matching what go-git
// expects from a normal repo. Any other symbolic ref resolves to its target
// hash, which is all bw ever needs from one.
func (s *reftableRefStorer) Reference(name plumbing.ReferenceName) (*plumbing.Reference, error) {
	if isHeadRef(name) {
		if target, err := execGit(s.repoDir, "symbolic-ref", "--quiet", name.String()); err == nil {
			if t := strings.TrimSpace(target); t != "" {
				return plumbing.NewSymbolicReference(name, plumbing.ReferenceName(t)), nil
			}
		}
	}

	out, err := execGit(s.repoDir, "rev-parse", "--verify", "--quiet", name.String())
	if err != nil {
		return nil, plumbing.ErrReferenceNotFound
	}
	hash := strings.TrimSpace(out)
	if hash == "" {
		return nil, plumbing.ErrReferenceNotFound
	}
	return plumbing.NewHashReference(name, plumbing.NewHash(hash)), nil
}

// SetReference writes a ref unconditionally.
func (s *reftableRefStorer) SetReference(ref *plumbing.Reference) error {
	if ref.Type() == plumbing.SymbolicReference {
		_, err := execGit(s.repoDir, "symbolic-ref", ref.Name().String(), ref.Target().String())
		return err
	}
	_, err := execGit(s.repoDir, "update-ref", ref.Name().String(), ref.Hash().String())
	return err
}

// CheckAndSetReference writes new only if the stored value still matches old,
// delegating the compare-and-swap to `git update-ref <ref> <new> <old>`.
func (s *reftableRefStorer) CheckAndSetReference(new, old *plumbing.Reference) error {
	if old == nil {
		return s.SetReference(new)
	}
	_, err := execGit(s.repoDir, "update-ref",
		new.Name().String(), new.Hash().String(), old.Hash().String())
	return err
}

// IterReferences lists every ref, plus HEAD, which for-each-ref omits.
func (s *reftableRefStorer) IterReferences() (storer.ReferenceIter, error) {
	out, err := execGit(s.repoDir, "for-each-ref",
		"--format=%(objectname) %(refname) %(symref)")
	if err != nil {
		return nil, err
	}

	var refs []*plumbing.Reference
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := plumbing.ReferenceName(fields[1])
		if len(fields) > 2 {
			refs = append(refs, plumbing.NewSymbolicReference(name, plumbing.ReferenceName(fields[2])))
			continue
		}
		refs = append(refs, plumbing.NewHashReference(name, plumbing.NewHash(fields[0])))
	}

	if head, err := s.Reference(plumbing.HEAD); err == nil {
		refs = append(refs, head)
	}

	return storer.NewReferenceSliceIter(refs), nil
}

// RemoveReference deletes a ref. Deleting a ref that is already gone is not an
// error, matching go-git's filesystem storer.
func (s *reftableRefStorer) RemoveReference(name plumbing.ReferenceName) error {
	_, err := execGit(s.repoDir, "update-ref", "-d", name.String())
	return err
}

// CountLooseRefs reports zero: reftable has no loose refs by construction.
// go-git only uses the count to decide whether packing is worthwhile.
func (s *reftableRefStorer) CountLooseRefs() (int, error) {
	return 0, nil
}

// PackRefs is a no-op. reftable compacts its own tables.
func (s *reftableRefStorer) PackRefs() error {
	return nil
}
