// Package initialise — skill_artefact_test.go covers writeSkillArtefact.
//
// REQ coverage:
//   - REQ-SA-01 (SKILL.md written with locked bytes)
//   - REQ-SA-02 (pre-existing SKILL.md without --force → skip + warn, no error)
//   - REQ-SA-02 (pre-existing SKILL.md with --force → overwrite)
//   - REQ-SA-03 (--no-skill → no-op, nothing written)
package initialise

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init/template"
)

// skillArtefactPath returns the canonical .claude/skills/pbuilder/SKILL.md path
// under the given directory.
func skillArtefactPath(dir string) string {
	return filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
}

// Test_WriteSkillArtefact_WritesLockedBytes verifies that writeSkillArtefact
// writes the template.Skill bytes verbatim to .claude/skills/pbuilder/SKILL.md.
// REQ-SA-01.
func Test_WriteSkillArtefact_WritesLockedBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	req := InitRequest{Directory: dir, Force: false}

	path, err := writeSkillArtefact(ffs, req, template.Skill)
	if err != nil {
		t.Fatalf("writeSkillArtefact: unexpected error: %v", err)
	}

	want := skillArtefactPath(dir)
	if path != want {
		t.Errorf("returned path = %q, want %q", path, want)
	}

	got, readErr := ffs.ReadFile(want)
	if readErr != nil {
		t.Fatalf("SKILL.md not written: %v", readErr)
	}
	if !bytes.Equal(got, template.Skill) {
		t.Errorf("SKILL.md bytes mismatch:\ngot  len=%d\nwant len=%d", len(got), len(template.Skill))
	}
}

// Test_WriteSkillArtefact_PreexistingSkill_SkipsWithWarning verifies that when
// SKILL.md already exists and Force=false, writeSkillArtefact:
//   - returns the existing path (not empty string)
//   - returns ErrCodeInitSkillExists sentinel (the service downgrades to a warning)
//   - does NOT overwrite the file
//
// REQ-SA-02: "skip is not a failure" — the SERVICE converts this sentinel to
// a warning entry in InitResult.Warnings rather than propagating it as an error.
func Test_WriteSkillArtefact_PreexistingSkill_SkipsWithWarning(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	existing := []byte("existing content — must not be overwritten")
	target := skillArtefactPath(dir)
	if err := ffs.WriteFile(target, existing, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	req := InitRequest{Directory: dir, Force: false}
	path, err := writeSkillArtefact(ffs, req, template.Skill)

	// The sentinel error must be ErrCodeInitSkillExists (service downgrades to warn).
	if err == nil {
		t.Fatal("expected ErrCodeInitSkillExists sentinel when SKILL.md pre-exists; got nil")
	}
	if path != target {
		t.Errorf("returned path = %q, want %q", path, target)
	}

	// File must be untouched.
	got, readErr := ffs.ReadFile(target)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if !bytes.Equal(got, existing) {
		t.Errorf("SKILL.md was overwritten despite Force=false (REQ-SA-02)")
	}
}

// Test_WriteSkillArtefact_PreexistingSkill_Force_Overwrites verifies that when
// SKILL.md already exists and Force=true, writeSkillArtefact overwrites with
// the locked bytes. REQ-SA-02.
func Test_WriteSkillArtefact_PreexistingSkill_Force_Overwrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	target := skillArtefactPath(dir)
	if err := ffs.WriteFile(target, []byte("old content"), 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	req := InitRequest{Directory: dir, Force: true}
	path, err := writeSkillArtefact(ffs, req, template.Skill)
	if err != nil {
		t.Fatalf("writeSkillArtefact pre-existing force: unexpected error: %v", err)
	}
	if path != target {
		t.Errorf("returned path = %q, want %q", path, target)
	}

	got, readErr := ffs.ReadFile(target)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if !bytes.Equal(got, template.Skill) {
		t.Errorf("SKILL.md not overwritten with locked bytes after --force")
	}
}

// Test_WriteSkillArtefact_NoSkill_NoOp verifies that when req.NoSkill is true,
// writeSkillArtefact does nothing and returns an empty path and nil error.
// REQ-SA-03.
func Test_WriteSkillArtefact_NoSkill_NoOp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	req := InitRequest{Directory: dir, NoSkill: true}

	path, err := writeSkillArtefact(ffs, req, template.Skill)
	if err != nil {
		t.Fatalf("writeSkillArtefact no-skill: unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("returned path = %q, want empty string when --no-skill", path)
	}

	// Verify nothing was written to the fakeFS.
	target := skillArtefactPath(dir)
	if _, statErr := ffs.Stat(target); statErr == nil {
		t.Errorf("SKILL.md was written despite --no-skill (REQ-SA-03)")
	}
}

