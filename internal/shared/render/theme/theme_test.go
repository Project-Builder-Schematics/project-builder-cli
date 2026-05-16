package theme_test

import (
	"io"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// Test_Default_ReturnsNoColorTheme verifies that theme.Default returns a
// non-zero Theme whose internal profile is NoColor (S-000 skeleton acceptance).
//
// REQ render-pretty/REQ-01.1 (partial): theme injection compiles end-to-end.
func Test_Default_ReturnsNoColorTheme(t *testing.T) {
	t.Parallel()

	th, err := theme.Default(io.Discard, "", "")
	if err != nil {
		t.Fatalf("theme.Default returned error: %v", err)
	}

	if th.Profile() != theme.NoColor {
		t.Errorf("Default theme profile = %v; want NoColor", th.Profile())
	}

	if th.Appearance() != theme.Light {
		t.Errorf("Default theme appearance = %v; want Light", th.Appearance())
	}
}
