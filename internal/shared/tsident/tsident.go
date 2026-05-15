// Package tsident provides TypeScript identifier escaping for the builder-new
// codegen pipeline.
//
// ADR-025: hand-rolled strings.Builder in tsgen.go calls EscapeIdent for EVERY
// property name. No text/template — silent-injection risk is unacceptable for
// user-controlled names crossing into typed output.
//
// REQ-TI-01..10: full transformation contract enforced by S-003.
// ReservedWords (69 entries) defined in reserved.go.
package tsident

import (
	"slices"
	"strings"
	"unicode"
)

// EscapeIdent transforms an arbitrary string into a valid TypeScript identifier.
//
// Transformations applied in order:
//  1. Panic on empty input (programming error — caller must validate non-empty).
//  2. Replace runs of non-identifier characters (any char that is not an ASCII
//     letter, digit, or underscore, AND is not a multi-byte non-ASCII rune)
//     with a single underscore. Handles hyphen, space, dot, and all other
//     ASCII punctuation. Consecutive replaceable chars collapse to one _.
//  3. Replace non-ASCII (multi-byte) runes with a single underscore each.
//  4. If the resulting first character is a digit, prefix with _.
//  5. If the exact result equals a ReservedWords entry, append _.
//
// Steps 2+3 are combined in one rune-by-rune pass for efficiency.
// Consecutive replaceable chars (including non-ASCII) are collapsed to single _.
//
// Pure function; deterministic; no side effects; no I/O.
//
// REQ-TI-07: panics on empty input — caller MUST validate non-empty before
// calling. (Empty names are rejected by validate.RejectMetachars upstream.)
func EscapeIdent(s string) string {
	if s == "" {
		panic("tsident: cannot escape empty string")
	}

	// Pass 1: replace non-identifier runes. Consecutive replaceable runes
	// are collapsed to a single underscore.
	//
	// A "safe" rune is: ASCII letter (a-z, A-Z), ASCII digit (0-9), or _.
	// Everything else (hyphen, space, dot, non-ASCII, etc.) → _.
	var b strings.Builder
	b.Grow(len(s) + 2) // small headroom for prefix and suffix
	prevWasReplaced := false

	for _, r := range s {
		if isSafeRune(r) {
			b.WriteRune(r)
			prevWasReplaced = false
		} else {
			if !prevWasReplaced {
				b.WriteByte('_')
			}
			prevWasReplaced = true
		}
	}

	result := b.String()

	// Pass 2: if first character is a digit, prefix with _.
	if len(result) > 0 && unicode.IsDigit(rune(result[0])) {
		result = "_" + result
	}

	// Pass 3: if result exactly matches a reserved word, append _.
	if slices.Contains(ReservedWords, result) {
		result += "_"
	}

	return result
}

// IsReserved reports whether s exactly matches any entry in ReservedWords.
// Case-sensitive match. Use for diagnostic / introspection only;
// EscapeIdent uses this internally via slices.Contains.
func IsReserved(s string) bool {
	return slices.Contains(ReservedWords, s)
}

// isSafeRune reports whether r is a valid TypeScript identifier character
// that does NOT need escaping: ASCII letter, ASCII digit, or underscore.
// Non-ASCII runes return false (conservative ASCII-only policy per REQ-TI-08).
func isSafeRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}
