package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/session"
)

// setupHookTestServer builds a minimal chi router with the hook endpoint wired
// up, a real HubManager, and a SessionManager seeded with a ticket session.
// Returns the server, the session manager, and the cortex session UUID.
func setupHookTestServer(t *testing.T, ticketID string) (*httptest.Server, *SessionManager, string) {
	t.Helper()

	tmpDir := t.TempDir()
	sessPath := filepath.Join(tmpDir, ".sessions.json")
	sessStore := session.NewStore(sessPath)
	sess, err := sessStore.Create(ticketID, "codex", "test-window")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	cortexSessionID := sess.SessionID

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	hubMgr, err := NewHubManager(logger)
	if err != nil {
		t.Fatalf("NewHubManager: %v", err)
	}

	sessMgr := NewSessionManager(logger)
	sessMgr.mu.Lock()
	sessMgr.stores[tmpDir] = sessStore
	sessMgr.mu.Unlock()

	deps := &Dependencies{
		SessionManager: sessMgr,
		HubManager:     hubMgr,
		Bus:            events.NewBus(),
		Logger:         logger,
	}

	r := chi.NewRouter()
	hookHandlers := NewHookHandlers(deps)
	r.Post("/hook/{agent}", hookHandlers.IngestHook)

	return httptest.NewServer(r), sessMgr, cortexSessionID
}

// lookupSession retrieves the session for cortexSessionID from sessMgr.
func lookupSession(t *testing.T, sessMgr *SessionManager, cortexSessionID string) *session.Session {
	t.Helper()
	sessMgr.mu.RLock()
	defer sessMgr.mu.RUnlock()
	for _, store := range sessMgr.stores {
		if s, err := store.GetBySessionID(cortexSessionID); err == nil {
			return s
		}
	}
	t.Fatalf("session %q not found in any store", cortexSessionID)
	return nil
}

func TestIngestHook_Codex_BackCorrelatesSessionStart(t *testing.T) {
	codexNativeID := "codex-native-sess-xyz"

	srv, sessMgr, cortexSessionID := setupHookTestServer(t, "ticket-corr-1")
	defer srv.Close()

	payload := map[string]any{
		"hook_event_name": "SessionStart",
		"session_id":      codexNativeID,
		"model":           "gpt-5.4",
	}
	body, _ := json.Marshal(payload)

	url := srv.URL + "/hook/codex?cortex_session_id=" + cortexSessionID
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /hook/codex: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	updated := lookupSession(t, sessMgr, cortexSessionID)
	if updated.AgentSessionID != codexNativeID {
		t.Errorf("AgentSessionID = %q, want %q", updated.AgentSessionID, codexNativeID)
	}
}

func TestIngestHook_Codex_IgnoresNonSessionStart(t *testing.T) {
	srv, sessMgr, cortexSessionID := setupHookTestServer(t, "ticket-corr-2")
	defer srv.Close()

	payload := map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      "codex-native-should-not-correlate",
		"tool_name":       "Bash",
	}
	body, _ := json.Marshal(payload)

	url := srv.URL + "/hook/codex?cortex_session_id=" + cortexSessionID
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /hook/codex: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	// AgentSessionID must remain empty — only SessionStart triggers correlation.
	updated := lookupSession(t, sessMgr, cortexSessionID)
	if updated.AgentSessionID != "" {
		t.Errorf("AgentSessionID should be empty for non-SessionStart, got %q", updated.AgentSessionID)
	}
}

func TestIngestHook_Codex_MissingQueryParam_NoCorrelation(t *testing.T) {
	srv, sessMgr, cortexSessionID := setupHookTestServer(t, "ticket-corr-3")
	defer srv.Close()

	payload := map[string]any{
		"hook_event_name": "SessionStart",
		"session_id":      "codex-native-no-query-param",
	}
	body, _ := json.Marshal(payload)

	// No cortex_session_id query param.
	resp, err := http.Post(srv.URL+"/hook/codex", "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /hook/codex: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	updated := lookupSession(t, sessMgr, cortexSessionID)
	if updated.AgentSessionID != "" {
		t.Errorf("AgentSessionID should be empty without query param, got %q", updated.AgentSessionID)
	}
}
