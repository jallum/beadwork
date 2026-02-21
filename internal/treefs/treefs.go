// Package treefs provides a mutable, filesystem-like API backed by git tree
// objects. It reads the current tree from a ref, allows in-memory mutations,
// then writes the result as a new tree + commit atomically.
package treefs

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// DirEntry represents a single entry returned by ReadDir.
type DirEntry struct {
	name  string
	isDir bool
}

func (d DirEntry) Name() string { return d.name }
func (d DirEntry) IsDir() bool  { return d.isDir }

// FileInfo holds metadata about a file or directory.
type FileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fi FileInfo) Name() string { return fi.name }
func (fi FileInfo) Size() int64  { return fi.size }
func (fi FileInfo) IsDir() bool  { return fi.isDir }

// TreeFS is a mutable in-memory filesystem backed by a git tree.
type TreeFS struct {
	repo    *git.Repository
	ref     plumbing.ReferenceName
	baseRef plumbing.Hash // commit hash at Open time, used for CAS
	base    *object.Tree  // nil if branch doesn't exist yet

	// overlay tracks pending mutations: path → content (nil means delete)
	overlay map[string][]byte
	// dirs tracks explicitly created directories (for MkdirAll)
	dirs map[string]bool
}

// Open opens a git repository and loads the tree from the given ref.
// If the ref doesn't exist yet, the TreeFS starts empty.
func Open(gitDir string, ref string) (*TreeFS, error) {
	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}
	return OpenFromRepo(repo, ref)
}

// OpenFromRepo creates a TreeFS from an already-opened go-git Repository.
func OpenFromRepo(repo *git.Repository, ref string) (*TreeFS, error) {
	refName := plumbing.ReferenceName(ref)
	tfs := &TreeFS{
		repo:    repo,
		ref:     refName,
		overlay: make(map[string][]byte),
		dirs:    make(map[string]bool),
	}

	r, err := repo.Reference(refName, true)
	if err != nil {
		// Ref doesn't exist yet — start with empty tree
		return tfs, nil
	}

	tfs.baseRef = r.Hash()
	commit, err := repo.CommitObject(r.Hash())
	if err != nil {
		return nil, fmt.Errorf("read commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("read tree: %w", err)
	}
	tfs.base = tree
	return tfs, nil
}

// Repo returns the underlying go-git repository.
func (t *TreeFS) Repo() *git.Repository {
	return t.repo
}

// Refresh reloads the TreeFS from the current ref, discarding any pending
// overlay. Use this when another process may have committed to the same ref.
func (t *TreeFS) Refresh() error {
	t.overlay = make(map[string][]byte)
	t.dirs = make(map[string]bool)

	r, err := t.repo.Reference(t.ref, true)
	if err != nil {
		t.baseRef = plumbing.ZeroHash
		t.base = nil
		return nil
	}
	t.baseRef = r.Hash()
	return t.reloadBase()
}

// maybeRefresh reloads the base tree if the ref has moved and we have no
// pending overlay changes. This makes reads transparent when another TreeFS
// instance (or process) has committed to the same ref.
func (t *TreeFS) maybeRefresh() {
	if len(t.overlay) > 0 || len(t.dirs) > 0 {
		return
	}
	r, err := t.repo.Reference(t.ref, true)
	if err != nil {
		return
	}
	if r.Hash() == t.baseRef {
		return
	}
	t.baseRef = r.Hash()
	t.reloadBase()
}

// ReadFile reads the contents of a file at the given path.
func (t *TreeFS) ReadFile(p string) ([]byte, error) {
	t.maybeRefresh()
	p = clean(p)

	// Check overlay first
	if data, ok := t.overlay[p]; ok {
		if data == nil {
			return nil, fmt.Errorf("file not found: %s", p)
		}
		return append([]byte(nil), data...), nil
	}

	// Fall through to base tree
	if t.base == nil {
		return nil, fmt.Errorf("file not found: %s", p)
	}
	entry, err := t.base.FindEntry(p)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", p)
	}
	if entry.Mode == filemode.Dir {
		return nil, fmt.Errorf("is a directory: %s", p)
	}
	blob, err := t.repo.BlobObject(entry.Hash)
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// WriteFile writes data to the given path. Parent directories are created
// implicitly.
func (t *TreeFS) WriteFile(p string, data []byte) error {
	p = clean(p)
	if p == "" {
		return fmt.Errorf("empty path")
	}
	// Store a copy. Use non-nil empty slice for zero-length data so it
	// isn't confused with nil (which means "deleted" in the overlay).
	stored := make([]byte, len(data))
	copy(stored, data)
	t.overlay[p] = stored
	// Ensure parent directories exist
	dir := path.Dir(p)
	for dir != "." && dir != "" {
		t.dirs[dir] = true
		dir = path.Dir(dir)
	}
	return nil
}

// Remove removes a file at the given path.
func (t *TreeFS) Remove(p string) error {
	p = clean(p)
	// Mark as deleted in overlay (nil value = deleted)
	t.overlay[p] = nil
	return nil
}

// ReadDir lists entries in a directory.
func (t *TreeFS) ReadDir(p string) ([]DirEntry, error) {
	t.maybeRefresh()
	p = clean(p)

	entries := make(map[string]DirEntry)

	// Gather from base tree
	if t.base != nil {
		if p == "." || p == "" {
			// Root level
			for _, e := range t.base.Entries {
				entries[e.Name] = DirEntry{
					name:  e.Name,
					isDir: e.Mode == filemode.Dir,
				}
			}
		} else {
			subtree, err := t.base.Tree(p)
			if err == nil {
				for _, e := range subtree.Entries {
					entries[e.Name] = DirEntry{
						name:  e.Name,
						isDir: e.Mode == filemode.Dir,
					}
				}
			}
		}
	}

	// Apply overlay
	prefix := p + "/"
	if p == "." || p == "" {
		prefix = ""
	}

	for overlayPath, data := range t.overlay {
		if prefix == "" {
			// Root-level: check for direct children
			if !strings.Contains(overlayPath, "/") {
				if data == nil {
					delete(entries, overlayPath)
				} else {
					entries[overlayPath] = DirEntry{name: overlayPath, isDir: false}
				}
			} else {
				// It's a nested path — the first segment is a directory at root
				topDir := strings.SplitN(overlayPath, "/", 2)[0]
				if _, exists := entries[topDir]; !exists {
					entries[topDir] = DirEntry{name: topDir, isDir: true}
				}
			}
		} else if strings.HasPrefix(overlayPath, prefix) {
			rest := overlayPath[len(prefix):]
			if !strings.Contains(rest, "/") {
				// Direct child
				if data == nil {
					delete(entries, rest)
				} else {
					entries[rest] = DirEntry{name: rest, isDir: false}
				}
			} else {
				// Nested — first segment is a subdirectory
				subDir := strings.SplitN(rest, "/", 2)[0]
				if _, exists := entries[subDir]; !exists {
					entries[subDir] = DirEntry{name: subDir, isDir: true}
				}
			}
		}
	}

	// Also include explicitly created directories
	for dirPath := range t.dirs {
		if prefix == "" {
			if !strings.Contains(dirPath, "/") {
				if _, exists := entries[dirPath]; !exists {
					entries[dirPath] = DirEntry{name: dirPath, isDir: true}
				}
			} else {
				topDir := strings.SplitN(dirPath, "/", 2)[0]
				if _, exists := entries[topDir]; !exists {
					entries[topDir] = DirEntry{name: topDir, isDir: true}
				}
			}
		} else if strings.HasPrefix(dirPath, prefix) {
			rest := dirPath[len(prefix):]
			if !strings.Contains(rest, "/") {
				if _, exists := entries[rest]; !exists {
					entries[rest] = DirEntry{name: rest, isDir: true}
				}
			}
		}
	}

	var result []DirEntry
	for _, e := range entries {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].name < result[j].name
	})
	return result, nil
}

// MkdirAll records that a directory (and its parents) should exist.
func (t *TreeFS) MkdirAll(p string) error {
	p = clean(p)
	for p != "." && p != "" {
		t.dirs[p] = true
		p = path.Dir(p)
	}
	return nil
}

// Stat returns file info for the given path.
func (t *TreeFS) Stat(p string) (FileInfo, error) {
	t.maybeRefresh()
	p = clean(p)

	// Check overlay for files
	if data, ok := t.overlay[p]; ok {
		if data == nil {
			return FileInfo{}, fmt.Errorf("not found: %s", p)
		}
		return FileInfo{name: path.Base(p), size: int64(len(data)), isDir: false}, nil
	}

	// Check explicit dirs — but only if there's at least one non-deleted entry
	if t.dirs[p] {
		if t.dirHasContent(p) {
			return FileInfo{name: path.Base(p), isDir: true}, nil
		}
	}

	// Check base tree
	if t.base != nil {
		// Check if it's a file
		entry, err := t.base.FindEntry(p)
		if err == nil {
			if entry.Mode == filemode.Dir {
				return FileInfo{name: path.Base(p), isDir: true}, nil
			}
			blob, err := t.repo.BlobObject(entry.Hash)
			if err == nil {
				return FileInfo{name: path.Base(p), size: blob.Size, isDir: false}, nil
			}
		}
		// Check if it's a subtree
		_, err = t.base.Tree(p)
		if err == nil {
			return FileInfo{name: path.Base(p), isDir: true}, nil
		}
	}

	// Check if any overlay entry implies this directory exists
	prefix := p + "/"
	for overlayPath, data := range t.overlay {
		if strings.HasPrefix(overlayPath, prefix) && data != nil {
			return FileInfo{name: path.Base(p), isDir: true}, nil
		}
	}

	return FileInfo{}, fmt.Errorf("not found: %s", p)
}

// dirHasContent returns true if a directory in the dirs map should be reported
// as existing. A dir is considered empty only if every entry under it has been
// explicitly deleted in the overlay.
func (t *TreeFS) dirHasContent(p string) bool {
	prefix := p + "/"
	hasOverlayEntries := false
	for op, data := range t.overlay {
		if strings.HasPrefix(op, prefix) {
			hasOverlayEntries = true
			if data != nil {
				return true // at least one non-deleted file
			}
		}
	}
	if hasOverlayEntries {
		// All overlay entries are deletions — check if base has surviving files
		if t.base != nil {
			if _, err := t.base.Tree(p); err == nil {
				entries, _ := t.ReadDir(p)
				return len(entries) > 0
			}
		}
		return false // all entries deleted, no base content
	}
	// No overlay entries at all — dir was just MkdirAll'd, report as existing
	return true
}

// Commit materializes all pending changes into a git commit and updates the
// ref atomically. Returns an error if the ref has moved since Open (CAS).
func (t *TreeFS) Commit(msg string) error {
	if len(t.overlay) == 0 {
		return nil // nothing to commit
	}

	storer := t.repo.Storer

	// Build the new tree from base + overlay
	newTree, err := t.buildTree(storer)
	if err != nil {
		return fmt.Errorf("build tree: %w", err)
	}

	// Create commit
	commit := &object.Commit{
		Author: object.Signature{
			Name:  "beadwork",
			Email: "beadwork@localhost",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "beadwork",
			Email: "beadwork@localhost",
			When:  time.Now(),
		},
		Message:  msg,
		TreeHash: newTree,
	}

	// Set parent if we have a base commit
	if !t.baseRef.IsZero() {
		commit.ParentHashes = []plumbing.Hash{t.baseRef}
	}

	obj := storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return fmt.Errorf("encode commit: %w", err)
	}
	commitHash, err := storer.SetEncodedObject(obj)
	if err != nil {
		return fmt.Errorf("store commit: %w", err)
	}

	// CAS: update ref only if it still points to our base
	return t.casUpdateRef(commitHash)
}

// casUpdateRef atomically updates the ref to point to newHash, but only if
// the ref currently points to t.baseRef (or doesn't exist if baseRef is zero).
func (t *TreeFS) casUpdateRef(newHash plumbing.Hash) error {
	storer := t.repo.Storer

	// Read current ref
	currentRef, err := storer.Reference(t.ref)
	if err != nil {
		// Ref doesn't exist — only valid if we started with no base
		if t.baseRef.IsZero() {
			newRef := plumbing.NewHashReference(t.ref, newHash)
			if err := storer.SetReference(newRef); err != nil {
				return fmt.Errorf("create ref: %w", err)
			}
			t.baseRef = newHash
			t.overlay = make(map[string][]byte)
			t.dirs = make(map[string]bool)
			return t.reloadBase()
		}
		return fmt.Errorf("ref disappeared: %w", err)
	}

	// CAS check
	if currentRef.Hash() != t.baseRef {
		return fmt.Errorf("conflict: ref %s has moved (expected %s, got %s)",
			t.ref, t.baseRef.String()[:8], currentRef.Hash().String()[:8])
	}

	// Update ref
	newRef := plumbing.NewHashReference(t.ref, newHash)
	if err := storer.SetReference(newRef); err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	// Advance base
	t.baseRef = newHash
	t.overlay = make(map[string][]byte)
	t.dirs = make(map[string]bool)
	return t.reloadBase()
}

// reloadBase reloads the base tree from the current baseRef.
func (t *TreeFS) reloadBase() error {
	if t.baseRef.IsZero() {
		t.base = nil
		return nil
	}
	commit, err := t.repo.CommitObject(t.baseRef)
	if err != nil {
		return fmt.Errorf("reload commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("reload tree: %w", err)
	}
	t.base = tree
	return nil
}

// buildTree constructs a new git tree from the base tree + overlay.
func (t *TreeFS) buildTree(s storer.EncodedObjectStorer) (plumbing.Hash, error) {
	// Collect all file paths from base + overlay
	files := make(map[string][]byte) // path → content (nil means delete)

	// Start with base tree files
	if t.base != nil {
		if err := t.collectBaseFiles(t.base, "", files); err != nil {
			return plumbing.ZeroHash, err
		}
	}

	// Apply overlay
	for p, data := range t.overlay {
		if data == nil {
			delete(files, p)
		} else {
			files[p] = data
		}
	}

	// Build tree hierarchy bottom-up
	return t.writeTreeFromFiles(s, files)
}

// collectBaseFiles recursively reads all files from a base tree.
func (t *TreeFS) collectBaseFiles(tree *object.Tree, prefix string, out map[string][]byte) error {
	for _, entry := range tree.Entries {
		fullPath := entry.Name
		if prefix != "" {
			fullPath = prefix + "/" + entry.Name
		}

		if entry.Mode == filemode.Dir {
			subtree, err := t.repo.TreeObject(entry.Hash)
			if err != nil {
				return err
			}
			if err := t.collectBaseFiles(subtree, fullPath, out); err != nil {
				return err
			}
		} else {
			blob, err := t.repo.BlobObject(entry.Hash)
			if err != nil {
				return err
			}
			reader, err := blob.Reader()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				return err
			}
			out[fullPath] = data
		}
	}
	return nil
}

// writeTreeFromFiles builds a hierarchical git tree from a flat map of paths.
func (t *TreeFS) writeTreeFromFiles(s storer.EncodedObjectStorer, files map[string][]byte) (plumbing.Hash, error) {
	// Group files by top-level directory
	type treeNode struct {
		files    map[string][]byte // relative path → content
		subtrees map[string]bool   // subdirectory names
	}

	root := &treeNode{
		files:    make(map[string][]byte),
		subtrees: make(map[string]bool),
	}

	// Build the tree structure
	for p, data := range files {
		parts := strings.SplitN(p, "/", 2)
		if len(parts) == 1 {
			root.files[p] = data
		} else {
			root.subtrees[parts[0]] = true
		}
	}

	// Group files by subtree
	subtreeFiles := make(map[string]map[string][]byte)
	for p, data := range files {
		parts := strings.SplitN(p, "/", 2)
		if len(parts) == 2 {
			if subtreeFiles[parts[0]] == nil {
				subtreeFiles[parts[0]] = make(map[string][]byte)
			}
			subtreeFiles[parts[0]][parts[1]] = data
		}
	}

	var entries []object.TreeEntry

	// Add file entries
	for name, data := range root.files {
		blobHash, err := t.writeBlob(s, data)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		entries = append(entries, object.TreeEntry{
			Name: name,
			Mode: filemode.Regular,
			Hash: blobHash,
		})
	}

	// Recursively build subtrees
	for name, subFiles := range subtreeFiles {
		subHash, err := t.writeTreeFromFiles(s, subFiles)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		entries = append(entries, object.TreeEntry{
			Name: name,
			Mode: filemode.Dir,
			Hash: subHash,
		})
	}

	// Sort entries for deterministic trees
	sort.Slice(entries, func(i, j int) bool {
		// Git sorts directories with a trailing slash for comparison
		ni, nj := entries[i].Name, entries[j].Name
		if entries[i].Mode == filemode.Dir {
			ni += "/"
		}
		if entries[j].Mode == filemode.Dir {
			nj += "/"
		}
		return ni < nj
	})

	// Write tree object
	tree := &object.Tree{Entries: entries}
	obj := s.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode tree: %w", err)
	}
	return s.SetEncodedObject(obj)
}

// writeBlob stores a blob in the object store and returns its hash.
func (t *TreeFS) writeBlob(s storer.EncodedObjectStorer, data []byte) (plumbing.Hash, error) {
	obj := s.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(data)))
	w, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return plumbing.ZeroHash, err
	}
	w.Close()
	return s.SetEncodedObject(obj)
}

// Reset moves the ref to a new commit hash, discarding any pending overlay.
// Used by Sync to fast-forward or reset to the remote tip.
func (t *TreeFS) Reset(hash plumbing.Hash) error {
	newRef := plumbing.NewHashReference(t.ref, hash)
	if err := t.repo.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("reset ref: %w", err)
	}
	t.baseRef = hash
	t.overlay = make(map[string][]byte)
	t.dirs = make(map[string]bool)
	return t.reloadBase()
}

// CommitsBetween returns commits on localRef that are not ancestors of
// remoteRef, in oldest-first order. Used by Sync to find local-only commits.
func (t *TreeFS) CommitsBetween(localHash, remoteHash plumbing.Hash) ([]CommitInfo, error) {
	// Collect all ancestors of remoteHash into a set
	remoteSet := make(map[plumbing.Hash]bool)
	if !remoteHash.IsZero() {
		iter, err := t.repo.Log(&git.LogOptions{From: remoteHash})
		if err == nil {
			iter.ForEach(func(c *object.Commit) error {
				remoteSet[c.Hash] = true
				return nil
			})
		}
	}

	// Walk local commits, collecting those not in remoteSet
	var commits []CommitInfo
	iter, err := t.repo.Log(&git.LogOptions{From: localHash})
	if err != nil {
		return nil, fmt.Errorf("walk local commits: %w", err)
	}
	iter.ForEach(func(c *object.Commit) error {
		if remoteSet[c.Hash] {
			return storer.ErrStop
		}
		commits = append(commits, CommitInfo{
			Hash:    c.Hash.String(),
			Message: strings.TrimSpace(c.Message),
			Time:    c.Author.When,
			Author:  c.Author.Name,
		})
		return nil
	})

	// Reverse to oldest-first
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}
	return commits, nil
}

// AllCommits returns all commits on the tracked ref, newest-first.
func (t *TreeFS) AllCommits() ([]CommitInfo, error) {
	if t.baseRef.IsZero() {
		return nil, nil
	}
	var commits []CommitInfo
	iter, err := t.repo.Log(&git.LogOptions{From: t.baseRef})
	if err != nil {
		return nil, fmt.Errorf("walk commits: %w", err)
	}
	iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, CommitInfo{
			Hash:    c.Hash.String(),
			Message: strings.TrimSpace(c.Message),
			Time:    c.Author.When,
			Author:  c.Author.Name,
		})
		return nil
	})
	return commits, nil
}

// RefHash returns the current hash of the tracked ref.
func (t *TreeFS) RefHash() plumbing.Hash {
	return t.baseRef
}

// HasRef returns true if the tracked ref exists.
func (t *TreeFS) HasRef() bool {
	return !t.baseRef.IsZero()
}

// RefName returns the reference name this TreeFS tracks.
func (t *TreeFS) RefName() plumbing.ReferenceName {
	return t.ref
}

// LookupRef looks up a reference by name.
func (t *TreeFS) LookupRef(name string) (plumbing.Hash, error) {
	ref, err := t.repo.Reference(plumbing.ReferenceName(name), true)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return ref.Hash(), nil
}

// HasRemotes returns true if the repo has at least one remote configured.
func (t *TreeFS) HasRemotes() (bool, error) {
	remotes, err := t.repo.Remotes()
	if err != nil {
		return false, err
	}
	return len(remotes) > 0, nil
}

// SetRef directly sets a reference. Used by Init to create tracking branches.
func (t *TreeFS) SetRef(name string, hash plumbing.Hash) error {
	ref := plumbing.NewHashReference(plumbing.ReferenceName(name), hash)
	return t.repo.Storer.SetReference(ref)
}

// DeleteRef removes a reference.
func (t *TreeFS) DeleteRef(name string) error {
	return t.repo.Storer.RemoveReference(plumbing.ReferenceName(name))
}

// CommitInfo holds a commit hash and message.
type CommitInfo struct {
	Hash    string
	Message string
	Time    time.Time
	Author  string
}

// MergeCommit attempts a 3-way merge between the local tree, remote tree,
// and their common ancestor (base). If all changes are non-conflicting, it
// creates a commit with the merged tree on top of remoteHash and updates
// the local ref. Returns true if the merge succeeded, false if there were
// conflicts.
func (t *TreeFS) MergeCommit(localHash, remoteHash plumbing.Hash, localCommitMsgs []string) (bool, error) {
	// Find common ancestor by walking both commit histories
	baseHash, err := t.findMergeBase(localHash, remoteHash)
	if err != nil {
		return false, err
	}

	// Collect files from all three trees
	baseFiles := make(map[string][]byte)
	localFiles := make(map[string][]byte)
	remoteFiles := make(map[string][]byte)

	if !baseHash.IsZero() {
		if c, err := t.repo.CommitObject(baseHash); err == nil {
			if tree, err := c.Tree(); err == nil {
				t.collectBaseFiles(tree, "", baseFiles)
			}
		}
	}

	if c, err := t.repo.CommitObject(localHash); err == nil {
		if tree, err := c.Tree(); err == nil {
			t.collectBaseFiles(tree, "", localFiles)
		}
	}

	if c, err := t.repo.CommitObject(remoteHash); err == nil {
		if tree, err := c.Tree(); err == nil {
			t.collectBaseFiles(tree, "", remoteFiles)
		}
	}

	// 3-way merge
	merged := make(map[string][]byte)

	// Collect all paths
	allPaths := make(map[string]bool)
	for p := range baseFiles {
		allPaths[p] = true
	}
	for p := range localFiles {
		allPaths[p] = true
	}
	for p := range remoteFiles {
		allPaths[p] = true
	}

	for p := range allPaths {
		baseData, inBase := baseFiles[p]
		localData, inLocal := localFiles[p]
		remoteData, inRemote := remoteFiles[p]

		localChanged := !bytesEqual(baseData, localData) || (inBase != inLocal)
		remoteChanged := !bytesEqual(baseData, remoteData) || (inBase != inRemote)

		switch {
		case !localChanged && !remoteChanged:
			// No change on either side
			if inBase {
				merged[p] = baseData
			}
		case localChanged && !remoteChanged:
			// Only local changed
			if inLocal {
				merged[p] = localData
			}
			// else: local deleted, keep it deleted
		case !localChanged && remoteChanged:
			// Only remote changed
			if inRemote {
				merged[p] = remoteData
			}
			// else: remote deleted
		case localChanged && remoteChanged:
			// Both changed
			if bytesEqual(localData, remoteData) && inLocal == inRemote {
				// Same change on both sides
				if inLocal {
					merged[p] = localData
				}
			} else {
				// Conflict
				return false, nil
			}
		}
	}

	// Build merged tree and commit on top of remote
	treeHash, err := t.writeTreeFromFiles(t.repo.Storer, merged)
	if err != nil {
		return false, fmt.Errorf("build merged tree: %w", err)
	}

	// Create commit(s) — replay local commits on top of remote
	// For simplicity, create one commit per local intent
	parentHash := remoteHash
	for _, msg := range localCommitMsgs {
		commit := &object.Commit{
			Author: object.Signature{
				Name: "beadwork", Email: "beadwork@localhost", When: time.Now(),
			},
			Committer: object.Signature{
				Name: "beadwork", Email: "beadwork@localhost", When: time.Now(),
			},
			Message:      msg,
			TreeHash:     treeHash,
			ParentHashes: []plumbing.Hash{parentHash},
		}
		obj := t.repo.Storer.NewEncodedObject()
		if err := commit.Encode(obj); err != nil {
			return false, err
		}
		hash, err := t.repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return false, err
		}
		parentHash = hash
	}

	// Update ref to the final commit
	newRef := plumbing.NewHashReference(t.ref, parentHash)
	if err := t.repo.Storer.SetReference(newRef); err != nil {
		return false, err
	}
	t.baseRef = parentHash
	t.overlay = make(map[string][]byte)
	t.dirs = make(map[string]bool)
	if err := t.reloadBase(); err != nil {
		return false, err
	}

	return true, nil
}

// findMergeBase finds the common ancestor of two commits.
func (t *TreeFS) findMergeBase(a, b plumbing.Hash) (plumbing.Hash, error) {
	// Collect all ancestors of b
	bAncestors := make(map[plumbing.Hash]bool)
	iter, err := t.repo.Log(&git.LogOptions{From: b})
	if err != nil {
		return plumbing.ZeroHash, err
	}
	iter.ForEach(func(c *object.Commit) error {
		bAncestors[c.Hash] = true
		return nil
	})

	// Walk a's ancestors, first one in bAncestors is the merge base
	iter, err = t.repo.Log(&git.LogOptions{From: a})
	if err != nil {
		return plumbing.ZeroHash, err
	}
	var base plumbing.Hash
	iter.ForEach(func(c *object.Commit) error {
		if bAncestors[c.Hash] {
			base = c.Hash
			return storer.ErrStop
		}
		return nil
	})
	return base, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// clean normalizes a path: removes leading/trailing slashes, resolves . and ..
func clean(p string) string {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	if p == "." {
		return ""
	}
	return p
}
