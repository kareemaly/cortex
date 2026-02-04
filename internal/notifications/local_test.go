package notifications

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// mockRunner is a test implementation of CommandRunner.
type mockRunner struct {
	lookPathFunc func(file string) (string, error)
	runFunc      func(name string, args ...string) ([]byte, error)
	runCalls     []runCall
}

type runCall struct {
	name string
	args []string
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if m.lookPathFunc != nil {
		return m.lookPathFunc(file)
	}
	return "", errors.New("not found")
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	m.runCalls = append(m.runCalls, runCall{name: name, args: args})
	if m.runFunc != nil {
		return m.runFunc(name, args...)
	}
	return nil, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestLocalChannel_Name(t *testing.T) {
	ch := NewLocalChannel(testLogger())
	if ch.Name() != "local" {
		t.Errorf("Name() = %q, want %q", ch.Name(), "local")
	}
}

func TestLocalChannel_Available_TerminalNotifier(t *testing.T) {
	runner := &mockRunner{
		lookPathFunc: func(file string) (string, error) {
			if file == "terminal-notifier" {
				return "/usr/local/bin/terminal-notifier", nil
			}
			return "", errors.New("not found")
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	ch.tool = toolTerminalNotifier // Simulate darwin detection

	// Force detection for darwin
	ch.toolOnce.Do(func() {})
	ch.tool = toolTerminalNotifier

	if !ch.Available() {
		t.Error("Available() = false, want true when terminal-notifier found")
	}
}

func TestLocalChannel_Available_NoTools(t *testing.T) {
	runner := &mockRunner{
		lookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())

	if ch.Available() {
		t.Error("Available() = true, want false when no tools found")
	}
}

func TestLocalChannel_Send_TerminalNotifier(t *testing.T) {
	runner := &mockRunner{
		lookPathFunc: func(file string) (string, error) {
			if file == "terminal-notifier" {
				return "/usr/local/bin/terminal-notifier", nil
			}
			return "", errors.New("not found")
		},
		runFunc: func(name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	// Pre-set tool to avoid platform detection
	ch.toolOnce.Do(func() {})
	ch.tool = toolTerminalNotifier

	err := ch.Send(context.Background(), Notification{
		Title: "Test Title",
		Body:  "Test Body",
		Sound: false,
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}

	call := runner.runCalls[0]
	if call.name != "terminal-notifier" {
		t.Errorf("called %q, want terminal-notifier", call.name)
	}

	argsStr := strings.Join(call.args, " ")
	if !strings.Contains(argsStr, "-title") || !strings.Contains(argsStr, "Test Title") {
		t.Errorf("args %v missing title", call.args)
	}
	if !strings.Contains(argsStr, "-message") || !strings.Contains(argsStr, "Test Body") {
		t.Errorf("args %v missing message", call.args)
	}
}

func TestLocalChannel_Send_TerminalNotifier_WithSound(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	ch.toolOnce.Do(func() {})
	ch.tool = toolTerminalNotifier

	err := ch.Send(context.Background(), Notification{
		Title: "Test",
		Body:  "Body",
		Sound: true,
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}

	argsStr := strings.Join(runner.runCalls[0].args, " ")
	if !strings.Contains(argsStr, "-sound") || !strings.Contains(argsStr, "default") {
		t.Errorf("args %v missing sound flag", runner.runCalls[0].args)
	}
}

func TestLocalChannel_Send_Osascript(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	ch.toolOnce.Do(func() {})
	ch.tool = toolOsascript

	err := ch.Send(context.Background(), Notification{
		Title: "Test Title",
		Body:  "Test Body",
		Sound: false,
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}

	call := runner.runCalls[0]
	if call.name != "osascript" {
		t.Errorf("called %q, want osascript", call.name)
	}
	if len(call.args) < 2 || call.args[0] != "-e" {
		t.Errorf("expected -e flag, got %v", call.args)
	}

	script := call.args[1]
	if !strings.Contains(script, "display notification") {
		t.Errorf("script %q missing display notification", script)
	}
	if !strings.Contains(script, "Test Body") {
		t.Errorf("script %q missing body", script)
	}
	if !strings.Contains(script, "Test Title") {
		t.Errorf("script %q missing title", script)
	}
}

func TestLocalChannel_Send_Osascript_WithSound(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	ch.toolOnce.Do(func() {})
	ch.tool = toolOsascript

	err := ch.Send(context.Background(), Notification{
		Title: "Test",
		Body:  "Body",
		Sound: true,
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}

	script := runner.runCalls[0].args[1]
	if !strings.Contains(script, `sound name "default"`) {
		t.Errorf("script %q missing sound directive", script)
	}
}

func TestLocalChannel_Send_NotifySend(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	ch.toolOnce.Do(func() {})
	ch.tool = toolNotifySend

	err := ch.Send(context.Background(), Notification{
		Title: "Test Title",
		Body:  "Test Body",
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}

	call := runner.runCalls[0]
	if call.name != "notify-send" {
		t.Errorf("called %q, want notify-send", call.name)
	}

	// Check args: -u normal "Title" "Body"
	if len(call.args) < 4 {
		t.Fatalf("expected at least 4 args, got %d", len(call.args))
	}
	if call.args[0] != "-u" || call.args[1] != "normal" {
		t.Errorf("expected -u normal, got %v", call.args[:2])
	}
	if call.args[2] != "Test Title" {
		t.Errorf("title = %q, want Test Title", call.args[2])
	}
	if call.args[3] != "Test Body" {
		t.Errorf("body = %q, want Test Body", call.args[3])
	}
}

func TestLocalChannel_Send_NotifySend_Urgency(t *testing.T) {
	tests := []struct {
		name    string
		urgency string
		want    string
	}{
		{"low urgency", "low", "low"},
		{"normal urgency", "normal", "normal"},
		{"critical urgency", "critical", "critical"},
		{"empty defaults to normal", "", "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockRunner{
				runFunc: func(name string, args ...string) ([]byte, error) {
					return nil, nil
				},
			}
			ch := NewLocalChannelWithRunner(runner, testLogger())
			ch.toolOnce.Do(func() {})
			ch.tool = toolNotifySend

			err := ch.Send(context.Background(), Notification{
				Title:   "Test",
				Body:    "Body",
				Urgency: tt.urgency,
			})

			if err != nil {
				t.Errorf("Send() error = %v", err)
			}

			if len(runner.runCalls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(runner.runCalls))
			}

			args := runner.runCalls[0].args
			if args[1] != tt.want {
				t.Errorf("urgency = %q, want %q", args[1], tt.want)
			}
		})
	}
}

func TestLocalChannel_Send_NoTools_GracefulDegradation(t *testing.T) {
	runner := &mockRunner{
		lookPathFunc: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())

	// This should NOT error - graceful degradation
	err := ch.Send(context.Background(), Notification{
		Title: "Test",
		Body:  "Body",
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil for graceful degradation", err)
	}

	// No run calls should be made
	if len(runner.runCalls) != 0 {
		t.Errorf("expected 0 run calls, got %d", len(runner.runCalls))
	}
}

func TestLocalChannel_Send_ToolError(t *testing.T) {
	runner := &mockRunner{
		runFunc: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("tool failed")
		},
	}
	ch := NewLocalChannelWithRunner(runner, testLogger())
	ch.toolOnce.Do(func() {})
	ch.tool = toolTerminalNotifier

	err := ch.Send(context.Background(), Notification{
		Title: "Test",
		Body:  "Body",
	})

	if err == nil {
		t.Error("Send() error = nil, want error when tool fails")
	}
}

func TestLocalChannel_ImplementsChannel(t *testing.T) {
	var _ Channel = (*LocalChannel)(nil)
}
