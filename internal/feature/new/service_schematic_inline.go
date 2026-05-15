// Package newfeature — service_schematic_inline.go implements inline-mode schematic
// registration: embeds schematic definition directly in project-builder.json.
//
// REQ coverage:
//   - REQ-NSI-01: happy path inline — no files created; project-builder.json entry
//   - REQ-NSI-02: conflict without --force → ErrCodeNewSchematicExists
//   - REQ-NSI-03: --force overwrites existing inline entry
//   - REQ-NSI-04: soft warning at 10th inline schematic (via Renderer)
//   - REQ-NSI-05: soft warning when project-builder.json exceeds 20KB (via Renderer)
//   - REQ-PJ-06: inline entry shape {"inputs": {}}
package newfeature

import (
	"context"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// registerSchematicInline implements the inline-mode schematic creation flow.
// No files are created under schematics/ — the entry is embedded in project-builder.json.
func (s *Service) registerSchematicInline(ctx context.Context, req NewSchematicRequest) (*NewResult, error) {
	_ = ctx // reserved for future cancellation

	// 1. Validate name (REQ-NS-04).
	if err := validateSchematicName(req.Name); err != nil {
		return nil, err
	}

	collection := collectionName(req.Collection)

	// 2. Dry-run: record planned op and return (no writes; skip conflict check).
	if req.DryRun {
		_ = s.fs.PlannedOps() // trigger the dry-run recording
		return &NewResult{
			DryRun:        true,
			PlannedOps:    []PlannedOp{{Op: "mutate_json", Path: "project-builder.json"}},
			SchematicName: req.Name,
		}, nil
	}

	// 3. Conflict check (REQ-NSI-02 / REQ-NSI-03).
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		return nil, err
	}

	if SchematicExists(cfg, collection, req.Name) && !req.Force {
		return nil, &errs.Error{
			Code:    errs.ErrCodeNewSchematicExists,
			Op:      "new.registerSchematicInline",
			Message: "schematic '" + req.Name + "' already exists; use --force to overwrite",
			Suggestions: []string{
				"run with --force to overwrite the existing inline schematic",
				"choose a different name",
			},
		}
	}

	// 4. Register inline entry in config.
	if err := RegisterSchematicInline(cfg, collection, req.Name); err != nil {
		return nil, err
	}

	// 5. Write project-builder.json.
	if err := WriteConfig(req.WorkDir, cfg, s.fs); err != nil {
		return nil, err
	}

	result := &NewResult{
		SchematicName: req.Name,
		FilesCreated:  []string{},
	}

	// 6. Soft warning: 10th inline schematic threshold (REQ-NSI-04).
	inlineCount := CountInlineSchematics(cfg, collection)
	if inlineCount >= InlineSchematicThreshold {
		result.Warnings = append(result.Warnings, WarnApproachingSchematicLimit(collection, inlineCount))
	}

	// 7. Soft warning: file size threshold (REQ-NSI-05).
	pbPath := filepath.Join(req.WorkDir, "project-builder.json")
	if data, readErr := s.fs.ReadFile(pbPath); readErr == nil {
		if len(data) >= FileSizeThresholdBytes {
			result.Warnings = append(result.Warnings, WarnApproachingFileSize(len(data)))
		}
	}

	return result, nil
}
