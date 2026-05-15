// Package newfeature — service_publishable.go implements publishable collection
// scaffolding for `builder new collection <name> --publishable`.
//
// REQ coverage:
//   - REQ-NCP-01: lifecycle stubs generated (add/ + remove/ with factory.ts + schema.json + schema.d.ts)
//   - REQ-NCP-02: --force allows overwrite of existing publishable collection
//   - REQ-NCP-03: --publishable + --inline → ErrCodeModeConflict
package newfeature

import (
	"context"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// createPublishableCollection implements the --publishable collection creation flow.
// It writes collection.json + add/ and remove/ lifecycle stubs, then mutates
// project-builder.json with the collection path.
func (s *Service) createPublishableCollection(_ context.Context, _ NewCollectionRequest) (*NewResult, error) {
	// Stub — implementation follows in Task E (RED phase keeps it as not-implemented).
	return nil, &errs.Error{
		Code:    errs.ErrCodeNewNotImplemented,
		Op:      "new.createPublishableCollection",
		Message: "new collection --publishable: not yet implemented in this build",
	}
}
