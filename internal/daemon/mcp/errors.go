package mcp

import "fmt"

// ErrorCode represents MCP tool error codes.
type ErrorCode string

const (
	ErrorCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrorCodeValidation    ErrorCode = "VALIDATION_ERROR"
	ErrorCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	ErrorCodeInternal      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeStateConflict ErrorCode = "STATE_CONFLICT"
)

// ToolError represents an error returned from an MCP tool.
type ToolError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// Error implements the error interface.
func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewNotFoundError creates a NOT_FOUND error.
func NewNotFoundError(resource, id string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeNotFound,
		Message: fmt.Sprintf("%s not found: %s", resource, id),
	}
}

// NewValidationError creates a VALIDATION_ERROR.
func NewValidationError(field, message string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeValidation,
		Message: fmt.Sprintf("validation error for %s: %s", field, message),
	}
}

// NewUnauthorizedError creates an UNAUTHORIZED error.
func NewUnauthorizedError(message string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeUnauthorized,
		Message: message,
	}
}

// NewInternalError creates an INTERNAL_ERROR.
func NewInternalError(message string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeInternal,
		Message: message,
	}
}

// NewStateConflictError creates a STATE_CONFLICT error.
func NewStateConflictError(state, mode, message string) *ToolError {
	return &ToolError{
		Code:    ErrorCodeStateConflict,
		Message: fmt.Sprintf("state=%s mode=%s: %s", state, mode, message),
	}
}

// IsToolError checks if err is a ToolError.
func IsToolError(err error) bool {
	_, ok := err.(*ToolError)
	return ok
}
