package config

import "fmt"

// ProjectNotFoundError indicates no .cortex/ directory was found.
type ProjectNotFoundError struct {
	StartPath string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("project not found: no .cortex/ directory found starting from %s", e.StartPath)
}

// ConfigParseError indicates the config file could not be parsed.
type ConfigParseError struct {
	Path string
	Err  error
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse config %s: %v", e.Path, e.Err)
}

func (e *ConfigParseError) Unwrap() error {
	return e.Err
}

// ValidationError indicates a config validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid config: %s: %s", e.Field, e.Message)
}

// IsProjectNotFound returns true if err is a ProjectNotFoundError.
func IsProjectNotFound(err error) bool {
	_, ok := err.(*ProjectNotFoundError)
	return ok
}

// IsConfigParseError returns true if err is a ConfigParseError.
func IsConfigParseError(err error) bool {
	_, ok := err.(*ConfigParseError)
	return ok
}

// IsValidationError returns true if err is a ValidationError.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// ExtendPathNotFoundError indicates the extend path doesn't exist.
type ExtendPathNotFoundError struct {
	Path       string
	ResolvedTo string
}

func (e *ExtendPathNotFoundError) Error() string {
	if e.Path == e.ResolvedTo {
		return fmt.Sprintf("extend path not found: %s", e.Path)
	}
	return fmt.Sprintf("extend path not found: %s (resolved to %s)", e.Path, e.ResolvedTo)
}

// IsExtendPathNotFound returns true if err is an ExtendPathNotFoundError.
func IsExtendPathNotFound(err error) bool {
	_, ok := err.(*ExtendPathNotFoundError)
	return ok
}

// CircularExtendError indicates a circular reference in extend chain.
type CircularExtendError struct {
	Path string
}

func (e *CircularExtendError) Error() string {
	return fmt.Sprintf("circular extend reference detected at: %s", e.Path)
}

// IsCircularExtend returns true if err is a CircularExtendError.
func IsCircularExtend(err error) bool {
	_, ok := err.(*CircularExtendError)
	return ok
}
