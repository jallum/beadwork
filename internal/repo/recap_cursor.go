package repo

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const recapCursorRef = "refs/beadwork/recap-cursor"

// RecapCursor returns the commit hash stored in the recap cursor ref,
// or "" if no cursor has been set. The ref is local-only (never pushed).
func (r *Repo) RecapCursor() string {
	path := filepath.Join(r.GitDir, recapCursorRef)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// LastRecapAt returns the mtime of the recap cursor ref file, which
// records when recap last ran successfully (advancing the cursor or
// touching the ref on a no-event run). Returns the zero time if the
// ref doesn't exist.
func (r *Repo) LastRecapAt() time.Time {
	path := filepath.Join(r.GitDir, recapCursorRef)
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// SetRecapCursor writes a commit hash to the recap cursor ref.
// Creates parent directories as needed.
func (r *Repo) SetRecapCursor(hash string) error {
	path := filepath.Join(r.GitDir, recapCursorRef)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hash+"\n"), 0644)
}

// TouchRecapCursor bumps the mtime of the recap cursor ref to the
// current time without changing its content. Used to record that recap
// ran successfully even when no new events advanced the cursor, so that
// the "since last recap" header reflects the most recent run rather
// than the last cursor advance.
func (r *Repo) TouchRecapCursor() error {
	path := filepath.Join(r.GitDir, recapCursorRef)
	now := time.Now()
	return os.Chtimes(path, now, now)
}
