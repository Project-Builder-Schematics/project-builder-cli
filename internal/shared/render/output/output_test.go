package output_test

import (
	"reflect"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
)

// Test_Output_DeclaresTenMethods_ByReflection covers output-port/REQ-01.1.
//
// GIVEN the output package compiled at HEAD
// WHEN reflect.TypeOf((*output.Output)(nil)).Elem() is inspected
// THEN exactly the ten method names are present, no more, no less.
func Test_Output_DeclaresTenMethods_ByReflection(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf((*output.Output)(nil)).Elem()

	wantMethods := []string{
		"Heading",
		"Body",
		"Hint",
		"Success",
		"Warning",
		"Error",
		"Path",
		"Prompt",
		"Newline",
		"Stream",
	}

	if typ.NumMethod() != len(wantMethods) {
		t.Errorf("output.Output has %d methods; want %d\nmethods present: %v",
			typ.NumMethod(), len(wantMethods), methodNames(typ))
	}

	for _, name := range wantMethods {
		if _, ok := typ.MethodByName(name); !ok {
			t.Errorf("output.Output missing method %q (REQ-01.1)", name)
		}
	}
}

// methodNames returns the list of method names on an interface type —
// used for diagnostic output only.
func methodNames(typ reflect.Type) []string {
	names := make([]string, typ.NumMethod())
	for i := range typ.NumMethod() {
		names[i] = typ.Method(i).Name
	}
	return names
}
