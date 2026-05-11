package execute

import (
	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// RunE is the leaf command's RunE for `builder execute`.
//
// CONTRACT:STUB — returns ErrCodeNotImplemented at the skeleton phase.
// Real implementation lands at /plan #4 (Angular subprocess adapter).
//
// Op: "execute.handler" — matches OpRegex ^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$
func RunE(_ *cobra.Command, _ []string) error {
	return &errs.Error{
		Code:    errs.ErrCodeNotImplemented,
		Op:      "execute.handler",
		Message: "execute not yet implemented (planned for /plan #4)",
	}
}
