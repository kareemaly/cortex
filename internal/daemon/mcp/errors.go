package mcp

import (
	"fmt"

	"github.com/kareemaly/cortex1/internal/ticket"
)

// ErrorCode represents MCP tool error codes.
type ErrorCode string

const (
	ErrorCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrorCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeInternal     ErrorCode = "INTERNAL_ERROR"
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

// WrapTicketError converts ticket store errors to MCP tool errors.
func WrapTicketError(err error) *ToolError {
	if err == nil {
		return nil
	}

	// Check for specific ticket error types
	switch e := err.(type) {
	case *ticket.NotFoundError:
		return NewNotFoundError(e.Resource, e.ID)
	case *ticket.ValidationError:
		return NewValidationError(e.Field, e.Message)
	default:
		return NewInternalError(err.Error())
	}
}

// IsToolError checks if err is a ToolError.
func IsToolError(err error) bool {
	_, ok := err.(*ToolError)
	return ok
}
