package tmux

import (
	"os"
	"os/exec"
)

// ArchitectWindowIndex is the reserved window index for the architect agent.
const ArchitectWindowIndex = 0

// TmuxRunner executes tmux commands.
type TmuxRunner interface {
	Run(args ...string) ([]byte, error)
	RunInteractive(args ...string) error
	RunBackground(args ...string) error
}

// execRunner is the default implementation using exec.Command.
type execRunner struct {
	tmuxPath string
}

func (r *execRunner) Run(args ...string) ([]byte, error) {
	cmd := exec.Command(r.tmuxPath, args...)
	return cmd.CombinedOutput()
}

func (r *execRunner) RunInteractive(args ...string) error {
	cmd := exec.Command(r.tmuxPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *execRunner) RunBackground(args ...string) error {
	cmd := exec.Command(r.tmuxPath, args...)
	return cmd.Start()
}

// Manager handles tmux session and window operations.
type Manager struct {
	runner TmuxRunner
}

// NewManager creates a new tmux manager.
// Returns NotInstalledError if tmux is not available.
func NewManager() (*Manager, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, &NotInstalledError{}
	}
	return &Manager{runner: &execRunner{tmuxPath: path}}, nil
}

// NewManagerWithRunner creates a Manager with a custom runner (for testing).
func NewManagerWithRunner(runner TmuxRunner) *Manager {
	return &Manager{runner: runner}
}

// Available returns true if tmux is installed and available.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// run executes a tmux command and returns output.
func (m *Manager) run(args ...string) ([]byte, error) {
	return m.runner.Run(args...)
}

// runSilent executes a tmux command and discards output.
func (m *Manager) runSilent(args ...string) error {
	_, err := m.run(args...)
	return err
}

// runBackground executes a tmux command without waiting for it to finish.
// Returns an error if the command fails to start.
func (m *Manager) runBackground(args ...string) error {
	return m.runner.RunBackground(args...)
}

// sessionTarget returns a tmux target string that enforces exact session name matching.
// The "=" prefix disables tmux's default fnmatch and substring resolution (supported since tmux 2.1).
func sessionTarget(name string) string {
	return "=" + name
}
