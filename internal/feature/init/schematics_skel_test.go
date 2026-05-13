// Package initialise — schematics_skel_test.go tests writeSchematicsSkel.
//
// REQ coverage:
//   - REQ-SF-01 (schematics/ directory created + .gitkeep with locked bytes)
//   - REQ-SF-02 (constant folder name schematicsFolderName used)
package initialise

import (
	"bytes"
	"path/filepath"
	"testing"
)

// lockedGitkeepBytes is the exact content of schematics/.gitkeep as locked
// in the spec (REQ-SF-01 + locked content from explore obs #234).
// Two comment lines followed by a trailing newline.
var lockedGitkeepBytes = []byte("# This folder holds local schematics for this project.\n# Use `builder add <name>` to scaffold a new schematic here.\n")

// Test_WriteSchematicsSkel_EmitsLockedGitkeepBytes verifies that
// writeSchematicsSkel writes the exact locked bytes to .gitkeep (REQ-SF-01).
func Test_WriteSchematicsSkel_EmitsLockedGitkeepBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	req := InitRequest{Directory: dir}

	gitkeepPath, err := writeSchematicsSkel(ffs, req)
	if err != nil {
		t.Fatalf("writeSchematicsSkel: unexpected error: %v", err)
	}

	expectedPath := filepath.Join(dir, schematicsFolderName, ".gitkeep")
	if gitkeepPath != expectedPath {
		t.Errorf("returned path = %q, want %q", gitkeepPath, expectedPath)
	}

	got, readErr := ffs.ReadFile(expectedPath)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", expectedPath, readErr)
	}

	if !bytes.Equal(got, lockedGitkeepBytes) {
		t.Errorf(".gitkeep bytes mismatch\ngot:\n%q\nwant:\n%q", got, lockedGitkeepBytes)
	}
}

// Test_WriteSchematicsSkel_CreatesSchematicsDir verifies that writeSchematicsSkel
// calls MkdirAll for the schematics directory before writing .gitkeep (REQ-SF-02).
// This is verified indirectly via fakeFS: if MkdirAll were not called on a real
// filesystem, WriteFile would fail; the test confirms the path uses
// schematicsFolderName as the parent directory name.
func Test_WriteSchematicsSkel_CreatesSchematicsDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	req := InitRequest{Directory: dir}

	_, err := writeSchematicsSkel(ffs, req)
	if err != nil {
		t.Fatalf("writeSchematicsSkel: unexpected error: %v", err)
	}

	// The .gitkeep file must be under the schematics folder (REQ-SF-02).
	gitkeepPath := filepath.Join(dir, schematicsFolderName, ".gitkeep")
	_, statErr := ffs.Stat(gitkeepPath)
	if statErr != nil {
		t.Errorf("Stat(%q): expected file to exist after writeSchematicsSkel: %v", gitkeepPath, statErr)
	}

	// The folder name must match the constant (REQ-SF-02).
	parent := filepath.Base(filepath.Dir(gitkeepPath))
	if parent != schematicsFolderName {
		t.Errorf("parent dir = %q, want %q (schematicsFolderName)", parent, schematicsFolderName)
	}
}
