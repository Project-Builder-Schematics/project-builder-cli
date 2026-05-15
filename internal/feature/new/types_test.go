// Package newfeature — types_test.go covers the core request/result/op shapes
// for the builder new feature.
//
// REQ coverage: types structural smoke (zero-value correctness).
// The build failure (undefined package) is the RED state; types.go is the GREEN.
package newfeature_test

import (
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// compile-time alias assertion: newfeature.PlannedOp must be identical to
// fswriter.PlannedOp (type alias, not a distinct defined type). If this breaks,
// a type conversion would be required here and the build would fail.
var _ newfeature.PlannedOp = fswriter.PlannedOp{}

// Test_NewSchematicRequest_ZeroValue verifies the struct exists with expected fields.
func Test_NewSchematicRequest_ZeroValue(t *testing.T) {
	t.Parallel()

	var req newfeature.NewSchematicRequest
	// Name field must be a string (zero value is empty string).
	if req.Name != "" {
		t.Errorf("NewSchematicRequest.Name zero value = %q; want empty string", req.Name)
	}
	// DryRun field must be a bool (zero value is false).
	if req.DryRun {
		t.Errorf("NewSchematicRequest.DryRun zero value = true; want false")
	}
	// Force field must be a bool (zero value is false).
	if req.Force {
		t.Errorf("NewSchematicRequest.Force zero value = true; want false")
	}
}

// Test_NewCollectionRequest_ZeroValue verifies the struct exists with expected fields.
func Test_NewCollectionRequest_ZeroValue(t *testing.T) {
	t.Parallel()

	var req newfeature.NewCollectionRequest
	if req.Name != "" {
		t.Errorf("NewCollectionRequest.Name zero value = %q; want empty string", req.Name)
	}
	if req.DryRun {
		t.Errorf("NewCollectionRequest.DryRun zero value = true; want false")
	}
	if req.Force {
		t.Errorf("NewCollectionRequest.Force zero value = true; want false")
	}
}

// Test_NewResult_ZeroValue verifies the result struct exists and has expected fields.
func Test_NewResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var result newfeature.NewResult
	if result.DryRun {
		t.Errorf("NewResult.DryRun zero value = true; want false")
	}
	if result.PlannedOps != nil {
		t.Errorf("NewResult.PlannedOps zero value != nil; want nil")
	}
}

// Test_PlannedOp_IsAlias_ForFswriterPlannedOp verifies PlannedOp is type-compatible
// with fswriter.PlannedOp (type alias per design).
//
// The alias property is guaranteed at compile time: because PlannedOp = fswriter.PlannedOp
// (type alias, not a new defined type), the file will not compile if they diverge.
// This test documents the intent and exercises field access.
func Test_PlannedOp_IsAlias_ForFswriterPlannedOp(t *testing.T) {
	t.Parallel()

	// Direct construction via fswriter.PlannedOp — no conversion required.
	op := fswriter.PlannedOp{Op: "create_file", Path: "/tmp/test"}
	if op.Op != "create_file" {
		t.Errorf("PlannedOp.Op = %q; want %q", op.Op, "create_file")
	}
	if op.Path != "/tmp/test" {
		t.Errorf("PlannedOp.Path = %q; want %q", op.Path, "/tmp/test")
	}
}
