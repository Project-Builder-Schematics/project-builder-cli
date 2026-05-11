package update

import (
	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// RunE is the leaf command's RunE for `builder skill update`.
//
// CONTRACT:STUB — returns ErrCodeNotImplemented at the skeleton phase.
// Real implementation lands at a future /plan phase.
//
// Op: "skill_update.handler" — underscore separates compound feature name;
// matches OpRegex ^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$
func RunE(_ *cobra.Command, _ []string) error {
	return &errs.Error{
		Code:    errs.ErrCodeNotImplemented,
		Op:      "skill_update.handler",
		Message: "skill update not yet implemented (planned for a future /plan phase)",
	}
}
