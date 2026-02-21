package issue

import (
	"encoding/json"
	"fmt"
)

func (s *Store) readIssue(id string) (*Issue, error) {
	if s.cache != nil {
		if iss, ok := s.cache[id]; ok {
			return iss, nil
		}
	}
	data, err := s.FS.ReadFile("issues/" + id + ".json")
	if err != nil {
		return nil, fmt.Errorf("issue %s not found", id)
	}
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("corrupt issue %s: %w", id, err)
	}
	if s.cache == nil {
		s.cache = make(map[string]*Issue)
	}
	s.cache[id] = &issue
	return &issue, nil
}

func (s *Store) writeIssue(issue *Issue) error {
	data, err := json.MarshalIndent(issue, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := s.FS.WriteFile("issues/"+issue.ID+".json", data); err != nil {
		return err
	}
	if s.cache == nil {
		s.cache = make(map[string]*Issue)
	}
	s.cache[issue.ID] = issue
	return nil
}

func (s *Store) setStatus(id, status string) error {
	s.FS.MkdirAll("status/" + status)
	return s.FS.WriteFile("status/"+status+"/"+id, []byte{})
}

func (s *Store) moveStatus(id, from, to string) error {
	s.FS.Remove("status/" + from + "/" + id)
	return s.setStatus(id, to)
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeStr(slice []string, s string) []string {
	var result []string
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	if result == nil {
		return []string{}
	}
	return result
}
