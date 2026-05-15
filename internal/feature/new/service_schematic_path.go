// Package newfeature — service_schematic_path.go implements path-mode schematic
// scaffolding: factory.{ts|js} + schema.json + project-builder.json mutation.
//
// REQ coverage:
//   - REQ-NS-01: happy path TS — two files + JSON entry
//   - REQ-NS-02: conflict without --force → ErrCodeNewSchematicExists
//   - REQ-NS-03: --force overwrites existing schematic
//   - REQ-NS-04: name validation → ErrCodeInvalidSchematicName
//   - REQ-NS-05: --dry-run returns planned ops; zero FS writes
//   - REQ-NS-06: --language=js → factory.js
//   - REQ-PJ-01..08: project-builder.json mutation via projectconfig
//   - REQ-SJ-01/03/05: schema.json canonical bytes via schema.MarshalEmpty
package newfeature

import (
	"context"
	"path/filepath"
	"strings"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/validate"
)

// registerSchematicPath implements the path-mode schematic creation flow.
// Complexity is bounded by delegating each logical step to a helper.
func (s *Service) registerSchematicPath(ctx context.Context, req NewSchematicRequest) (*NewResult, error) {
	_ = ctx // reserved for future cancellation

	// 1. Validate name (REQ-NS-04).
	if err := validateSchematicName(req.Name); err != nil {
		return nil, err
	}

	collection := collectionName(req.Collection)
	lang := s.effectiveLang(req)

	// 2. Conflict check (skip in dry-run — dryRunFS Stat always returns ErrNotExist).
	if !req.DryRun {
		if err := s.checkSchematicConflict(req, collection); err != nil {
			return nil, err
		}
	}

	// 3. Write factory + schema files.
	schematicDir, factoryPath, schemaPath, err := s.writeSchematicFiles(req, lang)
	if err != nil {
		return nil, err
	}
	_ = schematicDir

	// 4. Mutate project-builder.json (skip in dry-run).
	if req.DryRun {
		return &NewResult{
			DryRun:        true,
			PlannedOps:    s.fs.PlannedOps(),
			SchematicName: req.Name,
		}, nil
	}

	pbPath, err := s.mutateProjectConfig(req, collection, req.Name)
	if err != nil {
		return nil, err
	}

	return &NewResult{
		SchematicName: req.Name,
		FilesCreated:  []string{factoryPath, schemaPath, pbPath},
	}, nil
}

// collectionName returns "default" if col is empty.
func collectionName(col string) string {
	if col == "" {
		return "default"
	}
	return col
}

// effectiveLang resolves the language for this request.
func (s *Service) effectiveLang(req NewSchematicRequest) string {
	if req.Language != "" {
		return req.Language
	}
	return resolveLanguage(req.WorkDir, s.fs)
}

// checkSchematicConflict returns ErrCodeNewSchematicExists if the schematic
// already exists and --force was not specified (REQ-NS-02).
func (s *Service) checkSchematicConflict(req NewSchematicRequest, collection string) error {
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		return err
	}
	if SchematicExists(cfg, collection, req.Name) && !req.Force {
		relDir := "./schematics/" + req.Name
		return &errs.Error{
			Code:    errs.ErrCodeNewSchematicExists,
			Op:      "new.registerSchematicPath",
			Message: "schematic '" + req.Name + "' already exists at " + relDir + "; use --force to overwrite",
			Suggestions: []string{
				"run with --force to overwrite the existing schematic",
				"choose a different name",
			},
		}
	}
	return nil
}

// writeSchematicFiles creates the schematic directory and writes factory + schema files.
// Returns (schematicDir, factoryPath, schemaPath, error).
func (s *Service) writeSchematicFiles(req NewSchematicRequest, lang string) (string, string, string, error) {
	schematicDir := filepath.Join(req.WorkDir, "schematics", req.Name)
	ext := "ts"
	if lang == "js" {
		ext = "js"
	}
	factoryPath := filepath.Join(schematicDir, "factory."+ext)
	schemaPath := filepath.Join(schematicDir, "schema.json")

	if err := s.fs.MkdirAll(schematicDir, 0o755); err != nil {
		return "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to create schematic directory",
			Cause:   err,
		}
	}

	factoryBytes, err := RenderFactoryTemplate(lang, req.Name)
	if err != nil {
		return "", "", "", err
	}

	if err := s.fs.WriteFile(factoryPath, factoryBytes, 0o644); err != nil {
		return "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to write factory." + ext,
			Cause:   err,
		}
	}

	if err := s.fs.WriteFile(schemaPath, MarshalEmpty(), 0o644); err != nil {
		return "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to write schema.json",
			Cause:   err,
		}
	}

	return schematicDir, factoryPath, schemaPath, nil
}

// mutateProjectConfig reads project-builder.json, registers the schematic path,
// and writes back. Returns the absolute path of project-builder.json.
func (s *Service) mutateProjectConfig(req NewSchematicRequest, collection, name string) (string, error) {
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		return "", err
	}
	relPath := filepath.ToSlash("./schematics/" + name)
	if err := RegisterSchematicPath(cfg, collection, name, relPath); err != nil {
		return "", err
	}
	if err := WriteConfig(req.WorkDir, cfg, s.fs); err != nil {
		return "", err
	}
	return filepath.Join(req.WorkDir, "project-builder.json"), nil
}

// validateSchematicName returns ErrCodeInvalidSchematicName if name is invalid.
// Checks: non-empty, no shell metacharacters, no path separators (REQ-NS-04).
func validateSchematicName(name string) error {
	if name == "" {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidSchematicName,
			Op:      "new.validateSchematicName",
			Message: "'': invalid schematic name — name is required; valid: kebab-case, no path separators, no shell metacharacters",
		}
	}

	// Reject path separators (subset of RejectMetachars but with dedicated message).
	if strings.ContainsAny(name, "/\\") {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidSchematicName,
			Op:      "new.validateSchematicName",
			Message: "'" + name + "': invalid schematic name — contains path separator; valid: kebab-case, no path separators, no shell metacharacters",
		}
	}

	// Reject shell metacharacters and null bytes (REQ-NS-04 / validate.RejectMetachars).
	if err := validate.RejectMetachars("new.validateSchematicName", "schematic name", name); err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidSchematicName,
			Op:      "new.validateSchematicName",
			Message: "'" + name + "': invalid schematic name — contains forbidden character; valid: kebab-case, no path separators, no shell metacharacters",
		}
	}

	return nil
}

// resolveLanguage returns "ts" or "js" based on workspace signals.
// For S-001: always defaults to "ts" (real auto-detect lands in S-005).
func resolveLanguage(_ string, _ interface{}) string {
	if languageDetectFn != nil {
		lang, err := languageDetectFn("")
		if err == nil && lang != "" {
			return lang
		}
	}
	return "ts"
}
