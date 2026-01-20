package lifecycle

import (
	"context"
	"errors"
	"testing"
)

// mockRunner is a test implementation of CommandRunner.
type mockRunner struct {
	results []mockResult
	calls   []mockCall
	index   int
}

type mockResult struct {
	stdout   string
	exitCode int
	err      error
}

type mockCall struct {
	dir     string
	command string
}

func newMockRunner(results ...mockResult) *mockRunner {
	return &mockRunner{results: results}
}

func (m *mockRunner) Run(ctx context.Context, dir, command string) (string, int, error) {
	m.calls = append(m.calls, mockCall{dir: dir, command: command})

	if m.index >= len(m.results) {
		return "", 0, nil
	}

	result := m.results[m.index]
	m.index++
	return result.stdout, result.exitCode, result.err
}

func TestExecutor_Execute_AllHooksSucceed(t *testing.T) {
	runner := newMockRunner(
		mockResult{stdout: "lint ok", exitCode: 0},
		mockResult{stdout: "test ok", exitCode: 0},
	)
	executor := NewExecutorWithRunner(runner)

	hooks := []HookDefinition{
		{Run: "npm run lint"},
		{Run: "npm run test"},
	}
	vars := NewTemplateVars("123", "my-ticket", "My Ticket")

	result, err := executor.Execute(context.Background(), "/project", HookOnSubmit, hooks, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if len(result.Hooks) != 2 {
		t.Fatalf("expected 2 hook results, got %d", len(result.Hooks))
	}

	if result.Hooks[0].Command != "npm run lint" {
		t.Errorf("expected command 'npm run lint', got %q", result.Hooks[0].Command)
	}
	if result.Hooks[0].Stdout != "lint ok" {
		t.Errorf("expected stdout 'lint ok', got %q", result.Hooks[0].Stdout)
	}
	if result.Hooks[0].ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.Hooks[0].ExitCode)
	}

	if result.Hooks[1].Command != "npm run test" {
		t.Errorf("expected command 'npm run test', got %q", result.Hooks[1].Command)
	}
}

func TestExecutor_Execute_StopsOnFirstFailure(t *testing.T) {
	runner := newMockRunner(
		mockResult{stdout: "lint failed", exitCode: 1},
		mockResult{stdout: "should not run", exitCode: 0},
	)
	executor := NewExecutorWithRunner(runner)

	hooks := []HookDefinition{
		{Run: "npm run lint"},
		{Run: "npm run test"},
	}
	vars := NewTemplateVars("123", "my-ticket", "My Ticket")

	result, err := executor.Execute(context.Background(), "/project", HookOnSubmit, hooks, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success to be false")
	}

	if len(result.Hooks) != 1 {
		t.Fatalf("expected 1 hook result (stopped on failure), got %d", len(result.Hooks))
	}

	if result.Hooks[0].ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.Hooks[0].ExitCode)
	}

	// Verify second command was not called
	if len(runner.calls) != 1 {
		t.Errorf("expected 1 call (stopped on failure), got %d", len(runner.calls))
	}
}

func TestExecutor_Execute_TemplateExpansion(t *testing.T) {
	runner := newMockRunner(
		mockResult{stdout: "ok", exitCode: 0},
	)
	executor := NewExecutorWithRunner(runner)

	hooks := []HookDefinition{
		{Run: "echo '{{ticket_id}} - {{ticket_slug}} - {{ticket_title}}'"},
	}
	vars := NewTemplateVars("abc-123", "fix-login", "Fix Login Bug")

	_, err := executor.Execute(context.Background(), "/project", HookOnSubmit, hooks, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}

	expected := "echo 'abc-123 - fix-login - Fix Login Bug'"
	if runner.calls[0].command != expected {
		t.Errorf("expected command %q, got %q", expected, runner.calls[0].command)
	}
}

func TestExecutor_Execute_CommitMessageInApprove(t *testing.T) {
	runner := newMockRunner(
		mockResult{stdout: "ok", exitCode: 0},
	)
	executor := NewExecutorWithRunner(runner)

	hooks := []HookDefinition{
		{Run: "git commit -m '{{commit_message}}'"},
	}
	vars := NewTemplateVars("123", "feat", "Feature").WithCommitMessage("Add new feature")

	_, err := executor.Execute(context.Background(), "/project", HookOnApprove, hooks, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "git commit -m 'Add new feature'"
	if runner.calls[0].command != expected {
		t.Errorf("expected command %q, got %q", expected, runner.calls[0].command)
	}
}

func TestExecutor_Execute_CommitMessageInNonApproveHook(t *testing.T) {
	runner := newMockRunner()
	executor := NewExecutorWithRunner(runner)

	hooks := []HookDefinition{
		{Run: "echo '{{commit_message}}'"},
	}
	vars := NewTemplateVars("123", "feat", "Feature")

	_, err := executor.Execute(context.Background(), "/project", HookOnSubmit, hooks, vars)
	if err == nil {
		t.Fatal("expected error for commit_message in non-approve hook")
	}

	if !IsInvalidVariable(err) {
		t.Errorf("expected InvalidVariableError, got %T", err)
	}
}

func TestExecutor_Execute_ExecutionError(t *testing.T) {
	runner := newMockRunner(
		mockResult{err: errors.New("command not found")},
	)
	executor := NewExecutorWithRunner(runner)

	hooks := []HookDefinition{
		{Run: "nonexistent-command"},
	}
	vars := NewTemplateVars("123", "feat", "Feature")

	_, err := executor.Execute(context.Background(), "/project", HookOnSubmit, hooks, vars)
	if err == nil {
		t.Fatal("expected error for execution failure")
	}

	if !IsExecutionError(err) {
		t.Errorf("expected ExecutionError, got %T", err)
	}
}

func TestExecutor_Execute_EmptyHooks(t *testing.T) {
	runner := newMockRunner()
	executor := NewExecutorWithRunner(runner)

	vars := NewTemplateVars("123", "feat", "Feature")

	result, err := executor.Execute(context.Background(), "/project", HookOnSubmit, []HookDefinition{}, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true for empty hooks")
	}

	if len(result.Hooks) != 0 {
		t.Errorf("expected 0 hook results, got %d", len(result.Hooks))
	}
}

func TestExecutor_Execute_NilHooks(t *testing.T) {
	runner := newMockRunner()
	executor := NewExecutorWithRunner(runner)

	vars := NewTemplateVars("123", "feat", "Feature")

	result, err := executor.Execute(context.Background(), "/project", HookOnSubmit, nil, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected Success to be true for nil hooks")
	}
}

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		vars     TemplateVars
		expected string
	}{
		{
			name:     "no variables",
			command:  "npm run test",
			vars:     NewTemplateVars("123", "slug", "Title"),
			expected: "npm run test",
		},
		{
			name:     "single variable",
			command:  "echo {{ticket_id}}",
			vars:     NewTemplateVars("abc-123", "slug", "Title"),
			expected: "echo abc-123",
		},
		{
			name:     "multiple variables",
			command:  "{{ticket_id}}-{{ticket_slug}}-{{ticket_title}}",
			vars:     NewTemplateVars("id", "slug", "title"),
			expected: "id-slug-title",
		},
		{
			name:     "unknown variable left unchanged",
			command:  "echo {{unknown_var}}",
			vars:     NewTemplateVars("123", "slug", "Title"),
			expected: "echo {{unknown_var}}",
		},
		{
			name:     "commit message",
			command:  "git commit -m '{{commit_message}}'",
			vars:     NewTemplateVars("123", "slug", "Title").WithCommitMessage("Fix bug"),
			expected: "git commit -m 'Fix bug'",
		},
		{
			name:     "repeated variable",
			command:  "{{ticket_id}} and {{ticket_id}}",
			vars:     NewTemplateVars("123", "slug", "Title"),
			expected: "123 and 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandTemplate(tt.command, tt.vars)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateHookType(t *testing.T) {
	tests := []struct {
		name     string
		hookType HookType
		command  string
		wantErr  bool
	}{
		{
			name:     "ticket_id in on_pickup",
			hookType: HookOnPickup,
			command:  "echo {{ticket_id}}",
			wantErr:  false,
		},
		{
			name:     "ticket_slug in on_submit",
			hookType: HookOnSubmit,
			command:  "echo {{ticket_slug}}",
			wantErr:  false,
		},
		{
			name:     "commit_message in on_approve",
			hookType: HookOnApprove,
			command:  "git commit -m '{{commit_message}}'",
			wantErr:  false,
		},
		{
			name:     "commit_message in on_pickup",
			hookType: HookOnPickup,
			command:  "echo {{commit_message}}",
			wantErr:  true,
		},
		{
			name:     "commit_message in on_submit",
			hookType: HookOnSubmit,
			command:  "echo {{commit_message}}",
			wantErr:  true,
		},
		{
			name:     "unknown variable allowed",
			hookType: HookOnSubmit,
			command:  "echo {{unknown}}",
			wantErr:  false,
		},
		{
			name:     "no variables",
			hookType: HookOnSubmit,
			command:  "npm run test",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHookType(tt.hookType, tt.command)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("ExecutionError", func(t *testing.T) {
		err := &ExecutionError{Command: "test", Err: errors.New("failed")}
		if !IsExecutionError(err) {
			t.Error("IsExecutionError should return true")
		}
		if err.Error() != `failed to execute command "test": failed` {
			t.Errorf("unexpected error message: %s", err.Error())
		}
		if err.Unwrap() == nil {
			t.Error("Unwrap should return underlying error")
		}
	})

	t.Run("TemplateError", func(t *testing.T) {
		err := &TemplateError{Template: "{{bad}}", Reason: "invalid"}
		if !IsTemplateError(err) {
			t.Error("IsTemplateError should return true")
		}
		if err.Error() != `template expansion failed for "{{bad}}": invalid` {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("InvalidVariableError", func(t *testing.T) {
		err := &InvalidVariableError{Variable: "commit_message", HookType: HookOnSubmit}
		if !IsInvalidVariable(err) {
			t.Error("IsInvalidVariable should return true")
		}
		if err.Error() != `variable "commit_message" is not available in on_submit hooks` {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})
}

func TestExtractTemplateVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no vars",
			input:    "npm run test",
			expected: []string{},
		},
		{
			name:     "single var",
			input:    "echo {{ticket_id}}",
			expected: []string{"ticket_id"},
		},
		{
			name:     "multiple vars",
			input:    "{{ticket_id}} {{ticket_slug}}",
			expected: []string{"ticket_id", "ticket_slug"},
		},
		{
			name:     "duplicate vars",
			input:    "{{ticket_id}} {{ticket_id}}",
			expected: []string{"ticket_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTemplateVars(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d vars, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("expected var %q at index %d, got %q", tt.expected[i], i, v)
				}
			}
		})
	}
}

func TestContainsTemplateVars(t *testing.T) {
	if ContainsTemplateVars("npm run test") {
		t.Error("should return false for no vars")
	}
	if !ContainsTemplateVars("echo {{ticket_id}}") {
		t.Error("should return true for vars")
	}
}

func TestIsKnownVariable(t *testing.T) {
	known := []string{"ticket_id", "ticket_slug", "ticket_title", "commit_message"}
	for _, v := range known {
		if !IsKnownVariable(v) {
			t.Errorf("%s should be known", v)
		}
	}
	if IsKnownVariable("unknown") {
		t.Error("unknown should not be known")
	}
}

func TestIsAvailableInHookType(t *testing.T) {
	// ticket_id available in all
	if !IsAvailableInHookType("ticket_id", HookOnPickup) {
		t.Error("ticket_id should be available in on_pickup")
	}
	if !IsAvailableInHookType("ticket_id", HookOnSubmit) {
		t.Error("ticket_id should be available in on_submit")
	}
	if !IsAvailableInHookType("ticket_id", HookOnApprove) {
		t.Error("ticket_id should be available in on_approve")
	}

	// commit_message only in on_approve
	if IsAvailableInHookType("commit_message", HookOnPickup) {
		t.Error("commit_message should not be available in on_pickup")
	}
	if IsAvailableInHookType("commit_message", HookOnSubmit) {
		t.Error("commit_message should not be available in on_submit")
	}
	if !IsAvailableInHookType("commit_message", HookOnApprove) {
		t.Error("commit_message should be available in on_approve")
	}

	// unknown not available anywhere
	if IsAvailableInHookType("unknown", HookOnApprove) {
		t.Error("unknown should not be available")
	}
}

func TestEscapeForShell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"hello world", "'hello world'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}

	for _, tt := range tests {
		result := EscapeForShell(tt.input)
		if result != tt.expected {
			t.Errorf("EscapeForShell(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewTemplateVars(t *testing.T) {
	vars := NewTemplateVars("id-123", "my-slug", "My Title")
	if vars.TicketID != "id-123" {
		t.Errorf("expected TicketID 'id-123', got %q", vars.TicketID)
	}
	if vars.TicketSlug != "my-slug" {
		t.Errorf("expected TicketSlug 'my-slug', got %q", vars.TicketSlug)
	}
	if vars.TicketTitle != "My Title" {
		t.Errorf("expected TicketTitle 'My Title', got %q", vars.TicketTitle)
	}
	if vars.CommitMessage != "" {
		t.Errorf("expected empty CommitMessage, got %q", vars.CommitMessage)
	}
}

func TestTemplateVars_WithCommitMessage(t *testing.T) {
	vars := NewTemplateVars("id", "slug", "title")
	vars2 := vars.WithCommitMessage("my message")

	// Original should be unchanged
	if vars.CommitMessage != "" {
		t.Error("original vars should be unchanged")
	}

	// New vars should have message
	if vars2.CommitMessage != "my message" {
		t.Errorf("expected 'my message', got %q", vars2.CommitMessage)
	}
}
