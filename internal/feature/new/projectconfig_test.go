// Package newfeature — projectconfig_test.go covers the in-line project-builder.json
// read/write/mutate helpers (ADR-027; F-01 marker present per FF-17).
//
// REQ coverage:
//   - REQ-PJ-01: atomic write (write-temp + rename via FSWriter)
//   - REQ-PJ-02: idempotent re-running produces byte-identical output
//   - REQ-PJ-03: version field preserved verbatim (R-RES-1 — init writes "1" as string)
//   - REQ-PJ-04: unknown top-level fields preserved after mutation
//   - REQ-PJ-05: path mode writes collections.default.<name>.path
//   - REQ-PJ-07: json.NewEncoder + SetEscapeHTML(false) + two-space indent
//   - REQ-PJ-08: ErrCorruptConfig on unparseable project-builder.json
package newfeature_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// minimalPBJSON is the project-builder.json written by `builder init`.
// Note: version is a string "1" — R-RES-1 requires this is preserved verbatim.
const minimalPBJSON = `{
  "$schema": "./node_modules/@pbuilder/sdk/schemas/project-builder.schema.json",
  "version": "1",
  "collections": {},
  "dependencies": {},
  "settings": {
    "autoInstall": true,
    "conflictPolicy": "child-wins",
    "depValidation": "dev"
  },
  "skill": {
    "enabled": true,
    "path": ".claude/skills/pbuilder/SKILL.md"
  }
}
`

// withPBJSON writes content to <dir>/project-builder.json via the given fakeFS.
func withPBJSON(t *testing.T, dir, content string, fs fswriter.FSWriter) {
	t.Helper()
	path := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: WriteFile project-builder.json: %v", err)
	}
}

// Test_ReadConfig_ParsesVersionAsRawMessage verifies that the version field
// is read as json.RawMessage, preserving the string "1" verbatim (R-RES-1 / REQ-PJ-03).
func Test_ReadConfig_ParsesVersionAsRawMessage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: unexpected error: %v", err)
	}

	// Version must be the raw JSON token "1" (string-quoted in JSON = `"1"`).
	// We check it did not coerce to an integer (which would be `1` without quotes).
	if string(cfg.Version) != `"1"` {
		t.Errorf("ReadConfig: version = %q; want %q (string verbatim per R-RES-1)", string(cfg.Version), `"1"`)
	}
}

// Test_WriteConfig_VersionPreservedVerbatim verifies that a read-mutate-write cycle
// keeps the version value byte-identical (R-RES-1 / REQ-PJ-03).
func Test_WriteConfig_VersionPreservedVerbatim(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// No mutation — just write back.
	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	// Read back the written bytes.
	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	// The version value in the written JSON must be `"1"` (string), not `1` (int).
	if !strings.Contains(string(written), `"version": "1"`) {
		t.Errorf("WriteConfig: version coerced from string to int (R-RES-1 violation);\nwrote: %s", written)
	}
}

// Test_WriteConfig_UnknownFieldsPreserved verifies that top-level fields not known
// to projectconfig are preserved after read-mutate-write (REQ-PJ-04).
func Test_WriteConfig_UnknownFieldsPreserved(t *testing.T) {
	t.Parallel()

	const withExtra = `{
  "version": "1",
  "collections": {},
  "customField": "preserve-me",
  "anotherField": 42
}
`
	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, withExtra, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	if !strings.Contains(string(written), `"customField"`) {
		t.Errorf("WriteConfig: unknown field 'customField' lost (REQ-PJ-04 violation);\nwrote: %s", written)
	}
	if !strings.Contains(string(written), `"anotherField"`) {
		t.Errorf("WriteConfig: unknown field 'anotherField' lost (REQ-PJ-04 violation);\nwrote: %s", written)
	}
}

// Test_WriteConfig_TwoSpaceIndent verifies the output uses two-space indentation
// (REQ-PJ-07 / L-builder-init-03).
func Test_WriteConfig_TwoSpaceIndent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	// Two-space indent: first nested key should start with "  " (two spaces).
	if !strings.Contains(string(written), "\n  \"") {
		t.Errorf("WriteConfig: output not two-space indented (REQ-PJ-07);\nwrote: %s", written)
	}
}

// Test_WriteConfig_TrailingNewline verifies the output ends with a newline
// (consistent with init's locked-bytes contract).
func Test_WriteConfig_TrailingNewline(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	if !strings.HasSuffix(string(written), "\n") {
		t.Errorf("WriteConfig: output missing trailing newline;\nwrote: %q", written)
	}
}

// Test_RegisterSchematicPath_WritesCollectionEntry verifies that after calling
// RegisterSchematicPath, the config has the expected entry (REQ-PJ-05).
func Test_RegisterSchematicPath_WritesCollectionEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if err := newfeature.RegisterSchematicPath(cfg, "default", "my-schematic", "./schematics/my-schematic"); err != nil {
		t.Fatalf("RegisterSchematicPath: unexpected error: %v", err)
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	// Parse the output to verify the mutation is present.
	var out map[string]json.RawMessage
	if err := json.Unmarshal(written, &out); err != nil {
		t.Fatalf("parse written JSON: %v", err)
	}

	var collections map[string]json.RawMessage
	if err := json.Unmarshal(out["collections"], &collections); err != nil {
		t.Fatalf("parse collections: %v", err)
	}

	defaultColl, ok := collections["default"]
	if !ok {
		t.Fatalf("collections.default missing from written JSON")
	}

	var defaultMap map[string]json.RawMessage
	if err := json.Unmarshal(defaultColl, &defaultMap); err != nil {
		t.Fatalf("parse default collection: %v", err)
	}

	schEntry, ok := defaultMap["my-schematic"]
	if !ok {
		t.Fatalf("collections.default.my-schematic missing from written JSON")
	}

	var entry map[string]string
	if err := json.Unmarshal(schEntry, &entry); err != nil {
		t.Fatalf("parse schematic entry: %v", err)
	}

	if got := entry["path"]; got != "./schematics/my-schematic" {
		t.Errorf("RegisterSchematicPath: path = %q; want %q", got, "./schematics/my-schematic")
	}
}

// Test_RegisterSchematicPath_Idempotent verifies that registering the same schematic
// twice (with the same path) produces the same result (REQ-PJ-02).
func Test_RegisterSchematicPath_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// Register twice.
	for range 2 {
		if err := newfeature.RegisterSchematicPath(cfg, "default", "my-schematic", "./schematics/my-schematic"); err != nil {
			t.Fatalf("RegisterSchematicPath: %v", err)
		}
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	// Count occurrences of "my-schematic" in the output — must be exactly 1 key.
	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	count := strings.Count(string(written), `"my-schematic"`)
	if count != 1 {
		t.Errorf("RegisterSchematicPath idempotent: found %d occurrences of my-schematic in JSON; want 1;\nwrote: %s", count, written)
	}
}

// Test_ReadConfig_ErrCorruptConfig verifies that unparseable JSON returns an error
// (REQ-PJ-08).
func Test_ReadConfig_ErrCorruptConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, `{ not valid json`, fs)

	_, err := newfeature.ReadConfig(dir, fs)
	if err == nil {
		t.Fatal("ReadConfig: expected error for corrupt JSON, got nil")
	}
}

// Test_RegisterSchematicInline_WritesCollectionEntry verifies that after calling
// RegisterSchematicInline, the config has the expected nested entry (REQ-PJ-06).
//
// Expected JSON shape:
//
//	"collections": { "default": { "schematics": { "<name>": { "inputs": {} } } } }
func Test_RegisterSchematicInline_WritesCollectionEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if err := newfeature.RegisterSchematicInline(cfg, "default", "my-inline"); err != nil {
		t.Fatalf("RegisterSchematicInline: unexpected error: %v", err)
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	// Parse the output to verify the mutation is present.
	var out map[string]json.RawMessage
	if err := json.Unmarshal(written, &out); err != nil {
		t.Fatalf("parse written JSON: %v", err)
	}

	var collections map[string]json.RawMessage
	if err := json.Unmarshal(out["collections"], &collections); err != nil {
		t.Fatalf("parse collections: %v", err)
	}

	defaultColl, ok := collections["default"]
	if !ok {
		t.Fatalf("collections.default missing from written JSON")
	}

	// The inline shape nests under "schematics" key inside the collection.
	var defaultMap map[string]json.RawMessage
	if err := json.Unmarshal(defaultColl, &defaultMap); err != nil {
		t.Fatalf("parse default collection: %v", err)
	}

	schematicsRaw, ok := defaultMap["schematics"]
	if !ok {
		t.Fatalf("collections.default.schematics missing from written JSON (inline mode nests under 'schematics' key)")
	}

	var schematicsMap map[string]json.RawMessage
	if err := json.Unmarshal(schematicsRaw, &schematicsMap); err != nil {
		t.Fatalf("parse schematics map: %v", err)
	}

	schEntry, ok := schematicsMap["my-inline"]
	if !ok {
		t.Fatalf("collections.default.schematics.my-inline missing from written JSON")
	}

	// The inline entry must be {"inputs": {}}.
	var entry map[string]json.RawMessage
	if err := json.Unmarshal(schEntry, &entry); err != nil {
		t.Fatalf("parse inline entry: %v", err)
	}

	if _, hasInputs := entry["inputs"]; !hasInputs {
		t.Errorf("inline entry missing 'inputs' key; got: %s", schEntry)
	}
}

// Test_RegisterSchematicInline_Idempotent verifies that registering the same inline
// schematic twice produces the same result (REQ-PJ-02).
func Test_RegisterSchematicInline_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// Register twice.
	for range 2 {
		if err := newfeature.RegisterSchematicInline(cfg, "default", "my-inline"); err != nil {
			t.Fatalf("RegisterSchematicInline: %v", err)
		}
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Count occurrences of "my-inline" in the output — must be exactly 1 key.
	count := strings.Count(string(written), `"my-inline"`)
	if count != 1 {
		t.Errorf("RegisterSchematicInline idempotent: found %d occurrences of my-inline in JSON; want 1;\nwrote: %s", count, written)
	}
}

// Test_SchematicExists_InlineMode verifies SchematicExists returns true when the
// schematic is registered in inline mode (nested under "schematics" key).
func Test_SchematicExists_InlineMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// Before registration: must not exist.
	if newfeature.SchematicExists(cfg, "default", "my-inline") {
		t.Error("SchematicExists: returned true before inline registration")
	}

	if err := newfeature.RegisterSchematicInline(cfg, "default", "my-inline"); err != nil {
		t.Fatalf("RegisterSchematicInline: %v", err)
	}

	// After registration: must exist (checks both path and inline modes).
	if !newfeature.SchematicExists(cfg, "default", "my-inline") {
		t.Error("SchematicExists: returned false after inline registration")
	}
}

// Test_SchematicExistsInPathMode verifies SchematicExistsInPathMode returns true
// only for path-mode entries, not inline entries.
func Test_SchematicExistsInPathMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// Path-mode entry: should be detected as path mode.
	if err := newfeature.RegisterSchematicPath(cfg, "default", "path-sch", "./schematics/path-sch"); err != nil {
		t.Fatalf("RegisterSchematicPath: %v", err)
	}
	if !newfeature.SchematicExistsInPathMode(cfg, "default", "path-sch") {
		t.Error("SchematicExistsInPathMode: returned false for path-mode entry")
	}

	// Inline entry: should NOT be detected as path mode.
	if err := newfeature.RegisterSchematicInline(cfg, "default", "inline-sch"); err != nil {
		t.Fatalf("RegisterSchematicInline: %v", err)
	}
	if newfeature.SchematicExistsInPathMode(cfg, "default", "inline-sch") {
		t.Error("SchematicExistsInPathMode: returned true for inline-mode entry (wrong)")
	}
}

// Test_SchematicExists_PathMode verifies SchematicExists returns true when the
// schematic is registered in path mode in the given collection.
func Test_SchematicExists_PathMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// Before registration: must not exist.
	if newfeature.SchematicExists(cfg, "default", "my-schematic") {
		t.Error("SchematicExists: returned true before registration")
	}

	if err := newfeature.RegisterSchematicPath(cfg, "default", "my-schematic", "./schematics/my-schematic"); err != nil {
		t.Fatalf("RegisterSchematicPath: %v", err)
	}

	// After registration: must exist.
	if !newfeature.SchematicExists(cfg, "default", "my-schematic") {
		t.Error("SchematicExists: returned false after path registration")
	}
}
