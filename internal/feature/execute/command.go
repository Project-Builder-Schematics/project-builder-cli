// Package execute wires the `builder execute` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at /plan #4 (Angular subprocess adapter).
package execute

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder execute`.
//
// execute runs a named schematic against an existing project workspace.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "execute",
		Short: "Execute a schematic against an existing project workspace",
		Long: `Execute runs a named schematic against an existing project workspace.

The schematic is resolved from the project's configured collection, inputs are
validated against its JSON schema, and execution is streamed as a live event
channel to the active renderer.

CONTRACT:STUB — not yet implemented (planned for /plan #4).`,
		RunE: RunE,
	}
}
