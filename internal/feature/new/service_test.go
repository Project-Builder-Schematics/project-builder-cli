// Package newfeature — service_test.go covers service-level contracts.
//
// REQ coverage (S-001 updates these from stub to real behaviour):
//   - REQ-NS-05: dry-run returns DryRun=true result
//   - REQ-EC-07: inline mode still returns ErrCodeNewNotImplemented (stub sentinel, S-002)
//   - REQ-EC-07: collection non-dry-run still returns ErrCodeNewNotImplemented (S-004)
package newfeature_test

import (
	"context"
	"errors"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
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

// Test_Service_RegisterSchematic_Inline_ReturnsErrNotImplemented verifies that
// --inline still returns ErrCodeNewNotImplemented (S-002 stub, REQ-EC-07).
func Test_Service_RegisterSchematic_Inline_ReturnsErrNotImplemented(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewSchematicRequest{Name: "my-schematic", Inline: true}

	_, err := svc.RegisterSchematic(context.Background(), req)
	if err == nil {
		t.Fatal("RegisterSchematic(inline): expected ErrCodeNewNotImplemented, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNewNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("RegisterSchematic(inline): errors.Is(ErrCodeNewNotImplemented) = false; got: %v", err)
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

// Test_Service_RegisterCollection_NonDryRun_ReturnsErrNewNotImplemented verifies
// that the S-000b stub returns ErrCodeNewNotImplemented for non-dry-run (REQ-EC-07).
func Test_Service_RegisterCollection_NonDryRun_ReturnsErrNewNotImplemented(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewCollectionRequest{Name: "my-collection", DryRun: false}

	_, err := svc.RegisterCollection(context.Background(), req)
	if err == nil {
		t.Fatal("RegisterCollection(non-dry-run): expected ErrCodeNewNotImplemented, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNewNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("RegisterCollection: errors.Is(ErrCodeNewNotImplemented) = false; got: %v", err)
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("RegisterCollection: errors.As(*errs.Error) failed; got: %T", err)
	}
	if e.Op != "new.handler" {
		t.Errorf("RegisterCollection error Op = %q; want %q", e.Op, "new.handler")
	}
}
