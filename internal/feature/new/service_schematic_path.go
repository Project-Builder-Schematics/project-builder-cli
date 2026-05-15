// Package newfeature — service_schematic_path.go implements path-mode schematic
// scaffolding: factory.{ts|js} + schema.json + schema.d.ts + project-builder.json.
//
// REQ coverage:
//   - REQ-NS-01: happy path TS — three files + JSON entry
//   - REQ-NS-02: conflict without --force → ErrCodeNewSchematicExists
//   - REQ-NS-03: --force overwrites existing schematic
//   - REQ-NS-04: name validation → ErrCodeInvalidSchematicName
//   - REQ-NS-05: --dry-run returns planned ops; zero FS writes
//   - REQ-NS-06: --language=js → factory.js
//   - REQ-EX-02/03: --extends grammar validated before FS writes (S-005)
//   - REQ-LG-01..06: language auto-detect + explicit override (S-005)
//   - REQ-PJ-01..08: project-builder.json mutation via projectconfig
//   - REQ-SJ-01/03/05: schema.json canonical bytes via schema.MarshalEmpty
//   - REQ-TG-01: schema.d.ts written alongside schema.json (S-003 wire-in)
package newfeature

import (
	"context"
	"io/fs"
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

	// 1b. Validate --extends grammar (REQ-EX-02/03), if provided.
	if req.Extends != "" {
		if err := ValidateExtendsGrammar(req.Extends); err != nil {
			return nil, err
		}
	}

	collection := collectionName(req.Collection)

	// 1c. Resolve language — may produce a warning (REQ-LG-03) and may fail
	// for invalid --language values (REQ-LG-06).
	lang, langWarn, err := s.resolveEffectiveLang(req)
	if err != nil {
		return nil, err
	}

	// 2. Conflict check (skip in dry-run — dryRunFS Stat always returns ErrNotExist).
	if !req.DryRun {
		if err := s.checkSchematicConflict(req, collection); err != nil {
			return nil, err
		}
	}

	// 3. Write factory + schema.json + schema.d.ts files.
	schematicDir, factoryPath, schemaPath, dtsPath, err := s.writeSchematicFiles(req, lang)
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

	pbPath, cfgWarns, err := s.mutateProjectConfig(req, collection, req.Name)
	if err != nil {
		return nil, err
	}

	result := &NewResult{
		SchematicName: req.Name,
		FilesCreated:  []string{factoryPath, schemaPath, dtsPath, pbPath},
	}

	// Propagate language warning (REQ-LG-03) through NewResult.Warnings (ADR-019).
	if langWarn != "" {
		result.Warnings = append(result.Warnings, langWarn)
	}

	// Propagate BOM-strip warning (ADV-06) from ReadConfig through NewResult.Warnings.
	result.Warnings = append(result.Warnings, cfgWarns...)

	return result, nil
}

// collectionName returns "default" if col is empty.
func collectionName(col string) string {
	if col == "" {
		return "default"
	}
	return col
}

// resolveEffectiveLang returns the effective language, any warning, and any error.
// Uses ResolveLanguage (REQ-LG-01..06) with the test-injection seam as fallback.
func (s *Service) resolveEffectiveLang(req NewSchematicRequest) (lang, warn string, err error) {
	// Test injection seam: if languageDetectFn is set, honour it for compatibility
	// with tests that use SetLanguageDetectFn (export_test.go / L-builder-init-01).
	if languageDetectFn != nil {
		detected, detErr := languageDetectFn(req.WorkDir)
		if detErr == nil && detected != "" {
			// Explicit flag still wins (REQ-LG-04/05).
			if req.Language != "" {
				return req.Language, "", validateExplicitLanguage(req.Language)
			}
			return detected, "", nil
		}
	}
	return ResolveLanguage(req.Language, req.WorkDir, s.fs)
}

// validateExplicitLanguage returns ErrCodeInvalidLanguage for unsupported values.
// Used by resolveEffectiveLang when languageDetectFn is active (test seam).
func validateExplicitLanguage(explicit string) error {
	switch explicit {
	case "ts", "js":
		return nil
	default:
		return &errs.Error{
			Code:    errs.ErrCodeInvalidLanguage,
			Op:      "new.resolveLanguage",
			Message: "--language '" + explicit + "': unsupported; valid values: ts, js",
		}
	}
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

// checkSymlinkSafety rejects schematicDir if it exists and is a symlink whose
// resolved target is outside the workspace root (ADV-08 / REQ-PJ-01).
// This prevents a malicious symlink from redirecting FS writes outside the project.
// Skip on Windows (symlink semantics are different and require elevated privileges).
func (s *Service) checkSymlinkSafety(workDir, schematicDir string) error {
	lfi, err := s.fs.Lstat(schematicDir)
	if err != nil {
		return nil // path doesn't exist — no symlink to check
	}
	if lfi.Mode()&fs.ModeSymlink == 0 {
		return nil // regular dir — no safety concern
	}
	// Path is a symlink — resolve the target.
	resolved, err := s.fs.EvalSymlinks(schematicDir)
	if err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidSchematicName,
			Op:      "new.registerSchematicPath",
			Message: "schematic path '" + schematicDir + "' is a broken symlink",
		}
	}
	// Ensure the resolved target is within the workspace.
	// Clean both paths to normalise separators.
	cleanResolved := filepath.Clean(resolved)
	cleanWorkDir := filepath.Clean(workDir)
	if len(cleanResolved) >= len(cleanWorkDir) && cleanResolved[:len(cleanWorkDir)] == cleanWorkDir {
		return nil // inside workspace — safe
	}
	return &errs.Error{
		Code:    errs.ErrCodeInvalidSchematicName,
		Op:      "new.registerSchematicPath",
		Message: "schematic path '" + schematicDir + "' is a symlink pointing outside the workspace; rejected for safety",
		Suggestions: []string{
			"remove the symlink and run again, or choose a different schematic name",
		},
	}
}

// writeSchematicFiles creates the schematic directory and writes factory + schema
// + schema.d.ts files. Returns (schematicDir, factoryPath, schemaPath, dtsPath, error).
func (s *Service) writeSchematicFiles(req NewSchematicRequest, lang string) (string, string, string, string, error) {
	schematicDir := filepath.Join(req.WorkDir, "schematics", req.Name)

	// ADV-08: reject symlinks pointing outside the workspace before any writes.
	if err := s.checkSymlinkSafety(req.WorkDir, schematicDir); err != nil {
		return "", "", "", "", err
	}
	ext := "ts"
	if lang == "js" {
		ext = "js"
	}
	factoryPath := filepath.Join(schematicDir, "factory."+ext)
	schemaPath := filepath.Join(schematicDir, "schema.json")
	dtsPath := filepath.Join(schematicDir, "schema.d.ts")

	if err := s.fs.MkdirAll(schematicDir, 0o755); err != nil {
		return "", "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to create schematic directory",
			Cause:   err,
		}
	}

	factoryBytes, err := RenderFactoryTemplate(lang, req.Name)
	if err != nil {
		return "", "", "", "", err
	}

	if err := s.fs.WriteFile(factoryPath, factoryBytes, 0o644); err != nil {
		return "", "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to write factory." + ext,
			Cause:   err,
		}
	}

	if err := s.fs.WriteFile(schemaPath, MarshalEmpty(), 0o644); err != nil {
		return "", "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to write schema.json",
			Cause:   err,
		}
	}

	// Generate schema.d.ts via tsgen (REQ-TG-01 wire-in — S-003).
	emptySchema := Schema{Inputs: map[string]InputSpec{}}
	dtsBytes, err := GenerateDTS(req.Name, emptySchema)
	if err != nil {
		return "", "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to generate schema.d.ts",
			Cause:   err,
		}
	}

	if err := s.fs.WriteFile(dtsPath, dtsBytes, 0o644); err != nil {
		return "", "", "", "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "new.registerSchematicPath",
			Message: "failed to write schema.d.ts",
			Cause:   err,
		}
	}

	return schematicDir, factoryPath, schemaPath, dtsPath, nil
}

// mutateProjectConfig reads project-builder.json, registers the schematic path,
// and writes back. Returns (pbPath, warnings, error).
// warnings carries non-fatal issues from ReadConfig (e.g. BOM-strip per ADV-06).
func (s *Service) mutateProjectConfig(req NewSchematicRequest, collection, name string) (string, []string, error) {
	cfg, err := ReadConfig(req.WorkDir, s.fs)
	if err != nil {
		return "", nil, err
	}
	cfgWarns := cfg.Warnings
	relPath := filepath.ToSlash("./schematics/" + name)
	if err := RegisterSchematicPath(cfg, collection, name, relPath); err != nil {
		return "", cfgWarns, err
	}
	if err := WriteConfig(req.WorkDir, cfg, s.fs); err != nil {
		return "", cfgWarns, err
	}
	return filepath.Join(req.WorkDir, "project-builder.json"), cfgWarns, nil
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
