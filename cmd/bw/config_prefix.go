package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/registry"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/treefs"
)

// renamePrefix changes the repo's prefix in .bwconfig and migrates
// open/in_progress issues (plus their descendant subtrees) to the new
// prefix. Closed issues keep their old prefix as a historical record;
// references to renamed issues from closed-issue JSON are updated.
//
// All changes land as a single commit.
func renamePrefix(store *issue.Store, r *repo.Repo, newPrefix string, w Writer, cfg *config.Config) error {
	if err := repo.ValidatePrefix(newPrefix); err != nil {
		return err
	}
	if newPrefix == "" {
		return fmt.Errorf("prefix cannot be empty")
	}
	oldPrefix := r.Prefix
	if oldPrefix == newPrefix {
		return fmt.Errorf("prefix is already %q", newPrefix)
	}

	all, err := loadAllIssues(store, r.TreeFS())
	if err != nil {
		return err
	}

	// Build rename map: top-level open/in_progress with the old prefix,
	// plus their entire descendant subtrees (children's IDs structurally
	// depend on the parent's ID).
	renameMap := buildRenameMap(all, oldPrefix, newPrefix)

	// Update .bwconfig: new prefix + alias bookkeeping.
	//
	// The old prefix normally becomes an alias so closed issues with
	// the old prefix remain reachable cross-repo. BUT if another repo
	// already claims the old prefix (as its primary or an alias),
	// adding an alias here would create or prolong a collision — and
	// the user is often renaming specifically to escape one. Skip the
	// alias in that case and surface a note.
	collisionRepo := ""
	if oldPrefix != "" {
		collisionRepo = otherRepoClaiming(r, oldPrefix, cfg)
	}
	existingAliases := r.Aliases()
	var newAliases []string
	if collisionRepo != "" {
		// Don't alias a prefix owned elsewhere. Existing aliases that
		// don't conflict with newPrefix stay.
		newAliases = mergeAliases(existingAliases, "", newPrefix)
		fmt.Fprintf(os.Stderr,
			"note: prefix %q is also used by %s — not adding as alias for this repo\n",
			oldPrefix, collisionRepo)
	} else {
		newAliases = mergeAliases(existingAliases, oldPrefix, newPrefix)
	}
	if err := r.SetAliases(newAliases); err != nil {
		return err
	}
	if err := r.SetConfig("prefix", newPrefix); err != nil {
		return err
	}

	if len(renameMap) == 0 {
		intent := fmt.Sprintf("config prefix=%s", newPrefix)
		if err := r.Commit(intent); err != nil {
			return fmt.Errorf("commit failed: %w", err)
		}
		fmt.Fprintf(w, "prefix=%s (no open issues to rename)\n", newPrefix)
		return nil
	}

	fs := r.TreeFS()

	// 1. Rewrite issue JSONs.
	for _, iss := range all {
		newID, renamed := renameMap[iss.ID]

		// Update reference fields whether or not this issue is itself renamed.
		dirty := false
		if v, ok := renameMap[iss.Parent]; ok {
			iss.Parent = v
			dirty = true
		}
		for i, id := range iss.Blocks {
			if v, ok := renameMap[id]; ok {
				iss.Blocks[i] = v
				dirty = true
			}
		}
		for i, id := range iss.BlockedBy {
			if v, ok := renameMap[id]; ok {
				iss.BlockedBy[i] = v
				dirty = true
			}
		}

		if renamed {
			// Remove the old JSON file; write under the new ID.
			fs.Remove("issues/" + iss.ID + ".json")
			iss.ID = newID
			if err := writeJSON(fs, "issues/"+newID+".json", iss); err != nil {
				return err
			}
		} else if dirty {
			if err := writeJSON(fs, "issues/"+iss.ID+".json", iss); err != nil {
				return err
			}
		}
	}

	// 2. Move status markers for renamed issues.
	for _, status := range []string{"open", "in_progress", "closed", "deferred"} {
		entries, _ := fs.ReadDir("status/" + status)
		for _, e := range entries {
			if e.Name() == ".gitkeep" {
				continue
			}
			if newID, ok := renameMap[e.Name()]; ok {
				fs.Remove("status/" + status + "/" + e.Name())
				fs.WriteFile("status/"+status+"/"+newID, []byte{})
			}
		}
	}

	// 3. Move label markers for renamed issues.
	labelDirs, _ := fs.ReadDir("labels")
	for _, ld := range labelDirs {
		if !ld.IsDir() {
			continue
		}
		entries, _ := fs.ReadDir("labels/" + ld.Name())
		for _, e := range entries {
			if e.Name() == ".gitkeep" {
				continue
			}
			if newID, ok := renameMap[e.Name()]; ok {
				fs.Remove("labels/" + ld.Name() + "/" + e.Name())
				fs.WriteFile("labels/"+ld.Name()+"/"+newID, []byte{})
			}
		}
	}

	// 4. Move block edges. blocks/<blocker>/<blocked> — either side may
	//    have been renamed.
	blockerDirs, _ := fs.ReadDir("blocks")
	for _, bd := range blockerDirs {
		if !bd.IsDir() {
			continue
		}
		blocker := bd.Name()
		newBlocker, blockerRenamed := renameMap[blocker]
		entries, _ := fs.ReadDir("blocks/" + blocker)
		for _, e := range entries {
			if e.Name() == ".gitkeep" {
				continue
			}
			blocked := e.Name()
			newBlocked, blockedRenamed := renameMap[blocked]

			if !blockerRenamed && !blockedRenamed {
				continue
			}
			finalBlocker := blocker
			if blockerRenamed {
				finalBlocker = newBlocker
			}
			finalBlocked := blocked
			if blockedRenamed {
				finalBlocked = newBlocked
			}
			fs.Remove("blocks/" + blocker + "/" + blocked)
			fs.MkdirAll("blocks/" + finalBlocker)
			fs.WriteFile("blocks/"+finalBlocker+"/"+finalBlocked, []byte{})
		}
	}

	// 5. Commit everything atomically — .bwconfig (prefix + aliases),
	//    renamed issue JSONs, status/label/block edge files.
	intent := fmt.Sprintf("config prefix=%s (renamed %d issue(s))", newPrefix, len(renameMap))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if collisionRepo != "" {
		fmt.Fprintf(w, "prefix=%s (renamed %d open/in_progress issue(s); old prefix %q NOT aliased — claimed by %s)\n",
			newPrefix, len(renameMap), oldPrefix, collisionRepo)
	} else {
		fmt.Fprintf(w, "prefix=%s (renamed %d open/in_progress issue(s); %q kept as alias)\n",
			newPrefix, len(renameMap), oldPrefix)
	}
	return nil
}

// otherRepoClaiming returns the path of another registered repo that
// already has `prefix` as its primary or one of its aliases, or "" if
// none does. Used by the renamer to decide whether to add `prefix` to
// this repo's alias list.
func otherRepoClaiming(r *repo.Repo, prefix string, cfg *config.Config) string {
	selfPath, _ := filepath.EvalSymlinks(r.RepoDir())
	for _, path := range registry.ResolveAll(cfg, prefix) {
		canon, _ := filepath.EvalSymlinks(path)
		if canon != selfPath {
			return path
		}
	}
	return ""
}

// mergeAliases produces the new alias list after a prefix rename.
// Adds oldPrefix unless it's already present or empty; removes newPrefix
// if it was previously an alias (rename back to a former prefix).
func mergeAliases(existing []string, oldPrefix, newPrefix string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s == "" || s == newPrefix || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, a := range existing {
		add(a)
	}
	add(oldPrefix)
	return out
}

// buildRenameMap returns oldID → newID for every issue that should be
// renamed. The renamed set is: top-level (no parent, no '.') open or
// in_progress issues whose ID begins with oldPrefix+"-", plus all their
// descendants (regardless of status — IDs structurally include the
// parent's ID).
func buildRenameMap(all []*issue.Issue, oldPrefix, newPrefix string) map[string]string {
	renameMap := make(map[string]string)
	roots := []string{}

	for _, iss := range all {
		if iss.Status != "open" && iss.Status != "in_progress" {
			continue
		}
		if iss.Parent != "" {
			continue
		}
		if strings.Contains(iss.ID, ".") {
			continue
		}
		if !strings.HasPrefix(iss.ID, oldPrefix+"-") {
			continue
		}
		newID := newPrefix + iss.ID[len(oldPrefix):]
		renameMap[iss.ID] = newID
		roots = append(roots, iss.ID)
	}

	// Expand to descendants. Issue with ID like "<root>.<rest>" gets renamed.
	for _, iss := range all {
		for _, root := range roots {
			if strings.HasPrefix(iss.ID, root+".") {
				newID := renameMap[root] + iss.ID[len(root):]
				renameMap[iss.ID] = newID
				break
			}
		}
	}

	return renameMap
}

// loadAllIssues reads every issue JSON from issues/.
func loadAllIssues(_ *issue.Store, fs *treefs.TreeFS) ([]*issue.Issue, error) {
	entries, err := fs.ReadDir("issues")
	if err != nil {
		return nil, err
	}
	var out []*issue.Issue
	for _, e := range entries {
		if e.IsDir() || e.Name() == ".gitkeep" {
			continue
		}
		data, err := fs.ReadFile("issues/" + e.Name())
		if err != nil {
			continue
		}
		var iss issue.Issue
		if err := json.Unmarshal(data, &iss); err != nil {
			continue
		}
		out = append(out, &iss)
	}
	return out, nil
}

func writeJSON(fs *treefs.TreeFS, path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fs.WriteFile(path, data)
}
