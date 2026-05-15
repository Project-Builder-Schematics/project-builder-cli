// Package main — exit_test.go covers the exitCodeForErr helper (S-006 expansion).
//
// exitCodeForErr maps an error from fang.Execute to a process exit code:
//   - nil → 0 (success; helper is never called for nil, but defensive)
//   - *errs.Error (direct or wrapped) → 2 (structured user-facing error)
//   - any other error → 1 (unexpected / infrastructure error)
//
// REQ-NS-02, REQ-NS-04, REQ-NS-07, REQ-NSI-02, REQ-NCP-03, REQ-EX-02, REQ-EX-03,
// REQ-LG-06, REQ-NC-02, REQ-NC-05, REQ-EC-01..06 all require exit code 2 for
// structured errors. REQ-EC compliance moves from PARTIAL to COMPLIANT after this
// fix is wired into main().
package main

import (
	"errors"
	"fmt"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// Test_ExitCodeForErr covers all input variants of exitCodeForErr.
func Test_ExitCodeForErr(t *testing.T) {
	t.Parallel()

	structuredErr := &errs.Error{
		Code:    errs.ErrCodeNewSchematicExists,
		Op:      "new.handler",
		Message: "schematic 'foo' already exists",
	}

	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			name:     "nil error returns 0",
			err:      nil,
			wantCode: 0,
		},
		{
			name:     "generic errors.New returns 1",
			err:      errors.New("something went wrong"),
			wantCode: 1,
		},
		{
			name:     "direct *errs.Error returns 2",
			err:      structuredErr,
			wantCode: 2,
		},
		{
			name:     "wrapped *errs.Error via fmt.Errorf returns 2",
			err:      fmt.Errorf("handler: %w", structuredErr),
			wantCode: 2,
		},
		{
			name:     "double-wrapped *errs.Error returns 2",
			err:      fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", structuredErr)),
			wantCode: 2,
		},
		{
			name: "*errs.Error with ErrCodeInvalidSchematicName returns 2",
			err: &errs.Error{
				Code:    errs.ErrCodeInvalidSchematicName,
				Op:      "new.handler",
				Message: "invalid name",
			},
			wantCode: 2,
		},
		{
			name: "*errs.Error with ErrCodeModeConflict returns 2",
			err: &errs.Error{
				Code:    errs.ErrCodeModeConflict,
				Op:      "new.handler",
				Message: "mode conflict",
			},
			wantCode: 2,
		},
		{
			name: "*errs.Error with ErrCodeInvalidExtends returns 2",
			err: &errs.Error{
				Code:    errs.ErrCodeInvalidExtends,
				Op:      "new.handler",
				Message: "invalid extends",
			},
			wantCode: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := exitCodeForErr(tc.err)
			if got != tc.wantCode {
				t.Errorf("exitCodeForErr(%v) = %d; want %d", tc.err, got, tc.wantCode)
			}
		})
	}
}
