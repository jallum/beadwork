package issue

import (
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const base36 = "0123456789abcdefghijklmnopqrstuvwxyz"

// validateExplicitID checks that the given ID is well-formed and unique.
func (s *Store) validateExplicitID(id string) (string, error) {
	// Format check: no whitespace, non-empty.
	if strings.ContainsAny(id, " \t\n\r") {
		return "", fmt.Errorf("invalid ID %q: must not contain whitespace", id)
	}
	// Uniqueness check.
	existing := s.ExistingIDs()
	if existing[id] {
		return "", fmt.Errorf("ID %q already exists", id)
	}
	return id, nil
}

func (s *Store) generateID() (string, error) {
	existing := s.ExistingIDs()
	retries := s.IDRetries
	if retries <= 0 {
		retries = 10
	}
	rr := s.RandReader
	if rr == nil {
		rr = rand.Reader
	}
	for length := 3; length <= 8; length++ {
		for attempt := 0; attempt < retries; attempt++ {
			b := make([]byte, length)
			if _, err := io.ReadFull(rr, b); err != nil {
				return "", err
			}
			var suffix strings.Builder
			for _, v := range b {
				suffix.WriteByte(base36[int(v)%36])
			}
			id := s.Prefix + "-" + suffix.String()
			if !existing[id] {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after trying lengths 3-8")
}

// generateChildID returns the next sequential child ID for the given parent.
// Format: parentID.N where N is max(existing child numbers) + 1.
func (s *Store) generateChildID(parentID string) (string, error) {
	existing := s.ExistingIDs()
	prefix := parentID + "."
	maxN := 0
	for id := range existing {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		rest := id[len(prefix):]
		n, err := strconv.Atoi(rest)
		if err != nil {
			continue // skip grandchildren (e.g. "parent.1.1")
		}
		if n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("%s.%d", parentID, maxN+1), nil
}

// ensureIDSet lazily populates the ID set from disk on first access.
func (s *Store) ensureIDSet() {
	if s.idSet != nil {
		return
	}
	s.idSet = make(map[string]bool)
	entries, err := s.FS.ReadDir("issues")
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			s.idSet[strings.TrimSuffix(name, ".json")] = true
		}
	}
}

// trackID adds an ID to the lazy set (called after create/import).
func (s *Store) trackID(id string) {
	if s.idSet != nil {
		s.idSet[id] = true
	}
}

// untrackID removes an ID from the lazy set (called after delete).
func (s *Store) untrackID(id string) {
	if s.idSet != nil {
		delete(s.idSet, id)
	}
}

func (s *Store) ExistingIDs() map[string]bool {
	s.ensureIDSet()
	// Return a copy so callers can't mutate the internal set.
	ids := make(map[string]bool, len(s.idSet))
	for k := range s.idSet {
		ids[k] = true
	}
	return ids
}

func (s *Store) resolveID(partial string) (string, error) {
	s.ensureIDSet()
	// Exact match is the fast path.
	if s.idSet[partial] {
		return partial, nil
	}
	var matches []string
	for id := range s.idSet {
		if strings.HasPrefix(id, partial) || strings.HasSuffix(id, partial) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous ID %q: matches %s", partial, strings.Join(matches, ", "))
	}
	return "", fmt.Errorf("no issue found matching %q", partial)
}
