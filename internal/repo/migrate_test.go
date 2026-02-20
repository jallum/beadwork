package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func initTestRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	run("add", ".")
	run("commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(orig) })

	r, err := FindRepo()
	if err != nil {
		t.Fatalf("FindRepo: %v", err)
	}
	return r
}

func TestVersionUnset(t *testing.T) {
	r := initTestRepo(t)
	// Manually init without version key (simulates pre-versioning repo)
	r.tfs.WriteFile(".bwconfig", []byte("prefix=test\n"))
	r.tfs.WriteFile("issues/.gitkeep", []byte{})
	r.tfs.Commit("init beadwork")
	r.initialized = true

	if v := r.Version(); v != 0 {
		t.Errorf("Version() = %d, want 0", v)
	}
}

func TestVersionAfterInit(t *testing.T) {
	r := initTestRepo(t)
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if v := r.Version(); v != CurrentVersion {
		t.Errorf("Version() = %d, want %d", v, CurrentVersion)
	}
}

func TestVersionRoundTrip(t *testing.T) {
	r := initTestRepo(t)
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	r.SetConfig("version", "42")
	r.tfs.Commit("set version")

	if v := r.Version(); v != 42 {
		t.Errorf("Version() = %d, want 42", v)
	}
}

func TestVersionInvalidValue(t *testing.T) {
	r := initTestRepo(t)
	r.tfs.WriteFile(".bwconfig", []byte("prefix=test\nversion=abc\n"))
	r.tfs.Commit("bad version")
	r.initialized = true

	if v := r.Version(); v != 0 {
		t.Errorf("Version() = %d, want 0 for invalid value", v)
	}
}

// --- Upgrade ---

func TestUpgradeFromV0(t *testing.T) {
	r := initTestRepo(t)
	// Simulate pre-versioning repo
	r.tfs.WriteFile(".bwconfig", []byte("prefix=test\n"))
	r.tfs.WriteFile("issues/.gitkeep", []byte{})
	r.tfs.Commit("init beadwork")
	r.initialized = true
	r.Prefix = "test"

	from, to, err := r.Upgrade()
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if from != 0 {
		t.Errorf("from = %d, want 0", from)
	}
	if to != CurrentVersion {
		t.Errorf("to = %d, want %d", to, CurrentVersion)
	}
	if v := r.Version(); v != CurrentVersion {
		t.Errorf("Version() after upgrade = %d, want %d", v, CurrentVersion)
	}
}

func TestUpgradeAlreadyCurrent(t *testing.T) {
	r := initTestRepo(t)
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	from, to, err := r.Upgrade()
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if from != CurrentVersion || to != CurrentVersion {
		t.Errorf("got (%d, %d), want (%d, %d)", from, to, CurrentVersion, CurrentVersion)
	}
}

func TestUpgradeDescriptions(t *testing.T) {
	// Verify every migration has a non-empty description
	for i, m := range Migrations {
		if m.Description == "" {
			t.Errorf("migration %d has empty description", i)
		}
	}
}

func TestUpgradeV1ToV2PriorityMigration(t *testing.T) {
	r := initTestRepo(t)
	// Simulate a v1 repo with issues at old priority scale (1-5)
	r.tfs.WriteFile(".bwconfig", []byte("prefix=test\nversion=1\n"))
	r.tfs.WriteFile("issues/.gitkeep", []byte{})

	// Write issue files with v1-era priorities (1-5)
	cases := []struct {
		file     string
		oldPri   int
		wantPri  int
	}{
		{"issues/test-0001.json", 1, 0},
		{"issues/test-0002.json", 3, 2},
		{"issues/test-0003.json", 5, 4},
	}

	for _, tc := range cases {
		data := fmt.Sprintf(`{"id":"%s","title":"test","priority":%d,"type":"task","status":"open"}`,
			tc.file, tc.oldPri)
		r.tfs.WriteFile(tc.file, []byte(data))
	}
	r.tfs.Commit("init v1 repo with issues")
	r.initialized = true
	r.Prefix = "test"

	from, to, err := r.Upgrade()
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if from != 1 {
		t.Errorf("from = %d, want 1", from)
	}
	if to != CurrentVersion {
		t.Errorf("to = %d, want %d", to, CurrentVersion)
	}

	// Verify priorities were decremented
	for _, tc := range cases {
		data, err := r.tfs.ReadFile(tc.file)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", tc.file, err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal(%s): %v", tc.file, err)
		}
		var got int
		if err := json.Unmarshal(raw["priority"], &got); err != nil {
			t.Fatalf("Unmarshal priority(%s): %v", tc.file, err)
		}
		if got != tc.wantPri {
			t.Errorf("%s: priority = %d, want %d", tc.file, got, tc.wantPri)
		}
	}

	if v := r.Version(); v != CurrentVersion {
		t.Errorf("Version() after upgrade = %d, want %d", v, CurrentVersion)
	}
}

func TestInitStampsVersion(t *testing.T) {
	r := initTestRepo(t)
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	val, ok := r.GetConfig("version")
	if !ok {
		t.Fatal("version key not found in config")
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		t.Fatalf("version value %q not an integer", val)
	}
	if n != CurrentVersion {
		t.Errorf("config version = %d, want %d", n, CurrentVersion)
	}
}
