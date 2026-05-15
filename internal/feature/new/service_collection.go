// Package newfeature — service_collection.go implements collection scaffolding
// for `builder new collection <name>` (without --publishable).
//
// REQ coverage:
//   - REQ-NC-01: happy path — collection.json skeleton + project-builder.json entry
//   - REQ-NC-02: conflict without --force → ErrCodeNewCollectionExists
//   - REQ-NC-03: --force allows overwrite
//   - REQ-NC-04: NO add/ or remove/ dirs without --publishable (enforced here by omission)
//   - REQ-NC-05: name validation → ErrCodeInvalidSchematicName (reuses schematic validator)
//   - REQ-NC-06: --dry-run returns planned ops; zero writes
package newfeature

import (
	"context"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// createCollection implements the non-publishable collection creation flow.
// It writes collection.json (skeleton) and mutates project-builder.json.
// No lifecycle stubs (add/ / remove/) are created here — that belongs to
// createPublishableCollection (REQ-NC-04 enforcement by omission).
func (s *Service) createCollection(ctx context.Context, req NewCollectionRequest) (*NewResult, error) {
	_ = ctx // reserved for future cancellation

	// 1. Validate name (REQ-NC-05 — reuses schematic name validator per spec).
	if err := validateSchematicName(req.Name); err != nil {
		return nil, err
	}

	// 2. Dry-run: record planned ops and return (no writes; skip conflict check).
	if req.DryRun {
		_ = s.fs.MkdirAll(filepath.Join(req.WorkDir, "schematics", req.Name), 0o755)
		return &NewResult{
			DryRun:         true,
			PlannedOps:     []PlannedOp{{Op: "create_file", Path: "schematics/" + req.Name + "/collection.json"}},
			CollectionName: req.Name,
		}, nil
	}

	// 3. Conflict check (REQ-NC-02 / REQ-NC-03).
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		return nil, err
	}

	if CollectionExists(cfg, req.Name) && !req.Force {
		relPath := "./schematics/" + req.Name + "/collection.json"
		return nil, &errs.Error{
			Code:    errs.ErrCodeNewCollectionExists,
			Op:      "new.createCollection",
			Message: "collection '" + req.Name + "' already exists at " + relPath + "; use --force to overwrite",
			Suggestions: []string{
				"run with --force to overwrite the existing collection",
				"choose a different name",
			},
		}
	}

	// 4. Write collection.json skeleton.
	collectionDir := filepath.Join(req.WorkDir, "schematics", req.Name)
	colJSONPath := filepath.Join(collectionDir, "collection.json")

	if err := s.fs.MkdirAll(collectionDir, 0o755); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.createCollection",
			Message: "failed to create collection directory",
			Cause:   err,
		}
	}

	if err := s.fs.WriteFile(colJSONPath, MarshalCollectionSkeleton(), 0o644); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.createCollection",
			Message: "failed to write collection.json",
			Cause:   err,
		}
	}

	// 5. Mutate project-builder.json (REQ-NC-01).
	relPath := filepath.ToSlash("./schematics/" + req.Name + "/collection.json")
	if err := RegisterCollection(cfg, req.Name, relPath); err != nil {
		return nil, err
	}
	if err := WriteConfig(req.WorkDir, cfg, s.fs); err != nil {
		return nil, err
	}

	pbPath := filepath.Join(req.WorkDir, "project-builder.json")
	return &NewResult{
		CollectionName: req.Name,
		FilesCreated:   []string{colJSONPath, pbPath},
	}, nil
}
