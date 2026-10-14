package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestPromptOverridesPaths(t *testing.T) {
	for _, mode := range []string{"subdirectory", "explicit-repo", "absolute", "worktree"} {
		t.Run(mode, func(t *testing.T) {
			env := testutil.NewEnv(t)
			defer env.Cleanup()
			iss, err := env.Store.Create("Path resolution task", issue.CreateOpts{})
			if err != nil {
				t.Fatal(err)
			}
			env.CommitIntent("create " + iss.ID)

			root := env.Dir
			if mode == "worktree" {
				root = filepath.Join(t.TempDir(), "checkout")
				cmd := exec.Command("git", "worktree", "add", "-b", "custom-prompts", root)
				cmd.Dir = env.Dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git worktree add: %s: %v", out, err)
				}
			}
			promptDir := filepath.Join(root, ".beadwork", "prompts")
			if mode == "absolute" {
				promptDir = filepath.Join(t.TempDir(), "prompts with spaces")
			}
			if err := os.MkdirAll(promptDir, 0755); err != nil {
				t.Fatal(err)
			}
			for name, src := range map[string]string{
				"prime": "Custom prime {{ .Prefix }}\n{{ bw \"ready\" \"--no-context\" }}",
				"start": "Custom start {{ .ID }} {{ .Status }}",
			} {
				path := filepath.Join(promptDir, name+".md")
				if err := os.WriteFile(path, []byte(src), 0644); err != nil {
					t.Fatal(err)
				}
				configPath := filepath.Join(".beadwork", "prompts", name+".md")
				if mode == "absolute" {
					configPath = path
				}
				bw(t, env.Dir, "config", "set", "prompts."+name, configPath)
				if mode == "worktree" {
					// A file in the main checkout must not win over the worktree's.
					mainPath := filepath.Join(env.Dir, configPath)
					if err := os.MkdirAll(filepath.Dir(mainPath), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(mainPath, []byte("Wrong checkout"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}

			cwd := root
			var prefix []string
			if mode == "subdirectory" || mode == "worktree" {
				cwd = filepath.Join(root, "nested", "directory")
				if err := os.MkdirAll(cwd, 0755); err != nil {
					t.Fatal(err)
				}
			} else if mode == "explicit-repo" {
				cwd = t.TempDir()
				prefix = []string{"-C", root}
			}
			out := bw(t, cwd, append(prefix, "prime")...)
			assertContains(t, out, "Custom prime test")
			assertContains(t, out, iss.ID)
			assertNotContains(t, out, "How Should This Land?")
			out = bw(t, cwd, append(prefix, "start", iss.ID)...)
			assertContains(t, out, "Path resolution task")
			assertContains(t, out, "Custom start "+iss.ID+" in_progress")
			assertNotContains(t, out, "LANDING THE WORK")
		})
	}
}
