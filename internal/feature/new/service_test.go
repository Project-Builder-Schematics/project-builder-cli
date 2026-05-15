// Package newfeature — service_test.go covers the stub service methods.
//
// REQ coverage:
//   - REQ-EC-07: RegisterSchematic + RegisterCollection both return ErrCodeNewNotImplemented.
//
// These stubs are replaced with real logic in S-001 (schematic) and S-004 (collection).
// The test confirms the stub contract so downstream handlers can safely call the service.
package newfeature_test

import (
	"context"
	"errors"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// Test_Service_RegisterSchematic_ReturnsErrNewNotImplemented verifies that the
// S-000b stub returns ErrCodeNewNotImplemented (REQ-EC-07).
func Test_Service_RegisterSchematic_ReturnsErrNewNotImplemented(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewSchematicRequest{Name: "my-schematic", DryRun: true}

	_, err := svc.RegisterSchematic(context.Background(), req)
	if err == nil {
		t.Fatal("RegisterSchematic: expected ErrCodeNewNotImplemented, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNewNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("RegisterSchematic: errors.Is(ErrCodeNewNotImplemented) = false; got: %v", err)
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("RegisterSchematic: errors.As(*errs.Error) failed; got: %T", err)
	}
	if e.Op != "new.handler" {
		t.Errorf("RegisterSchematic error Op = %q; want %q", e.Op, "new.handler")
	}
}

// Test_Service_RegisterCollection_ReturnsErrNewNotImplemented verifies that the
// S-000b stub returns ErrCodeNewNotImplemented (REQ-EC-07).
func Test_Service_RegisterCollection_ReturnsErrNewNotImplemented(t *testing.T) {
	t.Parallel()

	svc := newfeature.NewService(fswriter.NewDryRunWriter())
	req := newfeature.NewCollectionRequest{Name: "my-collection", DryRun: true}

	_, err := svc.RegisterCollection(context.Background(), req)
	if err == nil {
		t.Fatal("RegisterCollection: expected ErrCodeNewNotImplemented, got nil")
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
