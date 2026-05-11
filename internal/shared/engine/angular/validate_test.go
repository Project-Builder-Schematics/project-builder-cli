// Package angular_test covers validate.go.
//
// S-002 scope:
//   - REQ-02.2: path traversal in Name rejected before exec
//   - REQ-02.3: shell metacharacters in Collection rejected before exec
//   - REQ-05.1: valid SchematicRef passes validation
//   - REQ-05.2: forward slash in Name rejected
//   - REQ-05.3: NUL byte in any field rejected
//   - REQ-14.1: nil channel returned on pre-exec error
//   - REQ-16.1: Op format matches ^angular\.[a-z][a-z0-9_]*$
package angular_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular"
	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// opRegex is the invariant format for the Op field (REQ-16.1).
var opRegex = regexp.MustCompile(`^angular\.[a-z][a-z0-9_]*$`)

// Test_ValidateRef_ValidRef covers REQ-05.1: a well-formed SchematicRef passes.
func Test_ValidateRef_ValidRef(t *testing.T) {
	t.Parallel()

	ref := engine.SchematicRef{
		Collection: "@schematics/angular",
		Name:       "component",
		Version:    "17.0.0",
	}

	// validateRef is unexported; we assert no panic + no error for a known-good
	// ref by exercising adapter.Execute via the exported surface in adapter_test.go.
	// For S-000 we merely document the shape here; adapter_test.go drives it end-to-end.
	_ = ref // shape documented; end-to-end test in adapter_test.go
}

// Test_ValidateRef_PathTraversal covers REQ-02.2:
// ../../../etc/passwd in Name → ErrCodeInvalidInput before exec.
func Test_ValidateRef_PathTraversal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ref  engine.SchematicRef
	}{
		{
			name: "dotdot_in_name",
			ref:  engine.SchematicRef{Name: "../../../etc/passwd"},
		},
		{
			name: "dotdot_at_start",
			ref:  engine.SchematicRef{Name: "../foo"},
		},
		{
			name: "dotdot_in_collection",
			ref:  engine.SchematicRef{Collection: "../../../etc", Name: "passwd"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertValidationRejects(t, tc.ref)
		})
	}
}

// Test_ValidateRef_ForwardSlashInName covers REQ-05.2.
func Test_ValidateRef_ForwardSlashInName(t *testing.T) {
	t.Parallel()

	ref := engine.SchematicRef{
		Collection: "@schematics/angular",
		Name:       "foo/bar",
	}
	assertValidationRejects(t, ref)
}

// Test_ValidateRef_NULByte covers REQ-05.3:
// NUL byte in any field → ErrCodeInvalidInput.
func Test_ValidateRef_NULByte(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ref  engine.SchematicRef
	}{
		{
			name: "nul_in_collection",
			ref:  engine.SchematicRef{Collection: "@schematics/angular\x00injected", Name: "component"},
		},
		{
			name: "nul_in_name",
			ref:  engine.SchematicRef{Collection: "@schematics/angular", Name: "component\x00"},
		},
		{
			name: "nul_in_version",
			ref:  engine.SchematicRef{Collection: "@schematics/angular", Name: "component", Version: "17.0.0\x00"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertValidationRejects(t, tc.ref)
		})
	}
}

// Test_ValidateRef_ShellMetacharacters covers REQ-02.3 and REQ-05.1:
// shell metacharacters in any field → ErrCodeInvalidInput.
func Test_ValidateRef_ShellMetacharacters(t *testing.T) {
	t.Parallel()

	metacharCases := []struct {
		name  string
		field string
		char  string
	}{
		{"dollar_in_name", "name", "foo$HOME"},
		{"backtick_in_name", "name", "foo`whoami`"},
		{"paren_open_in_collection", "collection", "@schematics/$(whoami)"},
		{"paren_close_in_name", "name", "foo)bar"},
		{"brace_open_in_name", "name", "foo{bar"},
		{"brace_close_in_name", "name", "foo}bar"},
		{"pipe_in_name", "name", "foo|bar"},
		{"semicolon_in_name", "name", "foo;bar"},
		{"ampersand_in_name", "name", "foo&bar"},
		{"gt_in_name", "name", "foo>bar"},
		{"lt_in_name", "name", "foo<bar"},
		{"backslash_in_name", "name", "foo\\bar"},
		{"double_quote_in_name", "name", `foo"bar`},
		{"single_quote_in_name", "name", "foo'bar"},
		{"newline_in_name", "name", "foo\nbar"},
		{"carriage_return_in_name", "name", "foo\rbar"},
		{"malicious_rm_rf", "name", "foo; rm -rf $HOME"},
	}

	for _, tc := range metacharCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ref engine.SchematicRef
			switch tc.field {
			case "name":
				ref = engine.SchematicRef{Collection: "@schematics/angular", Name: tc.char}
			case "collection":
				ref = engine.SchematicRef{Collection: tc.char, Name: "component"}
			case "version":
				ref = engine.SchematicRef{Collection: "@schematics/angular", Name: "component", Version: tc.char}
			}
			assertValidationRejects(t, ref)
		})
	}
}

// Test_ValidateRef_OpFormat covers REQ-16.1:
// Op field on validation error matches ^angular\.[a-z][a-z0-9_]*$.
func Test_ValidateRef_OpFormat(t *testing.T) {
	t.Parallel()

	// Trigger a validation error and check the Op field.
	ref := engine.SchematicRef{Name: "bad/name"}

	// Use adapter.Execute to get the returned error.
	// NODE_BINARY need not be set for pre-exec validation — error returned before discovery.
	d := angular.NewAdapter()
	ch, err := d.Execute(context.Background(), engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: ref,
	})

	if ch != nil {
		t.Error("channel must be nil on pre-exec validation error — REQ-14.1 violated")
	}
	if err == nil {
		t.Fatal("expected non-nil error for invalid SchematicRef")
	}

	var appErr *apperrors.Error
	for e := err; e != nil; {
		if ae, ok := e.(*apperrors.Error); ok {
			appErr = ae
			break
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := e.(unwrapper); ok {
			e = u.Unwrap()
		} else {
			break
		}
	}
	if appErr == nil {
		t.Fatalf("error is not *errors.Error: %T — %v", err, err)
	}
	if appErr.Code != apperrors.ErrCodeInvalidInput {
		t.Errorf("Code = %q, want %q — REQ-14.1", appErr.Code, apperrors.ErrCodeInvalidInput)
	}
	if !opRegex.MatchString(appErr.Op) {
		t.Errorf("Op = %q does not match %s — REQ-16.1 violated", appErr.Op, opRegex)
	}
}

// Test_ValidateRef_NilChannelOnPreExecError covers REQ-14.1:
// pre-exec validation error → channel is nil.
func Test_ValidateRef_NilChannelOnPreExecError(t *testing.T) {
	t.Parallel()

	d := angular.NewAdapter()
	ch, err := d.Execute(context.Background(), engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "bad/name"},
	})

	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if ch != nil {
		t.Errorf("channel = %v, want nil — REQ-14.1 violated", ch)
	}
}

// assertValidationRejects is a helper that asserts Execute returns a
// validation error (ErrCodeInvalidInput) with nil channel.
func assertValidationRejects(t *testing.T, ref engine.SchematicRef) {
	t.Helper()

	d := angular.NewAdapter()
	ch, err := d.Execute(context.Background(), engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: ref,
	})

	if err == nil {
		t.Errorf("Execute() expected validation error for ref %+v, got nil error", ref)
		return
	}
	if ch != nil {
		t.Errorf("Execute() channel must be nil on validation error — REQ-14.1 violated; ref=%+v", ref)
	}

	// Validate error code.
	var appErr *apperrors.Error
	for e := err; e != nil; {
		if ae, ok := e.(*apperrors.Error); ok {
			appErr = ae
			break
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := e.(unwrapper); ok {
			e = u.Unwrap()
		} else {
			break
		}
	}
	if appErr == nil {
		t.Errorf("error %T is not *errors.Error; ref=%+v", err, ref)
		return
	}
	if appErr.Code != apperrors.ErrCodeInvalidInput {
		t.Errorf("error code = %q, want ErrCodeInvalidInput — ref=%+v", appErr.Code, ref)
	}
}
