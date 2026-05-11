// Package initialise wires the `builder init` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at /plan #5 (project initialisation schematic).
package initialise

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder init`.
//
// init bootstraps a new project from a schematic collection.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialise a new project from a schematic collection",
		Long: `Initialise bootstraps a new project workspace from a schematic collection.

The schematic collection defines the project structure, files, and lifecycle
scripts to generate. Inputs are validated against the schematic's JSON schema
before execution begins.

CONTRACT:STUB — not yet implemented (planned for /plan #5).`,
		RunE: RunE,
	}
}
