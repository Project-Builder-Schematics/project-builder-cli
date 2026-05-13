// Package initialise — pkgmanager_stub.go provides a minimal PackageManagerRunner
// stub for S-000. The real implementation lands in S-005 (pkgmanager.go).
//
// This file exists so composeApp can wire NewService without a build error.
// The stub Detect and Install methods are never called in S-000 because:
//   - dry-run mode skips Install
//   - S-000 does not call Detect (PM detection is a S-005 concern)
package initialise

import (
	"context"
	"fmt"
)

// realPM is the S-000 stub implementation of PackageManagerRunner.
// Replace with the real implementation in S-005 (pkgmanager.go).
type realPM struct{}

// NewRealPM returns the PackageManagerRunner for use by composeApp.
// In S-000 this is a stub; it is replaced by the real implementation in S-005.
func NewRealPM() PackageManagerRunner { return &realPM{} }

// Detect returns PMNpm as a safe default stub.
// Real detection logic (lockfile scan + flag precedence) lands in S-005.
func (r *realPM) Detect(_ string, flag PackageManager) (PackageManager, error) {
	if flag != PMUnset {
		return flag, nil
	}
	return PMNpm, nil
}

// Install always returns an error in S-000 — it should never be called
// in dry-run mode. If it is called, the error makes the violation explicit.
func (r *realPM) Install(_ context.Context, _ string, pm PackageManager) error {
	return fmt.Errorf("realPM.Install called in S-000 stub for %s — install subprocess not yet implemented (S-005)", pm)
}
