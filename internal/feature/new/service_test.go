// Package newfeature — service_test.go covers service-level contracts.
//
// REQ coverage:
//   - REQ-NS-05: dry-run returns DryRun=true result
//   - REQ-NSI-01: inline mode now implemented (S-002) — stub sentinel test replaced
//   - REQ-NC-06: collection dry-run returns DryRun=true result (S-004 real impl)
package newfeature_test

import (
	"context"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// Test_Service_RegisterSchematic_DryRun_ReturnsResult verifies that dry-run mode
// returns a valid result with DryRun=true (REQ-NS-05).
func Test_Service_RegisterSchematic_DryRun_ReturnsResult(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewSchematicRequest{Name: "my-schematic", DryRun: true}

	result, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterSchematic(dry-run): unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RegisterSchematic(dry-run): result is nil")
	}
	if !result.DryRun {
		t.Errorf("RegisterSchematic(dry-run): result.DryRun = false; want true")
	}
}

// Test_Service_RegisterSchematic_Inline_DryRun_ReturnsResult verifies that
// --inline --dry-run returns a valid DryRun result (REQ-NSI-01, S-002 implemented).
// The stub sentinel test (ErrCodeNewNotImplemented) is replaced now that inline is live.
func Test_Service_RegisterSchematic_Inline_DryRun_ReturnsResult(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewSchematicRequest{Name: "my-schematic", Inline: true, DryRun: true}

	result, err := svc.RegisterSchematic(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterSchematic(inline, dry-run): unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RegisterSchematic(inline, dry-run): result is nil")
	}
	if !result.DryRun {
		t.Errorf("RegisterSchematic(inline, dry-run): result.DryRun = false; want true")
	}
}

// Test_Service_RegisterCollection_DryRun_ReturnsEmptyResult verifies that the
// S-000b dry-run path returns a valid result with DryRun=true (REQ-NC-06).
func Test_Service_RegisterCollection_DryRun_ReturnsEmptyResult(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewCollectionRequest{Name: "my-collection", DryRun: true}

	result, err := svc.RegisterCollection(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterCollection(dry-run): unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RegisterCollection(dry-run): result is nil")
	}
	if !result.DryRun {
		t.Errorf("RegisterCollection(dry-run): result.DryRun = false; want true")
	}
}

// Test_Service_RegisterCollection_DryRun_HasPlannedOps verifies that the S-004 real
// implementation returns DryRun=true with non-empty PlannedOps in dry-run mode (REQ-NC-06).
// Replaces the S-000b stub sentinel test (ErrCodeNewNotImplemented removed in S-004).
func Test_Service_RegisterCollection_DryRun_HasPlannedOps(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewCollectionRequest{Name: "my-collection", DryRun: true}

	result, err := svc.RegisterCollection(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterCollection(dry-run): unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("RegisterCollection(dry-run): result is nil")
	}
	if !result.DryRun {
		t.Errorf("RegisterCollection(dry-run): result.DryRun = false; want true")
	}
	if len(result.PlannedOps) == 0 {
		t.Error("RegisterCollection(dry-run): PlannedOps empty; expected at least 1 op")
	}
}
