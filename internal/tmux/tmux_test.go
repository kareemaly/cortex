package tmux

import (
	"errors"
	"testing"
)

func TestArchitectWindowIndex(t *testing.T) {
	if ArchitectWindowIndex != 0 {
		t.Errorf("ArchitectWindowIndex = %d, want 0", ArchitectWindowIndex)
	}
}

func TestNotInstalledError(t *testing.T) {
	err := &NotInstalledError{}
	expected := "tmux is not installed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestSessionNotFoundError(t *testing.T) {
	err := &SessionNotFoundError{Session: "my-session"}
	expected := "tmux session not found: my-session"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestWindowNotFoundError(t *testing.T) {
	err := &WindowNotFoundError{Session: "my-session", Window: "my-window"}
	expected := "tmux window not found: my-session:my-window"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestCommandError(t *testing.T) {
	t.Run("with output", func(t *testing.T) {
		err := &CommandError{Command: "new-session", Output: "session exists"}
		expected := "tmux command failed: new-session: session exists"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("without output", func(t *testing.T) {
		err := &CommandError{Command: "new-session", Output: ""}
		expected := "tmux command failed: new-session"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})
}

func TestIsNotInstalled(t *testing.T) {
	t.Run("matches NotInstalledError", func(t *testing.T) {
		err := &NotInstalledError{}
		if !IsNotInstalled(err) {
			t.Error("IsNotInstalled should return true for NotInstalledError")
		}
	})

	t.Run("does not match other errors", func(t *testing.T) {
		err := errors.New("some error")
		if IsNotInstalled(err) {
			t.Error("IsNotInstalled should return false for other errors")
		}
	})

	t.Run("does not match nil", func(t *testing.T) {
		if IsNotInstalled(nil) {
			t.Error("IsNotInstalled should return false for nil")
		}
	})
}

func TestIsSessionNotFound(t *testing.T) {
	t.Run("matches SessionNotFoundError", func(t *testing.T) {
		err := &SessionNotFoundError{Session: "test"}
		if !IsSessionNotFound(err) {
			t.Error("IsSessionNotFound should return true for SessionNotFoundError")
		}
	})

	t.Run("does not match other errors", func(t *testing.T) {
		err := errors.New("some error")
		if IsSessionNotFound(err) {
			t.Error("IsSessionNotFound should return false for other errors")
		}
	})
}

func TestIsWindowNotFound(t *testing.T) {
	t.Run("matches WindowNotFoundError", func(t *testing.T) {
		err := &WindowNotFoundError{Session: "test", Window: "win"}
		if !IsWindowNotFound(err) {
			t.Error("IsWindowNotFound should return true for WindowNotFoundError")
		}
	})

	t.Run("does not match other errors", func(t *testing.T) {
		err := errors.New("some error")
		if IsWindowNotFound(err) {
			t.Error("IsWindowNotFound should return false for other errors")
		}
	})
}

func TestSwitchClient(t *testing.T) {
	t.Run("calls switch-client with session target", func(t *testing.T) {
		runner := NewMockRunner()
		mgr := NewManagerWithRunner(runner)

		err := mgr.SwitchClient("myproject")
		if err != nil {
			t.Fatalf("SwitchClient() returned error: %v", err)
		}

		// Find the switch-client call
		found := false
		for _, call := range runner.Calls {
			if len(call) >= 3 && call[0] == "switch-client" && call[1] == "-t" && call[2] == "myproject:" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected switch-client -t myproject: call, got calls: %v", runner.Calls)
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		runner := &MockRunner{}
		runner.RunFunc = func(args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "switch-client" {
				return []byte("no clients"), &CommandError{Command: "switch-client", Output: "no clients"}
			}
			return []byte{}, nil
		}
		mgr := NewManagerWithRunner(runner)

		err := mgr.SwitchClient("myproject")
		if err == nil {
			t.Fatal("expected error from SwitchClient(), got nil")
		}
	})
}

func TestAvailable(t *testing.T) {
	// This test just verifies the function doesn't panic.
	// The actual result depends on whether tmux is installed.
	_ = Available()
}
