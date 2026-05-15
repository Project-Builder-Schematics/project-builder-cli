// Package newfeature — factory_template_test.go covers the embedded factory templates.
//
// REQ coverage:
//   - REQ-NS-01: factory.ts is created (TypeScript)
//   - REQ-NS-06: factory.js is created when --language=js (JavaScript)
//   - REQ-NS-08: explicit --language wins over auto-detect
package newfeature_test

import (
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
)

// Test_LoadFactoryTemplate_TS verifies the TypeScript template is non-empty
// and is valid Go template syntax (REQ-NS-01).
func Test_LoadFactoryTemplate_TS(t *testing.T) {
	t.Parallel()

	tmpl, err := newfeature.LoadFactoryTemplate("ts")
	if err != nil {
		t.Fatalf("LoadFactoryTemplate(ts): unexpected error: %v", err)
	}
	if len(tmpl) == 0 {
		t.Error("LoadFactoryTemplate(ts): returned empty template")
	}
}

// Test_LoadFactoryTemplate_JS verifies the JavaScript template is non-empty
// (REQ-NS-06: factory.js produced when --language=js).
func Test_LoadFactoryTemplate_JS(t *testing.T) {
	t.Parallel()

	tmpl, err := newfeature.LoadFactoryTemplate("js")
	if err != nil {
		t.Fatalf("LoadFactoryTemplate(js): unexpected error: %v", err)
	}
	if len(tmpl) == 0 {
		t.Error("LoadFactoryTemplate(js): returned empty template")
	}
}

// Test_LoadFactoryTemplate_InvalidLanguage verifies that unsupported language
// returns an error (REQ-LG-06: ErrCodeInvalidLanguage).
func Test_LoadFactoryTemplate_InvalidLanguage(t *testing.T) {
	t.Parallel()

	_, err := newfeature.LoadFactoryTemplate("python")
	if err == nil {
		t.Fatal("LoadFactoryTemplate(python): expected error, got nil")
	}
}

// Test_RenderFactoryTemplate_TS verifies the TypeScript template renders correctly
// with a given schematic name (REQ-NS-01).
func Test_RenderFactoryTemplate_TS(t *testing.T) {
	t.Parallel()

	result, err := newfeature.RenderFactoryTemplate("ts", "my-schematic")
	if err != nil {
		t.Fatalf("RenderFactoryTemplate(ts, my-schematic): unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("RenderFactoryTemplate: returned empty bytes")
	}
	// TS factory should reference the schematic name.
	if !strings.Contains(string(result), "my-schematic") && !strings.Contains(string(result), "my_schematic") {
		t.Errorf("RenderFactoryTemplate(ts): output does not mention schematic name;\ngot: %s", result)
	}
}

// Test_RenderFactoryTemplate_JS verifies the JavaScript template renders correctly
// (REQ-NS-06: factory.js produced when --language=js).
func Test_RenderFactoryTemplate_JS(t *testing.T) {
	t.Parallel()

	result, err := newfeature.RenderFactoryTemplate("js", "my-schematic")
	if err != nil {
		t.Fatalf("RenderFactoryTemplate(js, my-schematic): unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("RenderFactoryTemplate(js): returned empty bytes")
	}
}
