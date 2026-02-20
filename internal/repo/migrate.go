package repo

import (
	"fmt"
	"strconv"
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
