package issue

import (
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const base36 = "0123456789abcdefghijklmnopqrstuvwxyz"

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

func (s *Store) ExistingIDs() map[string]bool {
	ids := make(map[string]bool)
	entries, err := s.FS.ReadDir("issues")
	if err != nil {
		return ids
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			ids[strings.TrimSuffix(name, ".json")] = true
		}
	}
	return ids
}

func (s *Store) resolveID(partial string) (string, error) {
	entries, err := s.FS.ReadDir("issues")
	if err != nil {
		return "", fmt.Errorf("cannot read issues: %w", err)
	}
	var matches []string
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if name == partial {
			return name, nil
		}
		if strings.HasPrefix(name, partial) || strings.HasSuffix(name, partial) {
			matches = append(matches, name)
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
