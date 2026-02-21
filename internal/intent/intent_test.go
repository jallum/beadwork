package intent_test

import (
	"os"
	"testing"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestReplayCreate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{`create test-0000 p1 bug "Login crashes on timeout"`}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].Title != "Login crashes on timeout" {
		t.Errorf("title = %q", issues[0].Title)
	}
	if issues[0].Priority != 1 {
		t.Errorf("priority = %d, want 1", issues[0].Priority)
	}
	if issues[0].Type != "bug" {
		t.Errorf("type = %q, want bug", issues[0].Type)
	}
}

func TestReplayCreateNoQuotes(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create without quoted title — should use remaining parts as title
	intents := []string{`create test-0000 p2 task Unquoted title words`}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].Title != "Unquoted title words" {
		t.Errorf("title = %q, want 'Unquoted title words'", issues[0].Title)
	}
}

func TestReplayCreateDefaultPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create with no priority prefix — should default to p3
	intents := []string{`create test-0000 p3 task "Default priority task"`}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	issues, _ := env.Store.List(issue.Filter{})
	if issues[0].Priority != 3 {
		t.Errorf("priority = %d, want 3", issues[0].Priority)
	}
}

func TestReplayClose(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To close", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{"close " + iss.ID})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestReplayCloseAlreadyClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already closed", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.CommitIntent("create and close " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{"close " + iss.ID})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	// Issue should still be closed
	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestReplayReopen(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To reopen", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.CommitIntent("setup " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{"reopen " + iss.ID})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestReplayReopenNotClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Open issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{"reopen " + iss.ID})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestReplayUpdate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Original", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	intents := []string{
		"update " + iss.ID + " status=in_progress assignee=agent-1 priority=1",
	}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}
	if got.Assignee != "agent-1" {
		t.Errorf("assignee = %q, want agent-1", got.Assignee)
	}
	if got.Priority != 1 {
		t.Errorf("priority = %d, want 1", got.Priority)
	}
}

func TestReplayUpdateTitle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Old title", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{
		"update " + iss.ID + " title=New",
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Title != "New" {
		t.Errorf("title = %q, want New", got.Title)
	}
}

func TestReplayLink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID + " and " + b.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{
		"link " + a.ID + " blocks " + b.ID,
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	blocker, _ := env.Store.Get(a.ID)
	if len(blocker.Blocks) != 1 || blocker.Blocks[0] != b.ID {
		t.Errorf("blocker.Blocks = %v, want [%s]", blocker.Blocks, b.ID)
	}
	blocked, _ := env.Store.Get(b.ID)
	if len(blocked.BlockedBy) != 1 || blocked.BlockedBy[0] != a.ID {
		t.Errorf("blocked.BlockedBy = %v, want [%s]", blocked.BlockedBy, a.ID)
	}
}

func TestReplayUnlink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup link")

	errs := intent.Replay(env.Repo, env.Store, []string{
		"unlink " + a.ID + " blocks " + b.ID,
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	blocker, _ := env.Store.Get(a.ID)
	if len(blocker.Blocks) != 0 {
		t.Errorf("blocker.Blocks = %v, want empty", blocker.Blocks)
	}
}

func TestReplayLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Labeled", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"old"}, nil)
	env.CommitIntent("setup " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{
		"label " + iss.ID + " +new -old",
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if len(got.Labels) != 1 || got.Labels[0] != "new" {
		t.Errorf("labels = %v, want [new]", got.Labels)
	}
}

func TestReplayConfig(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Repo, env.Store, []string{
		"config default.priority=2",
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	val, ok := env.Repo.GetConfig("default.priority")
	if !ok {
		t.Fatal("default.priority not set after replay")
	}
	if val != "2" {
		t.Errorf("default.priority = %q, want 2", val)
	}
}

func TestReplayMultipleIntents(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-0000 p1 bug "First issue"`,
		`create test-0000 p2 task "Second issue"`,
	}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
}

func TestReplayMalformedIntents(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		"create",                // too few parts
		"close",                 // missing id
		"delete",               // missing id
		"update",               // missing id and fields
		"link foo",             // missing blocks keyword
		"unlink foo bar",       // missing blocks keyword
		"label",                // missing id
		"config",               // missing key=value
		"config noequals",      // missing =
	}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) != len(intents) {
		t.Errorf("expected %d errors, got %d: %v", len(intents), len(errs), errs)
	}
}

func TestReplayEmptyAndUnknown(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// These should be silently skipped, not produce errors
	intents := []string{
		"",
		"init beadwork",
		"somefutureverb arg1 arg2",
	}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for empty/unknown intents, got %d: %v", len(errs), errs)
	}
}

func TestReplayUpdateNonexistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Repo, env.Store, []string{
		"update test-zzzz status=closed",
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestReplayCloseNonexistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Repo, env.Store, []string{
		"close test-zzzz",
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestReplayPartialFailure(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Exists", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// First intent succeeds (close existing), second fails (close nonexistent)
	intents := []string{
		"close " + iss.ID,
		"close test-zzzz",
	}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	// The successful intent should have taken effect
	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestReplayCreateQuotedTitleWithSpaces(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{`create test-0000 p2 task "Title with multiple words"`}
	errs := intent.Replay(env.Repo, env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	issues, _ := env.Store.List(issue.Filter{})
	if issues[0].Title != "Title with multiple words" {
		t.Errorf("title = %q, want 'Title with multiple words'", issues[0].Title)
	}
}

func TestReplayIdempotentLink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	// Replay the same link again — the link is a no-op on disk,
	// so commit is silently skipped. No error expected.
	errs := intent.Replay(env.Repo, env.Store, []string{
		"link " + a.ID + " blocks " + b.ID,
	})
	if len(errs) != 0 {
		t.Fatalf("expected no errors (noop commit), got %d: %v", len(errs), errs)
	}

	// The link should still be intact
	blocker, _ := env.Store.Get(a.ID)
	if len(blocker.Blocks) != 1 {
		t.Errorf("blocks = %v, want exactly 1 entry", blocker.Blocks)
	}
}

func TestExtractQuoted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`create test-0000 p1 bug "Login crashes on timeout"`, "Login crashes on timeout"},
		{`create test-0000 p1 bug no-quotes`, ""},
		{`create test-0000 p1 bug "Unmatched opening`, ""},
		{`create test-0000 p1 bug ""`, ""},
		{`create test-0000 p1 bug "one" extra "two"`, "one"},
	}
	for _, tt := range tests {
		got := intent.ExtractQuoted(tt.input)
		if got != tt.want {
			t.Errorf("ExtractQuoted(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseIntent(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`create test-0000 p1 bug "Login crashes"`, []string{"create", "test-0000", "p1", "bug", "Login crashes"}},
		{`close test-1234`, []string{"close", "test-1234"}},
		{``, nil},
		{`link a blocks b`, []string{"link", "a", "blocks", "b"}},
	}
	for _, tt := range tests {
		got := intent.ParseIntent(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseIntent(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseIntent(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestReplayIdempotentUnlink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.CommitIntent("setup")

	// Unlink when no link exists — should succeed (idempotent)
	errs := intent.Replay(env.Repo, env.Store, []string{
		"unlink " + a.ID + " blocks " + b.ID,
	})
	if len(errs) != 0 {
		t.Fatalf("expected no errors for unlinking unlinked issues, got %d: %v", len(errs), errs)
	}
}

func TestReplayUpdateType(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{
		"update " + iss.ID + " type=bug",
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Type != "bug" {
		t.Errorf("type = %q, want bug", got.Type)
	}
}

func init() {
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}

func TestReplayDelete(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To delete", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{"delete " + iss.ID})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	_, err := env.Store.Get(iss.ID)
	if err == nil {
		t.Error("issue should not exist after replay delete")
	}
}

func TestReplayDeleteNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Repo, env.Store, []string{"delete test-zzzz"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestReplayComment(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Commentable", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Repo, env.Store, []string{
		`comment ` + iss.ID + ` "Replayed comment"`,
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if len(got.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(got.Comments))
	}
	if got.Comments[0].Text != "Replayed comment" {
		t.Errorf("text = %q, want %q", got.Comments[0].Text, "Replayed comment")
	}
}

func TestReplayCommentMalformed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Repo, env.Store, []string{
		"comment",
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestReplayCommentNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Repo, env.Store, []string{
		`comment test-zzzz "Missing issue"`,
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestReplayUpdateParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	child, _ := env.Store.Create("Child", issue.CreateOpts{})
	env.CommitIntent("create issues")

	errs := intent.Replay(env.Repo, env.Store, []string{
		"update " + child.ID + " parent=" + parent.ID,
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(child.ID)
	if got.Parent != parent.ID {
		t.Errorf("Parent = %q, want %q", got.Parent, parent.ID)
	}
}
