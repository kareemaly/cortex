//go:build darwin

package autostart

import "syscall"

// getSysProcAttr returns platform-specific process attributes for daemon spawning.
// On macOS, we use Setpgid to create a new process group.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
