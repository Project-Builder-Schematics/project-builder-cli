package theme_test

import (
	"io"
	"testing"

	"github.com/muesli/termenv"

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

// Test_MapToTermenv covers the full mapping from theme.Profile to termenv.Profile.
// Sequential — t.Setenv precludes t.Parallel().
//
// REQ output-port/REQ-05.1 (precondition): MapToTermenv must produce the correct
// termenv.Profile so composeApp can call lipgloss.SetColorProfile correctly.
func Test_MapToTermenv(t *testing.T) {
	cases := []struct {
		name string
		in   theme.Profile
		want termenv.Profile
	}{
		{"NoColor", theme.NoColor, termenv.Ascii},
		{"ANSI16", theme.ANSI16, termenv.ANSI},
		{"ANSI256", theme.ANSI256, termenv.ANSI256},
		{"TrueColor", theme.TrueColor, termenv.TrueColor},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := theme.MapToTermenv(tc.in)
			if got != tc.want {
				t.Errorf("MapToTermenv(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}
