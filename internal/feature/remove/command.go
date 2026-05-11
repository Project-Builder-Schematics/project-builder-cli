// Package remove wires the `builder remove` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at a future /plan phase.
package remove

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder remove`.
//
// remove deletes a generated artefact (component, module, service) from the
// project workspace, reversing the effects of a prior `add` invocation.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a generated artefact from the project workspace",
		Long: `Remove deletes a generated artefact (component, module, service) from the
project workspace by reversing the file changes produced by a prior add.

Only artefacts tracked in the workspace manifest can be removed. Untracked
files are not touched.

CONTRACT:STUB — not yet implemented (planned for a future /plan phase).`,
		RunE: RunE,
	}
}
