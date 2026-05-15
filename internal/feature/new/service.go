// Package newfeature — service.go contains Service, the orchestrator for
// the `builder new` feature.
//
// S-000b walking skeleton scope:
//   - RegisterSchematic and RegisterCollection return ErrCodeNewNotImplemented.
//   - Real logic for schematic path-mode lands in S-001.
//   - Real logic for collection lands in S-004.
//
// The stub sentinel ErrCodeNewNotImplemented (REQ-EC-07) will be REMOVED at
// archive time once all slices are complete.
package newfeature

import (
	"context"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// Service orchestrates the builder new outputs via FSWriter.
// Inject via NewService at composeApp (cmd/builder/main.go).
type Service struct {
	fs fswriter.FSWriter
}

// NewService constructs a Service with the given FSWriter dependency.
// In composeApp, pass fswriter.NewOSWriter() for production;
// pass fswriter.NewDryRunWriter() for dry-run mode.
func NewService(fs fswriter.FSWriter) *Service {
	return &Service{fs: fs}
}

// RegisterSchematic is the orchestration entry-point for `builder new schematic`.
//
// S-000b: dry-run returns an empty planned-ops result (exit 0) per REQ-NS-05 partial smoke.
// Non-dry-run returns ErrCodeNewNotImplemented. Real implementation lands in S-001.
func (s *Service) RegisterSchematic(_ context.Context, req NewSchematicRequest) (*NewResult, error) {
	if req.DryRun {
		return &NewResult{
			DryRun:        true,
			PlannedOps:    s.fs.PlannedOps(),
			SchematicName: req.Name,
		}, nil
	}
	return nil, notImplementedErr("new schematic")
}

// RegisterCollection is the orchestration entry-point for `builder new collection`.
//
// S-000b: dry-run returns an empty planned-ops result (exit 0) per REQ-NC-06.
// Non-dry-run returns ErrCodeNewNotImplemented. Real implementation lands in S-004.
func (s *Service) RegisterCollection(_ context.Context, req NewCollectionRequest) (*NewResult, error) {
	if req.DryRun {
		return &NewResult{
			DryRun:         true,
			PlannedOps:     s.fs.PlannedOps(),
			CollectionName: req.Name,
		}, nil
	}
	return nil, notImplementedErr("new collection")
}

// notImplementedErr returns the stub sentinel error for unimplemented subcommands.
// Op is "new.handler" — consistent with the handler origin convention.
func notImplementedErr(subcommand string) error {
	return &errs.Error{
		Code:    errs.ErrCodeNewNotImplemented,
		Op:      "new.handler",
		Message: subcommand + ": not yet implemented in this build",
		Suggestions: []string{
			"this feature is planned for a future slice (builder-new S-001..S-004)",
		},
	}
}
