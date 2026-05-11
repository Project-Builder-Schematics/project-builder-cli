package errors

import "encoding/json"

// OpRegex is the invariant format for the Op field.
// Every Op MUST match: exactly two dot-separated lowercase identifiers.
// e.g. "init.handler", "engine.execute", "skill_update.handler"
//
// This constant is used in test assertions and documented as a fitness
// function invariant (structured-error.REQ-01.3). Runtime enforcement
// is NOT performed here — it is enforced by fitness function FF at S-005.
const OpRegex = `^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`

// Error is the canonical error type for project-builder-cli.
//
// # Leak-safe defaults
//
// Error() returns exactly "<ErrCode>: <Message>" — Path, Cause, and
// Details are NEVER included in the default string representation.
//
// MarshalJSON emits {Code, Op, Message, Suggestions} ONLY. Cause, Details,
// and Path require verbose-mode opt-in (deferred — followup).
//
// # Op format invariant
//
// Op must match `^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$` exactly two
// dot-separated lowercase identifiers. Example: "init.handler".
// Fitness function REQ-01.3 enforces this at CI time.
//
// # errors.Is / errors.As compatibility
//
// errors.Is(err, &Error{Code: ErrCodeX}) returns true if err (or any
// error in its chain) is an *Error with Code == ErrCodeX.
type Error struct {
	// Code is the stable machine-readable identifier. Use ErrCode* constants.
	Code ErrCode

	// Op identifies the operation that failed, in "<package>.<function>" format.
	// e.g. "init.handler", "engine.execute".
	Op string

	// Path is an optional filesystem path relevant to the error.
	// OMITTED from Error() and MarshalJSON (default) — never echoed to users.
	Path string

	// Message is a user-safe description. MUST NOT contain raw paths,
	// argv values, environment variable values, or other sensitive data.
	Message string

	// Details contains extended diagnostic information.
	// OMITTED from MarshalJSON (default) — opt-in verbose mode deferred.
	Details []string

	// Suggestions contains optional remediation hints shown to the user.
	// Included in MarshalJSON output.
	Suggestions []string

	// Cause is the underlying error that triggered this Error.
	// OMITTED from Error() and MarshalJSON (default).
	// Accessible via errors.Unwrap / errors.As traversal.
	Cause error
}

// Error implements the error interface.
// Returns EXACTLY "<ErrCode>: <Message>" — leak-safe by design.
// Path, Cause, and Details are intentionally omitted.
func (e *Error) Error() string {
	return string(e.Code) + ": " + e.Message
}

// Unwrap returns the Cause error, enabling errors.Is / errors.As traversal
// through wrapped error chains.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is reports whether target matches this error by Code.
// This enables errors.Is(err, &Error{Code: ErrCodeX}) to work through
// fmt.Errorf("%w") wrapping without requiring exact pointer equality.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}

	return e.Code == t.Code
}

// SafeMessage returns the same safe string as Error(): "<ErrCode>: <Message>".
// It is an alias for Error() — provided for callers who need an explicit
// "safe to display" marker in the call site. Path, Cause, and Details are
// intentionally omitted.
func (e *Error) SafeMessage() string {
	return e.Error()
}

// jsonError is the shape emitted by MarshalJSON.
// Only the safe field set: Code, Op, Message, Suggestions.
type jsonError struct {
	Code        ErrCode  `json:"code"`
	Op          string   `json:"op"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// MarshalJSON emits ONLY the safe field set: Code, Op, Message, Suggestions.
// Cause, Details, and Path are explicitly omitted to prevent information leakage
// in API responses or AI agent output. This is the default rendering contract.
//
// security.REQ-04.1, .04.2: Cause, Details, Path MUST NOT appear in default JSON.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonError{
		Code:        e.Code,
		Op:          e.Op,
		Message:     e.Message,
		Suggestions: e.Suggestions,
	})
}
