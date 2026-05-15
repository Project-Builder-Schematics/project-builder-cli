// Package tsident — tsident_test.go covers the S-000a stub behaviour.
//
// S-000a ships a stub implementation: EscapeIdent returns its input unchanged
// and ReservedWords is empty. These tests assert the stub contract only.
//
// S-003 replaces both the implementation and these tests with the full
// table-driven matrix (ALL 69 reserved words + edge cases REQ-TI-01..10).
// That is expected and acceptable — stub tests are designed to be replaced.
//
// REQ-EC-07 dependency note: tsident is a shared dependency that S-000b
// and S-001..S-003 build on. This package must exist and compile before
// those slices can proceed.
package tsident_test

import (
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/tsident"
)

// Test_EscapeIdent_StubReturnsInputUnchanged verifies that the S-000a stub
// implementation returns its input string unchanged for non-reserved inputs.
// This assertion will be replaced in S-003 with the full transformation matrix.
func Test_EscapeIdent_StubReturnsInputUnchanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "simple lowercase", input: "myschematic"},
		{name: "snake_case", input: "my_schematic"},
		{name: "PascalCase", input: "MySchematic"},
		{name: "single char", input: "x"},
		{name: "with digits not leading", input: "schema1"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tsident.EscapeIdent(tt.input)
			if got != tt.input {
				t.Errorf("EscapeIdent(%q) = %q, want %q (stub must return input unchanged)", tt.input, got, tt.input)
			}
		})
	}
}

// Test_ReservedWords_StubIsEmpty verifies that the S-000a stub has an empty
// ReservedWords slice. S-003 replaces this with the full 69-word list.
func Test_ReservedWords_StubIsEmpty(t *testing.T) {
	t.Parallel()

	if len(tsident.ReservedWords) != 0 {
		t.Errorf("ReservedWords: got %d entries, want 0 (stub must be empty until S-003)", len(tsident.ReservedWords))
	}
}
