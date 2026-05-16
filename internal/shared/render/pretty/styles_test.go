// Package pretty_test — styles_test.go covers Styles struct vocabulary.
//
// REQ render-pretty/02.1 — Styles has exactly 8 semantic fields of type lipgloss.Style
package pretty_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"
)

// Test_Styles_HasEightSemanticFields_ByReflection asserts via reflection that
// pretty.Styles exposes exactly 8 exported fields, all of type lipgloss.Style,
// named exactly: Primary, Accent, Foreground, Muted, Background, Success,
// Warning, Error. This is the architectural gate against vocabulary drift
// (render-pretty/REQ-02.1).
func Test_Styles_HasEightSemanticFields_ByReflection(t *testing.T) {
	t.Parallel()

	wantNames := []string{
		"Accent",
		"Background",
		"Error",
		"Foreground",
		"Muted",
		"Primary",
		"Success",
		"Warning",
	}

	stylesType := reflect.TypeOf(pretty.Styles{})

	// Collect all exported fields.
	var gotNames []string
	for i := range stylesType.NumField() {
		f := stylesType.Field(i)
		if !f.IsExported() {
			continue
		}
		// Each exported field must be exactly lipgloss.Style.
		if f.Type != reflect.TypeOf(lipgloss.Style{}) {
			t.Errorf("field %s: expected type lipgloss.Style, got %v", f.Name, f.Type)
		}
		gotNames = append(gotNames, f.Name)
	}

	sort.Strings(gotNames)

	if len(gotNames) != len(wantNames) {
		t.Fatalf("Styles has %d exported fields, want %d\ngot:  %v\nwant: %v",
			len(gotNames), len(wantNames), gotNames, wantNames)
	}

	for i, name := range wantNames {
		if gotNames[i] != name {
			t.Errorf("field[%d]: got %q, want %q", i, gotNames[i], name)
		}
	}
}
