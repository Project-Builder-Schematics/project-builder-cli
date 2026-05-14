// Package angular implements AngularSubprocessAdapter — the first concrete
// implementation of engine.Engine that spawns Node.js via os/exec.
//
// SECURITY: This package MUST NOT invoke a shell. All exec calls use
// exec.CommandContext directly with typed arguments. SchematicRef fields are
// validated before reaching cmd.Args.
package angular

import (
	"strings"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/validate"
)

// validateRef validates a SchematicRef before allowing it to reach cmd.Args.
//
// Rules enforced (REQ-05):
//  1. Name MUST NOT contain ".." — no path traversal (REQ-02.2, REQ-05)
//  2. Name MUST NOT contain "/" — collection-relative resolution only (REQ-05.2)
//  3. All fields MUST NOT contain shell metacharacters: $ ` ( ) { } | ; & > < \ " ' \n \r NUL (REQ-02.3, REQ-05.3)
//  4. Collection MUST NOT contain ".." — no path traversal
//
// Returns *errors.Error{Code: ErrCodeInvalidInput, Op: "angular.validate_ref"}
// on any violation. Returns nil if the ref is valid.
func validateRef(ref engine.SchematicRef) error {
	// REQ-05.2, REQ-02.2: Name must not contain "/" (absolute path) or ".." (traversal).
	if strings.Contains(ref.Name, "/") {
		return &apperrors.Error{
			Code:    apperrors.ErrCodeInvalidInput,
			Op:      "angular.validate_ref",
			Message: "SchematicRef.Name must not contain '/' — use a collection-relative name",
		}
	}
	if strings.Contains(ref.Name, "..") {
		return &apperrors.Error{
			Code:    apperrors.ErrCodeInvalidInput,
			Op:      "angular.validate_ref",
			Message: "SchematicRef.Name must not contain '..' — path traversal is not permitted",
		}
	}

	// REQ-02.2 (collection): path traversal in Collection is also forbidden.
	if strings.Contains(ref.Collection, "..") {
		return &apperrors.Error{
			Code:    apperrors.ErrCodeInvalidInput,
			Op:      "angular.validate_ref",
			Message: "SchematicRef.Collection must not contain '..' — path traversal is not permitted",
		}
	}

	// REQ-02.3, REQ-05.3: forbid shell metacharacters and NUL byte in all fields.
	// Delegates to validate.RejectMetachars (promoted from this package in S-005 prep).
	if err := validate.RejectMetachars("angular.validate_ref", "SchematicRef.Collection", ref.Collection); err != nil {
		return err
	}
	if err := validate.RejectMetachars("angular.validate_ref", "SchematicRef.Name", ref.Name); err != nil {
		return err
	}
	if err := validate.RejectMetachars("angular.validate_ref", "SchematicRef.Version", ref.Version); err != nil {
		return err
	}

	return nil
}
