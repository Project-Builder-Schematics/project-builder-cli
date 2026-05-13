// Package validate provides input-validation helpers shared across features.
//
// The shell-metachar guard (RejectMetachars) was originally scoped to the
// angular sub-package. It is promoted here so any feature that spawns a
// subprocess can reuse the same defence-in-depth logic without duplication.
// (S-005 prep — see pkg ADR-023 for the rationale.)
package validate

import (
	"strings"

	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// ShellMetachars is the set of characters forbidden in any value that
// reaches exec.Cmd.Args.
//
// These are shell metacharacters that, if passed to a shell, would execute
// arbitrary code. Features use exec.CommandContext (no shell), but we reject
// them anyway as defence-in-depth: they have no legitimate purpose in a
// package-manager binary path or schematic name.
//
// Forbidden: $ ` ( ) { } | ; & > < \ " ' \n \r NUL
const ShellMetachars = "$`(){}|;&><\\\"'\n\r\x00"

// RejectMetachars returns a validation error if s contains any character from
// ShellMetachars. fieldName is included in the human-readable error message.
//
// Returns *errors.Error{Code: ErrCodeInvalidInput, Op: op} on violation.
// Returns nil when s is clean.
func RejectMetachars(op, fieldName, s string) error {
	if strings.ContainsAny(s, ShellMetachars) {
		return &apperrors.Error{
			Code:    apperrors.ErrCodeInvalidInput,
			Op:      op,
			Message: fieldName + " contains a forbidden character (shell metacharacter or NUL byte)",
		}
	}
	return nil
}
