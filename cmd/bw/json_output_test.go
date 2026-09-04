package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// tokenLookalike is real-world user content that collides with the display-token
// grammar: an Elixir map literal whose `{type: ...}` prefix matches the {type:...}
// token, and whose `<`/`>` are JSON-escaped to </> by encoding/json.
// Before the rawSink fix, token resolution rewrote this inside serialized JSON
// (uppercasing the escape to the invalid \U003C, dropping braces), producing
// output that fails json.Valid.
const tokenLookalike = `Backfill: copy entries keyed ` + "`" + `%{type: "MEMORY", id: <MEMORY_RECORD_ID>}` + "`" + ` into the map. Also {status:open} and {p:1} must survive verbatim.`

func TestFprintJSONBypassesTokenResolution(t *testing.T) {
	v := map[string]string{"description": tokenLookalike}

	cases := []struct {
		name string
		mk   func(buf *bytes.Buffer) Writer
	}{
		{"plain", func(buf *bytes.Buffer) Writer { return PlainWriter(buf) }},
		// Narrow width: proves JSON is neither wrapped nor colorized in TTY mode.
		{"color", func(buf *bytes.Buffer) Writer { return ColorWriter(buf, 20) }},
	}

	for _, tc := range cases {
		name := tc.name
		var buf bytes.Buffer
		w := tc.mk(&buf)

		fprintJSON(w, v)
		out := buf.Bytes()

		if !json.Valid(out) {
			t.Fatalf("[%s] fprintJSON produced invalid JSON:\n%s", name, out)
		}

		var decoded map[string]string
		if err := json.Unmarshal(out, &decoded); err != nil {
			t.Fatalf("[%s] unmarshal: %v", name, err)
		}
		if decoded["description"] != tokenLookalike {
			t.Errorf("[%s] content altered by output path:\n got: %q\nwant: %q",
				name, decoded["description"], tokenLookalike)
		}
		if strings.Contains(string(out), "\033[") {
			t.Errorf("[%s] ANSI escapes leaked into JSON output: %q", name, out)
		}
	}
}

func TestCmdListJSONRoundTripsTokenLookalikeContent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	created, err := env.Store.Create("Issue with tricky description",
		issue.CreateOpts{Description: tokenLookalike})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var buf bytes.Buffer
	if _, err := cmdList(env.Store, []string{"--json"}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdList: %v", err)
	}

	out := buf.Bytes()
	if !json.Valid(out) {
		t.Fatalf("list --json produced invalid JSON:\n%s", out)
	}

	var issues []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(out, &issues); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != created.ID {
		t.Fatalf("unexpected issues payload: %+v", issues)
	}
	if issues[0].Description != tokenLookalike {
		t.Errorf("description altered:\n got: %q\nwant: %q", issues[0].Description, tokenLookalike)
	}
}

func TestCmdExportEmitsValidJSONLWithoutXRaw(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	if _, err := env.Store.Create("Export victim", issue.CreateOpts{Description: tokenLookalike}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// PlainWriter (a resolving writer) is exactly what a piped `bw export`
	// gets — the case that used to corrupt without --x-raw.
	var buf bytes.Buffer
	if _, err := cmdExport(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSONL line, got %d:\n%s", len(lines), buf.String())
	}

	var rec struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("export line is not valid JSON: %v\n%s", err, lines[0])
	}
	if rec.Description != tokenLookalike {
		t.Errorf("description altered:\n got: %q\nwant: %q", rec.Description, tokenLookalike)
	}
}
