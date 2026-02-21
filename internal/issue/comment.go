package issue

import "time"

func (s *Store) Comment(id, text, author string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	issue.Comments = append(issue.Comments, Comment{
		Text:      text,
		Author:    author,
		Timestamp: now,
	})
	issue.UpdatedAt = now
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}
