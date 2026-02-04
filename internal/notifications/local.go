package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"sync"
)

// CommandRunner abstracts exec.Command for testing.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
	LookPath(file string) (string, error)
}

// execRunner is the default implementation using exec.Command.
type execRunner struct{}

func (r *execRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func (r *execRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// notifyTool represents a platform notification tool.
type notifyTool int

const (
	toolNone notifyTool = iota
	toolTerminalNotifier
	toolOsascript
	toolNotifySend
)

func (t notifyTool) String() string {
	switch t {
	case toolTerminalNotifier:
		return "terminal-notifier"
	case toolOsascript:
		return "osascript"
	case toolNotifySend:
		return "notify-send"
	default:
		return "none"
	}
}

// LocalChannel sends desktop notifications via platform-native tools.
type LocalChannel struct {
	runner CommandRunner
	logger *slog.Logger

	toolOnce sync.Once
	tool     notifyTool
}

// NewLocalChannel creates a LocalChannel with default exec runner.
func NewLocalChannel(logger *slog.Logger) *LocalChannel {
	return &LocalChannel{
		runner: &execRunner{},
		logger: logger,
	}
}

// NewLocalChannelWithRunner creates a LocalChannel with custom runner (for testing).
func NewLocalChannelWithRunner(runner CommandRunner, logger *slog.Logger) *LocalChannel {
	return &LocalChannel{
		runner: runner,
		logger: logger,
	}
}

// Name returns the channel name.
func (c *LocalChannel) Name() string {
	return "local"
}

// detectTool finds the best available notification tool for the platform.
func (c *LocalChannel) detectTool() notifyTool {
	switch runtime.GOOS {
	case "darwin":
		if _, err := c.runner.LookPath("terminal-notifier"); err == nil {
			return toolTerminalNotifier
		}
		if _, err := c.runner.LookPath("osascript"); err == nil {
			return toolOsascript
		}
	case "linux":
		if _, err := c.runner.LookPath("notify-send"); err == nil {
			return toolNotifySend
		}
	}
	return toolNone
}

// getTool returns the cached notification tool, detecting on first call.
func (c *LocalChannel) getTool() notifyTool {
	c.toolOnce.Do(func() {
		c.tool = c.detectTool()
		if c.tool != toolNone {
			c.logger.Info("notification tool detected", "tool", c.tool.String())
		} else {
			c.logger.Warn("no notification tool available")
		}
	})
	return c.tool
}

// Available returns true if at least one notification tool is found.
func (c *LocalChannel) Available() bool {
	return c.getTool() != toolNone
}

// Send delivers a notification using the platform-native tool.
// Returns nil (not error) if no tools available - daemon shouldn't crash.
func (c *LocalChannel) Send(ctx context.Context, n Notification) error {
	tool := c.getTool()
	if tool == toolNone {
		c.logger.Debug("skipping notification, no tool available", "title", n.Title)
		return nil
	}

	var err error
	switch tool {
	case toolTerminalNotifier:
		err = c.sendTerminalNotifier(n)
	case toolOsascript:
		err = c.sendOsascript(n)
	case toolNotifySend:
		err = c.sendNotifySend(n)
	}

	if err != nil {
		c.logger.Error("failed to send notification", "tool", tool.String(), "error", err)
		return err
	}

	c.logger.Debug("notification sent", "tool", tool.String(), "title", n.Title)
	return nil
}

// sendTerminalNotifier sends notification via terminal-notifier (macOS).
func (c *LocalChannel) sendTerminalNotifier(n Notification) error {
	args := []string{"-title", n.Title, "-message", n.Body}
	if n.Sound {
		args = append(args, "-sound", "default")
	}
	_, err := c.runner.Run("terminal-notifier", args...)
	return err
}

// sendOsascript sends notification via osascript (macOS fallback).
func (c *LocalChannel) sendOsascript(n Notification) error {
	script := fmt.Sprintf(`display notification %q with title %q`, n.Body, n.Title)
	if n.Sound {
		script += ` sound name "default"`
	}
	_, err := c.runner.Run("osascript", "-e", script)
	return err
}

// sendNotifySend sends notification via notify-send (Linux).
func (c *LocalChannel) sendNotifySend(n Notification) error {
	urgency := n.Urgency
	if urgency == "" {
		urgency = "normal"
	}
	args := []string{"-u", urgency, n.Title, n.Body}
	_, err := c.runner.Run("notify-send", args...)
	return err
}
