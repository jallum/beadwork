package repo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Migration defines a schema migration from version N to N+1.
type Migration struct {
	Description string
	Apply       func(r *Repo) error
}

// Migrations is the ordered list of migrations.
// Migrations[0] migrates version 0 → 1, Migrations[1] migrates 1 → 2, etc.
var Migrations = []Migration{
	{
		Description: "add version marker",
		Apply: func(r *Repo) error {
			// v0 → v1: no data changes, just stamps the version key.
			return nil
		},
	},
	{
		Description: "shift priority scale from 1-5 to 0-4",
		Apply: func(r *Repo) error {
			// v1 → v2: decrement all issue priorities by 1.
			entries, err := r.tfs.ReadDir("issues")
			if err != nil {
				// No issues directory is fine — nothing to migrate.
				return nil
			}
			for _, e := range entries {
				name := e.Name()
				if !strings.HasSuffix(name, ".json") || name == ".gitkeep" {
					continue
				}
				data, err := r.tfs.ReadFile("issues/" + name)
				if err != nil {
					return fmt.Errorf("read %s: %w", name, err)
				}
				var raw map[string]json.RawMessage
				if err := json.Unmarshal(data, &raw); err != nil {
					return fmt.Errorf("parse %s: %w", name, err)
				}
				if pRaw, ok := raw["priority"]; ok {
					var p int
					if err := json.Unmarshal(pRaw, &p); err == nil && p > 0 {
						p--
						newP, _ := json.Marshal(p)
						raw["priority"] = newP
						out, err := json.MarshalIndent(raw, "", "  ")
						if err != nil {
							return fmt.Errorf("marshal %s: %w", name, err)
						}
						out = append(out, '\n')
						if err := r.tfs.WriteFile("issues/"+name, out); err != nil {
							return fmt.Errorf("write %s: %w", name, err)
						}
					}
				}
			}
			return nil
		},
	},
}

// Upgrade migrates the repo from its current version to CurrentVersion.
// Returns (fromVersion, toVersion, error). All migrations are applied
// in a single atomic commit.
func (r *Repo) Upgrade() (int, int, error) {
	from := r.Version()
	if from >= CurrentVersion {
		return from, from, nil
	}
	if from >= len(Migrations) {
		return from, from, fmt.Errorf("repo version %d has no migration path", from)
	}

	for v := from; v < CurrentVersion; v++ {
		if err := Migrations[v].Apply(r); err != nil {
			r.tfs.Refresh()
			return from, v, fmt.Errorf("migration v%d → v%d (%s) failed: %w",
				v, v+1, Migrations[v].Description, err)
		}
	}

	if err := r.SetConfig("version", strconv.Itoa(CurrentVersion)); err != nil {
		r.tfs.Refresh()
		return from, from, fmt.Errorf("set version: %w", err)
	}

	msg := fmt.Sprintf("upgrade repo v%d -> v%d", from, CurrentVersion)
	if err := r.tfs.Commit(msg); err != nil {
		return from, from, fmt.Errorf("commit upgrade: %w", err)
	}

	return from, CurrentVersion, nil
}
