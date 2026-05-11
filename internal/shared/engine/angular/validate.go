// Package angular implements AngularSubprocessAdapter — the first concrete
// implementation of engine.Engine that spawns Node.js via os/exec.
//
// SECURITY: This package MUST NOT invoke a shell. All exec calls use
// exec.CommandContext directly with typed arguments. SchematicRef fields are
// validated before reaching cmd.Args.
package angular

import (
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
)

// validateRef validates a SchematicRef before allowing it to reach cmd.Args.
//
// S-000 skeleton: passes all refs (non-empty validation is a no-op).
// Full metacharacter + path-traversal rules are implemented in S-002.
//
// Returns *errors.Error{Code: ErrCodeInvalidInput, Op: "angular.validate_ref"}
// on violation (S-002+).
func validateRef(_ engine.SchematicRef) error {
	// S-000 skeleton: full validation (metacharacters, path traversal, NUL bytes)
	// is implemented in S-002. Returning nil here satisfies the walking-skeleton
	// acceptance criterion (FakeNode smoke test).
	return nil
}
