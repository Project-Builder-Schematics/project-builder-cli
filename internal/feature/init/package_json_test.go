// Package initialise — package_json_test.go tests the mutatePackageJSON function.
//
// # REQ coverage
//
//   - REQ-PM-01: adds @pbuilder/sdk ^1.0.0 to devDependencies; creates file if missing
//   - REQ-PM-02: additive only — never modifies a pre-existing @pbuilder/sdk entry
//   - REQ-PM-03: rewrites with 2-space indent + final newline unconditionally
//   - REQ-PM-04: atomic write via FSWriter.WriteFile
package initialise

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// --- helpers ---

// pkgJSONReq builds a minimal InitRequest pointing at dir.
func pkgJSONReq(dir string) InitRequest {
	return InitRequest{Directory: dir}
}

// --- Test cases ---

// Test_MutatePackageJSON_CreatesFile_IfMissing verifies that when no package.json
// exists, mutatePackageJSON creates a minimal file containing only the devDependencies
// entry for @pbuilder/sdk. (REQ-PM-01)
func Test_MutatePackageJSON_CreatesFile_IfMissing(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()

	path, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	wantPath := filepath.Join(dir, "package.json")
	if path != wantPath {
		t.Errorf("returned path = %q, want %q", path, wantPath)
	}

	data, readErr := ffs.ReadFile(wantPath)
	if readErr != nil {
		t.Fatalf("package.json not written: %v", readErr)
	}

	content := string(data)
	if !strings.Contains(content, `"@pbuilder/sdk": "^1.0.0"`) {
		t.Errorf("package.json missing @pbuilder/sdk entry\ngot:\n%s", content)
	}
}

// Test_MutatePackageJSON_AddsSDKDevDep_WhenAbsent verifies that when package.json
// exists but devDependencies does not contain @pbuilder/sdk, the entry is added.
// (REQ-PM-01)
func Test_MutatePackageJSON_AddsSDKDevDep_WhenAbsent(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	initial := []byte(`{"name":"my-project","devDependencies":{"typescript":"^5.0.0"}}`)
	if err := ffs.WriteFile(pkgPath, initial, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	if !strings.Contains(content, `"@pbuilder/sdk": "^1.0.0"`) {
		t.Errorf("package.json missing @pbuilder/sdk\ngot:\n%s", content)
	}
	if !strings.Contains(content, `"typescript": "^5.0.0"`) {
		t.Errorf("package.json missing pre-existing typescript entry\ngot:\n%s", content)
	}
}

// Test_MutatePackageJSON_CreatesDevDeps_IfMissing verifies that when the file
// exists but has no devDependencies key, it is created and the SDK entry added.
// (REQ-PM-01)
func Test_MutatePackageJSON_CreatesDevDeps_IfMissing(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	initial := []byte(`{"name":"no-dev-deps","version":"1.0.0"}`)
	if err := ffs.WriteFile(pkgPath, initial, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	if !strings.Contains(content, `"devDependencies"`) {
		t.Errorf("package.json missing devDependencies key\ngot:\n%s", content)
	}
	if !strings.Contains(content, `"@pbuilder/sdk": "^1.0.0"`) {
		t.Errorf("package.json missing @pbuilder/sdk\ngot:\n%s", content)
	}
}

// Test_MutatePackageJSON_PreservesExistingDeps verifies that dependencies other than
// the SDK are preserved in the output. (REQ-PM-02 — additive only)
func Test_MutatePackageJSON_PreservesExistingDeps(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	initial := []byte(`{
  "name": "my-project",
  "devDependencies": {
    "typescript": "^5.0.0",
    "eslint": "^8.0.0"
  },
  "dependencies": {
    "react": "^18.0.0"
  }
}`)
	if err := ffs.WriteFile(pkgPath, initial, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	for _, want := range []string{
		`"typescript": "^5.0.0"`,
		`"eslint": "^8.0.0"`,
		`"react": "^18.0.0"`,
		`"@pbuilder/sdk": "^1.0.0"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("package.json missing %q\ngot:\n%s", want, content)
		}
	}
}

// Test_MutatePackageJSON_PreservesUnknownTopLevelKeys verifies that unknown
// top-level keys (e.g. "scripts", "engines") appear in the output.
// (Forward-compat — top-level map[string]json.RawMessage preserves them.)
func Test_MutatePackageJSON_PreservesUnknownTopLevelKeys(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	initial := []byte(`{
  "name": "my-project",
  "scripts": {"build": "tsc"},
  "engines": {"node": ">=18"}
}`)
	if err := ffs.WriteFile(pkgPath, initial, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	for _, want := range []string{"scripts", "engines", "build", "tsc", "node", ">=18"} {
		if !strings.Contains(content, want) {
			t.Errorf("package.json missing %q after mutation\ngot:\n%s", want, content)
		}
	}
}

// Test_MutatePackageJSON_AlreadyPresent_NoVersionChange verifies that when
// @pbuilder/sdk already exists in devDependencies with any version, it is NOT
// modified. (REQ-PM-02 — additive only, never modifies pre-existing)
func Test_MutatePackageJSON_AlreadyPresent_NoVersionChange(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	// Pre-existing entry with a different version.
	initial := []byte(`{"devDependencies":{"@pbuilder/sdk":"^2.0.0","other":"^1.0.0"}}`)
	if err := ffs.WriteFile(pkgPath, initial, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	// The pre-existing ^2.0.0 must survive unchanged.
	if !strings.Contains(content, `"@pbuilder/sdk": "^2.0.0"`) {
		t.Errorf("@pbuilder/sdk version was changed; want ^2.0.0\ngot:\n%s", content)
	}
	// @pbuilder/sdk entry with ^1.0.0 must NOT be added (the key already exists).
	if strings.Contains(content, `"@pbuilder/sdk": "^1.0.0"`) {
		t.Errorf("@pbuilder/sdk ^1.0.0 must not be added when key already present\ngot:\n%s", content)
	}
}

// Test_MutatePackageJSON_MalformedJSON_ReturnsErrCodeInvalidInput_NoMutation
// verifies that malformed JSON causes an ErrCodeInvalidInput error and the file
// is not modified. (REQ-PM-02)
func Test_MutatePackageJSON_MalformedJSON_ReturnsErrCodeInvalidInput_NoMutation(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	malformed := []byte(`{name: "no-quotes"}`)
	if err := ffs.WriteFile(pkgPath, malformed, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) || e.Code != errs.ErrCodeInvalidInput {
		t.Errorf("expected ErrCodeInvalidInput; got: %v", err)
	} else {
		// Error message must name the file and state "not valid JSON" (REQ-PM-02 UX).
		msg := err.Error()
		if !strings.Contains(msg, "package.json") {
			t.Errorf("error message missing 'package.json': %q", msg)
		}
		if !strings.Contains(msg, "not valid JSON") {
			t.Errorf("error message missing 'not valid JSON': %q", msg)
		}
	}

	// File must not have been modified.
	got, _ := ffs.ReadFile(pkgPath)
	if string(got) != string(malformed) {
		t.Errorf("malformed file was modified; want unchanged:\ngot: %q\nwant: %q", string(got), string(malformed))
	}
}

// Test_MutatePackageJSON_Reformats2SpaceFinalNewline verifies that the output
// uses 2-space indentation and ends with a trailing newline. (REQ-PM-03)
func Test_MutatePackageJSON_Reformats2SpaceFinalNewline(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	// Minified input — no indentation at all.
	initial := []byte(`{"name":"compact","devDependencies":{}}`)
	if err := ffs.WriteFile(pkgPath, initial, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	// Must end with a trailing newline.
	if !strings.HasSuffix(content, "\n") {
		t.Errorf("package.json does not end with a trailing newline\ngot: %q", content)
	}

	// Must use 2-space indentation (at least one "  " occurrence before a key).
	if !strings.Contains(content, "\n  ") {
		t.Errorf("package.json does not use 2-space indentation\ngot:\n%s", content)
	}
}

// Test_MutatePackageJSON_VersionRange_ExactLiteral verifies the golden literal
// for the version range (REQ-PM-01 — exact lock: "^1.0.0").
// Any change to the version string breaks the version-range contract.
func Test_MutatePackageJSON_VersionRange_ExactLiteral(t *testing.T) {
	t.Parallel()

	ffs := newFakeFS()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	_, err := mutatePackageJSON(ffs, pkgJSONReq(dir))
	if err != nil {
		t.Fatalf("mutatePackageJSON: unexpected error: %v", err)
	}

	data, _ := ffs.ReadFile(pkgPath)
	content := string(data)

	// Golden literal: must be "^1.0.0" — not "latest", ">=1.0.0", or "1.0.0".
	const wantRange = `"@pbuilder/sdk": "^1.0.0"`
	if !strings.Contains(content, wantRange) {
		t.Errorf("version range mismatch; want exact %q\ngot:\n%s", wantRange, content)
	}
}
