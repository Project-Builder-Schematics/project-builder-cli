package add

import (
	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// RunE is the leaf command's RunE for `builder add`.
//
// CONTRACT:STUB — returns ErrCodeNotImplemented at the skeleton phase.
// Real implementation lands at a future /plan phase.
//
// Op: "add.handler" — matches OpRegex ^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$
func RunE(_ *cobra.Command, _ []string) error {
	return &errs.Error{
		Code:    errs.ErrCodeNotImplemented,
		Op:      "add.handler",
		Message: "add not yet implemented (planned for a future /plan phase)",
	}
}
