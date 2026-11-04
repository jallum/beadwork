package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestParseUIArgs(t *testing.T) {
	got, err := parseUIArgs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != 8787 {
		t.Fatalf("Port = %d, want 8787", got.Port)
	}

	got, err = parseUIArgs([]string{"--port", "0"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != 0 {
		t.Fatalf("Port = %d, want 0", got.Port)
	}

	for _, args := range [][]string{{"--port"}, {"--port", "bad"}, {"--port", "65536"}, {"extra"}} {
		if _, err := parseUIArgs(args); err == nil {
			t.Errorf("parseUIArgs(%q) succeeded, want error", args)
		}
	}
}

func TestUIIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	newUIHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Beadwork Queue") {
		t.Fatal("UI response does not contain the queue title")
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestUIStateEndpoint(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	created, err := env.Store.Create("Show this issue", issue.CreateOpts{Description: "UI test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Store.Commit("create " + created.ID + " p2 task \"Show this issue\""); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	rec := httptest.NewRecorder()
	newUIHandler(env.Store).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var got uiState
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Issues) != 1 || got.Issues[0].ID != created.ID {
		t.Fatalf("Issues = %#v, want %s", got.Issues, created.ID)
	}
	if len(got.Ready) != 1 || got.Ready[0].ID != created.ID {
		t.Fatalf("Ready = %#v, want %s", got.Ready, created.ID)
	}
	if got.Revision == "" {
		t.Fatal("Revision is empty")
	}
	if got.Repo == "" {
		t.Fatal("Repo is empty")
	}
}

func TestUIRefreshRequiresPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	rec := httptest.NewRecorder()
	newUIHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
