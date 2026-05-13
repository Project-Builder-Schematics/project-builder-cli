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
//   - Outputs 3..5 + install + MCP still return ErrCodeNotImplemented (S-002..S-005)
//
// Partial-write contract (Option A, S-001):
//   Outputs 1 and 2 are written to disk before ErrCodeNotImplemented is returned
//   for output 3. The error message names which slice is needed next and suggests
//   --dry-run as the stable preview path. Users can re-run with --force after
//   later slices land without data loss.
package initialise

import (
	"context"
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
		// --- Real-write path (partial, S-001) ---
		//
		// Outputs 1 and 2 are wired. Outputs 3..5 + install + MCP return
		// ErrCodeNotImplemented until later slices land.
		//
		// Partial-write caveat: if the service returns an error after writing
		// outputs 1 and 2, those files are already on disk. The user can re-run
		// with --force after later slices are installed to complete the init.
		// Use --dry-run to preview the full plan without any writes.

		// Output 1: project-builder.json (REQ-PJ-01..04, REQ-DV-04).
		if _, err := writeProjectConfig(s.fs, req); err != nil {
			return InitResult{}, err
		}

		// Output 2: schematics/.gitkeep (REQ-SF-01..02).
		if _, err := writeSchematicsSkel(s.fs, req); err != nil {
			return InitResult{}, err
		}

		// Outputs 3..5 + install + MCP — not yet implemented (S-002..S-005).
		return InitResult{}, &errs.Error{
			Code:    errs.ErrCodeNotImplemented,
			Op:      "init.handler",
			Message: "real-write of SKILL.md requires slice S-002; use --dry-run to preview the full plan, or run later slices first",
			Suggestions: []string{
				"use --dry-run to preview all planned operations without writing files",
				"project-builder.json and schematics/.gitkeep have been written; re-run with --force after S-002..S-005 land to complete init",
			},
		}
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
		agentsPath := filepath.Join(req.Directory, "AGENTS.md")
		if err := s.fs.AppendFile(agentsPath, []byte("\n<!-- pbuilder -->\n")); err != nil {
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

	// Note: install subprocess (REQ-PD-01) is NOT recorded as a PlannedOp in
	// S-000 dry-run — PM detection requires the real filesystem and the subprocess
	// cannot run in dry-run mode. The install_package op appears in S-005 when the
	// real PM runner is wired. The stable PlannedOp enum reserves install_package
	// for that slice (REQ-DR-02).

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
