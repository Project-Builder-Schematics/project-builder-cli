// Package newfeature — service_publishable.go implements publishable collection
// scaffolding for `builder new collection <name> --publishable`.
//
// REQ coverage:
//   - REQ-NCP-01: lifecycle stubs generated (add/ + remove/ with factory.ts + schema.json + schema.d.ts)
//   - REQ-NCP-02: --force allows overwrite of existing publishable collection
//   - REQ-NCP-03: --publishable + --inline → ErrCodeModeConflict
package newfeature

import (
	"context"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// CheckPublishableInlineConflict returns ErrCodeModeConflict when publishable and
// inline flags are both true (REQ-NCP-03 / REQ-EC-05 mode-conflict guard).
//
// Called by handler_collection.go before dispatching to RegisterCollection.
func CheckPublishableInlineConflict(publishable, inline bool) error {
	if publishable && inline {
		return &errs.Error{
			Code:    errs.ErrCodeModeConflict,
			Op:      "new.preflight",
			Message: "--publishable cannot be combined with --inline; publishable collections require file-system layout.",
			Suggestions: []string{
				"omit --inline to create a publishable collection with lifecycle stubs",
				"omit --publishable to create an inline collection (no lifecycle stubs)",
			},
		}
	}
	return nil
}

// createPublishableCollection implements the --publishable collection creation flow.
// Writes collection.json + add/ and remove/ lifecycle stubs, then mutates
// project-builder.json with the collection path.
//
// Lifecycle stubs per subdir (REQ-NCP-01):
//   - <subdir>/factory.ts  — lifecycle TS stub (via RenderLifecycleTemplate)
//   - <subdir>/schema.json — empty schema.json (via MarshalEmpty)
//   - <subdir>/schema.d.ts — TS codegen for empty schema (via GenerateDTS)
func (s *Service) createPublishableCollection(ctx context.Context, req NewCollectionRequest) (*NewResult, error) {
	_ = ctx // reserved for future cancellation

	// 1. Validate name (reuses schematic name validator per spec REQ-NC-05 note).
	if err := validateSchematicName(req.Name); err != nil {
		return nil, err
	}

	// 2. Dry-run: record planned ops and return (no writes).
	if req.DryRun {
		return &NewResult{
			DryRun: true,
			PlannedOps: []PlannedOp{
				{Op: "create_file", Path: "schematics/" + req.Name + "/collection.json"},
				{Op: "create_file", Path: "schematics/" + req.Name + "/add/factory.ts"},
				{Op: "create_file", Path: "schematics/" + req.Name + "/add/schema.json"},
				{Op: "create_file", Path: "schematics/" + req.Name + "/add/schema.d.ts"},
				{Op: "create_file", Path: "schematics/" + req.Name + "/remove/factory.ts"},
				{Op: "create_file", Path: "schematics/" + req.Name + "/remove/schema.json"},
				{Op: "create_file", Path: "schematics/" + req.Name + "/remove/schema.d.ts"},
			},
			CollectionName: req.Name,
		}, nil
	}

	// 3. Conflict check (REQ-NCP-02).
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		return nil, err
	}

	if CollectionExists(cfg, req.Name) && !req.Force {
		relPath := "./schematics/" + req.Name + "/collection.json"
		return nil, &errs.Error{
			Code:    errs.ErrCodeNewCollectionExists,
			Op:      "new.createPublishableCollection",
			Message: "collection '" + req.Name + "' already exists at " + relPath + "; use --force to overwrite",
			Suggestions: []string{
				"run with --force to overwrite the existing collection and lifecycle stubs",
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
			Op:      "new.createPublishableCollection",
			Message: "failed to create collection directory",
			Cause:   err,
		}
	}

	if err := s.fs.WriteFile(colJSONPath, MarshalCollectionSkeleton(), 0o644); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.createPublishableCollection",
			Message: "failed to write collection.json",
			Cause:   err,
		}
	}

	filesCreated := []string{colJSONPath}

	// 5. Write lifecycle stubs for each stage (REQ-NCP-01).
	for _, stage := range []string{"add", "remove"} {
		stageFiles, err := s.writeLifecycleStubs(req, stage)
		if err != nil {
			return nil, err
		}
		filesCreated = append(filesCreated, stageFiles...)
	}

	// 6. Mutate project-builder.json.
	relPath := filepath.ToSlash("./schematics/" + req.Name + "/collection.json")
	if err := RegisterCollection(cfg, req.Name, relPath); err != nil {
		return nil, err
	}
	if err := WriteConfig(req.WorkDir, cfg, s.fs); err != nil {
		return nil, err
	}

	pbPath := filepath.Join(req.WorkDir, "project-builder.json")
	filesCreated = append(filesCreated, pbPath)

	return &NewResult{
		CollectionName: req.Name,
		FilesCreated:   filesCreated,
	}, nil
}

// writeLifecycleStubs writes the three files for a single lifecycle stage
// (factory.ts + schema.json + schema.d.ts) under schematics/<name>/<stage>/.
// Returns the list of written absolute paths.
func (s *Service) writeLifecycleStubs(req NewCollectionRequest, stage string) ([]string, error) {
	stageDir := filepath.Join(req.WorkDir, "schematics", req.Name, stage)

	if err := s.fs.MkdirAll(stageDir, 0o755); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.writeLifecycleStubs",
			Message: "failed to create lifecycle directory for stage '" + stage + "'",
			Cause:   err,
		}
	}

	// factory.ts — lifecycle stub template.
	factoryPath := filepath.Join(stageDir, "factory.ts")
	factoryBytes, err := RenderLifecycleTemplate(stage, req.Name)
	if err != nil {
		return nil, err
	}
	if err := s.fs.WriteFile(factoryPath, factoryBytes, 0o644); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.writeLifecycleStubs",
			Message: "failed to write " + stage + "/factory.ts",
			Cause:   err,
		}
	}

	// schema.json — empty schema.json (REQ-SJ-03 canonical form).
	schemaPath := filepath.Join(stageDir, "schema.json")
	if err := s.fs.WriteFile(schemaPath, MarshalEmpty(), 0o644); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.writeLifecycleStubs",
			Message: "failed to write " + stage + "/schema.json",
			Cause:   err,
		}
	}

	// schema.d.ts — TS codegen for empty schema.
	dtsPath := filepath.Join(stageDir, "schema.d.ts")
	emptySchema := Schema{Inputs: map[string]InputSpec{}}
	// Interface name uses the stage as a suffix: e.g. "BarAddSchematicInputs".
	dtsName := req.Name + "-" + stage
	dtsBytes, err := GenerateDTS(dtsName, emptySchema)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.writeLifecycleStubs",
			Message: "failed to generate " + stage + "/schema.d.ts",
			Cause:   err,
		}
	}
	if err := s.fs.WriteFile(dtsPath, dtsBytes, 0o644); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.writeLifecycleStubs",
			Message: "failed to write " + stage + "/schema.d.ts",
			Cause:   err,
		}
	}

	return []string{factoryPath, schemaPath, dtsPath}, nil
}
