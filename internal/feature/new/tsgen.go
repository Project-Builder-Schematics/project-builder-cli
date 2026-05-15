// Package newfeature — tsgen.go generates TypeScript .d.ts declaration files
// from schema.json v1 inputs.
//
// ADR-025: hand-rolled strings.Builder (NO text/template — silent-injection risk
// is unacceptable for user-controlled names crossing into typed output).
//
// REQ coverage:
//   - REQ-TG-01..09: full .d.ts codegen contract
//   - REQ-TI-10: PascalCase applied HERE at the interface name level, NOT inside EscapeIdent
//   - REQ-TG-04: property names via EscapeIdent; renamed fields get // original comment
//
// S-003 stub: GenerateDTS returns empty placeholder. Real impl below is activated by tests.
package newfeature

// GenerateDTS generates a TypeScript .d.ts declaration file from a schema.
// The interface name is derived from schematicName via PascalCase + "SchematicInputs".
// Property names are escaped via tsident.EscapeIdent (REQ-TI-10 boundary).
//
// Returns a byte slice suitable for writing to schema.d.ts.
// Pure function; deterministic; no I/O.
//
// ADR-025: uses strings.Builder exclusively — NO text/template.
func GenerateDTS(_ string, _ Schema) ([]byte, error) {
	panic("tsgen.GenerateDTS: not yet implemented (S-003 stub)")
}
