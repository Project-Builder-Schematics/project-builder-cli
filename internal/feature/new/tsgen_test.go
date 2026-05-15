// Package newfeature — tsgen_test.go covers .d.ts code generation via golden
// snapshot assertions (REQ-TG-01..09, ADV-01).
//
// Each test case calls tsgen.Generate(name, schema) and compares byte-for-byte
// against a committed golden fixture under testdata/golden/tsgen/*.d.ts.
//
// Golden fixtures are hand-authored — deterministic, version-embedded, no timestamp.
// Changes to tsgen.go MUST trigger explicit golden updates (visible in PR diff).
//
// REQ coverage:
//   - REQ-TG-01: file location (caller's responsibility; test verifies byte content)
//   - REQ-TG-02: interface name is PascalCase(<name>)SchematicInputs
//   - REQ-TG-03: type mapping (string/number/boolean/enum/list)
//   - REQ-TG-04: property names via EscapeIdent; renamed fields get // original comment
//   - REQ-TG-05: header comment without timestamp (deterministic)
//   - REQ-TG-06: all fields optional (? suffix) — no required concept in schema v1
//   - REQ-TG-07: enum → union literal "a" | "b" | ...
//   - REQ-TG-08: list → Array<T>
//   - REQ-TG-09: golden byte-identical snapshots × 9 combinations
//   - ADV-01: reserved word as input name → escaped + comment (class → class_)
package newfeature_test

import (
	"os"
	"path/filepath"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
)

// goldenDir is the directory containing the committed .d.ts golden snapshots.
const tsgenGoldenDir = "testdata/golden/tsgen"

// readGolden reads a committed golden fixture for tsgen tests.
// Fails the test if the file cannot be read.
// path is assembled from a fixed directory constant (not user input — G304 safe).
func readGolden(t *testing.T, filename string) []byte {
	t.Helper()
	// path is constant-dir + test-controlled filename with no user input.
	path := filepath.Join(tsgenGoldenDir, filename) //nolint:gosec // path is test-internal constant + fixed filename
	b, err := os.ReadFile(path)                     //nolint:gosec // same
	if err != nil {
		t.Fatalf("readGolden(%q): %v", path, err)
	}
	return b
}

// assertGolden compares got bytes against the committed golden snapshot.
// Fails with a clear diff on mismatch.
func assertGolden(t *testing.T, filename string, got []byte) {
	t.Helper()
	want := readGolden(t, filename)
	if string(got) != string(want) {
		t.Errorf("tsgen output does not match golden %q:\n--- want ---\n%s\n--- got ---\n%s",
			filename, want, got)
	}
}

// ─── golden snapshot tests ────────────────────────────────────────────────────

// Test_TsGen_Empty verifies empty inputs → minimal interface (REQ-TG-01..02/05/09).
func Test_TsGen_Empty(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "empty.d.ts", got)
}

// Test_TsGen_StringField verifies string type mapping (REQ-TG-03/06/09).
func Test_TsGen_StringField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"name": {Type: "string"},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "string_field.d.ts", got)
}

// Test_TsGen_NumberField verifies number type mapping (REQ-TG-03/09).
func Test_TsGen_NumberField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"count": {Type: "number"},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "number_field.d.ts", got)
}

// Test_TsGen_BooleanField verifies boolean type mapping (REQ-TG-03/09).
func Test_TsGen_BooleanField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"enabled": {Type: "boolean"},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "boolean_field.d.ts", got)
}

// Test_TsGen_EnumField verifies enum → union literal type (REQ-TG-07/09).
func Test_TsGen_EnumField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"style": {Type: "enum", Enum: []string{"css", "scss", "sass"}},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "enum_field.d.ts", got)
}

// Test_TsGen_ListField verifies list → Array<T> (REQ-TG-08/09).
func Test_TsGen_ListField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"tags": {Type: "list", Items: &newfeature.ItemsSpec{Type: "string"}},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "list_field.d.ts", got)
}

// Test_TsGen_ReservedField verifies reserved word property name gets escaped
// with // original comment (REQ-TG-04/09, ADV-01).
// Schematic name "class" → interface ClassSchematicInputs (PascalCase, not reserved as name).
func Test_TsGen_ReservedField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"class": {Type: "string"},
	}}
	got, err := newfeature.GenerateDTS("class", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "reserved_field.d.ts", got)
}

// Test_TsGen_DigitField verifies digit-leading property name gets _ prefix
// with // original comment (REQ-TG-04/09, REQ-TI-05).
func Test_TsGen_DigitField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"123count": {Type: "number"},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "digit_field.d.ts", got)
}

// Test_TsGen_HyphenField verifies hyphen in property name is escaped to _
// with // original comment (REQ-TG-04/09, REQ-TI-04).
func Test_TsGen_HyphenField(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{
		"my-name": {Type: "string"},
	}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}
	assertGolden(t, "hyphen_field.d.ts", got)
}

// ─── interface name PascalCase tests ─────────────────────────────────────────

// Test_TsGen_PascalCaseName verifies interface name is PascalCase (REQ-TG-02).
func Test_TsGen_PascalCaseName(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{}}
	got, err := newfeature.GenerateDTS("my-schematic", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}

	content := string(got)
	const wantIface = "export interface MySchematicSchematicInputs {}"
	if !contains(content, wantIface) {
		t.Errorf("interface name not PascalCase; want %q in:\n%s", wantIface, content)
	}
}

// Test_TsGen_NoTimestamp verifies the header has no timestamp (REQ-TG-05).
func Test_TsGen_NoTimestamp(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{}}
	got, err := newfeature.GenerateDTS("foo", schema)
	if err != nil {
		t.Fatalf("GenerateDTS: %v", err)
	}

	// Must have the static header.
	content := string(got)
	const wantHeader = "// Auto-generated by builder new — do not edit manually"
	if !contains(content, wantHeader) {
		t.Errorf("header comment missing; want %q in:\n%s", wantHeader, content)
	}

	// Must NOT have a timestamp (any date-like pattern).
	// Simple check: no "2026" or "2025" in the output.
	if contains(content, "2025") || contains(content, "2026") {
		t.Errorf("timestamp detected in output (breaks reproducible builds):\n%s", content)
	}
}

// contains is a simple string containment helper.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsLoop(s, sub))
}

func containsLoop(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
