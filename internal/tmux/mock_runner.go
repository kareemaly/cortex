package tmux

// MockRunner is a test implementation of TmuxRunner.
type MockRunner struct {
	RunFunc            func(args ...string) ([]byte, error)
	RunInteractiveFunc func(args ...string) error
	Calls              [][]string
	windowExists       bool
}

// Run executes a tmux command and returns output.
func (m *MockRunner) Run(args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, args)
	if m.RunFunc != nil {
		return m.RunFunc(args...)
	}
	return []byte{}, nil
}

// RunInteractive executes an interactive tmux command.
func (m *MockRunner) RunInteractive(args ...string) error {
	m.Calls = append(m.Calls, args)
	if m.RunInteractiveFunc != nil {
		return m.RunInteractiveFunc(args...)
	}
	return nil
}

// SetWindowExists configures whether the mock reports windows as existing.
func (m *MockRunner) SetWindowExists(exists bool) {
	m.windowExists = exists
	m.RunFunc = func(args ...string) ([]byte, error) {
		if len(args) > 0 {
			switch args[0] {
			case "list-windows":
				if exists {
					return []byte("0:window:1"), nil
				}
				return []byte{}, nil // No windows
			case "has-session":
				return []byte{}, nil // Session exists
			case "new-session", "new-window", "rename-window", "send-keys", "select-window", "kill-window", "switch-client":
				if args[0] == "new-window" {
					return []byte("1"), nil
				}
				return []byte{}, nil
			}
		}
		return []byte{}, nil
	}
}

// NewMockRunner creates a MockRunner that succeeds by default.
func NewMockRunner() *MockRunner {
	runner := &MockRunner{
		windowExists: true,
	}
	runner.RunFunc = func(args ...string) ([]byte, error) {
		// Handle common tmux commands for testing
		if len(args) > 0 {
			switch args[0] {
			case "list-windows":
				if runner.windowExists {
					return []byte("0:window:1"), nil
				}
				return []byte{}, nil // No windows
			case "has-session":
				return []byte{}, nil // Session exists
			case "new-session", "new-window", "rename-window", "send-keys", "select-window", "kill-window", "switch-client":
				// Return window index for new-window
				if args[0] == "new-window" {
					return []byte("1"), nil
				}
				return []byte{}, nil
			}
		}
		return []byte{}, nil
	}
	return runner
}
