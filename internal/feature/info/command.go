// Package info wires the `builder info` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at a future /plan phase.
package info

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder info`.
//
// info displays metadata about the current project workspace and its
// configured schematic collection.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Display metadata about the current project and schematic collection",
		Long: `Info displays metadata about the current project workspace and its
configured schematic collection, including available schematics, version
pins, and workspace root detection.

CONTRACT:STUB — not yet implemented (planned for a future /plan phase).`,
		RunE: RunE,
	}
}
