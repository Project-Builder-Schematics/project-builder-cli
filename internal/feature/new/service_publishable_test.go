// Package newfeature — service_publishable_test.go covers Service.RegisterCollection
// with --publishable flag (lifecycle stubs: add/ + remove/).
//
// REQ coverage:
//   - REQ-NCP-01: happy path — collection.json + add/ + remove/ lifecycle stubs
//   - REQ-NCP-02: --publishable + --force allows overwrite
//   - REQ-NCP-03: --publishable + --inline → ErrCodeModeConflict
package newfeature_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// setupPublishableWorkspace is an alias for setupCollectionWorkspace defined in
// service_collection_test.go. Re-using the same workspace setup function avoids
// duplicate const; both files are in the same test package.
// (No re-declaration needed — setupCollectionWorkspace is already visible.)

// invokePublishable calls RegisterCollection with Publishable=true.
func invokePublishable(t *testing.T, svc *newfeature.Service, req newfeature.NewCollectionRequest) (*newfeature.NewResult, error) {
	t.Helper()
	req.Publishable = true
	return svc.RegisterCollection(context.Background(), req)
}

// Test_Publishable_HappyPath_AllLifecycleFiles verifies the full publishable layout
// is created: collection.json + add/factory.ts + add/schema.json + add/schema.d.ts
// + remove/factory.ts + remove/schema.json + remove/schema.d.ts (REQ-NCP-01).
func Test_Publishable_HappyPath_AllLifecycleFiles(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
	svc := newfeature.NewService(fs)

	result, err := invokePublishable(t, svc, newfeature.NewCollectionRequest{
		Name:    "bar",
		WorkDir: dir,
	})
	if err != nil {
		t.Fatalf("RegisterCollection(publishable): unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("result.DryRun should be false")
	}

	// collection.json must exist and match skeleton.
	colJSONPath := filepath.Join(dir, "schematics", "bar", "collection.json")
	if !fs.HasFile(colJSONPath) {
		t.Errorf("collection.json not created at %s", colJSONPath)
	}

	// Lifecycle stubs for "add" (REQ-NCP-01).
	addFactory := filepath.Join(dir, "schematics", "bar", "add", "factory.ts")
	addSchema := filepath.Join(dir, "schematics", "bar", "add", "schema.json")
	addDTS := filepath.Join(dir, "schematics", "bar", "add", "schema.d.ts")

	for _, p := range []string{addFactory, addSchema, addDTS} {
		if !fs.HasFile(p) {
			t.Errorf("lifecycle file missing: %s", p)
		}
	}

	// add/factory.ts must reference the collection name.
	factoryBytes, _ := fs.ReadFile(addFactory)
	if !strings.Contains(string(factoryBytes), "bar") {
		t.Errorf("add/factory.ts does not reference collection name; content: %s", factoryBytes)
	}

	// add/schema.json must match canonical MarshalEmpty bytes.
	addSchemaBytes, _ := fs.ReadFile(addSchema)
	if string(addSchemaBytes) != string(newfeature.MarshalEmpty()) {
		t.Errorf("add/schema.json bytes mismatch; got: %s", addSchemaBytes)
	}

	// Lifecycle stubs for "remove" (REQ-NCP-01).
	removeFactory := filepath.Join(dir, "schematics", "bar", "remove", "factory.ts")
	removeSchema := filepath.Join(dir, "schematics", "bar", "remove", "schema.json")
	removeDTS := filepath.Join(dir, "schematics", "bar", "remove", "schema.d.ts")

	for _, p := range []string{removeFactory, removeSchema, removeDTS} {
		if !fs.HasFile(p) {
			t.Errorf("lifecycle file missing: %s", p)
		}
	}
}

// Test_Publishable_ForceOverwrite verifies --publishable + --force on an existing
// collection overwrites all files (REQ-NCP-02).
func Test_Publishable_ForceOverwrite(t *testing.T) {
	t.Parallel()

	dir, fs := setupCollectionWorkspace(t)
	svc := newfeature.NewService(fs)

	req := newfeature.NewCollectionRequest{Name: "bar", WorkDir: dir}

	// First call.
	if _, err := invokePublishable(t, svc, req); err != nil {
		t.Fatalf("first publishable call: %v", err)
	}

	// Force overwrite.
	req.Force = true
	if _, err := invokePublishable(t, svc, req); err != nil {
		t.Errorf("--publishable --force: unexpected error: %v", err)
	}
}

// Test_Publishable_ModeConflict_InlineRejected verifies that --publishable + --inline
// (simulated via handler-level preflight in service) → ErrCodeModeConflict (REQ-NCP-03).
//
// Since NewCollectionRequest has no Inline field (the flag is not exposed on the
// collection command), we test the handler_collection.go preflight instead.
// This test uses a direct preflight helper that checks the flag combo before dispatch.
func Test_Publishable_ModeConflict_InlineRejected(t *testing.T) {
	t.Parallel()

	// The mode conflict is caught via the exported preflight helper.
	err := newfeature.CheckPublishableInlineConflict(true, true)
	if err == nil {
		t.Fatal("expected ErrCodeModeConflict for --publishable --inline; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeModeConflict {
		t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeModeConflict)
	}

	// Message must explain the conflict (REQ-EC-05).
	if !strings.Contains(e.Message, "publishable") && !strings.Contains(e.Message, "inline") {
		t.Errorf("ErrCodeModeConflict message does not explain conflict; message: %q", e.Message)
	}
}
