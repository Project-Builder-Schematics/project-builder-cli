// Package newfeature — handler_schematic.go contains the RunE handler logic
// for `builder new schematic`.
//
// Responsibilities (ADR-011: handler ≤ 80 SLOC soft / ≤ 100 hard):
//  1. Parse flags and positional argument (name)
//  2. Build NewSchematicRequest and call svc.RegisterSchematic
//  3. Render the result via RenderPretty or RenderJSON (ADR-019)
//
// S-000b: stub — delegates to RegisterSchematic which returns
// ErrCodeNewNotImplemented (REQ-EC-07). Real logic lands in S-001.
package newfeature

import (
	"github.com/spf13/cobra"

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

		// Swap FSWriter to dryRunFS when --dry-run is set.
		activeSvc := svc
		if dryRun {
			activeSvc = NewService(fswriter.NewDryRunWriter())
		}

		req := NewSchematicRequest{
			Name:       name,
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
		RenderPretty(cmd.OutOrStdout(), *result)
		return nil
	}
}
