// Package newfeature — extends.go implements the --extends flag grammar parser
// and TUI prompt fallback.
//
// REQ coverage:
//   - REQ-EX-01: valid grammar accepted: @scope/pkg:base
//   - REQ-EX-02: path traversal rejected → ErrCodeInvalidExtends
//   - REQ-EX-03: malformed grammar rejected (missing @, :, whitespace)
//   - REQ-EX-04: TUI prompt when interactive + flag absent (stub — deferred)
//   - REQ-EX-05: non-interactive without flag → skip extends silently
//   - ADV-04: path traversal in --extends rejected by grammar regex
//
// S-005 stub: ValidateExtendsGrammar returns nil (always valid) until the
// RED→GREEN cycle for REQ-EX-01..03 lands.
package newfeature

import (
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// ValidateExtendsGrammar checks that value matches the required grammar:
//
//	^@[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$
//
// Returns ErrCodeInvalidExtends if the grammar check fails.
// Returns nil if value is empty (caller skips validation for absent flag).
//
// S-005 stub: returns nil for all inputs (real implementation is TODO).
func ValidateExtendsGrammar(value string) error {
	_ = value
	_ = errs.ErrCodeInvalidExtends // ensure import is used
	return nil
}

// IsInteractiveTTY reports whether stdin is an interactive terminal.
// Used to gate the TUI prompt for --extends (REQ-EX-04/05).
// Returns false in non-TTY environments (CI, piped stdin, tests).
//
// S-005 stub: returns false (real os.Stdin TTY check is TODO).
func IsInteractiveTTY() bool {
	return false
}
