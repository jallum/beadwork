package issue_test

import (
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestGetNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Get("test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}


func TestReadCorruptJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a valid issue, then corrupt it on disk
	iss, _ := env.Store.Create("Valid", issue.CreateOpts{})
	env.Repo.TreeFS().WriteFile("issues/"+iss.ID+".json", []byte("{invalid json"))
	env.Store.ClearCache()

	_, err := env.Store.Get(iss.ID)
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("error = %q, want 'corrupt'", err.Error())
	}
}


func TestCacheHitReturnsSamePointer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Cache test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	a, err := env.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	b, err := env.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if a != b {
		t.Error("second Get should return the same pointer (cache hit)")
	}
}


func TestClearCacheInvalidates(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Clear test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	a, _ := env.Store.Get(iss.ID)
	env.Store.ClearCache()
	b, _ := env.Store.Get(iss.ID)

	if a == b {
		t.Error("after ClearCache, Get should return a new pointer")
	}
	if a.ID != b.ID {
		t.Errorf("IDs differ: %q vs %q", a.ID, b.ID)
	}
}


func TestDeleteEvictsFromCache(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Delete cache", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Populate cache
	_, _ = env.Store.Get(iss.ID)

	// Delete should evict
	_, err := env.Store.Delete(iss.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get should fail — not found (not served from stale cache)
	_, err = env.Store.Get(iss.ID)
	if err == nil {
		t.Error("Get after Delete should return error, not stale cached value")
	}
}

