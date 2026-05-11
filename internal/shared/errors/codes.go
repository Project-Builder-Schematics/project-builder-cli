// Package errors provides the canonical structured error type and stable
// error code constants for project-builder-cli.
//
// ErrCode is a named type (not a bare string) so that constants are
// the only valid construction — a mis-typed code string fails to compile.
package errors

// ErrCode is the typed identifier for an Error.
//
// Always compare via errors.Is against a sentinel *Error{Code: ...};
// never compare the string value directly.
type ErrCode string

const (
	// ErrCodeNotImplemented is returned by stub handlers before the feature
	// is implemented. Callers should not treat this as a permanent error.
	ErrCodeNotImplemented ErrCode = "not_implemented"

	// ErrCodeCancelled is returned when the operation was cancelled via
	// context cancellation before it could complete.
	ErrCodeCancelled ErrCode = "cancelled"

	// ErrCodeInvalidInput is returned when the provided input fails
	// schema validation or constraint checks.
	ErrCodeInvalidInput ErrCode = "invalid_input"

	// ErrCodeEngineNotFound is returned when no Engine implementation
	// is registered for the requested schematic collection.
	ErrCodeEngineNotFound ErrCode = "engine_not_found"

	// ErrCodeExecutionFailed is returned when the Engine's execution of
	// a schematic terminates with an error.
	ErrCodeExecutionFailed ErrCode = "execution_failed"
)
