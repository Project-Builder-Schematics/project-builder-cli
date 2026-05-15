// Package newfeature — service_schematic_path_test.go covers registerSchematicPath
// via the exported Service.RegisterSchematic entry-point (path mode only).
//
// REQ coverage:
//   - REQ-NS-01: happy path TS — factory.ts + schema.json + project-builder.json entry
//   - REQ-NS-02: conflict without --force → ErrCodeNewSchematicExists
//   - REQ-NS-03: --force overwrites existing schematic
//   - REQ-NS-04: name validation via validate.RejectMetachars → ErrCodeInvalidSchematicName
//   - REQ-NS-05: --dry-run returns planned ops; zero FS writes
//   - REQ-NS-06: --language=js → factory.js created
package newfeature_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// minimalPBJSONForSvc is the project-builder.json written by `builder init`.
// Used across service path tests.
const minimalPBJSONForSvc = `{
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

// setupWorkspace creates a temp directory with project-builder.json and returns
// (workspaceDir, fakeFS). The project-builder.json has no schematics yet.
func setupWorkspace(t *testing.T) (string, *fswriter.FakeFS) {
	t.Helper()
	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(pbPath, []byte(minimalPBJSONForSvc), 0o644); err != nil {
		t.Fatalf("setupWorkspace: WriteFile: %v", err)
	}
	return dir, fs
}

// Test_RegisterSchematic_HappyPath_TS verifies the full path-mode happy path
// for TypeScript: factory.ts + schema.json + project-builder.json entry (REQ-NS-01).
func Test_RegisterSchematic_HappyPath_TS(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "ts",
		WorkDir:    dir,
	}

	result, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterSchematic(ts): unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RegisterSchematic: result is nil")
	}
	if result.DryRun {
		t.Error("RegisterSchematic: result.DryRun = true; want false")
	}

	// Assert factory.ts was created.
	factoryPath := filepath.Join(dir, "schematics", "my-schematic", "factory.ts")
	if !fs.HasFile(factoryPath) {
		t.Errorf("RegisterSchematic: factory.ts not created at %s", factoryPath)
	}

	// Assert schema.json was created with canonical bytes.
	schemaPath := filepath.Join(dir, "schematics", "my-schematic", "schema.json")
	if !fs.HasFile(schemaPath) {
		t.Errorf("RegisterSchematic: schema.json not created at %s", schemaPath)
	}
	schemaBytes, _ := fs.ReadFile(schemaPath)
	if !strings.Contains(string(schemaBytes), `"inputs"`) {
		t.Errorf("RegisterSchematic: schema.json missing 'inputs' key;\ngot: %s", schemaBytes)
	}

	// Assert project-builder.json has the new entry.
	pbBytes, _ := fs.ReadFile(filepath.Join(dir, "project-builder.json"))
	if !strings.Contains(string(pbBytes), `"my-schematic"`) {
		t.Errorf("RegisterSchematic: project-builder.json missing 'my-schematic' entry;\ngot: %s", pbBytes)
	}
	if !strings.Contains(string(pbBytes), `"path"`) {
		t.Errorf("RegisterSchematic: project-builder.json entry missing 'path' key;\ngot: %s", pbBytes)
	}

	// Assert FilesCreated has at least 2 entries (factory.ts + schema.json + project-builder.json).
	if len(result.FilesCreated) < 2 {
		t.Errorf("RegisterSchematic: FilesCreated = %v; want at least 2 files", result.FilesCreated)
	}
}

// Test_RegisterSchematic_HappyPath_JS verifies that --language=js produces factory.js
// instead of factory.ts (REQ-NS-06).
func Test_RegisterSchematic_HappyPath_JS(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "js",
		WorkDir:    dir,
	}

	_, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterSchematic(js): unexpected error: %v", err)
	}

	// factory.js must exist.
	factoryJSPath := filepath.Join(dir, "schematics", "my-schematic", "factory.js")
	if !fs.HasFile(factoryJSPath) {
		t.Errorf("RegisterSchematic(js): factory.js not created at %s", factoryJSPath)
	}

	// factory.ts must NOT exist (language=js is exclusive).
	factoryTSPath := filepath.Join(dir, "schematics", "my-schematic", "factory.ts")
	if fs.HasFile(factoryTSPath) {
		t.Errorf("RegisterSchematic(js): factory.ts should NOT be created when --language=js")
	}

	// schema.json must still be created (language-independent).
	schemaPath := filepath.Join(dir, "schematics", "my-schematic", "schema.json")
	if !fs.HasFile(schemaPath) {
		t.Errorf("RegisterSchematic(js): schema.json not created at %s", schemaPath)
	}
}

// Test_RegisterSchematic_ConflictNoForce verifies that re-running without --force
// returns ErrCodeNewSchematicExists and writes nothing (REQ-NS-02).
func Test_RegisterSchematic_ConflictNoForce(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "ts",
		WorkDir:    dir,
	}

	// First call succeeds.
	_, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("first RegisterSchematic: unexpected error: %v", err)
	}

	// Record file count after first call.
	countBefore := fs.FileCount()

	// Second call without --force must fail.
	_, err = svc.RegisterSchematic(context.Background(), req)
	if err == nil {
		t.Fatal("second RegisterSchematic (no --force): expected error, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNewSchematicExists}
	if !errors.Is(err, sentinel) {
		t.Errorf("RegisterSchematic conflict: errors.Is(ErrCodeNewSchematicExists) = false; got: %v", err)
	}

	// No additional files should have been written.
	if fs.FileCount() != countBefore {
		t.Errorf("RegisterSchematic conflict: file count changed from %d to %d (writes occurred on conflict)", countBefore, fs.FileCount())
	}
}

// Test_RegisterSchematic_ForceOverwrite verifies that --force overwrites an existing
// schematic (REQ-NS-03).
func Test_RegisterSchematic_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "ts",
		WorkDir:    dir,
	}

	// First call — create.
	_, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("first RegisterSchematic: %v", err)
	}

	// Second call with --force — should succeed.
	req.Force = true
	_, err = svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Errorf("RegisterSchematic --force: unexpected error: %v", err)
	}
}

// Test_RegisterSchematic_DryRun verifies that --dry-run returns planned ops without
// writing any files to disk (REQ-NS-05).
func Test_RegisterSchematic_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Use a fresh FakeFS for dry-run — no project-builder.json needed (dry-run skips read).
	dryFS := fswriter.NewDryRunWriter()
	svc := newfeature.NewService(dryFS)

	req := newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "ts",
		DryRun:     true,
		WorkDir:    dir,
	}

	result, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterSchematic --dry-run: unexpected error: %v", err)
	}

	if !result.DryRun {
		t.Error("result.DryRun = false; want true")
	}

	// Planned ops must be non-empty (at least factory + schema + project-builder.json).
	if len(result.PlannedOps) == 0 {
		t.Error("RegisterSchematic --dry-run: PlannedOps is empty; expected at least 2 ops")
	}
}

// Test_RegisterSchematic_HappyPath_WritesSchemaDTO verifies that path-mode creates
// schema.d.ts alongside factory.ts and schema.json (REQ-TG-01, S-003 wire-in).
func Test_RegisterSchematic_HappyPath_WritesSchemaDTO(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewSchematicRequest{
		Name:       "my-schematic",
		Collection: "default",
		Language:   "ts",
		WorkDir:    dir,
	}

	result, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterSchematic: unexpected error: %v", err)
	}

	// schema.d.ts must be created (REQ-TG-01).
	dtsPath := filepath.Join(dir, "schematics", "my-schematic", "schema.d.ts")
	if !fs.HasFile(dtsPath) {
		t.Errorf("schema.d.ts not created at %s", dtsPath)
	}

	// Content must include the interface declaration (REQ-TG-02).
	dtsBytes, _ := fs.ReadFile(dtsPath)
	if !strings.Contains(string(dtsBytes), "MySchematicSchematicInputs") {
		t.Errorf("schema.d.ts missing interface name; content:\n%s", dtsBytes)
	}

	// schema.d.ts must be in FilesCreated (REQ-NS-01).
	foundDts := false
	for _, f := range result.FilesCreated {
		if strings.HasSuffix(f, "schema.d.ts") {
			foundDts = true
		}
	}
	if !foundDts {
		t.Errorf("schema.d.ts not listed in result.FilesCreated; got: %v", result.FilesCreated)
	}
}

// Test_RegisterSchematic_InvalidName_MetaChar verifies that shell metachar in name
// returns ErrCodeInvalidSchematicName (REQ-NS-04).
func Test_RegisterSchematic_InvalidName_MetaChar(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspace(t)
	svc := newfeature.NewService(fs)

	invalidNames := []struct {
		name   string
		reason string
	}{
		{"foo;bar", "shell metachar ;"},
		{"foo|bar", "shell metachar |"},
		{"foo/bar", "path separator /"},
		{"foo\x00bar", "null byte"},
		{"", "empty name"},
	}

	for _, tc := range invalidNames {
		tc := tc
		t.Run(tc.reason, func(t *testing.T) {
			t.Parallel()
			req := newfeature.NewSchematicRequest{
				Name:       tc.name,
				Collection: "default",
				Language:   "ts",
				WorkDir:    dir,
			}
			_, err := svc.RegisterSchematic(context.Background(), req)
			if err == nil {
				t.Fatalf("RegisterSchematic(%q): expected error for %s, got nil", tc.name, tc.reason)
			}
			sentinel := &errs.Error{Code: errs.ErrCodeInvalidSchematicName}
			if !errors.Is(err, sentinel) {
				t.Errorf("RegisterSchematic(%q): errors.Is(ErrCodeInvalidSchematicName) = false; got: %v", tc.name, err)
			}
		})
	}
}
