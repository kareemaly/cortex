package tmux

import "fmt"

// DisplayPopup opens a tmux popup window that closes when command exits.
// The popup is displayed in the current tmux session.
// Parameters:
//   - session: the tmux session name
//   - workingDir: the working directory for the command (can be empty)
//   - command: the command to execute in the popup
func (m *Manager) DisplayPopup(session, workingDir, command string) error {
	args := []string{"display-popup", "-E", "-w", "80%", "-h", "80%", "-t", sessionTarget(session) + ":"}
	if workingDir != "" {
		args = append(args, "-d", workingDir)
		command = fmt.Sprintf("cd %q && %s", workingDir, command)
	}
	args = append(args, command)
	return m.runBackground(args...)
}
