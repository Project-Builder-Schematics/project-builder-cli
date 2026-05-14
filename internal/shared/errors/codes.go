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

	// --- init feature error codes (REQ-EC-01) ---

	// ErrCodeInitDirNotEmpty is returned when the target directory is not empty
	// and --force was not supplied.
	ErrCodeInitDirNotEmpty ErrCode = "init_dir_not_empty"

	// ErrCodeInitConfigExists is returned when project-builder.json already
	// exists in the target directory and --force was not supplied.
	ErrCodeInitConfigExists ErrCode = "init_config_exists"

	// ErrCodeInitAgentFileAmbiguous is returned when both AGENTS.md and
	// CLAUDE.md exist in the target directory (selection precedence violated).
	ErrCodeInitAgentFileAmbiguous ErrCode = "init_agent_file_ambiguous"

	// ErrCodeInitPackageManagerNotFound is returned when no supported package
	// manager can be detected or resolved for the target directory.
	ErrCodeInitPackageManagerNotFound ErrCode = "init_package_manager_not_found"

	// ErrCodeInitSkillExists is returned when the SKILL.md artefact already
	// exists and --force was not supplied.
	ErrCodeInitSkillExists ErrCode = "init_skill_exists"

	// ErrCodeInitNotImplemented is returned for init sub-features that are
	// planned but not yet available (e.g. --publishable mode).
	// Distinct from ErrCodeNotImplemented which is the generic stub sentinel.
	ErrCodeInitNotImplemented ErrCode = "init_not_implemented"
)
