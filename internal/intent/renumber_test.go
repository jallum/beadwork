package intent_test

// Regression tests for replaying a create whose ID already exists.
//
// Child IDs are sequential (parent.N), so two clones adding children to
// the same epic allocate the same IDs. Replay must keep the local issue,
// and every later intent that referred to it, rather than drop it.
//
// Tests that expose unfixed bugs are marked with "BUG:" comments and use
// t.Skip() so the suite stays green until the corresponding fix lands.

import (
	"testing"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func createChildren(t *testing.T, env *testutil.Env, parent string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := env.Store.Create("Remote child", issue.CreateOpts{Parent: parent}); err != nil {
			t.Fatalf("Create child: %v", err)
		}
	}
	env.CommitIntent("create remote children")
}

// byTitle returns the only issue with the given title.
func byTitle(t *testing.T, env *testutil.Env, title string) *issue.Issue {
	t.Helper()
	env.Store.ClearCache()
	all, err := env.Store.List(issue.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found []*issue.Issue
	for _, iss := range all {
		if iss.Title == title {
			found = append(found, iss)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one issue titled %q, found %d", title, len(found))
	}
	return found[0]
}

func commentTexts(iss *issue.Issue) []string {
	var texts []string
	for _, c := range iss.Comments {
		texts = append(texts, c.Text)
	}
	return texts
}

// ---------------------------------------------------------------------------
// BUG: the local .1 collides with the remote .1 and is dropped; the local
// .2 is then created unchallenged, and both comments land on the wrong
// issues. A fix that moves the local .1 to a new ID must also not chain
// renames: the comment on the old .1 must follow the first local child,
// not whichever issue ends up at .2.
// ---------------------------------------------------------------------------
func TestReplayCollidingChildrenKeepTheirComments(t *testing.T) {
	t.Skip("BUG: colliding child ID drops the local child on replay")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	env.CommitIntent("create " + epic.ID)
	createChildren(t, env, epic.ID, 1) // the remote already has .1

	errs := intent.Replay(env.Store, []string{
		`create ` + epic.ID + `.1 p2 task "Local A" parent=` + epic.ID,
		`create ` + epic.ID + `.2 p2 task "Local B" parent=` + epic.ID,
		`comment ` + epic.ID + `.1 "on A"`,
		`comment ` + epic.ID + `.2 "on B"`,
	})
	t.Logf("replay reported: %v", errs)

	if rc, _ := env.Store.Get(epic.ID + ".1"); rc == nil || rc.Title != "Remote child" || len(rc.Comments) != 0 {
		t.Errorf("remote child %s.1 should be untouched, got %+v", epic.ID, rc)
	}
	for title, want := range map[string]string{"Local A": "on A", "Local B": "on B"} {
		iss := byTitle(t, env, title)
		if iss.Parent != epic.ID {
			t.Errorf("%s parent = %q, want %s", title, iss.Parent, epic.ID)
		}
		if got := commentTexts(iss); len(got) != 1 || got[0] != want {
			t.Errorf("%s comments = %v, want [%s]", title, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// BUG: as above, and a fix that rewrites references to a moved ID must
// match whole IDs: .10 is not a reference to .1, but .1.1 (a grandchild
// created under the local .1) belongs under wherever .1 went.
// ---------------------------------------------------------------------------
func TestReplayCollidingChildRespectsIDBoundaries(t *testing.T) {
	t.Skip("BUG: colliding child ID drops the local child on replay")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	env.CommitIntent("create " + epic.ID)
	createChildren(t, env, epic.ID, 10) // .1 to .10 exist remotely

	errs := intent.Replay(env.Store, []string{
		`create ` + epic.ID + `.1 p2 task "Local child" parent=` + epic.ID,
		`create ` + epic.ID + `.1.1 p2 task "Local grandchild" parent=` + epic.ID + `.1`,
		`comment ` + epic.ID + `.10 "must stay on .10"`,
	})
	t.Logf("replay reported: %v", errs)

	child := byTitle(t, env, "Local child")
	if child.ID == epic.ID+".1" {
		t.Fatalf("local child took the remote's ID %s", child.ID)
	}
	if grandchild := byTitle(t, env, "Local grandchild"); grandchild.Parent != child.ID {
		t.Errorf("grandchild parent = %q, want the local child %s", grandchild.Parent, child.ID)
	}
	if ten, _ := env.Store.Get(epic.ID + ".10"); ten == nil || len(ten.Comments) != 1 {
		t.Errorf("%s.10 should keep its comment, got %+v", epic.ID, ten)
	}
}

// ---------------------------------------------------------------------------
// Not a current bug (replay ignores parent= today, so the issue survives
// without one), but a guard for the fix: honouring parent= must not make
// a create fail outright when the remote deleted the parent.
// ---------------------------------------------------------------------------
func TestReplayCreateKeepsIssueWhenParentGone(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	errs := intent.Replay(env.Store, []string{
		`create test-orph p2 task "Orphan" parent=test-gone`,
	})
	t.Logf("replay reported: %v", errs)

	env.Store.ClearCache()
	iss, err := env.Store.Get("test-orph")
	if err != nil {
		t.Fatalf("the issue must survive its parent being deleted: %v", err)
	}
	if iss.Parent != "" {
		t.Errorf("parent = %q, want none", iss.Parent)
	}
}

// ---------------------------------------------------------------------------
// BUG: an attachment on a colliding child is lost with the child. A fix
// must read the blob from the local child's original path in the source
// tree, not from the same path in the current tree, which belongs to the
// remote's child.
// ---------------------------------------------------------------------------
func TestReplayAttachFollowsCollidingChild(t *testing.T) {
	t.Skip("BUG: colliding child ID drops the local child on replay")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	env.CommitIntent("create " + epic.ID)
	base := env.Repo.TreeFS().RefHash()
	childID := epic.ID + ".1"

	// Local side: a child with an attachment.
	if _, err := env.Store.Create("Local child", issue.CreateOpts{Parent: epic.ID}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Store.Attach(childID, "notes.txt", []byte("local")); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	env.CommitIntent("create " + childID)
	local := env.Repo.TreeFS().RefHash()

	// Remote side, diverged from base: its own .1 with an attachment at
	// the same path.
	if err := env.Repo.TreeFS().Reset(base); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	env.Store.ClearCache()
	if _, err := env.Store.Create("Remote child", issue.CreateOpts{Parent: epic.ID}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Store.Attach(childID, "notes.txt", []byte("remote")); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	env.CommitIntent("create remote " + childID)
	env.Store.ClearCache()

	env.Store.SourceHash = local
	errs := intent.Replay(env.Store, []string{
		"create " + childID + ` p2 task "Local child" parent=` + epic.ID + "\nattach " + childID + " notes.txt",
	})
	t.Logf("replay reported: %v", errs)

	localChild := byTitle(t, env, "Local child")
	for id, want := range map[string]string{childID: "remote", localChild.ID: "local"} {
		got, err := env.Store.GetAttachment(id, "notes.txt")
		if err != nil {
			t.Fatalf("GetAttachment(%s): %v", id, err)
		}
		if string(got) != want {
			t.Errorf("%s attachment = %q, want %q", id, got, want)
		}
	}
}
