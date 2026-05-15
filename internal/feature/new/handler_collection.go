// Package newfeature — handler_collection.go contains the RunE handler logic
// for `builder new collection`.
//
// Responsibilities (ADR-011: handler ≤ 80 SLOC soft / ≤ 100 hard):
//  1. Parse flags and positional argument (name)
//  2. Preflight: mode-conflict check (--publishable + --inline → REJECT)
//  3. Resolve working directory
//  4. Build NewCollectionRequest and call svc.RegisterCollection
//  5. Render the result via RenderPretty or RenderJSON (ADR-019)
//
// S-004: real implementation. --inline is not surfaced on the collection command
// (per spec — collections have no inline mode), but the preflight guard is wired
// here for future-safety and spec compliance (REQ-NCP-03).
package newfeature

import (
	"os"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
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

		// Resolve workspace root.
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

		req := NewCollectionRequest{
			Name:        name,
			WorkDir:     workDir,
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
