package repo

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestUsesReftable(t *testing.T) {
	cases := []struct {
		name string
		cfg  string
		want bool
	}{
		{"reftable backend", "[extensions]\n\trefstorage = reftable\n", true},
		{"files backend", "[extensions]\n\trefstorage = files\n", false},
		{"no extensions section", "[core]\n\tbare = false\n", false},
		{"unrelated extension", "[extensions]\n\tworktreeConfig = true\n", false},
		{"key and value casing", "[extensions]\n\trefStorage = RefTable\n", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := usesReftable(decodeRaw(t, tc.cfg)); got != tc.want {
				t.Errorf("usesReftable = %v, want %v", got, tc.want)
			}
		})
	}
}

// reftableRepo creates a repo using the reftable backend and returns its path,
// skipping the test when the local git is too old to make one.
func reftableRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := exec.Command("git", "init", "--ref-format=reftable", dir).Run(); err != nil {
		t.Skipf("git does not support --ref-format=reftable: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-m", "initial")

	return dir
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

func TestFindRepoAtReftableRepo(t *testing.T) {
	dir := reftableRepo(t)

	r, err := FindRepoAt(dir)
	if err != nil {
		t.Fatalf("FindRepoAt on reftable repo: %v", err)
	}
	if r.IsInitialized() {
		t.Error("should not be initialized before Init")
	}
}

func TestReftableRefStorerRoundTrip(t *testing.T) {
	dir := reftableRepo(t)

	goRepo, err := openGitRepo(dir)
	if err != nil {
		t.Fatalf("openGitRepo: %v", err)
	}
	st := goRepo.Storer
	hash := plumbing.NewHash(gitOutput(t, dir, "rev-parse", "HEAD"))

	// HEAD must come back symbolic, as it would from a files-backend repo.
	head, err := st.Reference(plumbing.HEAD)
	if err != nil {
		t.Fatalf("Reference(HEAD): %v", err)
	}
	if head.Type() != plumbing.SymbolicReference {
		t.Errorf("HEAD type = %v, want symbolic", head.Type())
	}

	const name = plumbing.ReferenceName("refs/heads/beadwork")

	if _, err := st.Reference(name); err != plumbing.ErrReferenceNotFound {
		t.Errorf("Reference(missing) error = %v, want ErrReferenceNotFound", err)
	}

	if err := st.SetReference(plumbing.NewHashReference(name, hash)); err != nil {
		t.Fatalf("SetReference: %v", err)
	}
	got, err := st.Reference(name)
	if err != nil {
		t.Fatalf("Reference after set: %v", err)
	}
	if got.Hash() != hash {
		t.Errorf("hash = %s, want %s", got.Hash(), hash)
	}

	// The ref really landed in reftable, not as a loose file.
	if out := gitOutput(t, dir, "rev-parse", name.String()); out != hash.String() {
		t.Errorf("git rev-parse %s = %s, want %s", name, out, hash.String())
	}

	// CAS must reject a stale expected value; beadwork's retry loop depends on
	// a failed compare-and-swap actually failing.
	stale := plumbing.NewHashReference(name, plumbing.NewHash("0000000000000000000000000000000000000001"))
	if err := st.CheckAndSetReference(plumbing.NewHashReference(name, hash), stale); err == nil {
		t.Error("CheckAndSetReference with stale old value succeeded, want error")
	}

	iter, err := st.IterReferences()
	if err != nil {
		t.Fatalf("IterReferences: %v", err)
	}
	seen := map[string]bool{}
	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		seen[ref.Name().String()] = true
		return nil
	}); err != nil {
		t.Fatalf("ForEach: %v", err)
	}
	if !seen[name.String()] {
		t.Errorf("IterReferences missing %s, saw %v", name, seen)
	}
	if !seen[plumbing.HEAD.String()] {
		t.Errorf("IterReferences missing HEAD, saw %v", seen)
	}

	if err := st.RemoveReference(name); err != nil {
		t.Fatalf("RemoveReference: %v", err)
	}
	if _, err := st.Reference(name); err != plumbing.ErrReferenceNotFound {
		t.Errorf("Reference after remove = %v, want ErrReferenceNotFound", err)
	}
}
