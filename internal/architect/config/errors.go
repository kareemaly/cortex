package config

import "fmt"

// ArchitectNotFoundError indicates no cortex.yaml was found.
type ArchitectNotFoundError struct {
	StartPath string
}

func (e *ArchitectNotFoundError) Error() string {
	return fmt.Sprintf("architect not found: no cortex.yaml found starting from %s", e.StartPath)
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

// IsArchitectNotFound returns true if err is an ArchitectNotFoundError.
func IsArchitectNotFound(err error) bool {
	_, ok := err.(*ArchitectNotFoundError)
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
