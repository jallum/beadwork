package registry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	r, err := Load(file)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("Paths = %v, want empty", r.Paths())
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	r, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Add("/home/user/project-a"); err != nil {
		t.Fatal(err)
	}
	if err := r.Add("/home/user/project-b"); err != nil {
		t.Fatal(err)
	}

	r2, err := Load(file)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	paths := r2.Paths()
	if len(paths) != 2 {
		t.Fatalf("Paths count = %d, want 2", len(paths))
	}
	if paths[0] != "/home/user/project-a" || paths[1] != "/home/user/project-b" {
		t.Errorf("Paths = %v", paths)
	}
}

func TestAddIdempotent(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	r, _ := Load(file)

	r.Add("/repo")
	r.Add("/repo")

	if len(r.Paths()) != 1 {
		t.Errorf("Paths count = %d, want 1", len(r.Paths()))
	}
}

func TestRemove(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	r, _ := Load(file)
	r.Add("/keep")
	r.Add("/remove-me")

	if !r.Remove("/remove-me") {
		t.Error("Remove returned false for existing path")
	}
	if r.Remove("/nonexistent") {
		t.Error("Remove returned true for nonexistent path")
	}
	if len(r.Paths()) != 1 {
		t.Errorf("Paths count = %d, want 1", len(r.Paths()))
	}
}

func TestConcurrentAdd(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	r, _ := Load(file)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r.Add("/repo/" + string(rune('a'+n)))
		}(i)
	}
	wg.Wait()

	r2, err := Load(file)
	if err != nil {
		t.Fatalf("Load after concurrent adds: %v", err)
	}
	if len(r2.Paths()) != 20 {
		t.Errorf("Paths count = %d, want 20", len(r2.Paths()))
	}
}

func TestPathsSorted(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	r, _ := Load(file)
	r.Add("/z")
	r.Add("/a")
	r.Add("/m")

	paths := r.Paths()
	if paths[0] != "/a" || paths[1] != "/m" || paths[2] != "/z" {
		t.Errorf("Paths not sorted: %v", paths)
	}
}

func TestLoadIgnoresBlankLines(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reg")
	os.WriteFile(file, []byte("/a\n\n  \n/b\n"), 0644)

	r, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Paths()) != 2 {
		t.Errorf("Paths count = %d, want 2", len(r.Paths()))
	}
}
