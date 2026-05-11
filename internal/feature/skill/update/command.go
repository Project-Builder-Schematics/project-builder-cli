// Package update wires the `builder skill update` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at a future /plan phase.
package update

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder skill update`.
//
// skill update upgrades the registered skills and extensions in the current
// project workspace to their latest published versions.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update registered skills and extensions to their latest versions",
		Long: `Update upgrades the registered schematic skills and extensions in the
current project workspace to their latest published versions.

Each skill is checked against its configured registry; updates are applied
only when the new version satisfies the project's compatibility constraints.

CONTRACT:STUB — not yet implemented (planned for a future /plan phase).`,
		RunE: RunE,
	}
}
