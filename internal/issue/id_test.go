package issue_test

import (
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestIDPrefixMatch(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Test issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	// Should resolve by suffix
	suffix := iss.ID[len("test-"):]
	got, err := env.Store.Get(suffix)
	if err != nil {
		t.Fatalf("Get by suffix %q: %v", suffix, err)
	}
	if got.ID != iss.ID {
		t.Errorf("resolved to %q, want %q", got.ID, iss.ID)
	}
}

func TestAmbiguousID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create issues with similar IDs that share a prefix
	for _, id := range []string{"test-ab01", "test-ab02"} {
		env.Store.Import(&issue.Issue{
			ID:        id,
			Title:     "Issue " + id,
			Status:    "open",
			Priority:  3,
			Type:      "task",
			Created:   "2026-01-01T00:00:00Z",
			Labels:    []string{},
			Blocks:    []string{},
			BlockedBy: []string{},
		})
	}
	env.CommitIntent("import two similar issues")

	// Searching for "test-ab0" should be ambiguous (matches both via prefix)
	_, err := env.Store.Get("test-ab0")
	if err == nil {
		t.Error("expected ambiguous ID error")
	}
	if err != nil && !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}

	// Exact match should work
	got, err := env.Store.Get("test-ab01")
	if err != nil {
		t.Fatalf("Get exact: %v", err)
	}
	if got.ID != "test-ab01" {
		t.Errorf("id = %q", got.ID)
	}
}

func TestMultipleIDsNeverCollide(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		iss, err := env.Store.Create("Issue", issue.CreateOpts{})
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		if ids[iss.ID] {
			t.Fatalf("duplicate ID: %s", iss.ID)
		}
		ids[iss.ID] = true
	}
}

func TestIDIsBase36(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	for i := 0; i < 20; i++ {
		iss, err := env.Store.Create("Base36 test", issue.CreateOpts{})
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		suffix := iss.ID[len("test-"):]
		for _, c := range suffix {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')) {
				t.Errorf("ID %q contains non-base36 char %q", iss.ID, string(c))
			}
		}
	}
}

func TestIDStartsAt3Chars(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Short ID", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	suffix := iss.ID[len("test-"):]
	if len(suffix) != 3 {
		t.Errorf("suffix %q length = %d, want 3", suffix, len(suffix))
	}
}

func TestIDDeterministicRand(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Inject a deterministic reader that always returns the same bytes.
	// Two creates with the same rand source should collide on the first
	// attempt at length 3, then succeed at length 3 on the second attempt
	// with different bytes.
	calls := 0
	env.Store.RandReader = readerFunc(func(b []byte) (int, error) {
		calls++
		for i := range b {
			b[i] = byte(calls) // deterministic, changes each call
		}
		return len(b), nil
	})

	iss1, err := env.Store.Create("First", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	iss2, err := env.Store.Create("Second", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if iss1.ID == iss2.ID {
		t.Errorf("expected different IDs, both got %q", iss1.ID)
	}
}

func TestIDAdaptiveLength(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Inject a reader that always returns zeros. Every attempt at a given
	// length produces the same ID, forcing a collision and length bump.
	env.Store.RandReader = readerFunc(func(b []byte) (int, error) {
		for i := range b {
			b[i] = 0
		}
		return len(b), nil
	})
	env.Store.IDRetries = 2

	// First create succeeds at length 3 (no collision yet).
	iss1, err := env.Store.Create("A", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	suffix1 := iss1.ID[len("test-"):]
	if len(suffix1) != 3 {
		t.Errorf("first suffix %q length = %d, want 3", suffix1, len(suffix1))
	}

	// Second create: all length-3 attempts collide (same bytes), bumps to 4.
	iss2, err := env.Store.Create("B", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	suffix2 := iss2.ID[len("test-"):]
	if len(suffix2) != 4 {
		t.Errorf("second suffix %q length = %d, want 4", suffix2, len(suffix2))
	}
}

// readerFunc adapts a function to io.Reader.
type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(b []byte) (int, error) { return f(b) }

