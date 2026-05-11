// Package discoverer_test covers the walking-skeleton subset of Discoverer.
//
// S-000 scope: FindNode() via NODE_BINARY env var only.
// Full priority chain + version validation tests live in S-004.
package discoverer_test

import (
	"os"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/discoverer"
	appErrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// Test_FindNode_NodeBinary_Set covers the S-000 skeleton path: when NODE_BINARY
// is set to a real executable, FindNode returns that path with nil error.
func Test_FindNode_NodeBinary_Set(t *testing.T) {
	// t.Setenv requires no t.Parallel — it mutates process env and Go 1.21+
	// panics if t.Parallel and t.Setenv are combined.

	// Use a real executable that exists on every Linux/macOS system.
	// /bin/sh is always present and executable; it is not Node but it is
	// discoverable. The skeleton only checks NODE_BINARY — version validation
	// is S-004's responsibility.
	const fakeBin = "/bin/sh"
	if _, err := os.Stat(fakeBin); err != nil {
		t.Skipf("test requires %s to exist: %v", fakeBin, err)
	}

	t.Setenv("NODE_BINARY", fakeBin)

	d := discoverer.New()
	got, err := d.FindNode()
	if err != nil {
		t.Fatalf("FindNode() returned unexpected error: %v", err)
	}
	if got == "" {
		t.Fatal("FindNode() returned empty path")
	}
}

// Test_FindNode_NodeBinary_Empty_ReturnsError covers the skeleton: when
// NODE_BINARY is not set and no full discovery chain exists yet (S-004),
// FindNode returns a structured *errors.Error with ErrCodeEngineNotFound.
//
// This test unsets NODE_BINARY to isolate the skeleton error path.
// The full priority chain (PATH lookup + well-known paths) is wired in S-004.
func Test_FindNode_NodeBinary_Empty_ReturnsError(t *testing.T) {
	// t.Setenv requires no t.Parallel.

	// Unset NODE_BINARY to trigger the not-found path in the skeleton.
	t.Setenv("NODE_BINARY", "")

	d := discoverer.New()
	_, err := d.FindNode()
	if err == nil {
		t.Fatal("FindNode() expected non-nil error when NODE_BINARY is empty")
	}

	var appErr *appErrors.Error
	if !asError(err, &appErr) {
		t.Fatalf("FindNode() error is %T, want *errors.Error; got: %v", err, err)
	}
	if appErr.Code != appErrors.ErrCodeEngineNotFound {
		t.Errorf("FindNode() error code = %q, want %q", appErr.Code, appErrors.ErrCodeEngineNotFound)
	}
}

// asError is a local helper that mimics errors.As for *appErrors.Error,
// avoiding an import of the standard "errors" package name clash.
func asError(err error, target **appErrors.Error) bool {
	for err != nil {
		if e, ok := err.(*appErrors.Error); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}
