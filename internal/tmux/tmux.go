package tmux

import (
	"os/exec"
)

// ArchitectWindowIndex is the reserved window index for the architect agent.
const ArchitectWindowIndex = 0

// Manager handles tmux session and window operations.
type Manager struct {
	tmuxPath string
}

// NewManager creates a new tmux manager.
// Returns NotInstalledError if tmux is not available.
func NewManager() (*Manager, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, &NotInstalledError{}
	}
	return &Manager{tmuxPath: path}, nil
}

// Available returns true if tmux is installed and available.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// run executes a tmux command and returns output.
func (m *Manager) run(args ...string) ([]byte, error) {
	cmd := exec.Command(m.tmuxPath, args...)
	return cmd.CombinedOutput()
}

// runSilent executes a tmux command and discards output.
func (m *Manager) runSilent(args ...string) error {
	_, err := m.run(args...)
	return err
}
