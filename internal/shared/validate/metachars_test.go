package validate_test

import (
	"errors"
	"testing"

	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/validate"
)

func Test_RejectMetachars_CleanString_ReturnsNil(t *testing.T) {
	t.Parallel()

	clean := []string{
		"npm",
		"/usr/local/bin/npm",
		"pnpm",
		"bun",
		"yarn",
		"@pbuilder/sdk",
		"my-project",
		"1.2.3",
	}

	for _, s := range clean {
		s := s
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			if err := validate.RejectMetachars("test.op", "field", s); err != nil {
				t.Errorf("RejectMetachars(%q): expected nil, got: %v", s, err)
			}
		})
	}
}

func Test_RejectMetachars_ForbiddenChar_ReturnsInvalidInput(t *testing.T) {
	t.Parallel()

	forbidden := []struct {
		name  string
		input string
	}{
		{name: "dollar", input: "npm$inject"},
		{name: "backtick", input: "`whoami`"},
		{name: "pipe", input: "npm|sh"},
		{name: "semicolon", input: "npm;rm -rf /"},
		{name: "ampersand", input: "npm&&evil"},
		{name: "redirect", input: "npm>out"},
		{name: "newline", input: "npm\ninjected"},
		{name: "NUL", input: "npm\x00"},
		{name: "singlequote", input: "npm'"},
		{name: "doublequote", input: `npm"`},
		{name: "backslash", input: `npm\evil`},
		{name: "openParen", input: "npm("},
	}

	for _, tc := range forbidden {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validate.RejectMetachars("test.op", "field", tc.input)
			if err == nil {
				t.Errorf("RejectMetachars(%q): expected error, got nil", tc.input)
				return
			}
			sentinel := &apperrors.Error{Code: apperrors.ErrCodeInvalidInput}
			if !errors.Is(err, sentinel) {
				t.Errorf("RejectMetachars(%q): errors.Is(ErrCodeInvalidInput) = false; got: %v", tc.input, err)
			}
		})
	}
}
