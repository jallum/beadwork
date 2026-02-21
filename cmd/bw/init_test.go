package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdInitFresh(t *testing.T) {
	// Create a bare git repo (not initialized with beadwork)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{"--prefix", "fresh"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output = %q, want 'initialized'", buf.String())
	}
	if !strings.Contains(buf.String(), "fresh") {
		t.Errorf("output = %q, want prefix 'fresh'", buf.String())
	}
}

func TestCmdInitAlreadyInitialized(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for already initialized")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestCmdInitForce(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{"--force", "--prefix", "newpfx"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit --force: %v", err)
	}
	if !strings.Contains(buf.String(), "reinitialized") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdInitDefaultPrefix(t *testing.T) {
	// Init without --prefix should derive prefix from repo name
	dir := t.TempDir()
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	runInDir("git", "init")
	runInDir("git", "config", "user.email", "test@test.com")
	runInDir("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	runInDir("git", "add", ".")
	runInDir("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output = %q, want 'initialized'", buf.String())
	}
}

func TestCmdInitForceInvalidPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{"--force", "--prefix", "has space"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestCmdInitUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{"--verbose"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCmdInitForceDefaultPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Force reinit without specifying prefix (uses empty string -> derives from repo)
	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{"--force"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit --force: %v", err)
	}
	if !strings.Contains(buf.String(), "reinitialized") {
		t.Errorf("output = %q, want 'reinitialized'", buf.String())
	}
}

func TestCmdInitInvalidPrefix(t *testing.T) {
	dir := t.TempDir()
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	runInDir("git", "init")
	runInDir("git", "config", "user.email", "test@test.com")
	runInDir("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	runInDir("git", "add", ".")
	runInDir("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit(nil, nil, []string{"--prefix", "invalid prefix!"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}
