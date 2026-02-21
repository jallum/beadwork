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

// --- Lazy ID set tests (bw-ccl.4) ---

func TestExistingIDsCached(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	b, _ := env.Store.Create("Issue B", issue.CreateOpts{})
	env.CommitIntent("create issues")

	ids1 := env.Store.ExistingIDs()
	if !ids1[a.ID] || !ids1[b.ID] {
		t.Errorf("first call missing IDs: %v", ids1)
	}

	// Second call should return the same data (from cache)
	ids2 := env.Store.ExistingIDs()
	if !ids2[a.ID] || !ids2[b.ID] {
		t.Errorf("second call missing IDs: %v", ids2)
	}
}

func TestIDSetUpdatedOnCreate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Prime the ID set
	a, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	_ = env.Store.ExistingIDs()

	// Create another issue — should appear in the set without re-reading disk
	b, _ := env.Store.Create("Issue B", issue.CreateOpts{})
	env.CommitIntent("create issues")

	ids := env.Store.ExistingIDs()
	if !ids[a.ID] {
		t.Error("A should be in ID set")
	}
	if !ids[b.ID] {
		t.Error("B should be in ID set after create")
	}
}

func TestIDSetUpdatedOnDelete(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	env.CommitIntent("create issue")

	// Prime the ID set
	ids := env.Store.ExistingIDs()
	if !ids[a.ID] {
		t.Fatal("A should be in ID set")
	}

	// Delete the issue
	env.Store.Delete(a.ID)
	env.CommitIntent("delete issue")

	ids = env.Store.ExistingIDs()
	if ids[a.ID] {
		t.Error("A should be gone from ID set after delete")
	}
}

func TestIDSetUpdatedOnImport(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Prime the ID set
	_ = env.Store.ExistingIDs()

	env.Store.Import(&issue.Issue{
		ID:        "test-import1",
		Title:     "Imported",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2026-01-01T00:00:00Z",
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.CommitIntent("import issue")

	ids := env.Store.ExistingIDs()
	if !ids["test-import1"] {
		t.Error("imported ID should be in ID set")
	}
}

func TestClearCacheClearsIDSet(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	env.CommitIntent("create issue")

	// Prime the ID set
	ids := env.Store.ExistingIDs()
	if !ids[a.ID] {
		t.Fatal("A should be in ID set")
	}

	// Clear cache should force a re-read on next call
	env.Store.ClearCache()

	ids = env.Store.ExistingIDs()
	if !ids[a.ID] {
		t.Error("A should still be found after ClearCache + re-read")
	}
}

func TestExistingIDsReturnsCopy(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	env.CommitIntent("create issue")

	ids := env.Store.ExistingIDs()
	// Mutate the returned map
	ids["fake-id"] = true

	// The internal set should not be affected
	ids2 := env.Store.ExistingIDs()
	if ids2["fake-id"] {
		t.Error("mutation of returned map should not affect internal ID set")
	}
	if !ids2[a.ID] {
		t.Error("A should still be in ID set")
	}
}

// readerFunc adapts a function to io.Reader.
type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(b []byte) (int, error) { return f(b) }

