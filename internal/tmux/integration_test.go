//go:build integration

package tmux

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func skipIfCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI environment")
	}
}

func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("tmux not available")
	}
}

func testSessionName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("cortex-test-%d", time.Now().UnixNano())
}

// parsePaneWidths parses pane width info from list-panes output.
// Expected format: "0:100\n1:200" where format is "pane_index:pane_width"
func parsePaneWidths(panes []string) []int {
	widths := make([]int, len(panes))
	for _, pane := range panes {
		parts := strings.Split(pane, ":")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx < 0 || idx >= len(widths) {
			continue
		}
		width, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		widths[idx] = width
	}
	return widths
}

func TestIntegrationNewManager(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	if m == nil {
		t.Error("manager should not be nil")
	}
}

func TestIntegrationSessionLifecycle(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	// Session should not exist initially
	exists, err := m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if exists {
		t.Error("session should not exist initially")
	}

	// Create session
	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Session should exist now
	exists, err = m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if !exists {
		t.Error("session should exist after creation")
	}

	// Creating again should be no-op
	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession (idempotent) failed: %v", err)
	}

	// Kill session
	if err := m.KillSession(session); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}

	// Session should not exist after kill
	exists, err = m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if exists {
		t.Error("session should not exist after kill")
	}
}

func TestIntegrationKillSessionNotFound(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = m.KillSession("nonexistent-session-xyz")
	if !IsSessionNotFound(err) {
		t.Errorf("expected SessionNotFoundError, got %T: %v", err, err)
	}
}

func TestIntegrationWindowLifecycle(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// List windows (should have 1 default window)
	windows, err := m.ListWindows(session)
	if err != nil {
		t.Fatalf("ListWindows failed: %v", err)
	}
	if len(windows) != 1 {
		t.Errorf("expected 1 window, got %d", len(windows))
	}

	// Create new window
	windowName := "test-window"
	index, err := m.CreateWindow(session, windowName, "")
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}
	if index == 0 {
		t.Error("new window should not have index 0")
	}

	// Window should exist
	exists, err := m.WindowExists(session, windowName)
	if err != nil {
		t.Fatalf("WindowExists failed: %v", err)
	}
	if !exists {
		t.Error("window should exist after creation")
	}

	// List windows again
	windows, err = m.ListWindows(session)
	if err != nil {
		t.Fatalf("ListWindows failed: %v", err)
	}
	if len(windows) != 2 {
		t.Errorf("expected 2 windows, got %d", len(windows))
	}

	// Get window by name
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		t.Fatalf("GetWindowByName failed: %v", err)
	}
	if window.Name != windowName {
		t.Errorf("window name = %q, want %q", window.Name, windowName)
	}
	if window.Index != index {
		t.Errorf("window index = %d, want %d", window.Index, index)
	}

	// Kill window
	if err := m.KillWindow(session, windowName); err != nil {
		t.Fatalf("KillWindow failed: %v", err)
	}

	// Window should not exist
	exists, err = m.WindowExists(session, windowName)
	if err != nil {
		t.Fatalf("WindowExists failed: %v", err)
	}
	if exists {
		t.Error("window should not exist after kill")
	}
}

func TestIntegrationCreateArchitectWindow(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	architectName := "architect"
	if err := m.CreateArchitectWindow(session, architectName); err != nil {
		t.Fatalf("CreateArchitectWindow failed: %v", err)
	}

	// Window 0 should have the architect name
	window, err := m.GetWindowByName(session, architectName)
	if err != nil {
		t.Fatalf("GetWindowByName failed: %v", err)
	}
	if window.Index != ArchitectWindowIndex {
		t.Errorf("architect window index = %d, want %d", window.Index, ArchitectWindowIndex)
	}
}

func TestIntegrationFocusWindow(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Create second window
	windowName := "second-window"
	_, err = m.CreateWindow(session, windowName, "")
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}

	// Focus first window (rename it first so we have a known name)
	firstName := "first"
	if err := m.CreateArchitectWindow(session, firstName); err != nil {
		t.Fatalf("CreateArchitectWindow failed: %v", err)
	}

	if err := m.FocusWindow(session, firstName); err != nil {
		t.Fatalf("FocusWindow failed: %v", err)
	}

	active, err := m.GetActiveWindow(session)
	if err != nil {
		t.Fatalf("GetActiveWindow failed: %v", err)
	}
	if active.Name != firstName {
		t.Errorf("active window = %q, want %q", active.Name, firstName)
	}
}

func TestIntegrationRunCommand(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	windowName := "cmd-window"
	_, err = m.CreateWindow(session, windowName, "")
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}

	// Run a simple command (just echo)
	if err := m.RunCommand(session, windowName, "echo hello"); err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
}

func TestIntegrationSendKeys(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	windowName := "keys-window"
	_, err = m.CreateWindow(session, windowName, "")
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}

	// Send C-c (should not error)
	if err := m.SendKeys(session, windowName, "C-c"); err != nil {
		t.Fatalf("SendKeys failed: %v", err)
	}
}

func TestIntegrationSpawnAgent(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	windowName := "agent-window"
	index, err := m.SpawnAgent(session, windowName, "echo agent started", "", "", "")
	if err != nil {
		t.Fatalf("SpawnAgent failed: %v", err)
	}

	// Session should exist
	exists, err := m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if !exists {
		t.Error("session should exist after SpawnAgent")
	}

	// Window should exist
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		t.Fatalf("GetWindowByName failed: %v", err)
	}
	if window.Index != index {
		t.Errorf("window index = %d, want %d", window.Index, index)
	}
}

func TestIntegrationSpawnAgentWithCompanion(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	windowName := "agent-with-companion"
	index, err := m.SpawnAgent(session, windowName, "echo agent", "echo companion", "", "")
	if err != nil {
		t.Fatalf("SpawnAgent with companion failed: %v", err)
	}

	// Session should exist
	exists, err := m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if !exists {
		t.Error("session should exist after SpawnAgent")
	}

	// Window should exist
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		t.Fatalf("GetWindowByName failed: %v", err)
	}
	if window.Index != index {
		t.Errorf("window index = %d, want %d", window.Index, index)
	}

	// Verify panes exist and have correct width ratio (30%/70%)
	output, err := m.run("list-panes", "-t", fmt.Sprintf("%s:%d", session, index), "-F", "#{pane_index}:#{pane_width}")
	if err != nil {
		t.Fatalf("list-panes failed: %v", err)
	}
	panes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(panes) != 2 {
		t.Errorf("expected 2 panes, got %d", len(panes))
	}

	// Verify pane width percentages (allowing some tolerance)
	widths := parsePaneWidths(panes)
	if len(widths) == 2 {
		totalWidth := widths[0] + widths[1]
		pane0Pct := float64(widths[0]) / float64(totalWidth) * 100
		pane1Pct := float64(widths[1]) / float64(totalWidth) * 100
		// Agent pane (0) should be ~30%, companion pane (1) should be ~70%
		// Allow 5% tolerance for rounding
		if pane0Pct < 25 || pane0Pct > 35 {
			t.Errorf("agent pane (0) width = %.1f%%, want ~30%%", pane0Pct)
		}
		if pane1Pct < 65 || pane1Pct > 75 {
			t.Errorf("companion pane (1) width = %.1f%%, want ~70%%", pane1Pct)
		}
	}
}

func TestIntegrationSpawnArchitect(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	windowName := "architect"
	if err := m.SpawnArchitect(session, windowName, "echo architect started", "", "", ""); err != nil {
		t.Fatalf("SpawnArchitect failed: %v", err)
	}

	// Session should exist
	exists, err := m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if !exists {
		t.Error("session should exist after SpawnArchitect")
	}

	// Window 0 should be the architect
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		t.Fatalf("GetWindowByName failed: %v", err)
	}
	if window.Index != ArchitectWindowIndex {
		t.Errorf("architect window index = %d, want %d", window.Index, ArchitectWindowIndex)
	}
}

func TestIntegrationSpawnArchitectWithCompanion(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	windowName := "architect"
	if err := m.SpawnArchitect(session, windowName, "echo architect", "echo kanban", "", ""); err != nil {
		t.Fatalf("SpawnArchitect with companion failed: %v", err)
	}

	// Session should exist
	exists, err := m.SessionExists(session)
	if err != nil {
		t.Fatalf("SessionExists failed: %v", err)
	}
	if !exists {
		t.Error("session should exist after SpawnArchitect")
	}

	// Window 0 should be the architect
	window, err := m.GetWindowByName(session, windowName)
	if err != nil {
		t.Fatalf("GetWindowByName failed: %v", err)
	}
	if window.Index != ArchitectWindowIndex {
		t.Errorf("architect window index = %d, want %d", window.Index, ArchitectWindowIndex)
	}

	// Verify panes exist and have correct width ratio (30%/70%)
	output, err := m.run("list-panes", "-t", fmt.Sprintf("%s:%d", session, ArchitectWindowIndex), "-F", "#{pane_index}:#{pane_width}")
	if err != nil {
		t.Fatalf("list-panes failed: %v", err)
	}
	panes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(panes) != 2 {
		t.Errorf("expected 2 panes, got %d", len(panes))
	}

	// Verify pane width percentages (allowing some tolerance)
	widths := parsePaneWidths(panes)
	if len(widths) == 2 {
		totalWidth := widths[0] + widths[1]
		pane0Pct := float64(widths[0]) / float64(totalWidth) * 100
		pane1Pct := float64(widths[1]) / float64(totalWidth) * 100
		// Agent pane (0) should be ~30%, companion pane (1) should be ~70%
		// Allow 5% tolerance for rounding
		if pane0Pct < 25 || pane0Pct > 35 {
			t.Errorf("architect pane (0) width = %.1f%%, want ~30%%", pane0Pct)
		}
		if pane1Pct < 65 || pane1Pct > 75 {
			t.Errorf("kanban pane (1) width = %.1f%%, want ~70%%", pane1Pct)
		}
	}
}

func TestIntegrationSplitWindowHorizontal(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Split the default window (index 0)
	if err := m.SplitWindowHorizontal(session, 0, ""); err != nil {
		t.Fatalf("SplitWindowHorizontal failed: %v", err)
	}

	// Verify we now have 2 panes with correct width ratio (30%/70%)
	output, err := m.run("list-panes", "-t", fmt.Sprintf("%s:0", session), "-F", "#{pane_index}:#{pane_width}")
	if err != nil {
		t.Fatalf("list-panes failed: %v", err)
	}
	panes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(panes) != 2 {
		t.Errorf("expected 2 panes after split, got %d", len(panes))
	}

	// Verify pane width percentages
	widths := parsePaneWidths(panes)
	if len(widths) == 2 {
		totalWidth := widths[0] + widths[1]
		pane0Pct := float64(widths[0]) / float64(totalWidth) * 100
		pane1Pct := float64(widths[1]) / float64(totalWidth) * 100
		// Original pane (0) should be ~30%, new pane (1) should be ~70%
		// Allow 5% tolerance for rounding
		if pane0Pct < 25 || pane0Pct > 35 {
			t.Errorf("original pane (0) width = %.1f%%, want ~30%%", pane0Pct)
		}
		if pane1Pct < 65 || pane1Pct > 75 {
			t.Errorf("new pane (1) width = %.1f%%, want ~70%%", pane1Pct)
		}
	}
}

func TestIntegrationRunCommandInPane(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Split the window first
	if err := m.SplitWindowHorizontal(session, 0, ""); err != nil {
		t.Fatalf("SplitWindowHorizontal failed: %v", err)
	}

	// Run command in pane 0
	if err := m.RunCommandInPane(session, 0, 0, "echo pane0"); err != nil {
		t.Fatalf("RunCommandInPane (pane 0) failed: %v", err)
	}

	// Run command in pane 1
	if err := m.RunCommandInPane(session, 0, 1, "echo pane1"); err != nil {
		t.Fatalf("RunCommandInPane (pane 1) failed: %v", err)
	}
}

func TestIntegrationSelectPane(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Split the window first
	if err := m.SplitWindowHorizontal(session, 0, ""); err != nil {
		t.Fatalf("SplitWindowHorizontal failed: %v", err)
	}

	// Select pane 0
	if err := m.SelectPane(session, 0, 0); err != nil {
		t.Fatalf("SelectPane (pane 0) failed: %v", err)
	}

	// Select pane 1
	if err := m.SelectPane(session, 0, 1); err != nil {
		t.Fatalf("SelectPane (pane 1) failed: %v", err)
	}
}

func TestIntegrationWindowNotFoundError(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session := testSessionName(t)
	defer func() {
		_ = m.KillSession(session)
	}()

	if err := m.CreateSession(session, ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	_, err = m.GetWindowByName(session, "nonexistent-window")
	if !IsWindowNotFound(err) {
		t.Errorf("expected WindowNotFoundError, got %T: %v", err, err)
	}
}

func TestIntegrationSessionNotFoundForWindow(t *testing.T) {
	skipIfCI(t)
	skipIfNoTmux(t)

	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = m.ListWindows("nonexistent-session-xyz")
	if !IsSessionNotFound(err) {
		t.Errorf("expected SessionNotFoundError, got %T: %v", err, err)
	}
}
