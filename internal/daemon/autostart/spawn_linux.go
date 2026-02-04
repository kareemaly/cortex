//go:build linux

package autostart

import "syscall"

// getSysProcAttr returns platform-specific process attributes for daemon spawning.
// On Linux, we use Setsid to create a new session (process group leader).
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
