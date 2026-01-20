package lifecycle

import "fmt"

// ExecutionError indicates a command could not be executed (not found, permission denied).
type ExecutionError struct {
	Command string
	Err     error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("failed to execute command %q: %v", e.Command, e.Err)
}

func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// TemplateError indicates template expansion failed.
type TemplateError struct {
	Template string
	Reason   string
}

func (e *TemplateError) Error() string {
	return fmt.Sprintf("template expansion failed for %q: %s", e.Template, e.Reason)
}

// InvalidVariableError indicates a variable was used in the wrong hook type.
type InvalidVariableError struct {
	Variable string
	HookType HookType
}

func (e *InvalidVariableError) Error() string {
	return fmt.Sprintf("variable %q is not available in %s hooks", e.Variable, e.HookType)
}

// IsExecutionError returns true if err is an ExecutionError.
func IsExecutionError(err error) bool {
	_, ok := err.(*ExecutionError)
	return ok
}

// IsTemplateError returns true if err is a TemplateError.
func IsTemplateError(err error) bool {
	_, ok := err.(*TemplateError)
	return ok
}

// IsInvalidVariable returns true if err is an InvalidVariableError.
func IsInvalidVariable(err error) bool {
	_, ok := err.(*InvalidVariableError)
	return ok
}
