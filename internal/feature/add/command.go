// Package add wires the `builder add` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at a future /plan phase.
package add

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder add`.
//
// add generates a new artefact (component, module, service) within an
// existing project workspace using a schematic.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new artefact to an existing project workspace",
		Long: `Add generates a new artefact (component, module, service) within an
existing project workspace by running a schematic.

The schematic is resolved from the project's configured collection; inputs
are validated against its JSON schema before any file changes occur.

CONTRACT:STUB — not yet implemented (planned for a future /plan phase).`,
		RunE: RunE,
	}
}
