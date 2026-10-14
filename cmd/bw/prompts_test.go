package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func configurePrompt(t *testing.T, env *testutil.Env, name, path string) {
	t.Helper()
	if err := env.Repo.SetConfig("prompts."+name, path); err != nil {
		t.Fatal(err)
	}
	env.CommitIntent("config prompts." + name + "=" + path)
}

func writePrompt(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCmdPrimeCustomPrompt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()
	iss, err := env.Store.Create("Custom ready task", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	env.CommitIntent("create " + iss.ID)
	writePrompt(t, filepath.Join(env.Dir, "prime.md"), "Custom {{ .Prefix }}\nBranch: {{ .Git.Branch }}\n{{ bw \"ready\" \"--no-context\" }}")
	configurePrompt(t, env, "prime", "prime.md")

	var buf bytes.Buffer
	if _, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Custom test", "Branch: ", iss.ID, "Custom ready task"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output missing %q: %s", want, &buf)
		}
	}
	if strings.Contains(buf.String(), "How Should This Land?") {
		t.Errorf("embedded prompt included in custom output: %s", &buf)
	}
}

func TestCmdPrimeCustomPromptErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"missing", ""},
		{"syntax", "{{ if }}"},
		{"execution", "{{ .UnknownField }}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := testutil.NewEnv(t)
			defer env.Cleanup()
			path := filepath.Join(env.Dir, "prime.md")
			if tc.name != "missing" {
				writePrompt(t, path, tc.src)
			}
			configurePrompt(t, env, "prime", "prime.md")
			var buf bytes.Buffer
			_, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil)
			if err == nil || !strings.Contains(err.Error(), "prompts.prime") || !strings.Contains(err.Error(), path) {
				t.Fatalf("expected error naming config key and file, got %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("error produced output: %s", &buf)
			}
		})
	}
}

func TestCmdPrimeEmptyPrompt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()
	writePrompt(t, filepath.Join(env.Dir, "empty.md"), "")
	configurePrompt(t, env, "prime", "empty.md")
	var buf bytes.Buffer
	if _, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("empty file should replace the default prompt: %s", &buf)
	}

	configurePrompt(t, env, "prime", "")
	buf.Reset()
	if _, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "How Should This Land?") {
		t.Errorf("empty setting should restore the default prompt: %s", &buf)
	}
}

func TestCmdStartCustomPrompt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()
	parent, err := env.Store.Create("Parent", issue.CreateOpts{Type: "epic"})
	if err != nil {
		t.Fatal(err)
	}
	iss, err := env.Store.Create("Custom task", issue.CreateOpts{Parent: parent.ID, Description: "Keep issue context"})
	if err != nil {
		t.Fatal(err)
	}
	env.CommitIntent("setup")
	if err := env.Repo.SetConfig("workflow.review", "pr"); err != nil {
		t.Fatal(err)
	}
	writePrompt(t, filepath.Join(env.Dir, "start.md"), "Custom {{ .ID }} {{ .Type }} {{ .Status }} {{ .Parent }} {{ .WorkflowReview }}\n{{ bw \"show\" .ID \"--json\" }}")
	configurePrompt(t, env, "start", "start.md")
	var buf bytes.Buffer
	if _, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Custom " + iss.ID + " task in_progress " + parent.ID + " pr", "Keep issue context", `"assignee": "alice"`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output missing %q: %s", want, &buf)
		}
	}
	if strings.Contains(buf.String(), "LANDING THE WORK") {
		t.Errorf("embedded prompt included in custom output: %s", &buf)
	}
}

func TestCmdStartPromptErrorsBeforeClaim(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"missing", ""},
		{"syntax", "{{ if }}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := testutil.NewEnv(t)
			defer env.Cleanup()
			iss, err := env.Store.Create("Unclaimed task", issue.CreateOpts{})
			if err != nil {
				t.Fatal(err)
			}
			env.CommitIntent("create " + iss.ID)
			path := filepath.Join(env.Dir, "start.md")
			if tc.name != "missing" {
				writePrompt(t, path, tc.src)
			}
			configurePrompt(t, env, "start", "start.md")
			var buf bytes.Buffer
			_, err = cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
			if err == nil || !strings.Contains(err.Error(), "prompts.start") || !strings.Contains(err.Error(), path) {
				t.Fatalf("expected error naming config key and file, got %v", err)
			}
			got, err := env.Store.Get(iss.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != "open" || got.Assignee != "" {
				t.Errorf("failed start claimed issue: %+v", got)
			}
			if buf.Len() != 0 {
				t.Errorf("error produced output: %s", &buf)
			}
		})
	}
}
