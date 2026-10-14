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
	for _, key := range []string{"prompts.prime", "prompts.start"} {
		if !strings.Contains(buf.String(), key+"=\n") {
			t.Errorf("output missing default %s: %q", key, buf.String())
		}
		if _, ok := env.Repo.GetConfig(key); ok {
			t.Errorf("listing persisted default %s", key)
		}
	}
}

func TestCmdConfigPromptDefaultsAndOverrides(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()
	for _, key := range []string{"prompts.prime", "prompts.start"} {
		var buf bytes.Buffer
		if _, err := cmdConfig(env.Store, []string{"get", key}, PlainWriter(&buf), nil); err != nil {
			t.Fatalf("get default %s: %v", key, err)
		}
		if buf.String() != "\n" {
			t.Errorf("default %s = %q, want empty value", key, buf.String())
		}
		path := ".beadwork/prompts/custom.md"
		buf.Reset()
		if _, err := cmdConfig(env.Store, []string{"set", key, path}, PlainWriter(&buf), nil); err != nil {
			t.Fatal(err)
		}
		buf.Reset()
		if _, err := cmdConfig(env.Store, []string{"list"}, PlainWriter(&buf), nil); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), key+"="+path+"\n") {
			t.Errorf("listing lost configured path: %s", &buf)
		}
		buf.Reset()
		if _, err := cmdConfig(env.Store, []string{"set", key, ""}, PlainWriter(&buf), nil); err != nil {
			t.Fatal(err)
		}
		buf.Reset()
		if _, err := cmdConfig(env.Store, []string{"get", key}, PlainWriter(&buf), nil); err != nil {
			t.Fatal(err)
		}
		if buf.String() != "\n" {
			t.Errorf("reset %s = %q, want empty value", key, buf.String())
		}
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
