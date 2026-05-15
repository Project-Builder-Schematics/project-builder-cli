// Package tsident provides TypeScript identifier escaping for the builder-new
// codegen pipeline.
//
// S-000a ships a stub implementation: EscapeIdent returns its input unchanged
// and ReservedWords is empty. The real implementation (ADR-025) lands in S-003
// with all 69 reserved words and transformation rules (REQ-TI-01..10).
//
// ADR-025: hand-rolled strings.Builder in tsgen.go calls EscapeIdent for EVERY
// property name. No text/template — silent-injection risk is unacceptable for
// user-controlled names crossing into typed output.
package tsident

// EscapeIdent transforms an arbitrary string into a valid TypeScript identifier.
//
// S-000a stub: returns s unchanged. The real implementation lands in S-003.
//
// Contract (enforced by S-003):
//   - Replaces runs of [-. ] with single _
//   - Replaces non-ASCII bytes with _
//   - Prefixes with _ if first char is a digit
//   - Appends _ if result matches a ReservedWords entry
//   - Panics on empty input (programming error; caller must validate non-empty)
//
// Pure function; deterministic; no side effects; no I/O.
func EscapeIdent(s string) string {
	return s
}

// ReservedWords is the canonical TypeScript reserved-word list.
// S-000a stub: empty. The full 69-word list lands in S-003 (reserved.go).
// EXPORTED for FF-14 fitness function to enumerate against the test matrix.
var ReservedWords = []string{}
