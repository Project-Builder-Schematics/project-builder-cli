// Package newfeature — handler_schematic.go contains the RunE handler logic
// for `builder new schematic`.
//
// Responsibilities (ADR-011: handler ≤ 80 SLOC soft / ≤ 100 hard):
//  1. Parse flags and positional argument (name)
//  2. Resolve working directory (os.Getwd)
//  3. Build NewSchematicRequest and call svc.RegisterSchematic
//  4. Render the result via RenderPretty or RenderJSON (ADR-019)
//
// S-001: real implementation. Delegates to service.RegisterSchematic which
// dispatches to registerSchematicPath. --dry-run swaps in dryRunFS.
// .d.ts generation is STUBBED with a WARN until S-003.
package newfeature

import (
	"os"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// handleSchematic returns the RunE closure for `builder new schematic`.
// svc is the wired Service. All flag values are parsed inside the closure.
//
// Signature matches command.go's RunE adapter pattern:
// func(svc, args, dryRun, jsonOut) error.
func handleSchematic(svc *Service) func(cmd *cobra.Command, args []string, dryRun, jsonOut bool) error {
	return func(cmd *cobra.Command, args []string, dryRun, jsonOut bool) error {
		flags := cmd.Flags()

		var name string
		if len(args) > 0 {
			name = args[0]
		}

		force, _ := flags.GetBool("force")
		inline, _ := flags.GetBool("inline")
		language, _ := flags.GetString("language")
		extends, _ := flags.GetString("extends")

		// Resolve workspace root (cwd by default; positional dir not supported in `new`).
		workDir, err := os.Getwd()
		if err != nil {
			return &errs.Error{
				Code:    errs.ErrCodeInvalidInput,
				Op:      "new.handler",
				Message: "could not determine current working directory",
				Cause:   err,
			}
		}

		// Swap FSWriter to dryRunFS when --dry-run is set.
		activeSvc := svc
		if dryRun {
			activeSvc = NewService(fswriter.NewDryRunWriter())
		}

		req := NewSchematicRequest{
			Name:       name,
			WorkDir:    workDir,
			Force:      force,
			DryRun:     dryRun,
			Inline:     inline,
			Language:   language,
			Extends:    extends,
			OutputJSON: jsonOut,
		}

		result, err := activeSvc.RegisterSchematic(cmd.Context(), req)
		if err != nil {
			return err
		}

		// S-001: .d.ts generation is STUBBED — emit WARN via Renderer (REQ-TG via S-003).
		if !dryRun && !inline {
			result.Warnings = append(result.Warnings,
				"schema.d.ts generation pending next slice (S-003)")
		}

		if jsonOut {
			return RenderJSON(cmd.OutOrStdout(), *result)
		}
		RenderPretty(cmd.OutOrStdout(), *result)
		return nil
	}
}
