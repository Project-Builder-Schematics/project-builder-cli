// Package skill wires the `builder skill` parent command and registers its
// `update` sub-command.
//
// CONTRACT:STUB — skill has no standalone handler; invoked with no args it
// prints help and returns nil (cobra-command-tree.REQ-02.1). The `update`
// child carries the real (stub) RunE.
package skill

import (
	"github.com/spf13/cobra"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/skill/update"
)

// NewCommand returns the Cobra parent command for `builder skill`.
//
// skill is a command group for skill-management operations. Invoked with no
// sub-command (or --help), it prints usage and exits 0.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage schematic skills and extensions",
		Long: `Skill provides a set of sub-commands for managing schematic skills and
extensions registered in the current project workspace.

Run 'builder skill --help' to see available sub-commands.

CONTRACT:STUB — sub-command implementations are planned for future /plan phases.`,
		// RunE prints help when invoked with no sub-command, then returns nil
		// so the process exits 0 (cobra-command-tree.REQ-02.1).
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(update.NewCommand())

	return cmd
}
