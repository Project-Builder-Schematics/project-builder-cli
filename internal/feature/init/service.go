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
//
//	project-builder.json → schematics/.gitkeep → SKILL.md → AGENTS/CLAUDE → package.json
//	→ install subprocess → MCP instructions (flag in result; renderer prints)
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
		return InitResult{}, notImplementedErr()
	}
	if req.DryRun {
		return s.dryRun(req)
	}
	return s.realWrite(ctx, req)
}

// notImplementedErr returns the REQ-CS-05 error for --publishable.
func notImplementedErr() error {
	return &errs.Error{
		Code:    errs.ErrCodeInitNotImplemented,
		Op:      "init.handler",
		Message: "--publishable mode is not yet implemented (planned for builder-init-publishable)",
		Suggestions: []string{
			"re-run without --publishable for the standard init flow",
			"track progress: https://github.com/Project-Builder-Schematics/project-builder-cli",
		},
	}
}

// realWrite orchestrates the five outputs + install + MCP for non-dry-run mode.
//
// Partial-write caveat: if an error occurs after earlier outputs land, those
// files remain on disk. Re-run with --force to complete (or use --dry-run first).
func (s *Service) realWrite(ctx context.Context, req InitRequest) (InitResult, error) {
	result := InitResult{Directory: req.Directory}

	// Outputs 1+2: always run (REQ-PJ-01..04, REQ-DV-04, REQ-SF-01..02).
	if _, err := writeProjectConfig(s.fs, req); err != nil {
		return InitResult{}, err
	}
	if _, err := writeSchematicsSkel(s.fs, req); err != nil {
		return InitResult{}, err
	}

	// --no-skill atomically skips outputs 3+4+5+install+MCP (REQ-SA-03).
	if req.NoSkill {
		return result, nil
	}

	// Output 3: SKILL.md (REQ-SA-01..03). ErrInitSkillExists is a warning, not fatal.
	if err := s.writeSkillWithWarning(req, &result); err != nil {
		return InitResult{}, err
	}

	// Output 4: AGENTS.md/CLAUDE.md marker (REQ-AR-01..05).
	if _, err := appendAgentsMarker(req.Directory, req.Force, s.fs); err != nil {
		return InitResult{}, err
	}

	// Output 5: package.json @pbuilder/sdk dev-dep (REQ-PM-01..04).
	if _, err := mutatePackageJSON(s.fs, req); err != nil {
		return InitResult{}, err
	}

	// Install + MCP flag.
	return s.runInstallAndMCP(ctx, req, result)
}

// writeSkillWithWarning runs writeSkillArtefact and turns ErrCodeInitSkillExists
// into a warning on result.Warnings rather than a hard error (REQ-SA-02).
func (s *Service) writeSkillWithWarning(req InitRequest, result *InitResult) error {
	if _, err := writeSkillArtefact(s.fs, req, s.skill); err != nil {
		var e *errs.Error
		if errors.As(err, &e) && e.Code == errs.ErrCodeInitSkillExists {
			result.Warnings = append(result.Warnings, e.Message)
			return nil
		}
		return err
	}
	return nil
}

// runInstallAndMCP handles the PM detection, optional install subprocess, and
// MCP flag finalisation. Detect runs even when --no-install so the renderer can
// show the recommended PM in the "run X install" hint.
func (s *Service) runInstallAndMCP(ctx context.Context, req InitRequest, result InitResult) (InitResult, error) {
	detectedPM, err := s.pm.Detect(req.Directory, req.PackageManagerFlag)
	if err != nil {
		return InitResult{}, err
	}
	result.PackageManager = detectedPM

	if !req.NoInstall {
		if err := s.pm.Install(ctx, req.Directory, detectedPM); err != nil {
			return InitResult{}, err
		}
		result.Installed = true
	}

	if req.MCP == MCPYes {
		result.MCPSetupOffered = true
	}
	return result, nil
}

// dryRun records the full PlannedOps sequence (REQ-DR-01..02) without writing
// to the real filesystem. The handler swaps to a dryRunFS at request time so
// every WriteFile / MkdirAll / AppendFile call is captured as a PlannedOp.
func (s *Service) dryRun(req InitRequest) (InitResult, error) {
	// Outputs 1+2: always recorded.
	pbJSON := filepath.Join(req.Directory, "project-builder.json")
	if err := s.fs.WriteFile(pbJSON, []byte("{}"), 0o644); err != nil {
		return InitResult{}, err
	}

	schematicsDir := filepath.Join(req.Directory, schematicsFolderName)
	gitkeep := filepath.Join(schematicsDir, ".gitkeep")
	if err := s.fs.MkdirAll(schematicsDir, 0o755); err != nil {
		return InitResult{}, err
	}
	if err := s.fs.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
		return InitResult{}, err
	}

	// Outputs 3+4+5+install are gated by --no-skill / --no-install.
	if !req.NoSkill {
		if err := s.dryRunSkillBlock(req); err != nil {
			return InitResult{}, err
		}
		if !req.NoInstall {
			recordInstallOp(s.fs)
		}
	}

	// Output 6: MCP op.
	mcpOffered := req.MCP == MCPYes
	if mcpOffered {
		recordMCPOp(s.fs)
	}

	return InitResult{
		Directory:       req.Directory,
		DryRun:          true,
		PlannedOps:      s.fs.PlannedOps(),
		MCPSetupOffered: mcpOffered,
	}, nil
}

// dryRunSkillBlock records the three "--no-skill-gated" outputs (SKILL.md,
// AGENTS marker, package.json mutation). All three skip atomically with --no-skill.
func (s *Service) dryRunSkillBlock(req InitRequest) error {
	skillPath := filepath.Join(req.Directory, ".claude", "skills", "pbuilder", "SKILL.md")
	if err := s.fs.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		return err
	}
	if err := s.fs.WriteFile(skillPath, s.skill, 0o644); err != nil {
		return err
	}

	// AGENTS marker (dry-run records for AGENTS.md by default; real target
	// selection happens in the realWrite path via appendAgentsMarker).
	agentsPath := filepath.Join(req.Directory, "AGENTS.md")
	if err := s.fs.AppendFile(agentsPath, []byte(agentMarkerBlock)); err != nil {
		return err
	}

	pkgJSON := filepath.Join(req.Directory, "package.json")
	return recordDevDepOp(s.fs, pkgJSON)
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
