package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

//go:embed ui/index.html
var uiIndex []byte

type UIArgs struct {
	Port int
}

func parseUIArgs(raw []string) (UIArgs, error) {
	for i, arg := range raw {
		if arg == "--port" && i+1 == len(raw) {
			return UIArgs{}, fmt.Errorf("--port requires a value")
		}
	}
	a, err := ParseArgs(raw, []string{"--port"}, nil)
	if err != nil {
		return UIArgs{}, err
	}
	port := 8787
	if a.Has("--port") {
		port, err = strconv.Atoi(a.String("--port"))
		if err != nil || port < 0 || port > 65535 {
			return UIArgs{}, fmt.Errorf("invalid --port: expected 0-65535")
		}
	}
	if len(a.Pos()) > 0 {
		return UIArgs{}, fmt.Errorf("unexpected argument: %s", a.PosFirst())
	}
	return UIArgs{Port: port}, nil
}

type uiState struct {
	Issues      []*issue.Issue `json:"issues"`
	Ready       []*issue.Issue `json:"ready"`
	Repo        string         `json:"repo"`
	Revision    string         `json:"revision"`
	GeneratedAt time.Time      `json:"generatedAt"`
}

type uiHandler struct {
	store *issue.Store
	mu    sync.Mutex
}

func newUIHandler(store *issue.Store) http.Handler {
	return &uiHandler{store: store}
}

func (h *uiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/index.html":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(uiIndex)
	case "/api/state":
		if r.Method != http.MethodGet {
			h.writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.writeState(w)
	case "/api/refresh":
		if r.Method != http.MethodPost {
			h.writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.writeState(w)
	case "/events":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.events(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *uiHandler) state() (uiState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.store.ReopenFS(); err != nil {
		return uiState{}, err
	}
	issues, err := h.store.List(issue.Filter{Statuses: issue.StatusNames()})
	if err != nil {
		return uiState{}, err
	}
	ready, err := h.store.Ready()
	if err != nil {
		return uiState{}, err
	}
	if issues == nil {
		issues = []*issue.Issue{}
	}
	if ready == nil {
		ready = []*issue.Issue{}
	}

	r, ok := h.store.Committer.(*repo.Repo)
	if !ok {
		return uiState{}, fmt.Errorf("UI requires a git-backed store")
	}
	return uiState{
		Issues:      issues,
		Ready:       ready,
		Repo:        filepath.Base(r.RepoDir()),
		Revision:    r.TreeFS().RefHash().String(),
		GeneratedAt: h.store.Now(),
	}, nil
}

func (h *uiHandler) writeState(w http.ResponseWriter) {
	state, err := h.state()
	if err != nil {
		h.writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, state)
}

func (h *uiHandler) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lastRevision := ""
	for {
		state, err := h.state()
		if err != nil {
			data, _ := json.Marshal(map[string]string{"message": err.Error()})
			fmt.Fprintf(w, "event: queue-error\ndata: %s\n\n", data)
		} else if state.Revision != lastRevision {
			data, _ := json.Marshal(state)
			fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
			lastRevision = state.Revision
		}
		flusher.Flush()

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *uiHandler) writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func cmdUI(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ua, err := parseUIArgs(args)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", ua.Port))
	if err != nil {
		return nil, fmt.Errorf("start UI: %w", err)
	}
	defer listener.Close()

	url := "http://" + listener.Addr().String()
	fmt.Fprintf(w, "Beadwork UI: %s\n", url)
	fmt.Fprintln(w, "Press Ctrl-C to stop.")

	server := &http.Server{
		Handler:           newUIHandler(store),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return nil, err
	}
	return nil, nil
}
