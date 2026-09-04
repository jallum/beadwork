package repo

import (
	"bytes"
	"slices"
	"testing"

	format "github.com/go-git/go-git/v5/plumbing/format/config"
)

func decodeRaw(t *testing.T, s string) *format.Config {
	t.Helper()

	raw := format.New()
	if err := format.NewDecoder(bytes.NewReader([]byte(s))).Decode(raw); err != nil {
		t.Fatalf("decode raw config: %v", err)
	}
	return raw
}

func TestStripNegativeFetchRefSpecs(t *testing.T) {
	raw := decodeRaw(t, `[remote "origin"]
	url = git@github.com:example/example.git
	fetch = +refs/heads/*:refs/remotes/origin/*
	fetch = ^refs/heads/skip/*
	fetch = +refs/tags/*:refs/tags/*
`)

	stripNegativeFetchRefSpecs(raw)

	origin := raw.Section("remote").Subsection("origin")
	want := []string{"+refs/heads/*:refs/remotes/origin/*", "+refs/tags/*:refs/tags/*"}
	if got := origin.OptionAll("fetch"); !slices.Equal(got, want) {
		t.Errorf("fetch = %q, want %q", got, want)
	}
	if got := origin.Option("url"); got != "git@github.com:example/example.git" {
		t.Errorf("url = %q, want it left intact", got)
	}
}

func TestStripNegativeFetchRefSpecsNoRemotes(t *testing.T) {
	// Must not create a bare [remote] section as a side effect.
	raw := decodeRaw(t, "[core]\n\tbare = false\n")

	stripNegativeFetchRefSpecs(raw)

	if raw.HasSection("remote") {
		t.Error("stripNegativeFetchRefSpecs added a remote section")
	}
}
