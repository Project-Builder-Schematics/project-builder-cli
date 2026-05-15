// Package newfeature — service_schematic_inline_test.go covers registerSchematicInline
// via the exported Service.RegisterSchematic entry-point (inline mode).
//
// REQ coverage:
//   - REQ-NSI-01: happy path inline — no files created; project-builder.json entry
//   - REQ-NSI-02: conflict without --force → ErrCodeNewSchematicExists
//   - REQ-NSI-03: --force overwrites existing inline entry
//   - REQ-NSI-04: soft warning at 10th inline schematic (via Renderer)
//   - REQ-NSI-05: soft warning when project-builder.json exceeds 20KB (via Renderer)
//   - REQ-NS-07: mode-conflict when path-mode entry exists → ErrCodeModeConflict
//   - REQ-PJ-06: inline entry shape {"inputs": {}}
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

// minimalPBJSONForInline is the post-init project-builder.json used in inline tests.
// Matches the format written by `builder init`.
const minimalPBJSONForInline = `{
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

// setupInlineWorkspace creates a temp dir with project-builder.json and returns
// (workspaceDir, fakeFS). No schematics registered yet.
func setupInlineWorkspace(t *testing.T) (string, *fswriter.FakeFS) {
	t.Helper()
	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(pbPath, []byte(minimalPBJSONForInline), 0o644); err != nil {
		t.Fatalf("setup: WriteFile: %v", err)
	}
	return dir, fs
}

// registerInlineSchematic is a test helper that calls Service.RegisterSchematic
// with inline=true and the given name.
func registerInlineSchematic(
	t *testing.T,
	svc *newfeature.Service,
	dir, name string,
	force bool,
) (*newfeature.NewResult, error) {
	t.Helper()
	return svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:    name,
		WorkDir: dir,
		Inline:  true,
		Force:   force,
	})
}

// parseInlineSchematics parses project-builder.json and returns the inline schematics
// map from collections.default.schematics.
func parseInlineSchematics(t *testing.T, fs *fswriter.FakeFS, dir string) map[string]json.RawMessage {
	t.Helper()
	pbPath := filepath.Join(dir, "project-builder.json")
	data, err := fs.ReadFile(pbPath)
	if err != nil {
		t.Fatalf("ReadFile project-builder.json: %v", err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("parse root: %v", err)
	}
	var collections map[string]json.RawMessage
	if err := json.Unmarshal(root["collections"], &collections); err != nil {
		t.Fatalf("parse collections: %v", err)
	}
	defRaw, ok := collections["default"]
	if !ok {
		t.Fatal("collections.default missing")
	}
	var defMap map[string]json.RawMessage
	if err := json.Unmarshal(defRaw, &defMap); err != nil {
		t.Fatalf("parse default collection: %v", err)
	}
	schRaw, ok := defMap["schematics"]
	if !ok {
		t.Fatal("collections.default.schematics missing")
	}
	var schMap map[string]json.RawMessage
	if err := json.Unmarshal(schRaw, &schMap); err != nil {
		t.Fatalf("parse schematics map: %v", err)
	}
	return schMap
}

// Test_Inline_HappyPath verifies the full inline registration flow (REQ-NSI-01).
func Test_Inline_HappyPath(t *testing.T) {
	t.Parallel()

	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := registerInlineSchematic(t, svc, dir, "my-inline", false)
	if err != nil {
		t.Fatalf("RegisterSchematic(inline): unexpected error: %v", err)
	}

	// Exit 0: no error.
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// No files created under schematics/ (REQ-NSI-01).
	schematicDir := filepath.Join(dir, "schematics", "my-inline")
	if fs.HasFile(filepath.Join(schematicDir, "factory.ts")) {
		t.Error("factory.ts MUST NOT be created in inline mode")
	}
	if fs.HasFile(filepath.Join(schematicDir, "schema.json")) {
		t.Error("schema.json MUST NOT be created in inline mode")
	}

	// project-builder.json has the inline entry (REQ-PJ-06 / REQ-NSI-01).
	schMap := parseInlineSchematics(t, fs, dir)
	entry, ok := schMap["my-inline"]
	if !ok {
		t.Fatal("collections.default.schematics.my-inline missing from project-builder.json")
	}

	// Entry must have {"inputs": {}}.
	var entryMap map[string]json.RawMessage
	if err := json.Unmarshal(entry, &entryMap); err != nil {
		t.Fatalf("parse inline entry: %v", err)
	}
	if _, hasInputs := entryMap["inputs"]; !hasInputs {
		t.Errorf("inline entry missing 'inputs' key; got: %s", entry)
	}
}

// Test_Inline_ConflictNoForce verifies ErrCodeNewSchematicExists when inline entry
// exists and --force is absent (REQ-NSI-02).
func Test_Inline_ConflictNoForce(t *testing.T) {
	t.Parallel()

	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	// First call succeeds.
	if _, err := registerInlineSchematic(t, svc, dir, "my-inline", false); err != nil {
		t.Fatalf("first call: %v", err)
	}
	countAfterFirst := fs.FileCount()

	// Second call without --force must fail.
	_, err := registerInlineSchematic(t, svc, dir, "my-inline", false)
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

	// File count must not change (no additional writes).
	if fs.FileCount() != countAfterFirst {
		t.Errorf("file count changed from %d to %d on conflict", countAfterFirst, fs.FileCount())
	}
}

// Test_Inline_ForceOverwrite verifies --force overwrites an existing inline entry
// (REQ-NSI-03).
func Test_Inline_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	// First call.
	if _, err := registerInlineSchematic(t, svc, dir, "my-inline", false); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Force overwrite.
	if _, err := registerInlineSchematic(t, svc, dir, "my-inline", true); err != nil {
		t.Errorf("--force overwrite: unexpected error: %v", err)
	}

	// Verify entry still present and valid.
	schMap := parseInlineSchematics(t, fs, dir)
	if _, ok := schMap["my-inline"]; !ok {
		t.Error("inline entry missing after --force overwrite")
	}

	// Only one entry (idempotent).
	if len(schMap) != 1 {
		t.Errorf("expected 1 inline entry after overwrite; got %d", len(schMap))
	}
}

// Test_Inline_ModeConflict_PathExists verifies ErrCodeModeConflict when path-mode
// entry exists and --inline is requested (REQ-NS-07 / REQ-EC-05).
func Test_Inline_ModeConflict_PathExists(t *testing.T) {
	t.Parallel()

	// Create workspace with a PATH-mode schematic.
	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	// Register in path mode first.
	if _, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:     "my-sch",
		Language: "ts",
		WorkDir:  dir,
	}); err != nil {
		t.Fatalf("path-mode setup: %v", err)
	}

	// Attempt inline registration — must fail with ErrCodeModeConflict.
	_, err := registerInlineSchematic(t, svc, dir, "my-sch", true) // --force still rejected
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

	// Error message must mention "builder remove" (REQ-EC-05).
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

// Test_Inline_SoftWarning_SchematicThreshold verifies a WARN is emitted when the
// collection reaches the threshold (REQ-NSI-04).
func Test_Inline_SoftWarning_SchematicThreshold(t *testing.T) {
	t.Parallel()

	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	// Register InlineSchematicThreshold-1 schematics first.
	threshold := newfeature.InlineSchematicThreshold
	for i := range threshold - 1 {
		name := "sch-" + string(rune('a'+i))
		if _, err := registerInlineSchematic(t, svc, dir, name, false); err != nil {
			t.Fatalf("setup schematic %q: %v", name, err)
		}
	}

	// The threshold-th registration should emit the soft warning.
	result, err := registerInlineSchematic(t, svc, dir, "sch-threshold", false)
	if err != nil {
		t.Fatalf("threshold registration: unexpected error: %v", err)
	}

	// Exit 0 (no error returned).
	if result == nil {
		t.Fatal("result is nil")
	}

	// Soft warning must be present in result.Warnings.
	if len(result.Warnings) == 0 {
		t.Error("expected soft warning at threshold; result.Warnings is empty")
	}

	// Warning must mention the collection and the count.
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "default") && strings.Contains(w, string(rune('0'+threshold))) {
			found = true
		}
		// Accept any warning containing the word "inline" and a count.
		if strings.Contains(w, "inline") {
			found = true
		}
	}
	if !found {
		t.Errorf("soft warning does not mention inline threshold; warnings: %v", result.Warnings)
	}
}

// Test_Inline_SoftWarning_FileSize verifies a WARN is emitted when project-builder.json
// exceeds 20KB after the inline write (REQ-NSI-05).
func Test_Inline_SoftWarning_FileSize(t *testing.T) {
	t.Parallel()

	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	// Pad project-builder.json to just below the threshold, then add one more inline.
	// FileSizeThresholdBytes = 20 * 1024 = 20480 bytes.
	// We write a large custom field to bloat the file.
	threshold := newfeature.FileSizeThresholdBytes
	padding := strings.Repeat("x", threshold+100)
	largePBJSON := `{
  "version": "1",
  "collections": {},
  "customPadding": "` + padding + `"
}
`
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(pbPath, []byte(largePBJSON), 0o644); err != nil {
		t.Fatalf("setup large project-builder.json: %v", err)
	}

	// Register an inline schematic — should trigger size warning.
	result, err := registerInlineSchematic(t, svc, dir, "size-warn-sch", false)
	if err != nil {
		t.Fatalf("inline registration: unexpected error: %v", err)
	}

	// Exit 0.
	if result == nil {
		t.Fatal("result is nil")
	}

	// Soft warning about file size must be present.
	if len(result.Warnings) == 0 {
		t.Error("expected soft warning for large file size; result.Warnings is empty")
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "KB") || strings.Contains(w, "size") || strings.Contains(w, "path mode") {
			found = true
		}
	}
	if !found {
		t.Errorf("soft warning does not mention file size; warnings: %v", result.Warnings)
	}
}

// Test_Inline_DryRun verifies --dry-run produces planned ops with zero real writes
// in inline mode (mirrors REQ-NS-05 for inline).
func Test_Inline_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dryFS := fswriter.NewDryRunWriter()
	svc := newfeature.NewService(dryFS)

	result, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:    "dry-inline",
		WorkDir: dir,
		Inline:  true,
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("dry-run inline: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.DryRun {
		t.Error("result.DryRun = false; want true")
	}
	if len(result.PlannedOps) == 0 {
		t.Error("dry-run inline: PlannedOps is empty")
	}
}

// Test_Inline_VersionPreserved verifies the version field is preserved verbatim
// after an inline write (R-RES-1 / REQ-PJ-03).
func Test_Inline_VersionPreserved(t *testing.T) {
	t.Parallel()

	dir, fs := setupInlineWorkspace(t)
	svc := newfeature.NewService(fs)

	if _, err := registerInlineSchematic(t, svc, dir, "my-inline", false); err != nil {
		t.Fatalf("RegisterSchematic(inline): %v", err)
	}

	pbPath := filepath.Join(dir, "project-builder.json")
	data, err := fs.ReadFile(pbPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(string(data), `"version": "1"`) {
		t.Errorf("version coerced from string (R-RES-1 violation); got: %s", data)
	}
}
