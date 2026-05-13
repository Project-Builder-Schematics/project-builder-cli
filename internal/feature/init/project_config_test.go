// Package initialise — project_config_test.go tests writeProjectConfig.
//
// REQ coverage:
//   - REQ-PJ-01 (locked v1 bytes — golden byte-equality)
//   - REQ-PJ-02 (atomic write via FSWriter, no direct os.* call)
//   - REQ-PJ-03 (--no-skill flips skill.enabled to false)
//   - REQ-PJ-04 (forward-compat unknown-keys — doc-only; see implementation comment)
//   - REQ-DV-04 (pre-existing project-builder.json → ErrCodeInitConfigExists)
package initialise

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// lockedProjectBuilderJSON is the exact v1 byte content of project-builder.json
// as locked in the spec (REQ-PJ-01). The test asserts byte-for-byte equality.
//
// Field order MUST match the locked layout:
//
//	$schema → version → collections → dependencies → settings → skill
//
// 2-space indent, trailing newline.
var lockedProjectBuilderJSON = []byte(`{
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
`)

// lockedProjectBuilderJSONNoSkill is the v1 content when --no-skill is passed
// (REQ-PJ-03): skill.enabled is false.
var lockedProjectBuilderJSONNoSkill = []byte(`{
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
    "enabled": false,
    "path": ".claude/skills/pbuilder/SKILL.md"
  }
}
`)

// Test_WriteProjectConfig_EmitsLockedV1Bytes verifies that writeProjectConfig
// writes the exact locked v1 bytes to project-builder.json (REQ-PJ-01).
func Test_WriteProjectConfig_EmitsLockedV1Bytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	req := InitRequest{Directory: dir}

	path, err := writeProjectConfig(ffs, req)
	if err != nil {
		t.Fatalf("writeProjectConfig: unexpected error: %v", err)
	}

	want := filepath.Join(dir, "project-builder.json")
	if path != want {
		t.Errorf("returned path = %q, want %q", path, want)
	}

	got, readErr := ffs.ReadFile(want)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", want, readErr)
	}

	if !bytes.Equal(got, lockedProjectBuilderJSON) {
		t.Errorf("project-builder.json bytes mismatch\ngot:\n%s\nwant:\n%s", got, lockedProjectBuilderJSON)
	}
}

// Test_WriteProjectConfig_NoSkill_FlipsSkillEnabledFalse verifies that when
// NoSkill is true, the written config has skill.enabled = false (REQ-PJ-03).
func Test_WriteProjectConfig_NoSkill_FlipsSkillEnabledFalse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	req := InitRequest{Directory: dir, NoSkill: true}

	_, err := writeProjectConfig(ffs, req)
	if err != nil {
		t.Fatalf("writeProjectConfig(no-skill): unexpected error: %v", err)
	}

	path := filepath.Join(dir, "project-builder.json")
	got, readErr := ffs.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}

	if !bytes.Equal(got, lockedProjectBuilderJSONNoSkill) {
		t.Errorf("project-builder.json (no-skill) bytes mismatch\ngot:\n%s\nwant:\n%s", got, lockedProjectBuilderJSONNoSkill)
	}
}

// Test_WriteProjectConfig_PreexistingConfig_ReturnsErrInitConfigExists verifies
// that if project-builder.json already exists and Force is false, the function
// returns ErrCodeInitConfigExists without overwriting the file (REQ-DV-04).
func Test_WriteProjectConfig_PreexistingConfig_ReturnsErrInitConfigExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	// Pre-seed an existing config.
	existingContent := []byte(`{"version":"existing"}`)
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := ffs.WriteFile(pbPath, existingContent, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	req := InitRequest{Directory: dir, Force: false}
	_, err := writeProjectConfig(ffs, req)
	if err == nil {
		t.Fatal("expected ErrCodeInitConfigExists, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInitConfigExists}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, ErrCodeInitConfigExists) = false; got: %v", err)
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("errors.As(*errs.Error) failed")
	}
	if len(e.Suggestions) == 0 {
		t.Error("Suggestions must be non-empty for ErrCodeInitConfigExists (REQ-EC-02)")
	}

	// File must not have been overwritten.
	got, readErr := ffs.ReadFile(pbPath)
	if readErr != nil {
		t.Fatalf("ReadFile after error: %v", readErr)
	}
	if !bytes.Equal(got, existingContent) {
		t.Error("pre-existing file was overwritten despite no --force")
	}
}

// Test_WriteProjectConfig_PreexistingWithForce_Overwrites verifies that when
// Force is true, an existing project-builder.json is overwritten with locked
// v1 bytes (REQ-DV-04, REQ-EC-03).
func Test_WriteProjectConfig_PreexistingWithForce_Overwrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	pbPath := filepath.Join(dir, "project-builder.json")
	if err := ffs.WriteFile(pbPath, []byte(`{"version":"old"}`), 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	req := InitRequest{Directory: dir, Force: true}
	path, err := writeProjectConfig(ffs, req)
	if err != nil {
		t.Fatalf("writeProjectConfig with --force: unexpected error: %v", err)
	}
	if path != pbPath {
		t.Errorf("returned path = %q, want %q", path, pbPath)
	}

	got, readErr := ffs.ReadFile(pbPath)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if !bytes.Equal(got, lockedProjectBuilderJSON) {
		t.Errorf("file after force overwrite:\ngot:\n%s\nwant:\n%s", got, lockedProjectBuilderJSON)
	}
}
