// Package pathutil — canonicalise_test.go covers Canonicalise.
//
// REQ coverage:
//   - REQ-DV-01 (directory canonicalised: filepath.Abs + filepath.Clean)
//   - REQ-DV-02 (reject path with .. traversal outside cwd)
package pathutil

import (
	"errors"
	"path/filepath"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// Test_Canonicalise_AbsoluteAndClean verifies that Canonicalise returns an
// absolute, clean path for valid inputs. REQ-DV-01.
func Test_Canonicalise_AbsoluteAndClean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantAbs bool
	}{
		{name: "absolute path", input: "/tmp/my-project", wantAbs: true},
		{name: "temp dir", input: t.TempDir(), wantAbs: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Canonicalise(tt.input)
			if err != nil {
				t.Fatalf("Canonicalise(%q): unexpected error: %v", tt.input, err)
			}
			if tt.wantAbs && !filepath.IsAbs(got) {
				t.Errorf("Canonicalise(%q) = %q: expected absolute path", tt.input, got)
			}
			if got != filepath.Clean(got) {
				t.Errorf("Canonicalise(%q) = %q: not Clean", tt.input, got)
			}
		})
	}
}

// Test_Canonicalise_TraversalOutsideCwd_Rejected verifies that paths
// containing .. that resolve outside the cwd are rejected. REQ-DV-02.
func Test_Canonicalise_TraversalOutsideCwd_Rejected(t *testing.T) {
	t.Parallel()

	_, err := Canonicalise("../../../etc")
	if err == nil {
		t.Fatal("Canonicalise with .. traversal: expected error, got nil (REQ-DV-02)")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInvalidInput}
	if !errors.Is(err, sentinel) {
		t.Errorf("Canonicalise traversal error: errors.Is(ErrCodeInvalidInput) = false; got: %v", err)
	}

	var e *errs.Error
	if errors.As(err, &e) {
		if e.Op != "pathutil.Canonicalise" {
			t.Errorf("error Op = %q, want %q", e.Op, "pathutil.Canonicalise")
		}
	}
}
