// Package newfeature — handler_collection.go contains the RunE handler logic
// for `builder new collection`.
//
// Responsibilities (ADR-011: handler ≤ 80 SLOC soft / ≤ 100 hard):
//  1. Parse flags and positional argument (name)
//  2. Build NewCollectionRequest and call svc.RegisterCollection
//  3. Render the result via RenderPretty or RenderJSON (ADR-019)
//
// S-000b: stub — delegates to RegisterCollection which returns
// ErrCodeNewNotImplemented (REQ-EC-07). Real logic lands in S-004.
package newfeature

import (
	"github.com/spf13/cobra"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// handleCollection returns the RunE closure for `builder new collection`.
// svc is the wired Service. All flag values are parsed inside the closure.
func handleCollection(svc *Service) func(cmd *cobra.Command, args []string, dryRun, jsonOut bool) error {
	return func(cmd *cobra.Command, args []string, dryRun, jsonOut bool) error {
		flags := cmd.Flags()

		var name string
		if len(args) > 0 {
			name = args[0]
		}

		force, _ := flags.GetBool("force")
		publishable, _ := flags.GetBool("publishable")

		// Swap FSWriter to dryRunFS when --dry-run is set.
		activeSvc := svc
		if dryRun {
			activeSvc = NewService(fswriter.NewDryRunWriter())
		}

		req := NewCollectionRequest{
			Name:        name,
			Force:       force,
			DryRun:      dryRun,
			Publishable: publishable,
			OutputJSON:  jsonOut,
		}

		result, err := activeSvc.RegisterCollection(cmd.Context(), req)
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
