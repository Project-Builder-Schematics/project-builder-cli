// Package validate wires the `builder validate` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at a future /plan phase.
package validate

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder validate`.
//
// validate checks that the current project workspace conforms to its
// schematic collection's constraints and schema definitions.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the project workspace against its schematic constraints",
		Long: `Validate checks that the current project workspace conforms to its
schematic collection's constraints, file layout rules, and schema definitions.

Exits non-zero if any violations are found; prints a structured report of
each violation with remediation hints.

CONTRACT:STUB — not yet implemented (planned for a future /plan phase).`,
		RunE: RunE,
	}
}
