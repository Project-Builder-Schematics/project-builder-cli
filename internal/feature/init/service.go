// Package initialise — service.go contains Service.Init, the orchestrator
// for the `builder init` feature.
//
// S-000 walking skeleton scope:
//   - Validates --publishable guard (REQ-CS-05)
//   - Implements the full dry-run path (REQ-DR-01, REQ-DR-02, REQ-EC-05)
//   - Records all 5 output PlannedOps in REQ-EC-05 write order
//   - Records mcp_setup_offered op when MCP=yes (REQ-MCP-02)
//   - Returns ErrCodeNotImplemented for non-dry-run (real writes land S-001..S-005)
//
// S-001 real-write additions:
//   - Output 1: project-builder.json — writeProjectConfig (REQ-PJ-01..04, REQ-DV-04)
//   - Output 2: schematics/.gitkeep — writeSchematicsSkel (REQ-SF-01..02)
//
// S-002 real-write additions:
//   - Output 3: SKILL.md — writeSkillArtefact (REQ-SA-01..03)
//   - --no-skill skips output 3 entirely; outputs 4+SDK also skipped atomically
//   - Outputs 4..5 + install + MCP still return ErrCodeNotImplemented (S-003..S-005)
//
// S-003 real-write additions:
//   - Output 4: AGENTS.md/CLAUDE.md marker — appendAgentsMarker (REQ-AR-01..05)
//   - Outputs 5 + install + MCP still return ErrCodeNotImplemented (S-004..S-005)
//
// S-004 real-write additions:
//   - Output 5: package.json @pbuilder/sdk dev-dep — mutatePackageJSON (REQ-PM-01..04)
//
// S-005 real-write additions (COMPLETE):
//   - PM detection: s.pm.Detect(dir, flag) — priority chain per REQ-PD-01
//   - --no-install: detect for pretty message, skip Install (REQ-PD-04)
//   - Install subprocess: s.pm.Install(ctx, dir, pm) — 120s timeout (REQ-PD-02, ADR-023)
//   - MCP instructions print: sets InitResult.MCPSetupOffered=true when MCP=yes (REQ-MCP-02)
//
// Write order LOCKED (REQ-EC-05):
//   project-builder.json → schematics/.gitkeep → SKILL.md → AGENTS/CLAUDE → package.json
//   → install subprocess → MCP instructions (flag in result; renderer prints)
package initialise

import (
	"context"
	"errors"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// Init orchestrates the five init outputs in write order per REQ-EC-05:
//  1. project-builder.json
//  2. schematics/.gitkeep
//  3. SKILL.md
//  4. AGENTS.md / CLAUDE.md marker
//  5. package.json (add @pbuilder/sdk dev-dep)
//  6. install subprocess
//  7. MCP instructions print (if MCP=yes)
//
// In S-000 (walking skeleton), dry-run is fully implemented. Real-write mode
// returns ErrCodeNotImplemented and will be filled in slice by slice (S-001..S-005).
func (s *Service) Init(ctx context.Context, req InitRequest) (InitResult, error) {
	// Guard: --publishable not yet implemented (REQ-CS-05).
	if req.Publishable {
		return InitResult{}, &errs.Error{
			Code:    errs.ErrCodeInitNotImplemented,
			Op:      "init.handler",
			Message: "--publishable mode is not yet implemented (planned for builder-init-publishable)",
			Suggestions: []string{
				"re-run without --publishable for the standard init flow",
				"track progress: https://github.com/Project-Builder-Schematics/project-builder-cli",
			},
		}
	}

	if !req.DryRun {
		// --- Real-write path (partial, S-004) ---
		//
		// Outputs 1, 2, 3, 4, and 5 are wired. Install subprocess + MCP return
		// ErrCodeNotImplemented until S-005 lands.
		//
		// Partial-write caveat: if the service returns an error after writing
		// earlier outputs, those files are already on disk. The user can re-run
		// with --force after later slices are installed to complete the init.
		// Use --dry-run to preview the full plan without any writes.

		var result InitResult
		result.Directory = req.Directory

		// Output 1: project-builder.json (REQ-PJ-01..04, REQ-DV-04).
		if _, err := writeProjectConfig(s.fs, req); err != nil {
			return InitResult{}, err
		}

		// Output 2: schematics/.gitkeep (REQ-SF-01..02).
		if _, err := writeSchematicsSkel(s.fs, req); err != nil {
			return InitResult{}, err
		}

		// Output 3: SKILL.md (REQ-SA-01..03) — skip if --no-skill.
		if !req.NoSkill {
			skillPath, skillErr := writeSkillArtefact(s.fs, req, s.skill)
			if skillErr != nil {
				// ErrCodeInitSkillExists means the file was pre-existing and Force=false.
				// Per REQ-SA-02, this is a WARNING not a failure: record it and continue.
				skipSentinel := &errs.Error{Code: errs.ErrCodeInitSkillExists}
				if errors.Is(skillErr, skipSentinel) {
					var e *errs.Error
					if errors.As(skillErr, &e) {
						result.Warnings = append(result.Warnings, e.Message)
					}
					// skillPath is still set to the existing file path — record it.
					_ = skillPath
				} else {
					// Any other error is fatal.
					return InitResult{}, skillErr
				}
			}
		}

		// --no-skill skips outputs 3, 4, and the SDK dev-dep atomically (REQ-SA-03).
		// When NoSkill=true, we jump past outputs 4+5 entirely.
		if !req.NoSkill {
			// Output 4: AGENTS.md/CLAUDE.md marker (REQ-AR-01..05).
			_, agentErr := appendAgentsMarker(req.Directory, req.Force, s.fs)
			if agentErr != nil {
				return InitResult{}, agentErr
			}

			// Output 5: package.json @pbuilder/sdk dev-dep (REQ-PM-01..04).
			_, pkgErr := mutatePackageJSON(s.fs, req)
			if pkgErr != nil {
				return InitResult{}, pkgErr
			}

			// PM detection (REQ-PD-01) — run even when --no-install so we can
			// show the recommended PM in the pretty-mode "run X install" hint.
			detectedPM, detectErr := s.pm.Detect(req.Directory, req.PackageManagerFlag)
			if detectErr != nil {
				return InitResult{}, detectErr
			}
			result.PackageManager = detectedPM

			// Install subprocess (REQ-PD-02..04, ADR-023).
			if !req.NoInstall {
				if installErr := s.pm.Install(ctx, req.Directory, detectedPM); installErr != nil {
					return InitResult{}, installErr
				}
				result.Installed = true
			}
			// --no-install: Detect still ran (for pretty message), Install skipped,
			// Installed remains false.

			// MCP instructions (REQ-MCP-02): set flag so renderer prints instructions.
			if req.MCP == MCPYes {
				result.MCPSetupOffered = true
			}
		}

		// --no-skill path: outputs 4+5+install+MCP are all skipped.
		// Outputs 1 and 2 were written above; output 3 was skipped.
		return result, nil
	}

	// --- Dry-run path ---

	// We need a dryRunFS to record ops. The service receives the fs dependency
	// from the constructor. In dry-run mode the handler swaps to a dryRunFS
	// so s.fs here IS a dryRunFS. However, to be defensive (the service should
	// not depend on the handler's swap), we work through s.fs's FSWriter methods
	// which in dry-run mode will record the ops correctly.
	//
	// The dryRunFS.recordOp method is used for ops that don't map naturally to
	// Stat/Write/Append calls (install_package, mcp_setup_offered). We use a
	// type assertion to access it — if the underlying fs does not expose it,
	// we fall back to recording nothing (fakeFS in tests uses its own recordOp).

	// Output 1: project-builder.json (REQ-PJ-01)
	pbJSON := filepath.Join(req.Directory, "project-builder.json")
	if err := s.fs.WriteFile(pbJSON, []byte("{}"), 0o644); err != nil {
		return InitResult{}, err
	}

	// Output 2: schematics/.gitkeep (REQ-SF-01)
	schematicsDir := filepath.Join(req.Directory, schematicsFolderName)
	gitkeep := filepath.Join(schematicsDir, ".gitkeep")
	if err := s.fs.MkdirAll(schematicsDir, 0o755); err != nil {
		return InitResult{}, err
	}
	if err := s.fs.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
		return InitResult{}, err
	}

	// Output 3: SKILL.md (REQ-SA-01) — skip if --no-skill.
	if !req.NoSkill {
		skillPath := filepath.Join(req.Directory, ".claude", "skills", "pbuilder", "SKILL.md")
		skillDir := filepath.Dir(skillPath)
		if err := s.fs.MkdirAll(skillDir, 0o755); err != nil {
			return InitResult{}, err
		}
		if err := s.fs.WriteFile(skillPath, s.skill, 0o644); err != nil {
			return InitResult{}, err
		}

		// Output 4: AGENTS.md marker (REQ-AR-01) — skip if --no-skill.
		// In dry-run mode, record the append_marker op for AGENTS.md (default
		// precedence). The real target selection (AGENTS vs CLAUDE) happens in
		// the real-write path via appendAgentsMarker.
		agentsPath := filepath.Join(req.Directory, "AGENTS.md")
		if err := s.fs.AppendFile(agentsPath, []byte(agentMarkerBlock)); err != nil {
			return InitResult{}, err
		}
	}

	// Output 5: package.json @pbuilder/sdk dev-dep (REQ-PM-01) — skip if --no-skill.
	if !req.NoSkill {
		pkgJSON := filepath.Join(req.Directory, "package.json")
		// In dry-run mode, record the modify_devdep op via the FSWriter.
		// The actual JSON mutation is implemented in S-004.
		if err := recordDevDepOp(s.fs, pkgJSON); err != nil {
			return InitResult{}, err
		}
	}

	// Install subprocess op (REQ-PD-01, REQ-DR-02) — dry-run records the op
	// so callers can preview that install will run. Skip when --no-install or
	// --no-skill (package.json is also skipped in those cases).
	if !req.NoSkill && !req.NoInstall {
		recordInstallOp(s.fs)
	}

	// Output 6: MCP instructions (REQ-MCP-02) — dry-run records op, no print.
	mcpOffered := false
	if req.MCP == MCPYes {
		recordMCPOp(s.fs)
		mcpOffered = true
	}

	return InitResult{
		Directory:       req.Directory,
		DryRun:          true,
		PlannedOps:      s.fs.PlannedOps(),
		MCPSetupOffered: mcpOffered,
	}, nil
}

// recordDevDepOp records a modify_devdep PlannedOp via the FSWriter.
// In dry-run mode, the dryRunFS intercepts this as a custom op type.
// The actual package.json mutation is implemented in S-004.
func recordDevDepOp(fs FSWriter, pkgJSONPath string) error {
	type opRecorder interface {
		recordOp(PlannedOp)
	}
	if r, ok := fs.(opRecorder); ok {
		r.recordOp(PlannedOp{Op: "modify_devdep", Path: pkgJSONPath, Details: "@pbuilder/sdk ^1.0.0"})
		return nil
	}
	// Fallback: use AppendFile to trigger a recorded op (for fakeFS tests).
	return fs.AppendFile(pkgJSONPath, nil)
}

// recordInstallOp records an install_package PlannedOp via the FSWriter.
// Records the intent to run `<pm> install --save-dev @pbuilder/sdk`.
// The exact PM is omitted here (not yet detected in dry-run mode).
func recordInstallOp(fs FSWriter) {
	type opRecorder interface {
		recordOp(PlannedOp)
	}
	if r, ok := fs.(opRecorder); ok {
		r.recordOp(PlannedOp{Op: "install_package", Details: "@pbuilder/sdk (dev-dep) via detected PM"})
	}
}

// recordMCPOp records a mcp_setup_offered PlannedOp via the FSWriter.
// No path field — REQ-DR-02.
func recordMCPOp(fs FSWriter) {
	type opRecorder interface {
		recordOp(PlannedOp)
	}
	if r, ok := fs.(opRecorder); ok {
		r.recordOp(PlannedOp{Op: "mcp_setup_offered"})
	}
}
