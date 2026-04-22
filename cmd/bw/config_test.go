package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdConfigSetAndGet(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdConfig(env.Store, []string{"set", "default.priority", "2"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if !strings.Contains(buf.String(), "default.priority=2") {
		t.Errorf("set output = %q", buf.String())
	}

	buf.Reset()
	_, err = cmdConfig(env.Store, []string{"get", "default.priority"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(buf.String(), "2") {
		t.Errorf("get output = %q", buf.String())
	}
}

func TestCmdConfigList(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdConfig(env.Store, []string{"list"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("config list: %v", err)
	}
	if !strings.Contains(buf.String(), "prefix=test") {
		t.Errorf("output = %q, want prefix=test", buf.String())
	}
}

func TestCmdConfigGetMissing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdConfig(env.Store, []string{"get", "nonexistent"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestCmdConfigUnsetRemovesKey(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	if err := cmdConfig(env.Store, []string{"set", "remote", "upstream"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("config set: %v", err)
	}

	buf.Reset()
	if err := cmdConfig(env.Store, []string{"unset", "remote"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("config unset: %v", err)
	}
	if !strings.Contains(buf.String(), "unset remote") {
		t.Errorf("unset output = %q", buf.String())
	}

	buf.Reset()
	if err := cmdConfig(env.Store, []string{"list"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("config list: %v", err)
	}
	if strings.Contains(buf.String(), "remote=") {
		t.Errorf("list still contains remote: %q", buf.String())
	}
	// Other keys preserved.
	if !strings.Contains(buf.String(), "prefix=test") {
		t.Errorf("list missing prefix after unset: %q", buf.String())
	}
}

func TestCmdConfigUnsetMissing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdConfig(env.Store, []string{"unset", "nonexistent"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error when unsetting a missing key")
	}
}
