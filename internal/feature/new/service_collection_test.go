// Package newfeature — service_collection_test.go covers Service.RegisterCollection
// (non-publishable path) via the service layer.
//
// REQ coverage:
//   - REQ-NC-01: happy path — collection.json skeleton + project-builder.json entry; NO add/remove dirs
//   - REQ-NC-02: conflict without --force → ErrCodeNewCollectionExists
//   - REQ-NC-03: --force overwrites existing collection
//   - REQ-NC-04: EXPLICIT negative — no add/ or remove/ dirs without --publishable
//   - REQ-NC-05: invalid name → ErrCodeInvalidSchematicName
//   - REQ-NC-06: --dry-run returns planned ops; zero FS writes
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

// setupCollectionWorkspace creates a temp dir with project-builder.json in a FakeFS.
// Reuses the same minimal JSON from handler_schematic_test.go but avoids import cycle
// by redefining the constant locally.
func setupCollectionWorkspace(t *testing.T) (string, *fswriter.FakeFS) {
	t.Helper()
	const minimalJSON = `{
  "$schema": "./node_modules/@pbuilder/sdk/schemas/project-builder.schema.json",
  "version": "1",
  "collections": {},
  "dependencies": {},
  "settings": {
    "autoInstall": true,
    "conflictPolicy": "child-wins",
    "depValidation": "dev"
  }
}
`
	dir := t.TempDir()
	fs := fswriter.NewFakeFS()
	pbPath := filepath.Join(dir, "project-builder.json")
	if err := fs.WriteFile(pbPath, []byte(minimalJSON), 0o644); err != nil {
		t.Fatalf("setup: WriteFile project-builder.json: %v", err)
	}
	return dir, fs
}

// invokeCollection is a test helper that calls Service.RegisterCollection.
func invokeCollection(t *testing.T, svc *newfeature.Service, req newfeature.NewCollectionRequest) (*newfeature.NewResult, error) {
	t.Helper()
	return svc.RegisterCollection(context.Background(), req)
}

// Test_Collection_HappyPath_CollectionJSONAndPBJSON verifies the happy path:
// collection.json skeleton created + project-builder.json entry added (REQ-NC-01).
func Test_Collection_HappyPath_CollectionJSONAndPBJSON(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := invokeCollection(t, svc, newfeature.NewCollectionRequest{
		Name:    "bar",
		WorkDir: dir,
	})
	if err != nil {
		t.Fatalf("RegisterCollection: unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// collection.json must exist at schematics/bar/collection.json.
	colJSONPath := filepath.Join(dir, "schematics", "bar", "collection.json")
	if !fs.HasFile(colJSONPath) {
		t.Fatalf("collection.json not created at %s", colJSONPath)
	}

	// collection.json must match canonical bytes (REQ-NC-01).
	colBytes, _ := fs.ReadFile(colJSONPath)
	want := string(newfeature.MarshalCollectionSkeleton())
	if string(colBytes) != want {
		t.Errorf("collection.json bytes mismatch:\nwant: %q\n got: %q", want, string(colBytes))
	}

	// project-builder.json must have collections.bar = {"path": "./schematics/bar/collection.json"}.
	pbBytes, _ := fs.ReadFile(filepath.Join(dir, "project-builder.json"))
	var pbMap map[string]json.RawMessage
	if err := json.Unmarshal(pbBytes, &pbMap); err != nil {
		t.Fatalf("parse project-builder.json: %v", err)
	}
	var cols map[string]json.RawMessage
	if err := json.Unmarshal(pbMap["collections"], &cols); err != nil {
		t.Fatalf("parse collections: %v", err)
	}
	barRaw, ok := cols["bar"]
	if !ok {
		t.Fatalf("collections.bar missing from project-builder.json (REQ-NC-01 violation); written: %s", pbBytes)
	}
	var barEntry map[string]string
	if err := json.Unmarshal(barRaw, &barEntry); err != nil {
		t.Fatalf("parse bar entry: %v", err)
	}
	if got := barEntry["path"]; got != "./schematics/bar/collection.json" {
		t.Errorf("collections.bar.path = %q; want %q", got, "./schematics/bar/collection.json")
	}
}

// Test_Collection_NC04_ExplicitNegative_NoLifecycleDirs asserts that without
// --publishable, NO add/ or remove/ directories are created (REQ-NC-04).
// This is an EXPLICIT test, not merely the absence of a positive.
func Test_Collection_NC04_ExplicitNegative_NoLifecycleDirs(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
	svc := newfeature.NewService(fs)

	_, err := invokeCollection(t, svc, newfeature.NewCollectionRequest{
		Name:    "bar",
		WorkDir: dir,
	})
	if err != nil {
		t.Fatalf("RegisterCollection: unexpected error: %v", err)
	}

	// EXPLICIT negative: add/ and remove/ must NOT exist (REQ-NC-04).
	addDir := filepath.Join(dir, "schematics", "bar", "add")
	removeDir := filepath.Join(dir, "schematics", "bar", "remove")
	addFactory := filepath.Join(addDir, "factory.ts")
	removeFactory := filepath.Join(removeDir, "factory.ts")

	for _, forbidden := range []string{addFactory, removeFactory} {
		if fs.HasFile(forbidden) {
			t.Errorf("REQ-NC-04 violation: lifecycle file %q MUST NOT exist without --publishable", forbidden)
		}
	}
}

// Test_Collection_ConflictNoForce verifies ErrCodeNewCollectionExists when
// collection already exists and --force is absent (REQ-NC-02).
func Test_Collection_ConflictNoForce(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewCollectionRequest{Name: "bar", WorkDir: dir}

	// First call succeeds.
	if _, err := invokeCollection(t, svc, req); err != nil {
		t.Fatalf("first call: %v", err)
	}
	countAfterFirst := fs.FileCount()

	// Second call without --force must fail.
	_, err := invokeCollection(t, svc, req)
	if err == nil {
		t.Fatal("second call: expected ErrCodeNewCollectionExists, got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeNewCollectionExists {
		t.Errorf("error code = %q; want %q", e.Code, errs.ErrCodeNewCollectionExists)
	}

	// No additional files written (REQ-NC-02 no-write guarantee).
	if fs.FileCount() != countAfterFirst {
		t.Errorf("file count changed on conflict (%d→%d)", countAfterFirst, fs.FileCount())
	}
}

// Test_Collection_ForceOverwrite verifies --force allows replacement (REQ-NC-03).
func Test_Collection_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewCollectionRequest{Name: "bar", WorkDir: dir}
	if _, err := invokeCollection(t, svc, req); err != nil {
		t.Fatalf("first call: %v", err)
	}

	req.Force = true
	if _, err := invokeCollection(t, svc, req); err != nil {
		t.Errorf("--force overwrite: unexpected error: %v", err)
	}
}

// Test_Collection_InvalidName verifies ErrCodeInvalidSchematicName for bad names (REQ-NC-05).
func Test_Collection_InvalidName(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
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
			_, err := invokeCollection(t, svc, newfeature.NewCollectionRequest{
				Name: tc.name, WorkDir: dir,
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

// Test_Collection_DryRun verifies --dry-run returns planned ops with zero real writes (REQ-NC-06).
func Test_Collection_DryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dryFS := fswriter.NewDryRunWriter()
	svc := newfeature.NewService(dryFS)

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
		t.Error("dry-run: PlannedOps is empty; expected at least 1 op")
	}
}
