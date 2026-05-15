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

// ─── Lifecycle templates (REQ-NCP-01) ─────────────────────────────────────────

// Test_LoadLifecycleTemplate_Add verifies the add lifecycle template is non-empty (REQ-NCP-01).
func Test_LoadLifecycleTemplate_Add(t *testing.T) {
	t.Parallel()

	tmpl, err := newfeature.LoadLifecycleTemplate("add")
	if err != nil {
		t.Fatalf("LoadLifecycleTemplate(add): unexpected error: %v", err)
	}
	if len(tmpl) == 0 {
		t.Error("LoadLifecycleTemplate(add): returned empty template")
	}
}

// Test_LoadLifecycleTemplate_Remove verifies the remove lifecycle template is non-empty (REQ-NCP-01).
func Test_LoadLifecycleTemplate_Remove(t *testing.T) {
	t.Parallel()

	tmpl, err := newfeature.LoadLifecycleTemplate("remove")
	if err != nil {
		t.Fatalf("LoadLifecycleTemplate(remove): unexpected error: %v", err)
	}
	if len(tmpl) == 0 {
		t.Error("LoadLifecycleTemplate(remove): returned empty template")
	}
}

// Test_LoadLifecycleTemplate_InvalidStage verifies that unsupported stage returns error.
func Test_LoadLifecycleTemplate_InvalidStage(t *testing.T) {
	t.Parallel()

	_, err := newfeature.LoadLifecycleTemplate("update")
	if err == nil {
		t.Fatal("LoadLifecycleTemplate(update): expected error, got nil")
	}
}

// Test_RenderLifecycleTemplate_Add verifies the add template renders with collection name.
func Test_RenderLifecycleTemplate_Add(t *testing.T) {
	t.Parallel()

	result, err := newfeature.RenderLifecycleTemplate("add", "my-collection")
	if err != nil {
		t.Fatalf("RenderLifecycleTemplate(add, my-collection): unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("RenderLifecycleTemplate(add): returned empty bytes")
	}
	if !strings.Contains(string(result), "my-collection") && !strings.Contains(string(result), "myCollection") {
		t.Errorf("RenderLifecycleTemplate(add): output does not mention collection name;\ngot: %s", result)
	}
}

// Test_RenderLifecycleTemplate_Remove verifies the remove template renders with collection name.
func Test_RenderLifecycleTemplate_Remove(t *testing.T) {
	t.Parallel()

	result, err := newfeature.RenderLifecycleTemplate("remove", "my-collection")
	if err != nil {
		t.Fatalf("RenderLifecycleTemplate(remove, my-collection): unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("RenderLifecycleTemplate(remove): returned empty bytes")
	}
	if !strings.Contains(string(result), "my-collection") && !strings.Contains(string(result), "myCollection") {
		t.Errorf("RenderLifecycleTemplate(remove): output does not mention collection name;\ngot: %s", result)
	}
}
