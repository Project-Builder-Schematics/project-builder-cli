// Package newfeature — extends.go implements the --extends flag grammar parser
// and TUI prompt fallback.
//
// REQ coverage:
//   - REQ-EX-01: valid grammar accepted: @scope/pkg:collection
//   - REQ-EX-02: path traversal rejected → ErrCodeInvalidExtends
//   - REQ-EX-03: malformed grammar rejected (missing @, :, whitespace)
//   - REQ-EX-04: TUI prompt when interactive + flag absent (gated by IsInteractiveTTY)
//   - REQ-EX-05: non-interactive without flag → skip extends silently
//   - ADV-04: path traversal in --extends rejected by grammar regex
package newfeature

import (
	"os"
	"regexp"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// extendsGrammar is the canonical regex for --extends values.
// Pattern: ^@[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$
//
// Rationale:
//   - Must start with @ (rules out path traversal like ../ and ./)
//   - Scope segment: one or more alphanumeric / underscore / hyphen chars
//   - Slash separator (NOT backslash — rules out Windows-style paths)
//   - Package segment: same char class as scope
//   - Colon separator
//   - Collection segment: same char class
//   - End of string — no trailing chars, no whitespace anywhere
//
// This regex automatically rejects:
//   - Path traversal: ../evil (no @, and . not allowed), @scope/../evil (. not allowed)
//   - Spaces/whitespace (not in char class)
//   - Backslashes (not in char class — rules out @scope\evil)
//   - Absolute paths (/etc/passwd — no @ prefix)
//   - Dots in segments (dot not in [a-zA-Z0-9_-])
var extendsGrammar = regexp.MustCompile(`^@[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$`)

// extendsFormatHint is the human-readable format hint included in error messages.
const extendsFormatHint = "@scope/pkg:collection (e.g. @my-org/my-pkg:base)"

// ValidateExtendsGrammar checks that value matches the required grammar:
//
//	^@[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$
//
// Returns ErrCodeInvalidExtends (REQ-EC-04) if the grammar check fails.
// Returns nil if value is empty (caller must skip validation for absent flag).
//
// Security: the regex structure guarantees:
//   - No path separators (/ allowed only as the scope/pkg delimiter)
//   - No backslashes (rules out Windows paths)
//   - No dot sequences (rules out .. traversal even inside @-prefixed values)
//   - No whitespace
//
// Deep validation (does the package actually exist?) is deferred to
// `builder validate` (REQ-EX-01 / D-11 — lazy validation policy).
func ValidateExtendsGrammar(value string) error {
	if extendsGrammar.MatchString(value) {
		return nil
	}
	return &errs.Error{
		Code:    errs.ErrCodeInvalidExtends,
		Op:      "new.validateExtends",
		Message: "--extends '" + value + "': invalid format; expected " + extendsFormatHint,
		Suggestions: []string{
			"use the format @scope/pkg:collection, e.g. @my-org/my-pkg:base",
		},
	}
}

// ttyCheckFn is the package-level TTY detection function.
// Tests override this via SetTTYCheckFn (export_test.go) to simulate TTY/non-TTY
// environments without depending on real stdin state (which varies by shell/CI).
var ttyCheckFn func() bool

// promptExtendsFn is the package-level extends prompt function.
// Signature: func(externals []string) (selected string, skipped bool, err error).
//
// The default implementation is a V1 stub that always returns skipped=true with
// a notice WARN (design §9 R-RES-2: full Bubble Tea polish is post-v1).
// Tests override via SetPromptExtendsFn (export_test.go) to inject deterministic
// selections (REQ-EX-04 integration test).
//
// The function is called by the handler when:
//   - IsInteractiveTTY() returns true, AND
//   - the --extends flag is absent
var promptExtendsFn = func(_ []string) (string, bool, error) {
	// V1 stub: always skip the extends prompt.
	// Post-v1: replace with real Bubble Tea list prompt (design §9 R-RES-2).
	return "", true, nil
}

// IsInteractiveTTY reports whether os.Stdin is an interactive terminal (TTY).
// Used to gate the TUI prompt for --extends (REQ-EX-04/05):
//   - Interactive (true): show TUI prompt listing externals from project-builder.json
//   - Non-interactive (false): skip prompt; continue without extends
//
// Returns false in non-TTY environments: CI pipelines, piped stdin, test processes.
// Tests inject via SetTTYCheckFn (export_test.go) for deterministic control.
func IsInteractiveTTY() bool {
	if ttyCheckFn != nil {
		return ttyCheckFn()
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// ModeCharDevice is set on a real TTY. If unset, stdin is a pipe or file.
	return (fi.Mode() & os.ModeCharDevice) != 0
}
