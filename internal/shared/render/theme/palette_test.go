package theme_test

import (
	"reflect"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// Test_DefaultPalette_ExposesEightTokens asserts the Palette struct has exactly
// 8 exported fields, named and typed per the design contract (theme-tokens/REQ-01.1).
func Test_DefaultPalette_ExposesEightTokens(t *testing.T) {
	t.Parallel()

	wantNames := []string{
		"Primary", "Accent", "Foreground", "Muted",
		"Background", "Success", "Warning", "Error",
	}

	typ := reflect.TypeOf(theme.Palette{})
	if typ.NumField() != len(wantNames) {
		t.Fatalf("Palette has %d fields, want %d", typ.NumField(), len(wantNames))
	}

	for i, want := range wantNames {
		f := typ.Field(i)
		if f.Name != want {
			t.Errorf("field[%d]: got %q, want %q", i, f.Name, want)
		}
		if f.Type != reflect.TypeOf(theme.Hex{}) {
			t.Errorf("field[%d] %q: type = %v, want theme.Hex", i, f.Name, f.Type)
		}
	}
}

// canonical hex values from design/color-palette — byte-for-byte source of truth.
// Duplicating here is intentional: tests catch palette.go drift from canonical.
const (
	primaryLight    = "#8B5CF6"
	primaryDark     = "#A78BFA"
	accentLight     = "#0D9488"
	accentDark      = "#2DD4BF"
	foregroundLight = "#0F172A"
	foregroundDark  = "#F8FAFC"
	mutedLight      = "#64748B"
	mutedDark       = "#94A3B8"
	backgroundLight = "#FFFFFF"
	backgroundDark  = "#0F172A"
	successLight    = "#16A34A"
	successDark     = "#22C55E"
	warningLight    = "#D97706"
	warningDark     = "#F59E0B"
	errorLight      = "#E11D48"
	errorDark       = "#F43F5E"
)

// Test_DefaultPalette_HexMatchesCanonical asserts every token light/dark hex
// matches the canonical design/color-palette entry byte-for-byte (theme-tokens/REQ-02.1, REQ-04.1).
func Test_DefaultPalette_HexMatchesCanonical(t *testing.T) {
	t.Parallel()

	p := theme.DefaultPalette()

	cases := []struct {
		name  string
		token theme.Hex
		light string
		dark  string
	}{
		{"Primary", p.Primary, primaryLight, primaryDark},
		{"Accent", p.Accent, accentLight, accentDark},
		{"Foreground", p.Foreground, foregroundLight, foregroundDark},
		{"Muted", p.Muted, mutedLight, mutedDark},
		{"Background", p.Background, backgroundLight, backgroundDark},
		{"Success", p.Success, successLight, successDark},
		{"Warning", p.Warning, warningLight, warningDark},
		{"Error", p.Error, errorLight, errorDark},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.token.Light != tc.light {
				t.Errorf("%s.Light = %q, want %q", tc.name, tc.token.Light, tc.light)
			}
			if tc.token.Dark != tc.dark {
				t.Errorf("%s.Dark = %q, want %q", tc.name, tc.token.Dark, tc.dark)
			}
		})
	}
}

// Test_DefaultPalette_DefensiveCopy asserts that mutating one returned Palette does
// not affect another (value-type semantics — theme-tokens/REQ-04.1).
func Test_DefaultPalette_DefensiveCopy(t *testing.T) {
	t.Parallel()

	p1 := theme.DefaultPalette()
	p2 := theme.DefaultPalette()

	original := p2.Primary.Light

	// Mutate p1 — must not affect p2.
	p1.Primary.Light = "#000000"

	if p2.Primary.Light != original {
		t.Errorf("p2.Primary.Light changed to %q after mutating p1 — Palette is not value-safe", p2.Primary.Light)
	}
}
