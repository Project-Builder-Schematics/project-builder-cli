// Package initialise — skill_artefact.go writes the SKILL.md artefact to the
// user's project under .claude/skills/pbuilder/SKILL.md.
//
// # Locked bytes (REQ-SA-01)
//
// The SKILL.md bytes are passed in from the service (originating from
// template.Skill, the //go:embed bundle). This function never hard-codes
// the content — it writes exactly what it receives.
//
// # Pre-existing artefact policy (REQ-SA-02)
//
// When .claude/skills/pbuilder/SKILL.md already exists:
//   - Force=false → return ErrCodeInitSkillExists sentinel (message is non-empty).
//     The service treats this sentinel as a WARNING appended to InitResult.Warnings,
//     NOT as a hard failure (exit code stays 0, "skip is not a failure").
//   - Force=true  → overwrite unconditionally.
//
// # --no-skill path (REQ-SA-03)
//
// When req.NoSkill is true, this function is a no-op and returns ("", nil).
// The service MUST also skip output 4 (AGENTS marker) and the SDK dev-dep
// when NoSkill is true — those are enforced in service.go.
//
// # No direct os.* calls (FF-init-02)
//
// All I/O goes through the FSWriter port.
package initialise

import (
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// skillArtefactDir is the canonical directory for the SKILL.md artefact under
// a project root. The path is REQ-SA-01 locked — do not change post-v1.0.0.
const skillArtefactDir = ".claude/skills/pbuilder"

// skillArtefactFilename is the filename written by writeSkillArtefact.
const skillArtefactFilename = "SKILL.md"

// writeSkillArtefact writes skill bytes to .claude/skills/pbuilder/SKILL.md
// inside req.Directory.
//
// Returns:
//   - (path, nil) on successful write.
//   - (path, ErrCodeInitSkillExists) when the file exists and Force=false.
//     Callers MUST treat this as a warning (not a fatal error).
//   - ("", nil) when req.NoSkill is true (no-op).
//   - ("", err) on any I/O failure.
//
// The path return value is always the canonical skill path when NoSkill=false,
// regardless of whether the file was written or skipped. This allows the
// caller to append the path to InitResult.OutputsCreated or log the skip.
func writeSkillArtefact(fs FSWriter, req InitRequest, skill []byte) (string, error) {
	if req.NoSkill {
		return "", nil
	}

	dir := filepath.Join(req.Directory, skillArtefactDir)
	target := filepath.Join(dir, skillArtefactFilename)

	// Check for pre-existing artefact (REQ-SA-02).
	if _, err := fs.Stat(target); err == nil {
		// File exists.
		if !req.Force {
			return target, &errs.Error{
				Code:    errs.ErrCodeInitSkillExists,
				Op:      "init.writeSkillArtefact",
				Message: "SKILL.md already exists at " + target + "; skipping (run with --force to overwrite)",
				Suggestions: []string{
					"run with --force to overwrite the existing SKILL.md",
					"remove " + target + " manually and re-run",
				},
			}
		}
		// Force=true: fall through to overwrite.
	}

	if err := fs.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	if err := fs.WriteFile(target, skill, 0o644); err != nil {
		return "", err
	}

	return target, nil
}
