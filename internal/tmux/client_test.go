package tmux

import (
	"errors"
	"os/exec"
	"testing"
)

func TestListClientsParseMultiple(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil // Session exists
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte("/dev/ttys000:mysession:0:architect\n/dev/ttys001:mysession:1:ticket-123\n"), nil
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	clients, err := mgr.ListClients("mysession")
	if err != nil {
		t.Fatalf("ListClients() returned error: %v", err)
	}

	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}

	// Verify first client
	if clients[0].TTY != "/dev/ttys000" {
		t.Errorf("client[0].TTY = %q, want /dev/ttys000", clients[0].TTY)
	}
	if clients[0].Session != "mysession" {
		t.Errorf("client[0].Session = %q, want mysession", clients[0].Session)
	}
	if clients[0].Window != 0 {
		t.Errorf("client[0].Window = %d, want 0", clients[0].Window)
	}
	if clients[0].WindowName != "architect" {
		t.Errorf("client[0].WindowName = %q, want architect", clients[0].WindowName)
	}

	// Verify second client
	if clients[1].TTY != "/dev/ttys001" {
		t.Errorf("client[1].TTY = %q, want /dev/ttys001", clients[1].TTY)
	}
	if clients[1].Window != 1 {
		t.Errorf("client[1].Window = %d, want 1", clients[1].Window)
	}
	if clients[1].WindowName != "ticket-123" {
		t.Errorf("client[1].WindowName = %q, want ticket-123", clients[1].WindowName)
	}
}

func TestListClientsNoClients(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil // Session exists
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte{}, errors.New("no clients") // tmux returns error when no clients
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	clients, err := mgr.ListClients("mysession")
	if err != nil {
		t.Fatalf("ListClients() returned error: %v", err)
	}

	if len(clients) != 0 {
		t.Errorf("expected 0 clients, got %d", len(clients))
	}
}

func TestListClientsEmptyOutput(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil // Session exists
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte(""), nil // Empty output
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	clients, err := mgr.ListClients("mysession")
	if err != nil {
		t.Fatalf("ListClients() returned error: %v", err)
	}

	if len(clients) != 0 {
		t.Errorf("expected 0 clients, got %d", len(clients))
	}
}

func TestListClientsSessionNotFound(t *testing.T) {
	// Create a real *exec.ExitError by running a command that fails
	// The "false" command always exits with code 1
	_, exitErr := exec.Command("false").CombinedOutput()

	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte("session not found"), exitErr
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	_, err := mgr.ListClients("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}

	if !IsSessionNotFound(err) {
		t.Errorf("expected SessionNotFoundError, got %T: %v", err, err)
	}
}

func TestListClientsWindowNameWithColons(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil // Session exists
		}
		if len(args) > 0 && args[0] == "list-clients" {
			// Window name contains colons: "ticket:foo:bar"
			return []byte("/dev/ttys000:mysession:0:ticket:foo:bar\n"), nil
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	clients, err := mgr.ListClients("mysession")
	if err != nil {
		t.Fatalf("ListClients() returned error: %v", err)
	}

	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}

	// Window name should include the colons
	if clients[0].WindowName != "ticket:foo:bar" {
		t.Errorf("WindowName = %q, want ticket:foo:bar", clients[0].WindowName)
	}
}

func TestIsUserAttachedTrue(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte("/dev/ttys000:mysession:0:architect\n/dev/ttys001:mysession:1:ticket-123\n"), nil
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	if !mgr.IsUserAttached("mysession", "architect") {
		t.Error("IsUserAttached should return true for attached window")
	}
	if !mgr.IsUserAttached("mysession", "ticket-123") {
		t.Error("IsUserAttached should return true for attached window")
	}
}

func TestIsUserAttachedFalse(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte("/dev/ttys000:mysession:0:architect\n"), nil
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	if mgr.IsUserAttached("mysession", "ticket-456") {
		t.Error("IsUserAttached should return false for unattached window")
	}
}

func TestIsUserAttachedReturnsFalseOnError(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte("session not found"), errors.New("exit status 1")
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	if mgr.IsUserAttached("nonexistent", "window") {
		t.Error("IsUserAttached should return false on error")
	}
}

func TestIsUserAttachedToWindowTrue(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte("/dev/ttys000:mysession:0:architect\n/dev/ttys001:mysession:1:ticket-123\n"), nil
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	if !mgr.IsUserAttachedToWindow("mysession", 0) {
		t.Error("IsUserAttachedToWindow should return true for attached window")
	}
	if !mgr.IsUserAttachedToWindow("mysession", 1) {
		t.Error("IsUserAttachedToWindow should return true for attached window")
	}
}

func TestIsUserAttachedToWindowFalse(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte{}, nil
		}
		if len(args) > 0 && args[0] == "list-clients" {
			return []byte("/dev/ttys000:mysession:0:architect\n"), nil
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	if mgr.IsUserAttachedToWindow("mysession", 2) {
		t.Error("IsUserAttachedToWindow should return false for unattached window")
	}
}

func TestIsUserAttachedToWindowReturnsFalseOnError(t *testing.T) {
	runner := &MockRunner{}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "has-session" {
			return []byte("session not found"), errors.New("exit status 1")
		}
		return []byte{}, nil
	}
	mgr := NewManagerWithRunner(runner)

	if mgr.IsUserAttachedToWindow("nonexistent", 0) {
		t.Error("IsUserAttachedToWindow should return false on error")
	}
}
