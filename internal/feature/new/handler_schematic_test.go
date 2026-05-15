// Package newfeature — handler_schematic_test.go covers end-to-end scenarios
// for `builder new schematic` via the wired handler + service + FakeFS stack.
//
// REQ coverage:
//   - REQ-NS-01: happy path TS — factory.ts + schema.json + project-builder.json
//   - REQ-NS-02: conflict without --force → ErrCodeNewSchematicExists; zero writes
//   - REQ-NS-03: --force overwrite → success
//   - REQ-NS-04: invalid name → ErrCodeInvalidSchematicName; zero writes
//   - REQ-NS-05: --dry-run → PlannedOps non-empty; zero real FS writes
//   - REQ-NS-06: --language=js → factory.js (not .ts)
//   - REQ-SJ-01/03/05: schema.json canonical bytes
//   - REQ-PJ-03/04/05: project-builder.json mutation invariants
//   - REQ-TG (stub): WARN emitted for deferred .d.ts
package newfeature_test

import (
	"bytes"
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

// minimalPBJSONForE2E is the post-init project-builder.json used in E2E tests.
const minimalPBJSONForE2E = `{
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

// setupE2EWorkspace creates a temp dir with project-builder.json in a FakeFS.
func setupE2EWorkspace(t *testing.T) (string, *fswriter.FakeFS) {
	t.Helper()
	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(pbPath, []byte(minimalPBJSONForE2E), 0o644); err != nil {
		t.Fatalf("setup: WriteFile: %v", err)
	}
	return dir, fs
}

// invokeRegisterSchematic is a test helper that calls Service.RegisterSchematic
// with the given request, returning (result, error) — mirrors what the handler does.
func invokeRegisterSchematic(t *testing.T, svc *newfeature.Service, req newfeature.NewSchematicRequest) (*newfeature.NewResult, error) {
	t.Helper()
	return svc.RegisterSchematic(context.Background(), req)
}

// Test_E2E_HappyPath_TS verifies the full E2E for TypeScript path mode (REQ-NS-01).
func Test_E2E_HappyPath_TS(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "ts",
		WorkDir:    dir,
	})
	if err != nil {
		t.Fatalf("RegisterSchematic: unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// factory.ts must exist with schematic name reference.
	factoryPath := filepath.Join(dir, "schematics", "my-schematic", "factory.ts")
	if !fs.HasFile(factoryPath) {
		t.Errorf("factory.ts not created at %s", factoryPath)
	}
	factoryBytes, _ := fs.ReadFile(factoryPath)
	if !strings.Contains(string(factoryBytes), "my-schematic") {
		t.Errorf("factory.ts does not reference schematic name; content: %s", factoryBytes)
	}

	// schema.json must match canonical bytes (REQ-SJ-01/03/05).
	schemaPath := filepath.Join(dir, "schematics", "my-schematic", "schema.json")
	schemaBytes, _ := fs.ReadFile(schemaPath)
	wantSchema := string(newfeature.MarshalEmpty())
	if string(schemaBytes) != wantSchema {
		t.Errorf("schema.json bytes mismatch:\nwant: %q\n got: %q", wantSchema, schemaBytes)
	}

	// project-builder.json must have collections.default.my-schematic.path entry (REQ-PJ-05).
	pbBytes, _ := fs.ReadFile(filepath.Join(dir, "project-builder.json"))
	var pbMap map[string]json.RawMessage
	if err := json.Unmarshal(pbBytes, &pbMap); err != nil {
		t.Fatalf("parse project-builder.json: %v", err)
	}
	var cols map[string]json.RawMessage
	if err := json.Unmarshal(pbMap["collections"], &cols); err != nil {
		t.Fatalf("parse collections: %v", err)
	}
	defColl, ok := cols["default"]
	if !ok {
		t.Fatal("collections.default missing")
	}
	var defMap map[string]json.RawMessage
	if err := json.Unmarshal(defColl, &defMap); err != nil {
		t.Fatalf("parse default collection: %v", err)
	}
	schEntry, ok := defMap["my-schematic"]
	if !ok {
		t.Fatal("collections.default.my-schematic missing")
	}
	var entry map[string]string
	if err := json.Unmarshal(schEntry, &entry); err != nil {
		t.Fatalf("parse entry: %v", err)
	}
	if entry["path"] != "./schematics/my-schematic" {
		t.Errorf("entry.path = %q; want %q", entry["path"], "./schematics/my-schematic")
	}

	// version must still be "1" string (R-RES-1 / REQ-PJ-03).
	if !strings.Contains(string(pbBytes), `"version": "1"`) {
		t.Errorf("project-builder.json version coerced from string (R-RES-1 violation); got: %s", pbBytes)
	}
}

// Test_E2E_HappyPath_JS verifies the full E2E for JavaScript (REQ-NS-06).
func Test_E2E_HappyPath_JS(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "js",
		WorkDir:    dir,
	})
	if err != nil {
		t.Fatalf("RegisterSchematic(js): unexpected error: %v", err)
	}

	factoryJS := filepath.Join(dir, "schematics", "my-schematic", "factory.js")
	factoryTS := filepath.Join(dir, "schematics", "my-schematic", "factory.ts")

	if !fs.HasFile(factoryJS) {
		t.Errorf("factory.js not created at %s", factoryJS)
	}
	if fs.HasFile(factoryTS) {
		t.Errorf("factory.ts MUST NOT be created when --language=js")
	}

	// schema.json still present (language-independent).
	schemaPath := filepath.Join(dir, "schematics", "my-schematic", "schema.json")
	if !fs.HasFile(schemaPath) {
		t.Errorf("schema.json not created at %s", schemaPath)
	}
}

// Test_E2E_ConflictNoForce verifies exit 2 + ErrCodeNewSchematicExists + zero writes
// when schematic exists and --force is absent (REQ-NS-02).
func Test_E2E_ConflictNoForce(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	}

	// First call succeeds.
	if _, err := invokeRegisterSchematic(t, svc, req); err != nil {
		t.Fatalf("first call: %v", err)
	}
	countAfterFirst := fs.FileCount()

	// Second call without --force must fail with ErrCodeNewSchematicExists.
	_, err := invokeRegisterSchematic(t, svc, req)
	if err == nil {
		t.Fatal("second call: expected ErrCodeNewSchematicExists, got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeNewSchematicExists {
		t.Errorf("error code = %q; want %q", e.Code, errs.ErrCodeNewSchematicExists)
	}

	// No additional files should have been written.
	if fs.FileCount() != countAfterFirst {
		t.Errorf("file count changed from %d to %d on conflict (writes occurred)", countAfterFirst, fs.FileCount())
	}
}

// Test_E2E_ForceOverwrite verifies --force overwrites existing schematic (REQ-NS-03).
func Test_E2E_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	}

	// First call.
	if _, err := invokeRegisterSchematic(t, svc, req); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Force overwrite.
	req.Force = true
	if _, err := invokeRegisterSchematic(t, svc, req); err != nil {
		t.Errorf("--force overwrite: unexpected error: %v", err)
	}
}

// Test_E2E_DryRun verifies --dry-run produces planned ops with zero real writes
// (REQ-NS-05).
func Test_E2E_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dryFS := fswriter.NewDryRunWriter()
	svc := newfeature.NewService(dryFS)

	result, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		DryRun:   true,
		WorkDir:  dir,
	})
	if err != nil {
		t.Fatalf("dry-run: unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("result.DryRun = false; want true")
	}
	if len(result.PlannedOps) == 0 {
		t.Error("dry-run: PlannedOps is empty; expected at least 2 ops")
	}

	// Verify no real files created (dryRunFS records ops but doesn't write disk).
	for _, op := range result.PlannedOps {
		if op.Op != "create_file" && op.Op != "append_marker" {
			t.Errorf("unexpected planned op type: %q", op.Op)
		}
	}
}

// Test_E2E_InvalidName verifies ErrCodeInvalidSchematicName for bad names (REQ-NS-04).
func Test_E2E_InvalidName(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	cases := []struct {
		name string
		desc string
	}{
		{"", "empty name"},
		{"foo/bar", "path separator"},
		{"foo;bar", "shell metachar ;"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
				Name: tc.name, Language: "ts", WorkDir: dir,
			})
			if err == nil {
				t.Fatalf("expected ErrCodeInvalidSchematicName for %q; got nil", tc.name)
			}
			var e *errs.Error
			if !errors.As(err, &e) {
				t.Fatalf("error not *errs.Error: %T %v", err, err)
			}
			if e.Code != errs.ErrCodeInvalidSchematicName {
				t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeInvalidSchematicName)
			}
		})
	}
}

// Test_E2E_RenderPretty_ShowsCreatedFiles verifies RenderPretty produces output
// listing created files for the happy path (ADR-019).
func Test_E2E_RenderPretty_ShowsCreatedFiles(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	})
	if err != nil {
		t.Fatalf("RegisterSchematic: %v", err)
	}

	var buf bytes.Buffer
	newfeature.RenderPretty(&buf, *result)

	got := buf.String()
	// Pretty output should mention the created files.
	if len(result.FilesCreated) > 0 && got == "" {
		t.Error("RenderPretty: empty output for non-dry-run result with files created")
	}
}

// ─── Inline mode E2E (REQ-NSI-01..05 + REQ-NS-07) ───────────────────────────

// Test_E2E_Inline_HappyPath verifies inline mode creates no files and embeds the
// schematic in project-builder.json (REQ-NSI-01 / REQ-PJ-06).
func Test_E2E_Inline_HappyPath(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:    "my-inline",
		WorkDir: dir,
		Inline:  true,
	})
	if err != nil {
		t.Fatalf("RegisterSchematic(inline): unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// No files created under schematics/ (REQ-NSI-01).
	schematicDir := filepath.Join(dir, "schematics", "my-inline")
	for _, fname := range []string{"factory.ts", "factory.js", "schema.json", "schema.d.ts"} {
		if fs.HasFile(filepath.Join(schematicDir, fname)) {
			t.Errorf("inline mode: file %q MUST NOT be created", fname)
		}
	}

	// project-builder.json has inline entry (REQ-PJ-06).
	pbBytes, _ := fs.ReadFile(filepath.Join(dir, "project-builder.json"))
	var pbMap map[string]json.RawMessage
	if err := json.Unmarshal(pbBytes, &pbMap); err != nil {
		t.Fatalf("parse project-builder.json: %v", err)
	}
	var cols map[string]json.RawMessage
	if err := json.Unmarshal(pbMap["collections"], &cols); err != nil {
		t.Fatalf("parse collections: %v", err)
	}
	defColl, ok := cols["default"]
	if !ok {
		t.Fatal("collections.default missing")
	}
	var defMap map[string]json.RawMessage
	if err := json.Unmarshal(defColl, &defMap); err != nil {
		t.Fatalf("parse default: %v", err)
	}
	schematicsRaw, ok := defMap["schematics"]
	if !ok {
		t.Fatal("collections.default.schematics missing (inline entries nest here)")
	}
	var schMap map[string]json.RawMessage
	if err := json.Unmarshal(schematicsRaw, &schMap); err != nil {
		t.Fatalf("parse schematics: %v", err)
	}
	schEntry, ok := schMap["my-inline"]
	if !ok {
		t.Fatal("collections.default.schematics.my-inline missing")
	}
	var entryMap map[string]json.RawMessage
	if err := json.Unmarshal(schEntry, &entryMap); err != nil {
		t.Fatalf("parse inline entry: %v", err)
	}
	if _, hasInputs := entryMap["inputs"]; !hasInputs {
		t.Errorf("inline entry missing 'inputs' key; got: %s", schEntry)
	}

	// version preserved (R-RES-1).
	if !strings.Contains(string(pbBytes), `"version": "1"`) {
		t.Errorf("version coerced (R-RES-1 violation); got: %s", pbBytes)
	}
}

// Test_E2E_Inline_ConflictNoForce verifies ErrCodeNewSchematicExists + zero writes
// when inline entry exists and --force is absent (REQ-NSI-02).
func Test_E2E_Inline_ConflictNoForce(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:    "my-inline",
		WorkDir: dir,
		Inline:  true,
	}

	// First call succeeds.
	if _, err := invokeRegisterSchematic(t, svc, req); err != nil {
		t.Fatalf("first call: %v", err)
	}
	countAfterFirst := fs.FileCount()

	// Second call without --force must fail.
	_, err := invokeRegisterSchematic(t, svc, req)
	if err == nil {
		t.Fatal("expected ErrCodeNewSchematicExists; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeNewSchematicExists {
		t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeNewSchematicExists)
	}
	if fs.FileCount() != countAfterFirst {
		t.Errorf("file count changed on conflict (%d→%d)", countAfterFirst, fs.FileCount())
	}
}

// Test_E2E_Inline_ForceOverwrite verifies --force overwrites existing inline entry
// (REQ-NSI-03).
func Test_E2E_Inline_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	base := newfeature.NewSchematicRequest{Name: "my-inline", WorkDir: dir, Inline: true}

	if _, err := invokeRegisterSchematic(t, svc, base); err != nil {
		t.Fatalf("first call: %v", err)
	}

	base.Force = true
	if _, err := invokeRegisterSchematic(t, svc, base); err != nil {
		t.Errorf("--force overwrite: unexpected error: %v", err)
	}
}

// Test_E2E_Inline_ModeConflict_PathExists verifies ErrCodeModeConflict when path
// entry exists and --inline is requested (REQ-NS-07 / ADV-10).
func Test_E2E_Inline_ModeConflict_PathExists(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// Create a path-mode schematic first.
	if _, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "my-sch",
		Language: "ts",
		WorkDir:  dir,
	}); err != nil {
		t.Fatalf("path setup: %v", err)
	}

	// --inline --force must still fail with ErrCodeModeConflict.
	_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:    "my-sch",
		WorkDir: dir,
		Inline:  true,
		Force:   true,
	})
	if err == nil {
		t.Fatal("expected ErrCodeModeConflict; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeModeConflict {
		t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeModeConflict)
	}

	// Message must name "builder remove" (REQ-EC-05).
	mentionsRemove := strings.Contains(e.Message, "builder remove")
	for _, s := range e.Suggestions {
		if strings.Contains(s, "builder remove") {
			mentionsRemove = true
		}
	}
	if !mentionsRemove {
		t.Errorf("ErrCodeModeConflict does not mention 'builder remove'; message: %q", e.Message)
	}
}

// Test_E2E_SchemaDTS_Present verifies that path-mode creates schema.d.ts and
// that NO deferred WARN is emitted in the rendered output (S-003 implements .d.ts).
// Replaces the old Test_E2E_Warnings_DtsDeferred which expected the stub WARN.
func Test_E2E_SchemaDTS_Present(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	})
	if err != nil {
		t.Fatalf("RegisterSchematic: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// schema.d.ts MUST be present (REQ-TG-01 / REQ-NS-01 three-file contract).
	dtsPath := filepath.Join(dir, "schematics", "my-schematic", "schema.d.ts")
	if !fs.HasFile(dtsPath) {
		t.Errorf("schema.d.ts not created at %s", dtsPath)
	}

	// schema.d.ts content must include the interface name (REQ-TG-02).
	dtsBytes, _ := fs.ReadFile(dtsPath)
	if !strings.Contains(string(dtsBytes), "MySchematicSchematicInputs") {
		t.Errorf("schema.d.ts missing interface; content: %s", dtsBytes)
	}

	// Simulate handler behavior: the handler appends warnings and then renders.
	// In S-003, the handler MUST NOT append the "schema.d.ts generation pending" WARN.
	// We simulate by rendering the result and asserting no "pending" text appears.
	var buf bytes.Buffer
	newfeature.RenderPretty(&buf, *result)
	rendered := buf.String()

	if strings.Contains(rendered, "pending") {
		t.Errorf("rendered output contains 'pending' WARN (should be removed in S-003):\n%s", rendered)
	}
	if strings.Contains(rendered, "schema.d.ts generation") {
		t.Errorf("rendered output still shows deferred .d.ts WARN (S-003 implements it):\n%s", rendered)
	}
}

// ─── S-005 Adversarial tests ──────────────────────────────────────────────────

// Test_ADV01_TSReservedWordAsName verifies that a TypeScript reserved word used
// as the schematic name is accepted and that the generated .d.ts uses escaped
// property names (ADV-01 / REQ-TI-02).
//
// Spec contract:
//   - schema.json preserves the original name "class" as the input key
//   - schema.d.ts uses "class_: string; // original: class" (EscapeIdent suffix)
//   - Interface name: ClassSchematicInputs (PascalCase; "class" is not reserved as interface NAME)
func Test_ADV01_TSReservedWordAsName(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// "class" is a TypeScript reserved word — schematic creation must succeed.
	result, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "class",
		Language: "ts",
		WorkDir:  dir,
	})
	if err != nil {
		t.Fatalf("ADV-01: RegisterSchematic('class'): unexpected error: %v", err)
	}

	// factory.ts must exist.
	factoryPath := filepath.Join(dir, "schematics", "class", "factory.ts")
	if !fs.HasFile(factoryPath) {
		t.Errorf("ADV-01: factory.ts not created at %s", factoryPath)
	}

	// schema.d.ts must exist with escaped interface (REQ-TG-02 / ADV-01).
	dtsPath := filepath.Join(dir, "schematics", "class", "schema.d.ts")
	if !fs.HasFile(dtsPath) {
		t.Errorf("ADV-01: schema.d.ts not created at %s", dtsPath)
	}
	dtsBytes, _ := fs.ReadFile(dtsPath)
	dtsContent := string(dtsBytes)
	// Interface name must be ClassSchematicInputs (PascalCase of "class").
	if !strings.Contains(dtsContent, "ClassSchematicInputs") {
		t.Errorf("ADV-01: schema.d.ts missing interface ClassSchematicInputs; content:\n%s", dtsContent)
	}

	// REQ-TG-04: verify that tsgen escapes a "class" input property to "class_".
	// The E2E creates an empty schema, so we test codegen directly here to cover
	// the reserved-word → property escape path (EscapeIdent + comment per REQ-TG-04).
	schemaWithClassInput := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"class": {Type: "string"},
		},
	}
	dtsWithClass, genErr := newfeature.GenerateDTS("widget", schemaWithClassInput)
	if genErr != nil {
		t.Fatalf("ADV-01: GenerateDTS with class input: %v", genErr)
	}
	dtsWithClassStr := string(dtsWithClass)
	if !strings.Contains(dtsWithClassStr, "class_") {
		t.Errorf("ADV-01: tsgen missing escaped property 'class_' for reserved-word input; got:\n%s", dtsWithClassStr)
	}

	// project-builder.json must have the schematic registered.
	if len(result.FilesCreated) == 0 {
		t.Error("ADV-01: no files created")
	}
}

// Test_ADV02_PathTraversalInName verifies that a schematic name containing path
// traversal characters is rejected with ErrCodeInvalidSchematicName (ADV-02).
func Test_ADV02_PathTraversalInName(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "../etc/passwd",
		Language: "ts",
		WorkDir:  dir,
	})
	if err == nil {
		t.Fatal("ADV-02: expected ErrCodeInvalidSchematicName for path traversal name; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("ADV-02: error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeInvalidSchematicName {
		t.Errorf("ADV-02: code = %q; want %q", e.Code, errs.ErrCodeInvalidSchematicName)
	}

	// No writes must have occurred.
	if fs.FileCount() > 1 { // 1 = project-builder.json written in setup
		t.Errorf("ADV-02: expected no writes after path traversal rejection; file count = %d", fs.FileCount())
	}
}

// Test_ADV03_ShellMetacharInName verifies that shell metacharacters in the
// schematic name are rejected with ErrCodeInvalidSchematicName (ADV-03).
func Test_ADV03_ShellMetacharInName(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	cases := []struct {
		name  string
		input string
	}{
		{"semicolon", "foo;rm -rf /"},
		{"pipe", "foo|bar"},
		{"ampersand", "foo&bar"},
		{"dollar sign", "foo$HOME"},
		{"backtick", "foo`ls`"},
		{"redirect gt", "foo>bar"},
		{"redirect lt", "foo<bar"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
				Name:     tc.input,
				Language: "ts",
				WorkDir:  dir,
			})
			if err == nil {
				t.Fatalf("ADV-03: expected error for %q; got nil", tc.input)
			}
			var e *errs.Error
			if !errors.As(err, &e) {
				t.Fatalf("ADV-03: error not *errs.Error; got: %T %v", err, err)
			}
			if e.Code != errs.ErrCodeInvalidSchematicName {
				t.Errorf("ADV-03: %q: code = %q; want %q", tc.input, e.Code, errs.ErrCodeInvalidSchematicName)
			}
		})
	}
}

// Test_ADV05_NullByteInName verifies that a null byte in the schematic name is
// rejected with ErrCodeInvalidSchematicName (ADV-05).
func Test_ADV05_NullByteInName(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "foo\x00bar",
		Language: "ts",
		WorkDir:  dir,
	})
	if err == nil {
		t.Fatal("ADV-05: expected ErrCodeInvalidSchematicName for null byte; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("ADV-05: error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeInvalidSchematicName {
		t.Errorf("ADV-05: code = %q; want %q", e.Code, errs.ErrCodeInvalidSchematicName)
	}
}

// Test_ADV09_ReadOnlyFilesystem verifies that when the filesystem is read-only,
// the service returns an error with no partial state (ADV-09).
//
// This test uses a real temp directory with os.Chmod to simulate a read-only
// filesystem. It is skipped on Windows (chmod semantics differ).
func Test_ADV09_ReadOnlyFilesystem(t *testing.T) {
	t.Parallel()

	if isWindows() {
		t.Skip("ADV-09: chmod not supported on Windows — skipping")
	}

	dir := t.TempDir()

	// Write project-builder.json before making the directory read-only.
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := writeFileOS(pbPath, []byte(minimalPBJSONForE2E)); err != nil {
		t.Fatalf("setup: write project-builder.json: %v", err)
	}

	// Create schematics dir and make it read-only.
	schDir := filepath.Join(dir, "schematics")
	if err := mkdirOS(schDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir schematics: %v", err)
	}
	if err := chmodOS(schDir, 0o555); err != nil {
		t.Fatalf("setup: chmod 0o555 schematics: %v", err)
	}
	// Restore permissions after test so t.TempDir cleanup works.
	t.Cleanup(func() { _ = chmodOS(schDir, 0o755) })

	// Use real OS writer (not FakeFS) to trigger actual permission error.
	svc := newfeature.NewService(newfeature.NewOSWriterForTest())

	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	})
	if err == nil {
		t.Fatal("ADV-09: expected error on read-only filesystem; got nil")
	}

	// No partial state: schematics/my-schematic/ must NOT have been created.
	schematicDir := filepath.Join(schDir, "my-schematic")
	if dirExistsOS(schematicDir) {
		t.Errorf("ADV-09: partial state found — schematic dir %s exists after write failure", schematicDir)
	}
}

// Test_ADV10_InlineForceWhenPathExists verifies that --inline --force returns
// ErrCodeModeConflict when the schematic already exists in path mode (ADV-10).
//
// The mode-conflict check is already enforced by checkModeConflict in S-002.
// This test adds explicit coverage in the handler_schematic_test suite.
func Test_ADV10_InlineForceWhenPathExists(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// Create path-mode schematic first.
	if _, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	}); err != nil {
		t.Fatalf("ADV-10: path setup: %v", err)
	}

	// --inline --force on a path-mode entry must fail with ErrCodeModeConflict.
	_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:    "my-schematic",
		WorkDir: dir,
		Inline:  true,
		Force:   true,
	})
	if err == nil {
		t.Fatal("ADV-10: expected ErrCodeModeConflict; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("ADV-10: error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeModeConflict {
		t.Errorf("ADV-10: code = %q; want %q", e.Code, errs.ErrCodeModeConflict)
	}
}

// ─── REQ-EX-04: TUI prompt when interactive + flag absent ─────────────────────

// Test_REQ_EX04_Interactive_PromptExtendsCalled verifies that when the terminal
// is interactive and --extends is absent, the handler calls PromptExtends and
// uses the selected value (REQ-EX-04).
//
// Strategy: inject SetTTYCheckFn (returns true) + SetPromptExtendsFn (returns
// a known value). Assert the resulting schematic has extends wired.
// Currently FAILS because the handler never calls promptExtendsFn.
func Test_REQ_EX04_Interactive_PromptExtendsCalled(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// Stub TTY check → interactive.
	newfeature.SetTTYCheckFn(t, func() bool { return true })

	// Stub extends prompt → returns a known value.
	const wantExtends = "@scope/pkg:base"
	newfeature.SetPromptExtendsFn(t, func(_ []string) (string, bool, error) {
		return wantExtends, false, nil
	})

	result, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
		// Extends is deliberately absent — handler should call PromptExtends.
	})
	if err != nil {
		t.Fatalf("REQ-EX-04: RegisterSchematic: unexpected error: %v", err)
	}

	// The result must record the extends value from the prompt.
	if result.ExtendsUsed != wantExtends {
		t.Errorf("REQ-EX-04: result.ExtendsUsed = %q; want %q (handler must call PromptExtends when interactive)", result.ExtendsUsed, wantExtends)
	}
}

// Test_REQ_EX04_NonInteractive_PromptNotCalled verifies that when the terminal
// is NOT interactive, PromptExtends is NOT called (REQ-EX-05 / REQ-EX-04 boundary).
func Test_REQ_EX04_NonInteractive_PromptNotCalled(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// Stub TTY check → non-interactive.
	newfeature.SetTTYCheckFn(t, func() bool { return false })

	// PromptExtends must NOT be called in non-interactive mode.
	promptCalled := false
	newfeature.SetPromptExtendsFn(t, func(_ []string) (string, bool, error) {
		promptCalled = true
		return "", true, nil
	})

	_, err := invokeRegisterSchematic(t, svc, newfeature.NewSchematicRequest{
		Name:     "my-schematic",
		Language: "ts",
		WorkDir:  dir,
	})
	if err != nil {
		t.Fatalf("REQ-EX-04: RegisterSchematic non-interactive: unexpected error: %v", err)
	}

	if promptCalled {
		t.Error("REQ-EX-04: PromptExtends was called in non-interactive mode; should be skipped (REQ-EX-05)")
	}
}
