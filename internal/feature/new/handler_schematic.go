// Package newfeature — handler_schematic.go contains the RunE handler logic
// for `builder new schematic`.
//
// Responsibilities (ADR-011: handler ≤ 80 SLOC soft / ≤ 100 hard):
//  1. Parse flags and positional argument (name)
//  2. Resolve working directory (os.Getwd)
//  3. Build NewSchematicRequest and call svc.RegisterSchematic
//  4. Render the result via RenderPretty or RenderJSON (ADR-019)
//
// ADR-03: output.Output is captured in the closure; all render calls go through
// out, never os.Stdout or fmt.Print* directly (FF-25 gate).
//
// S-001+S-003: real implementation. Delegates to service.RegisterSchematic which
// dispatches to registerSchematicPath. --dry-run swaps in dryRunFS.
// .d.ts generation is wired in service_schematic_path.go via tsgen (S-003).
// S-004: out output.Output added per ADR-03.
package newfeature

import (
	"os"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
)

// handleSchematic returns the RunE closure for `builder new schematic`.
// svc is the wired Service. out is the unified output port (ADR-03).
// All flag values are parsed inside the closure.
func handleSchematic(svc *Service, out output.Output) func(cmd *cobra.Command, args []string, dryRun, jsonOut bool) error {
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

		if jsonOut {
			return RenderJSON(cmd.OutOrStdout(), *result)
		}
		RenderPretty(out, *result)
		return nil
	}
}
