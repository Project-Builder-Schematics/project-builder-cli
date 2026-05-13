// Package initialise — project_config.go writes the project-builder.json
// config file atomically via the FSWriter port.
//
// # Locked v1 bytes (REQ-PJ-01)
//
// The exact field order, 2-space indentation, and trailing newline are part
// of the v1 spec contract. Any change to the output bytes after v1.0.0 is a
// BREAKING CHANGE and requires an explicit spec amendment.
//
// # Forward compatibility (REQ-PJ-04)
//
// This function WRITES the v1 schema — it does not read or validate existing
// configs. Future readers (builder execute, builder validate) MUST NOT use
// json.Decoder with DisallowUnknownFields, because unknown top-level keys are
// legal under the forward-compat invariant (Discussion #3). Adding a new key
// to the CLI-generated config in a later version is a MINOR (non-breaking)
// change as long as the new key is optional.
//
// # Atomic write (REQ-PJ-02, ADR-020)
//
// All writes go through FSWriter.WriteFile. The osFS implementation uses
// temp-file + os.Rename for atomicity. This function never calls os.* directly
// (FF-init-02 invariant).
package initialise

import (
	"encoding/json"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// projectBuilderConfig is the typed representation of project-builder.json v1.
//
// Field declaration order MUST match the locked output order (REQ-PJ-01).
// encoding/json.MarshalIndent preserves struct field declaration order, so
// the field order here is load-bearing — do not reorder without a spec amendment.
//
// To add a new top-level key in a later version: append a new field to this
// struct (at the end) and update the locked bytes in project_config_test.go.
// Appending is forward-compat; inserting in the middle changes the serialised
// key order and breaks the byte-equality contract.
type projectBuilderConfig struct {
	Schema       string                    `json:"$schema"`
	Version      string                    `json:"version"`
	Collections  map[string]any            `json:"collections"`
	Dependencies map[string]any            `json:"dependencies"`
	Settings     projectBuilderSettings    `json:"settings"`
	Skill        projectBuilderSkillConfig `json:"skill"`
}

// projectBuilderSettings is the settings block of project-builder.json v1.
// Field order is load-bearing (locked bytes).
type projectBuilderSettings struct {
	AutoInstall    bool   `json:"autoInstall"`
	ConflictPolicy string `json:"conflictPolicy"`
	DepValidation  string `json:"depValidation"`
}

// projectBuilderSkillConfig is the skill block of project-builder.json v1.
// Field order is load-bearing (locked bytes).
type projectBuilderSkillConfig struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

// writeProjectConfig writes the locked v1 project-builder.json to
// req.Directory via the FSWriter port.
//
// Returns the absolute path of the written file on success.
//
// Pre-existing file behaviour:
//   - req.Force = false → returns ErrCodeInitConfigExists (REQ-DV-04).
//   - req.Force = true  → overwrites unconditionally.
//
// No direct os.* calls (FF-init-02 invariant).
func writeProjectConfig(fs FSWriter, req InitRequest) (string, error) {
	target := filepath.Join(req.Directory, "project-builder.json")

	// Check for pre-existing config (REQ-DV-04).
	if _, err := fs.Stat(target); err == nil {
		// File exists.
		if !req.Force {
			return "", &errs.Error{
				Code:    errs.ErrCodeInitConfigExists,
				Op:      "init.writeProjectConfig",
				Message: "project-builder.json already exists in " + req.Directory,
				Suggestions: []string{
					"run with --force to overwrite the existing configuration",
					"remove project-builder.json manually and re-run",
				},
			}
		}
		// Force=true: fall through to overwrite.
	}

	data, err := marshalProjectConfig(req.NoSkill)
	if err != nil {
		return "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.writeProjectConfig",
			Message: "failed to serialise project-builder.json",
			Cause:   err,
		}
	}

	if err := fs.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}

	return target, nil
}

// marshalProjectConfig returns the locked v1 bytes for project-builder.json.
// When noSkill is true, skill.enabled is set to false (REQ-PJ-03).
//
// The output always ends with a trailing newline (part of the locked v1 contract).
func marshalProjectConfig(noSkill bool) ([]byte, error) {
	cfg := projectBuilderConfig{
		Schema:       "./node_modules/@pbuilder/sdk/schemas/project-builder.schema.json",
		Version:      "1",
		Collections:  map[string]any{},
		Dependencies: map[string]any{},
		Settings: projectBuilderSettings{
			AutoInstall:    true,
			ConflictPolicy: "child-wins",
			DepValidation:  "dev",
		},
		Skill: projectBuilderSkillConfig{
			Enabled: !noSkill,
			Path:    ".claude/skills/pbuilder/SKILL.md",
		},
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}

	// Append the required trailing newline (part of the locked v1 byte contract).
	return append(b, '\n'), nil
}
