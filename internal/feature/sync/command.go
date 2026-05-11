// Package sync wires the `builder sync` leaf command.
//
// CONTRACT:STUB — handler returns ErrCodeNotImplemented at this phase.
// Real implementation lands at a future /plan phase.
package sync

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder sync`.
//
// sync reconciles an existing project workspace against its schematic
// collection, applying updates from the collection without losing local changes.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Reconcile a project workspace against its schematic collection",
		Long: `Sync reconciles an existing project workspace against its schematic
collection, applying upstream updates without losing local customisations.

The merge strategy is defined by the schematic's conflict-resolution rules.
Dry-run output shows planned changes before applying them.

CONTRACT:STUB — not yet implemented (planned for a future /plan phase).`,
		RunE: RunE,
	}
}
