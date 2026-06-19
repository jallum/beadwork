package intent_test

import (
	"testing"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestReplayArchiveMovesClosedIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-aaaa p1 task "Done thing"`,
		`close test-aaaa`,
		`archive test-aaaa`,
	}
	if errs := intent.Replay(env.Store, intents); len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	if _, err := env.Store.Get("test-aaaa"); err == nil {
		t.Error("archived issue should not resolve via Get after replay")
	}
	a, err := env.Store.ArchivedIssue("test-aaaa")
	if err != nil {
		t.Fatalf("ArchivedIssue: %v", err)
	}
	if a.Title != "Done thing" {
		t.Errorf("title = %q, want 'Done thing'", a.Title)
	}
	if a.ArchivedAt == "" {
		t.Error("ArchivedAt not stamped on replay")
	}
}

func TestReplayArchiveWithCloseFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-bbbb p2 task "Still open"`,
		`archive test-bbbb --close`,
	}
	if errs := intent.Replay(env.Store, intents); len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}
	a, err := env.Store.ArchivedIssue("test-bbbb")
	if err != nil {
		t.Fatalf("ArchivedIssue: %v", err)
	}
	if a.Status != "closed" {
		t.Errorf("status = %q, want closed", a.Status)
	}
}

func TestReplayArchiveWithDetachSeversEdge(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-aaaa p2 task "Blocker"`,
		`create test-bbbb p2 task "Blocked"`,
		`link test-aaaa blocks test-bbbb`,
		`close test-aaaa`,
		`archive test-aaaa --detach`,
	}
	if errs := intent.Replay(env.Store, intents); len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}
	b, err := env.Store.Get("test-bbbb")
	if err != nil {
		t.Fatalf("Get blocked: %v", err)
	}
	if len(b.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty after detach archive", b.BlockedBy)
	}
}
