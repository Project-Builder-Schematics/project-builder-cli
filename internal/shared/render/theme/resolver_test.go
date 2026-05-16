// Package theme — resolver_test.go covers token-to-color mapping per profile.
//
// REQ theme-profile-detection/04.1 — TrueColor profile emits hex RGB SGR
// REQ theme-profile-detection/04.2 — NoColor profile emits zero SGR for all tokens
package theme_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// applyResolverHygiene clears env that can bleed into lipgloss global profile
// detection. These tests pin the profile explicitly via SetColorProfile, so
// env values must not interfere.
//
// NOTE: t.Setenv is used here — per Go 1.26 discipline these tests are
// SEQUENTIAL (no t.Parallel()). Do not add t.Parallel() to any test in this
// file.
func applyResolverHygiene(t *testing.T) {
	t.Helper()
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TTY_FORCE", "")
	t.Setenv("TERM", "dumb")
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-04.1 — TrueColor Dark: Primary token resolves to #A78BFA RGB SGR
// ──────────────────────────────────────────────────────────────────────────────

// Test_Resolver_TrueColor_Dark_Primary_Hex pins the global lipgloss color
// profile to TrueColor, constructs a Theme with Profile=TrueColor and
// Appearance=Dark, resolves TokPrimary, and asserts the rendered SGR bytes
// include the truecolor escape for #A78BFA (RGB 167,139,250).
func Test_Resolver_TrueColor_Dark_Primary_Hex(t *testing.T) {
	// Sequential — uses t.Setenv + global lipgloss state mutation.
	applyResolverHygiene(t)

	// Pin global lipgloss profile to TrueColor so Style.Render() emits 24-bit SGR.
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)
	c := th.Resolve(theme.TokPrimary)

	got := lipgloss.NewStyle().Foreground(c).Render("x")

	// #A78BFA = RGB(167, 139, 250) — truecolor SGR: ESC[38;2;167;139;250m
	const wantSGR = "\x1b[38;2;167;139;250m"
	if !strings.Contains(got, wantSGR) {
		t.Errorf("TrueColor/Dark/Primary: expected SGR %q in rendered string\ngot: %q", wantSGR, got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-04.1 (bonus) — TrueColor Light: Primary token resolves to #8B5CF6
// ──────────────────────────────────────────────────────────────────────────────

func Test_Resolver_TrueColor_Light_Primary_Hex(t *testing.T) {
	applyResolverHygiene(t)

	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Light)
	c := th.Resolve(theme.TokPrimary)

	got := lipgloss.NewStyle().Foreground(c).Render("x")

	// #8B5CF6 = RGB(139, 92, 246) — truecolor SGR: ESC[38;2;139;92;246m
	const wantSGR = "\x1b[38;2;139;92;246m"
	if !strings.Contains(got, wantSGR) {
		t.Errorf("TrueColor/Light/Primary: expected SGR %q in rendered string\ngot: %q", wantSGR, got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-04.2 — NoColor profile emits zero SGR for all 8 tokens
// ──────────────────────────────────────────────────────────────────────────────

// Test_Resolver_NoColor_EmitsNoSGR table-drives all 8 tokens and asserts that
// each Resolve() result, when used as a foreground on a lipgloss Style, produces
// output that equals exactly "x" — no escape bytes anywhere.
func Test_Resolver_NoColor_EmitsNoSGR(t *testing.T) {
	applyResolverHygiene(t)

	// Pin to Ascii (= lipgloss NoColor equivalent at render time) so we can
	// test that our lipgloss.NoColor{} values truly emit nothing.
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	tokens := []struct {
		name  string
		token theme.TokenName
	}{
		{"Primary", theme.TokPrimary},
		{"Accent", theme.TokAccent},
		{"Foreground", theme.TokForeground},
		{"Muted", theme.TokMuted},
		{"Background", theme.TokBackground},
		{"Success", theme.TokSuccess},
		{"Warning", theme.TokWarning},
		{"Error", theme.TokError},
	}

	for _, tt := range tokens {
		// Table-driven sub-tests: sequential (outer test uses t.Setenv).
		t.Run(tt.name, func(t *testing.T) {
			c := th.Resolve(tt.token)
			got := lipgloss.NewStyle().Foreground(c).Render("x")

			if got != "x" {
				t.Errorf("NoColor/%s: expected exactly \"x\", got %q (contains SGR escape)", tt.name, got)
			}
			if strings.Contains(got, "\x1b[") {
				t.Errorf("NoColor/%s: output contains ESC[ bytes, expected zero SGR\ngot: %q", tt.name, got)
			}
		})
	}
}
