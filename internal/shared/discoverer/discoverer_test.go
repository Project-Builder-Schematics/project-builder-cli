// Package discoverer_test covers the Discoverer struct.
//
// S-000 scope: FindNode() via NODE_BINARY env var only.
// S-004 scope:
//   - REQ-10.1: Node found on PATH accepted when version >= 18
//   - REQ-10.2: Node found on PATH rejected when version < 18
//   - REQ-10.3: Node not found → ErrCodeEngineNotFound
//   - REQ-11.1: project-local schematics preferred over PATH
//   - REQ-11.2: schematics not found → ErrCodeEngineNotFound with install hint
//   - REQ-11.3: schematics version below 17.0.0 rejected
//   - REQ-16.1: Op format on all error paths matches ^angular\.[a-z][a-z0-9_]*$
package discoverer_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/discoverer"
	appErrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// opRegex is the invariant format for the Op field (REQ-16.1).
var opRegex = regexp.MustCompile(`^angular\.[a-z][a-z0-9_]*$`)

// Test_FindNode_NodeBinary_Set covers the S-000 skeleton path: when NODE_BINARY
// is set to a real executable, FindNode returns that path with nil error.
func Test_FindNode_NodeBinary_Set(t *testing.T) {
	// t.Setenv requires no t.Parallel — it mutates process env and Go 1.21+
	// panics if t.Parallel and t.Setenv are combined.

	bin := fakeVersionBin(t, "v20.11.0")
	t.Setenv("NODE_BINARY", bin)

	d := discoverer.New()
	got, err := d.FindNode()
	if err != nil {
		t.Fatalf("FindNode() returned unexpected error: %v", err)
	}
	if got == "" {
		t.Fatal("FindNode() returned empty path")
	}
}

// Test_FindNode_NodeBinary_Empty_ReturnsError covers the skeleton: when
// NODE_BINARY is not set and no full discovery chain exists yet (S-004),
// FindNode returns a structured *errors.Error with ErrCodeEngineNotFound.
//
// This test unsets NODE_BINARY to isolate the skeleton error path.
// The full priority chain (PATH lookup + well-known paths) is wired in S-004.
func Test_FindNode_NodeBinary_Empty_ReturnsError(t *testing.T) {
	// t.Setenv requires no t.Parallel.

	// Unset NODE_BINARY to trigger the not-found path in the skeleton.
	t.Setenv("NODE_BINARY", "")

	// Also set PATH to empty so exec.LookPath("node") fails.
	t.Setenv("PATH", "")

	// Neutralise well-known paths — CI runners pre-install node at /usr/bin/node.
	discoverer.SetWellKnownNodePathsFn(t, func() []string { return nil })

	d := discoverer.New()
	_, err := d.FindNode()
	if err == nil {
		t.Fatal("FindNode() expected non-nil error when NODE_BINARY is empty and node not on PATH")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("FindNode() error is %T, want *errors.Error; got: %v", err, err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("FindNode() error code = %q, want %q", appErr.Code, appErrors.ErrCodeEngineNotFound)
	}
}

// Test_FindNode_PathLookup_VersionOK covers REQ-10.1:
// node on PATH with version >= 18 → path returned.
func Test_FindNode_PathLookup_VersionOK(t *testing.T) {
	// Create a fake node binary that reports v20.11.0.
	bin := fakeVersionBin(t, "v20.11.0")
	dir := filepath.Dir(bin)

	t.Setenv("NODE_BINARY", "") // disable NODE_BINARY override
	t.Setenv("PATH", dir)       // ensure fake node is on PATH

	d := discoverer.New()
	got, err := d.FindNode()
	if err != nil {
		t.Fatalf("FindNode() error: %v — REQ-10.1 violated", err)
	}
	if got == "" {
		t.Error("FindNode() returned empty path — REQ-10.1 violated")
	}
}

// Test_FindNode_PathLookup_VersionTooLow covers REQ-10.2:
// node on PATH with version < 18 → ErrCodeEngineNotFound.
func Test_FindNode_PathLookup_VersionTooLow(t *testing.T) {
	bin := fakeVersionBin(t, "v16.20.0")
	dir := filepath.Dir(bin)

	t.Setenv("NODE_BINARY", "")
	t.Setenv("PATH", dir)

	d := discoverer.New()
	_, err := d.FindNode()
	if err == nil {
		t.Fatal("FindNode() expected error for Node v16 — REQ-10.2 violated")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("error is %T, want *errors.Error", err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("Code = %q, want ErrCodeEngineNotFound — REQ-10.2", appErr.Code)
	}
	if !opRegex.MatchString(appErr.Op) {
		t.Errorf("Op = %q does not match %s — REQ-16.1", appErr.Op, opRegex)
	}
}

// Test_FindNode_NotFound covers REQ-10.3:
// node not on PATH and no NODE_BINARY → ErrCodeEngineNotFound.
func Test_FindNode_NotFound(t *testing.T) {
	t.Setenv("NODE_BINARY", "")
	t.Setenv("PATH", t.TempDir()) // dir with no node binary

	// Neutralise well-known paths — CI runners pre-install node at /usr/bin/node.
	discoverer.SetWellKnownNodePathsFn(t, func() []string { return nil })

	d := discoverer.New()
	_, err := d.FindNode()
	if err == nil {
		t.Fatal("FindNode() expected error when node not found — REQ-10.3 violated")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("error is %T, want *errors.Error", err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("Code = %q, want ErrCodeEngineNotFound — REQ-10.3", appErr.Code)
	}
}

// Test_FindSchematics_LocalPreferred covers REQ-11.1:
// {workspace}/node_modules/.bin/schematics preferred over PATH version.
func Test_FindSchematics_LocalPreferred(t *testing.T) {
	workspace := t.TempDir()

	// Create a local schematics binary in node_modules/.bin/.
	localBin := fakeSchematicsVersionBin(t, workspace, "17.3.0")
	_ = localBin

	// Create a different schematics binary on PATH.
	pathBin := fakeVersionBin(t, "17.3.0")
	pathDir := filepath.Dir(pathBin)
	// Rename to "schematics" in the PATH dir.
	schematicsPathBin := filepath.Join(pathDir, "schematics")
	if err := os.Rename(pathBin, schematicsPathBin); err != nil {
		t.Fatalf("failed to rename path bin: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(schematicsPathBin) })

	t.Setenv("PATH", pathDir)

	d := discoverer.New()
	got, err := d.FindSchematics(workspace)
	if err != nil {
		t.Fatalf("FindSchematics() error: %v — REQ-11.1 violated", err)
	}

	// Must return the local path, not the PATH one.
	expectedLocal := filepath.Join(workspace, "node_modules", ".bin", "schematics")
	if got != expectedLocal {
		t.Errorf("FindSchematics() = %q, want local %q — REQ-11.1 violated", got, expectedLocal)
	}
}

// Test_FindSchematics_NotFound covers REQ-11.2:
// neither local nor PATH schematics exists → ErrCodeEngineNotFound.
func Test_FindSchematics_NotFound(t *testing.T) {
	workspace := t.TempDir() // no node_modules/.bin/schematics

	t.Setenv("PATH", t.TempDir()) // no schematics on PATH either

	d := discoverer.New()
	_, err := d.FindSchematics(workspace)
	if err == nil {
		t.Fatal("FindSchematics() expected error when schematics not found — REQ-11.2 violated")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("error is %T, want *errors.Error", err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("Code = %q, want ErrCodeEngineNotFound — REQ-11.2", appErr.Code)
	}
	if len(appErr.Suggestions) == 0 {
		t.Error("Suggestions is empty — REQ-11.2 requires install hint")
	}
	if !opRegex.MatchString(appErr.Op) {
		t.Errorf("Op = %q does not match %s — REQ-16.1", appErr.Op, opRegex)
	}
}

// Test_FindSchematics_VersionTooLow covers REQ-11.3:
// schematics version < 17.0.0 → ErrCodeEngineNotFound.
func Test_FindSchematics_VersionTooLow(t *testing.T) {
	workspace := t.TempDir()
	_ = fakeSchematicsVersionBin(t, workspace, "15.2.0")

	d := discoverer.New()
	_, err := d.FindSchematics(workspace)
	if err == nil {
		t.Fatal("FindSchematics() expected error for version 15.2.0 — REQ-11.3 violated")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("error is %T, want *errors.Error", err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("Code = %q, want ErrCodeEngineNotFound — REQ-11.3", appErr.Code)
	}
}

// Test_FindNode_NodeBinary_Version_OK covers REQ-10.1 via NODE_BINARY:
// NODE_BINARY set to a fake binary that reports v18.0.0 → accepted.
func Test_FindNode_NodeBinary_Version_OK(t *testing.T) {
	bin := fakeVersionBin(t, "v18.0.0")
	t.Setenv("NODE_BINARY", bin)

	d := discoverer.New()
	got, err := d.FindNode()
	if err != nil {
		t.Fatalf("FindNode() error: %v — REQ-10.1 violated (v18 should be accepted)", err)
	}
	if got != bin {
		t.Errorf("FindNode() = %q, want %q", got, bin)
	}
}

// Test_FindNode_NodeBinary_Version_Rejected covers REQ-10.2 via NODE_BINARY:
// NODE_BINARY set to a fake binary that reports v17.9.0 → rejected.
func Test_FindNode_NodeBinary_Version_Rejected(t *testing.T) {
	bin := fakeVersionBin(t, "v17.9.0")
	t.Setenv("NODE_BINARY", bin)

	d := discoverer.New()
	_, err := d.FindNode()
	if err == nil {
		t.Fatal("FindNode() expected error for Node v17 — REQ-10.2 violated")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("error is %T, want *errors.Error", err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("Code = %q, want ErrCodeEngineNotFound — REQ-10.2", appErr.Code)
	}
}

// --- helpers ---

// fakeVersionBin creates a temp executable that, when run with --version,
// prints versionStr to stdout and exits 0. The file is cleaned up via t.Cleanup.
//
// The binary is a shell script on POSIX and a .cmd batch file on Windows.
// On POSIX, we create a shell script named "node" that just echoes the version.
func fakeVersionBin(t *testing.T, versionStr string) string {
	t.Helper()
	dir := t.TempDir()

	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.cmd"
	}
	binPath := filepath.Join(dir, name)

	var script string
	if runtime.GOOS == "windows" {
		script = fmt.Sprintf("@echo %s\r\n", versionStr)
	} else {
		script = fmt.Sprintf("#!/bin/sh\necho '%s'\n", versionStr)
	}

	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil { //nolint:gosec // test executable needs 0o755
		t.Fatalf("failed to write fake node binary: %v", err)
	}
	return binPath
}

// fakeSchematicsVersionBin creates a workspace node_modules/.bin/schematics
// executable that reports versionStr when run with --version.
//
//nolint:unparam // return value used by the caller that verifies local-preference
func fakeSchematicsVersionBin(t *testing.T, workspace, versionStr string) string {
	t.Helper()
	binDir := filepath.Join(workspace, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatalf("failed to create node_modules/.bin: %v", err)
	}

	name := "schematics"
	if runtime.GOOS == "windows" {
		name = "schematics.cmd"
	}
	binPath := filepath.Join(binDir, name)

	var script string
	if runtime.GOOS == "windows" {
		script = fmt.Sprintf("@echo %s\r\n", versionStr)
	} else {
		script = fmt.Sprintf("#!/bin/sh\necho '%s'\n", versionStr)
	}

	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil { //nolint:gosec // test executable needs 0o755
		t.Fatalf("failed to write fake schematics binary: %v", err)
	}
	return binPath
}

// asError is a local helper that mimics errors.As for *appErrors.Error,
// avoiding an import of the standard "errors" package name clash.
func asError(err error, target **appErrors.Error) bool {
	for err != nil {
		if e, ok := err.(*appErrors.Error); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}
