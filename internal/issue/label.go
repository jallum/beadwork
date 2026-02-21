package issue

import (
	"sort"
	"time"
)

func (s *Store) Label(id string, add, remove []string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	// Add labels
	for _, label := range add {
		s.FS.MkdirAll("labels/" + label)
		if err := s.FS.WriteFile("labels/"+label+"/"+id, []byte{}); err != nil {
			return nil, err
		}
		if !containsStr(issue.Labels, label) {
			issue.Labels = append(issue.Labels, label)
		}
	}

	// Remove labels
	for _, label := range remove {
		s.FS.Remove("labels/" + label + "/" + id)
		// Clean up empty label directory
		entries, _ := s.FS.ReadDir("labels/" + label)
		empty := true
		for _, e := range entries {
			if e.Name() != ".gitkeep" {
				empty = false
				break
			}
		}
		if empty {
			s.FS.Remove("labels/" + label + "/.gitkeep")
		}
		issue.Labels = removeStr(issue.Labels, label)
	}

	sort.Strings(issue.Labels)
	issue.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}
