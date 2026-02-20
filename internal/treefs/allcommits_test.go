package treefs

import (
	"testing"
)

func TestAllCommitsReturnsPopulatedFields(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Make a second commit so we have at least 2
	tfs.WriteFile("file1.txt", []byte("hello"))
	if err := tfs.Commit("bw-1234: add file1"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	commits, err := tfs.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits, got %d", len(commits))
	}

	for i, c := range commits {
		if c.Hash == "" {
			t.Errorf("commit[%d].Hash is empty", i)
		}
		if c.Message == "" {
			t.Errorf("commit[%d].Message is empty", i)
		}
		if c.Time.IsZero() {
			t.Errorf("commit[%d].Time is zero", i)
		}
		if c.Author == "" {
			t.Errorf("commit[%d].Author is empty", i)
		}
	}
}

func TestAllCommitsNewestFirst(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tfs.WriteFile("a.txt", []byte("a"))
	if err := tfs.Commit("first extra"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	tfs.WriteFile("b.txt", []byte("b"))
	if err := tfs.Commit("second extra"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	commits, err := tfs.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}

	// newest first means commits[0] should be "second extra"
	if commits[0].Message != "second extra" {
		t.Errorf("commits[0].Message = %q, want 'second extra'", commits[0].Message)
	}

	// Verify times are non-increasing
	for i := 1; i < len(commits); i++ {
		if commits[i].Time.After(commits[i-1].Time) {
			t.Errorf("commit[%d].Time (%v) is after commit[%d].Time (%v)", i, commits[i].Time, i-1, commits[i-1].Time)
		}
	}
}

func TestAllCommitsEmptyRepo(t *testing.T) {
	dir := initEmptyRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	commits, err := tfs.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits on empty repo, got %d", len(commits))
	}
}
