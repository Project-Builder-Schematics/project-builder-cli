// Package newfeature — service.go contains Service, the orchestrator for
// the `builder new` feature.
//
// Service dispatches to mode-specific helpers in service_schematic_path.go,
// service_schematic_inline.go, service_collection.go, and service_publishable.go.
// Preflight guards (mode-conflict detection) run inside Service methods before dispatch.
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
// Preflight order (ADR-026):
//  1. Mode-conflict check — if --inline and schematic exists in path mode → REJECT (REQ-NS-07)
//  2. Dispatch to registerSchematicPath or registerSchematicInline based on req.Inline
//  3. Conflict check + force resolution happen inside each mode-specific method
func (s *Service) RegisterSchematic(ctx context.Context, req NewSchematicRequest) (*NewResult, error) {
	// Shared preflight: mode-conflict detection (ADR-026 / REQ-NS-07 / REQ-EC-05).
	// Runs BEFORE dispatch; checked even when --dry-run is set.
	if err := s.checkModeConflict(req); err != nil {
		return nil, err
	}

	if req.Inline {
		return s.registerSchematicInline(ctx, req)
	}
	return s.registerSchematicPath(ctx, req)
}

// checkModeConflict returns ErrCodeModeConflict when the user requests --inline
// but the schematic already exists in path mode (REQ-NS-07 / ADR-026 REJECT policy).
//
// The check is always performed — --force does NOT override mode conflicts.
// The error message names "builder remove" so the user knows the recovery path (REQ-EC-05).
func (s *Service) checkModeConflict(req NewSchematicRequest) error {
	if !req.Inline {
		return nil // only relevant for inline requests
	}

	// Read the project config to detect path-mode entry.
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		// If the config is missing or unreadable, no conflict possible yet.
		return nil
	}

	collection := collectionName(req.Collection)
	if !SchematicExistsInPathMode(cfg, collection, req.Name) {
		return nil // no conflict
	}

	return &errs.Error{
		Code:    errs.ErrCodeModeConflict,
		Op:      "new.preflight",
		Message: "schematic '" + req.Name + "' exists in path mode; cannot register inline. Run 'builder remove' first.",
		Suggestions: []string{
			"run 'builder remove schematic " + req.Name + "' to remove the path-mode entry, then re-run with --inline",
			"omit --inline to overwrite the existing path-mode schematic",
		},
	}
}

// RegisterCollection is the orchestration entry-point for `builder new collection`.
//
// Preflight order (ADR-026):
//  1. Mode-conflict: --publishable + --inline → REJECT (REQ-NCP-03 / REQ-EC-05)
//  2. Dispatch to createCollection or createPublishableCollection based on req.Publishable
func (s *Service) RegisterCollection(ctx context.Context, req NewCollectionRequest) (*NewResult, error) {
	// Preflight guard for --publishable+--inline is in the handler (handler_collection.go),
	// not here, because NewCollectionRequest has no Inline field. The handler calls
	// CheckPublishableInlineConflict before building the request (ADR-026).

	if req.Publishable {
		return s.createPublishableCollection(ctx, req)
	}
	return s.createCollection(ctx, req)
}
