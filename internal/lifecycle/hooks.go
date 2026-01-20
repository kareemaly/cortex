package lifecycle

import (
	"bytes"
	"context"
	"os/exec"
)

// HookType represents the lifecycle point at which hooks are executed.
type HookType string

const (
	HookOnPickup  HookType = "on_pickup"
	HookOnSubmit  HookType = "on_submit"
	HookOnApprove HookType = "on_approve"
)

// HookDefinition represents a single hook command.
type HookDefinition struct {
	Run string `yaml:"run" json:"run"`
}

// HookResult represents the result of executing a single hook.
type HookResult struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	ExitCode int    `json:"exit_code"`
}

// ExecutionResult represents the overall result of executing hooks.
type ExecutionResult struct {
	Success bool         `json:"success"`
	Hooks   []HookResult `json:"hooks"`
}

// TemplateVars contains variables available for template expansion.
type TemplateVars struct {
	TicketID      string // All hooks
	TicketSlug    string // All hooks
	TicketTitle   string // All hooks
	CommitMessage string // on_approve only
}

// NewTemplateVars creates a TemplateVars with the given ticket information.
// Use ticket.GenerateSlug() to generate the slug from the title.
func NewTemplateVars(ticketID, ticketSlug, ticketTitle string) TemplateVars {
	return TemplateVars{
		TicketID:    ticketID,
		TicketSlug:  ticketSlug,
		TicketTitle: ticketTitle,
	}
}

// WithCommitMessage returns a copy of TemplateVars with the commit message set.
func (v TemplateVars) WithCommitMessage(message string) TemplateVars {
	v.CommitMessage = message
	return v
}

// CommandRunner executes shell commands.
type CommandRunner interface {
	Run(ctx context.Context, dir, command string) (stdout string, exitCode int, err error)
}

// shellRunner is the default CommandRunner implementation using sh -c.
type shellRunner struct{}

func (r *shellRunner) Run(ctx context.Context, dir, command string) (stdout string, exitCode int, err error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	stdout = stdoutBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return stdout, exitErr.ExitCode(), nil
		}
		return "", 0, runErr
	}

	return stdout, 0, nil
}

// Executor executes lifecycle hooks.
type Executor struct {
	runner CommandRunner
}

// NewExecutor creates a new Executor with the default shell runner.
func NewExecutor() *Executor {
	return &Executor{runner: &shellRunner{}}
}

// NewExecutorWithRunner creates a new Executor with a custom runner (for testing).
func NewExecutorWithRunner(runner CommandRunner) *Executor {
	return &Executor{runner: runner}
}

// Execute runs the given hooks sequentially in the specified directory.
// It stops on the first non-zero exit code and returns the results.
func (e *Executor) Execute(ctx context.Context, dir string, hookType HookType, hooks []HookDefinition, vars TemplateVars) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		Hooks:   make([]HookResult, 0, len(hooks)),
	}

	if len(hooks) == 0 {
		return result, nil
	}

	for _, hook := range hooks {
		// Validate template variables for this hook type
		if err := ValidateHookType(hookType, hook.Run); err != nil {
			return nil, err
		}

		// Expand template variables
		expandedCmd, err := expandTemplate(hook.Run, vars)
		if err != nil {
			return nil, err
		}

		// Execute the command
		stdout, exitCode, err := e.runner.Run(ctx, dir, expandedCmd)
		if err != nil {
			return nil, &ExecutionError{Command: expandedCmd, Err: err}
		}

		hookResult := HookResult{
			Command:  expandedCmd,
			Stdout:   stdout,
			ExitCode: exitCode,
		}
		result.Hooks = append(result.Hooks, hookResult)

		// Stop on failure
		if exitCode != 0 {
			result.Success = false
			break
		}
	}

	return result, nil
}
