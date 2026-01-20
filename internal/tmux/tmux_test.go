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

func TestAvailable(t *testing.T) {
	// This test just verifies the function doesn't panic.
	// The actual result depends on whether tmux is installed.
	_ = Available()
}
