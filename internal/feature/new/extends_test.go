// Package newfeature — extends_test.go covers --extends flag grammar validation
// and TTY detection behaviour.
//
// REQ coverage:
//   - REQ-EX-01: valid grammar @scope/pkg:base accepted
//   - REQ-EX-02: path traversal rejected (contains ../ or ./ or no @)
//   - REQ-EX-03: malformed grammar rejected (missing @, missing :, spaces)
//   - REQ-EX-04: TUI prompt shown when interactive + flag absent (stub/seam test)
//   - REQ-EX-05: non-interactive without flag → skip extends silently
//   - ADV-04: path traversal in --extends rejected
package newfeature_test

import (
	"errors"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// Test_ValidateExtendsGrammar_Valid verifies that well-formed @scope/pkg:collection
// values are accepted without error (REQ-EX-01).
func Test_ValidateExtendsGrammar_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
	}{
		{"simple", "@my-org/my-pkg:base"},
		{"underscores", "@my_org/my_pkg:my_collection"},
		{"numbers", "@org123/pkg456:col789"},
		{"hyphens in all parts", "@org-name/pkg-name:col-name"},
		{"alphanumeric scope", "@Org/Pkg:Col"},
		{"mixed case", "@My-Org/My-Pkg:Base"},
		{"single chars", "@a/b:c"},
		{"with numbers only in parts", "@123/456:789"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := newfeature.ValidateExtendsGrammar(tc.value); err != nil {
				t.Errorf("ValidateExtendsGrammar(%q): unexpected error: %v", tc.value, err)
			}
		})
	}
}

// Test_ValidateExtendsGrammar_PathTraversal verifies that values containing
// path traversal sequences are rejected (REQ-EX-02 / ADV-04).
func Test_ValidateExtendsGrammar_PathTraversal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
	}{
		{"relative up traversal", "../evil:base"},
		{"relative down traversal", "./evil:base"},
		{"traversal with @", "@../evil:base"},
		{"embedded traversal", "@scope/../evil:base"},
		{"absolute path", "/etc/passwd"},
		{"windows traversal", `@scope\evil:base`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertExtendsError(t, tc.value, errs.ErrCodeInvalidExtends)
		})
	}
}

// Test_ValidateExtendsGrammar_MalformedGrammar verifies that values not matching
// the required grammar are rejected with ErrCodeInvalidExtends (REQ-EX-03).
func Test_ValidateExtendsGrammar_MalformedGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
	}{
		{"missing @", "scope/pkg:base"},
		{"missing colon", "@scope/pkg"},
		{"missing slash", "@scopepkg:base"},
		{"spaces in value", "@scope / pkg : base"},
		{"empty string after @", "@/pkg:base"},
		{"empty pkg", "@scope/:base"},
		{"empty collection", "@scope/pkg:"},
		{"only @", "@"},
		{"special chars in scope", "@scope!name/pkg:base"},
		{"dot in scope", "@scope.name/pkg:base"},
		{"double colon", "@scope/pkg::base"},
		{"double at", "@@scope/pkg:base"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertExtendsError(t, tc.value, errs.ErrCodeInvalidExtends)
		})
	}
}

// Test_ValidateExtendsGrammar_ErrorMessage verifies the error message contains
// the invalid value and a format hint (REQ-EC-04 message contract).
func Test_ValidateExtendsGrammar_ErrorMessage(t *testing.T) {
	t.Parallel()

	value := "bad-extends-value"
	err := newfeature.ValidateExtendsGrammar(value)
	if err == nil {
		t.Fatal("expected error for malformed extends; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeInvalidExtends {
		t.Errorf("code = %q; want %q", e.Code, errs.ErrCodeInvalidExtends)
	}
	// Message must mention the bad value and the expected format (REQ-EC-04).
	if !containsAll(e.Message, value, "@") {
		t.Errorf("error message %q does not mention value %q or format hint", e.Message, value)
	}
}

// Test_IsInteractiveTTY_NonTTY verifies that IsInteractiveTTY returns false
// when stdin is not a TTY (REQ-EX-05 non-interactive path).
func Test_IsInteractiveTTY_NonTTY(t *testing.T) {
	t.Parallel()

	// In a test process, stdin is never a real TTY.
	// So IsInteractiveTTY() should return false.
	if newfeature.IsInteractiveTTY() {
		t.Error("IsInteractiveTTY(): expected false in test environment (not a real TTY)")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// assertExtendsError asserts that ValidateExtendsGrammar returns an *errs.Error
// with the expected code for the given value.
func assertExtendsError(t *testing.T, value string, wantCode errs.ErrCode) {
	t.Helper()
	err := newfeature.ValidateExtendsGrammar(value)
	if err == nil {
		t.Fatalf("ValidateExtendsGrammar(%q): expected error with code %q; got nil", value, wantCode)
	}
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("ValidateExtendsGrammar(%q): error not *errs.Error; got: %T %v", value, err, err)
	}
	if e.Code != wantCode {
		t.Errorf("ValidateExtendsGrammar(%q): code = %q; want %q", value, e.Code, wantCode)
	}
}
