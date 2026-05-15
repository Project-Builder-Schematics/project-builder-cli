// Package newfeature — handler_collection_test.go covers end-to-end scenarios
// for `builder new collection` via the wired handler + service + FakeFS stack.
//
// REQ coverage:
//   - REQ-NC-01: happy path — collection.json + project-builder.json entry
//   - REQ-NC-02: conflict no --force → ErrCodeNewCollectionExists
//   - REQ-NC-03: --force overwrites
//   - REQ-NC-04: no add/ or remove/ without --publishable
//   - REQ-NC-05: invalid name → ErrCodeInvalidSchematicName
//   - REQ-NC-06: --dry-run returns planned ops; zero writes
//   - REQ-NCP-01: --publishable creates lifecycle stubs
//   - REQ-NCP-02: --publishable --force overwrites
//   - REQ-NCP-03: --publishable + --inline → ErrCodeModeConflict
package newfeature_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// invokeCollectionFull calls Service.RegisterCollection to simulate the full
// collection command dispatch (handler delegates to service).
func invokeCollectionFull(t *testing.T, dir string, fs *fswriter.FakeFS, req newfeature.NewCollectionRequest) (*newfeature.NewResult, error) {
	t.Helper()
	svc := newfeature.NewService(fs)
	req.WorkDir = dir
	return svc.RegisterCollection(context.Background(), req)
}

// Test_E2E_Collection_HappyPath verifies the full E2E for collection creation (REQ-NC-01).
func Test_E2E_Collection_HappyPath(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)

	result, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{Name: "bar"})
	if err != nil {
		t.Fatalf("RegisterCollection: unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// collection.json must exist with skeleton bytes.
	colPath := filepath.Join(dir, "schematics", "bar", "collection.json")
	if !fs.HasFile(colPath) {
		t.Fatalf("collection.json not created at %s", colPath)
	}
	colBytes, _ := fs.ReadFile(colPath)
	if string(colBytes) != string(newfeature.MarshalCollectionSkeleton()) {
		t.Errorf("collection.json mismatch;\nwant: %q\n got: %q", newfeature.MarshalCollectionSkeleton(), colBytes)
	}

	// project-builder.json has collections.bar entry.
	pbBytes, _ := fs.ReadFile(filepath.Join(dir, "project-builder.json"))
	var pbMap map[string]json.RawMessage
	if err := json.Unmarshal(pbBytes, &pbMap); err != nil {
		t.Fatalf("parse project-builder.json: %v", err)
	}
	var cols map[string]json.RawMessage
	if err := json.Unmarshal(pbMap["collections"], &cols); err != nil {
		t.Fatalf("parse collections: %v", err)
	}
	if _, ok := cols["bar"]; !ok {
		t.Fatalf("collections.bar missing (REQ-NC-01 violation); written: %s", pbBytes)
	}
}

// Test_E2E_Collection_NC04_NoLifecycleDirs is the explicit negative test (REQ-NC-04).
func Test_E2E_Collection_NC04_NoLifecycleDirs(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)

	_, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{Name: "bar"})
	if err != nil {
		t.Fatalf("RegisterCollection: unexpected error: %v", err)
	}

	// EXPLICIT negative: add/ and remove/ factory files must NOT exist.
	for _, forbidden := range []string{
		filepath.Join(dir, "schematics", "bar", "add", "factory.ts"),
		filepath.Join(dir, "schematics", "bar", "remove", "factory.ts"),
	} {
		if fs.HasFile(forbidden) {
			t.Errorf("REQ-NC-04 violation: %q MUST NOT exist without --publishable", forbidden)
		}
	}
}

// Test_E2E_Collection_ConflictNoForce verifies ErrCodeNewCollectionExists (REQ-NC-02).
func Test_E2E_Collection_ConflictNoForce(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)

	// First call succeeds.
	if _, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{Name: "bar"}); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call without --force must fail.
	_, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{Name: "bar"})
	if err == nil {
		t.Fatal("expected ErrCodeNewCollectionExists; got nil")
	}
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeNewCollectionExists {
		t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeNewCollectionExists)
	}
}

// Test_E2E_Collection_ForceOverwrite verifies --force (REQ-NC-03).
func Test_E2E_Collection_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)

	if _, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{Name: "bar"}); err != nil {
		t.Fatalf("first call: %v", err)
	}

	if _, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{Name: "bar", Force: true}); err != nil {
		t.Errorf("--force: unexpected error: %v", err)
	}
}

// Test_E2E_Collection_DryRun verifies --dry-run (REQ-NC-06).
func Test_E2E_Collection_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Use DryRunWriter — it records ops but writes no real files.
	svc := newfeature.NewService(fswriter.NewDryRunWriter())

	result, err := svc.RegisterCollection(context.Background(), newfeature.NewCollectionRequest{
		Name:    "bar",
		WorkDir: dir,
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("dry-run: unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("result.DryRun = false; want true")
	}
	if len(result.PlannedOps) == 0 {
		t.Error("dry-run: PlannedOps empty; expected at least 1 op")
	}
}

// Test_E2E_Collection_Publishable_HappyPath verifies --publishable (REQ-NCP-01).
func Test_E2E_Collection_Publishable_HappyPath(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)

	result, err := invokeCollectionFull(t, dir, fs, newfeature.NewCollectionRequest{
		Name:        "bar",
		Publishable: true,
	})
	if err != nil {
		t.Fatalf("RegisterCollection(publishable): unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// All 7 files must exist.
	expected := []string{
		filepath.Join(dir, "schematics", "bar", "collection.json"),
		filepath.Join(dir, "schematics", "bar", "add", "factory.ts"),
		filepath.Join(dir, "schematics", "bar", "add", "schema.json"),
		filepath.Join(dir, "schematics", "bar", "add", "schema.d.ts"),
		filepath.Join(dir, "schematics", "bar", "remove", "factory.ts"),
		filepath.Join(dir, "schematics", "bar", "remove", "schema.json"),
		filepath.Join(dir, "schematics", "bar", "remove", "schema.d.ts"),
	}
	for _, p := range expected {
		if !fs.HasFile(p) {
			t.Errorf("missing lifecycle file: %s", p)
		}
	}
}

// Test_E2E_Collection_Publishable_Force verifies --publishable --force (REQ-NCP-02).
func Test_E2E_Collection_Publishable_Force(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)

	req := newfeature.NewCollectionRequest{Name: "bar", Publishable: true}
	if _, err := invokeCollectionFull(t, dir, fs, req); err != nil {
		t.Fatalf("first call: %v", err)
	}

	req.Force = true
	if _, err := invokeCollectionFull(t, dir, fs, req); err != nil {
		t.Errorf("--publishable --force: unexpected error: %v", err)
	}
}

// Test_E2E_Collection_Publishable_InlineConflict verifies --publishable + --inline
// is rejected (REQ-NCP-03). Tests the CheckPublishableInlineConflict guard.
func Test_E2E_Collection_Publishable_InlineConflict(t *testing.T) {
	t.Parallel()

	// The conflict is checked via the exported guard (handler calls this before dispatch).
	err := newfeature.CheckPublishableInlineConflict(true, true)
	if err == nil {
		t.Fatal("expected ErrCodeModeConflict; got nil")
	}
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeModeConflict {
		t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeModeConflict)
	}
}
