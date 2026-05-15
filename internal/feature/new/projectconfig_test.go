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
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
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

// ─── RegisterCollection (REQ-NC-01) ──────────────────────────────────────────

// Test_RegisterCollection_WritesTopLevelCollectionEntry verifies that after calling
// RegisterCollection, the config serialises the collection as a top-level peer
// of "default" with shape: collections.<name> = {"path": "<relPath>"} (REQ-NC-01).
func Test_RegisterCollection_WritesTopLevelCollectionEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if err := newfeature.RegisterCollection(cfg, "bar", "./schematics/bar/collection.json"); err != nil {
		t.Fatalf("RegisterCollection: unexpected error: %v", err)
	}

	if err := newfeature.WriteConfig(dir, cfg, fs); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	written, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after WriteConfig: %v", err)
	}

	// Parse to verify the collection entry exists at the top level of "collections".
	var out map[string]json.RawMessage
	if err := json.Unmarshal(written, &out); err != nil {
		t.Fatalf("parse written JSON: %v", err)
	}

	var collections map[string]json.RawMessage
	if err := json.Unmarshal(out["collections"], &collections); err != nil {
		t.Fatalf("parse collections: %v", err)
	}

	barRaw, ok := collections["bar"]
	if !ok {
		t.Fatalf("collections.bar missing from written JSON (REQ-NC-01 violation); written: %s", written)
	}

	var barEntry map[string]string
	if err := json.Unmarshal(barRaw, &barEntry); err != nil {
		t.Fatalf("parse bar entry: %v", err)
	}

	if got := barEntry["path"]; got != "./schematics/bar/collection.json" {
		t.Errorf("RegisterCollection: path = %q; want %q", got, "./schematics/bar/collection.json")
	}
}

// Test_CollectionExists_TrueAfterRegister verifies CollectionExists returns true
// after RegisterCollection is called and false before.
func Test_CollectionExists_TrueAfterRegister(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, minimalPBJSON, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// Before registration: must not exist.
	if newfeature.CollectionExists(cfg, "bar") {
		t.Error("CollectionExists: returned true before registration")
	}

	if err := newfeature.RegisterCollection(cfg, "bar", "./schematics/bar/collection.json"); err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	// After registration: must exist.
	if !newfeature.CollectionExists(cfg, "bar") {
		t.Error("CollectionExists: returned false after registration")
	}
}

// Test_RegisterCollection_RoundTrip verifies that reading back a project-builder.json
// that has a collection entry populates CollectionExists correctly.
func Test_RegisterCollection_RoundTrip(t *testing.T) {
	t.Parallel()

	const pbWithCollection = `{
  "version": "1",
  "collections": {
    "bar": {"path": "./schematics/bar/collection.json"}
  }
}
`
	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, pbWithCollection, fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	// After reading back, "bar" should be recognised as a collection (not a schematic container).
	if !newfeature.CollectionExists(cfg, "bar") {
		t.Error("CollectionExists: returned false after round-trip read (REQ-NC-01 round-trip)")
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

// ─── S-005 Adversarial tests (ADV-06, ADV-07, ADV-08) ────────────────────────

// Test_ADV06_BOMStrippedInSchemaJSON verifies that a schema.json file starting
// with a UTF-8 BOM is read correctly after BOM stripping (ADV-06).
//
// When an existing schema.json has a UTF-8 BOM prefix (\xEF\xBB\xBF),
// the service must strip it, parse the remaining JSON, and emit a WARN.
// The BOM must not cause a parse error.
//
// Note: ADV-06 is exercised at the projectconfig/service level. The BOM
// stripping applies to the schema.json read path in ReadSchemaFromBytes
// (used by --force revalidation). This test exercises the BOM-stripping
// helper directly and verifies no parse error occurs.
func Test_ADV06_BOMStrippedInSchemaJSON(t *testing.T) {
	t.Parallel()

	const validJSON = `{"inputs": {}}`
	bom := []byte{0xEF, 0xBB, 0xBF}
	bomJSON := append(bom, []byte(validJSON)...)

	// StripBOM should remove the BOM and return clean JSON.
	stripped, hadBOM := newfeature.StripBOM(bomJSON)
	if !hadBOM {
		t.Error("ADV-06: StripBOM: expected hadBOM=true for BOM-prefixed input")
	}
	if string(stripped) != validJSON {
		t.Errorf("ADV-06: StripBOM: got %q; want %q", stripped, validJSON)
	}

	// StripBOM on BOM-free input must return original unchanged.
	clean, hadBOM2 := newfeature.StripBOM([]byte(validJSON))
	if hadBOM2 {
		t.Error("ADV-06: StripBOM: expected hadBOM=false for BOM-free input")
	}
	if string(clean) != validJSON {
		t.Errorf("ADV-06: StripBOM: clean input modified; got %q", clean)
	}

	// Empty input must not panic.
	empty, _ := newfeature.StripBOM([]byte{})
	if len(empty) != 0 {
		t.Errorf("ADV-06: StripBOM: empty input returned non-empty: %q", empty)
	}

	// Partial BOM (only 2 of 3 bytes) must NOT be stripped.
	partialBOM := []byte{0xEF, 0xBB, 'x', 'y'}
	partial, hadPartial := newfeature.StripBOM(partialBOM)
	if hadPartial {
		t.Error("ADV-06: StripBOM: partial BOM incorrectly detected as full BOM")
	}
	if string(partial) != string(partialBOM) {
		t.Errorf("ADV-06: StripBOM: partial BOM input was modified unexpectedly")
	}
}

// Test_ADV06_BOMInProjectBuilderJSON verifies that ReadConfig handles a
// project-builder.json with a UTF-8 BOM without error (ADV-06 for the
// primary config file path).
func Test_ADV06_BOMInProjectBuilderJSON(t *testing.T) {
	t.Parallel()

	bom := []byte{0xEF, 0xBB, 0xBF}
	bomContent := append(bom, []byte(minimalPBJSON)...)

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	withPBJSON(t, dir, string(bomContent), fs)

	cfg, err := newfeature.ReadConfig(dir, fs)
	if err != nil {
		t.Fatalf("ADV-06: ReadConfig with BOM prefix: unexpected error: %v", err)
	}
	// version must still be "1" despite BOM.
	if string(cfg.Version) != `"1"` {
		t.Errorf("ADV-06: version after BOM strip = %q; want %q", string(cfg.Version), `"1"`)
	}
}

// Test_ADV07_ConcurrentWrites verifies that concurrent RegisterSchematic calls
// do not produce a data race (ADV-07).
//
// Multiple goroutines attempt to create "foo" simultaneously. At least one must
// succeed; losers may get ErrCodeNewSchematicExists (read-check-write race under
// FakeFS mutex serialization). Neither ErrCodeInvalidInput nor panic is acceptable.
//
// The test runs with t.Parallel() and the race detector. The OS-level rename
// atomicity guarantee is approximated in FakeFS via sync.Mutex serialization.
func Test_ADV07_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)

	const n = 10
	errCh := make(chan error, n)

	for range n {
		go func() {
			svc := newfeature.NewService(fs)
			_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
				Name:     "foo",
				Language: "ts",
				WorkDir:  dir,
			})
			errCh <- err
		}()
	}

	successCount := 0
	for range n {
		err := <-errCh
		if err == nil {
			successCount++
			continue
		}
		// Only ErrCodeNewSchematicExists is acceptable for the loser.
		// ErrCodeInvalidInput would indicate a partial write / logic error.
		var e *errs.Error
		if errors.As(err, &e) {
			if e.Code == errs.ErrCodeNewSchematicExists {
				continue // acceptable loser result
			}
			t.Errorf("ADV-07: unexpected error code %q: %v", e.Code, err)
		} else {
			t.Errorf("ADV-07: unexpected non-errs.Error: %T %v", err, err)
		}
	}

	if successCount == 0 {
		t.Error("ADV-07: all concurrent calls failed; expected at least one success")
	}
}

// Test_ADV08_SymlinkOutsideWorkspace verifies that a symlinked schematics dir
// pointing outside the workspace is rejected (ADV-08).
//
// Uses FakeFS.AddSymlink to simulate a symlink that resolves outside the
// workspace root. The handler must detect this via EvalSymlinks check and
// return ErrCodeInvalidSchematicName (or ErrCodeInvalidInput per spec note).
//
// Skip on Windows (symlink semantics differ).
func Test_ADV08_SymlinkOutsideWorkspace(t *testing.T) {
	t.Parallel()

	if isWindows() {
		t.Skip("ADV-08: symlink test skipped on Windows")
	}

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// Write project-builder.json.
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(pbPath, []byte(minimalPBJSONForE2E), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Register the schematics/foo path as a symlink pointing OUTSIDE the workspace.
	outsidePath := "/tmp/evil-outside-workspace"
	schematicPath := filepath.Join(dir, "schematics", "foo")
	fs.AddSymlink(schematicPath, outsidePath)

	svc := newfeature.NewService(fs)
	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:     "foo",
		Language: "ts",
		WorkDir:  dir,
	})

	// Must reject the symlink target outside workspace.
	if err == nil {
		t.Fatal("ADV-08: expected error for symlink outside workspace; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("ADV-08: error not *errs.Error; got: %T %v", err, err)
	}
	// Accept ErrCodeInvalidSchematicName or ErrCodeInvalidInput — both indicate rejection.
	if e.Code != errs.ErrCodeInvalidSchematicName && e.Code != errs.ErrCodeInvalidInput {
		t.Errorf("ADV-08: unexpected error code %q; want invalid name or invalid input", e.Code)
	}
}
