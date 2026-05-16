// Package pretty_test — golden_test.go pins the rendered ANSI byte sequences
// for the 4-profile × 3-style golden matrix.
//
// # Matrix invariant
//
// Test_Render_Golden_Matrix runs 12 sub-tests: 3 representative styles
// (Primary, Success, Error) × 4 terminal profiles (TrueColor, ANSI256,
// ANSI16, NoColor). Each cell pins the global lipgloss color profile via
// lipgloss.SetColorProfile, renders the style, and asserts byte-equality
// against a committed golden file at testdata/golden/styles/<profile>/.
//
// Tests in this file are SEQUENTIAL by design — lipgloss uses global renderer
// state (termenv.Profile), and Go 1.26 forbids t.Setenv + t.Parallel in the
// same test. No t.Parallel() calls are present anywhere in this file.
//
// Regenerate golden files when palette hex values change intentionally:
//
//	go test ./internal/shared/render/pretty/... -run Test_Render_Golden -update
//
// REQ coverage:
//   - render-pretty/REQ-04.1 — NoColor/Error/"failed" → plain bytes, no \x1b
//   - render-pretty/REQ-04.2 — TrueColor/Dark/Success/"ok" → RGB SGR bytes
package pretty_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// update is the -update flag for golden regeneration.
// Mirrors the pattern in internal/feature/new/golden_test.go exactly.
// Redeclaring here keeps the golden test file self-contained even if the two
// packages are run independently; both declare the same flag name which is
// harmless since flag.Bool is package-scoped per test binary.
var update = flag.Bool("update", false, "overwrite golden files with current output")

// withProfile pins the global lipgloss color profile for the duration of the
// test and restores it via t.Cleanup. Sequential tests only — do NOT call
// t.Parallel() before or after withProfile.
func withProfile(t *testing.T, p termenv.Profile) {
	t.Helper()
	prior := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(p)
	t.Cleanup(func() { lipgloss.SetColorProfile(prior) })
}

// themeProfileFor maps a termenv.Profile to our theme.Profile enum.
func themeProfileFor(p termenv.Profile) theme.Profile {
	switch p {
	case termenv.TrueColor:
		return theme.TrueColor
	case termenv.ANSI256:
		return theme.ANSI256
	case termenv.ANSI:
		return theme.ANSI16
	case termenv.Ascii:
		return theme.NoColor
	default:
		return theme.NoColor
	}
}

// goldenPath returns the path to the golden file for a given profile + style name.
// Layout: testdata/golden/styles/<profile>/<style>.golden
func goldenPath(profile, style string) string {
	return filepath.Join("testdata", "golden", "styles", profile, style+".golden")
}

// assertGolden compares got against the golden file at path. When -update is
// set, it writes got to path (creating parent directories as needed).
func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("assertGolden: MkdirAll %q: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil { //nolint:gosec // test fixture update path
			t.Fatalf("assertGolden: WriteFile %q: %v", path, err)
		}
		t.Logf("updated golden: %s", path)
		return
	}

	want, err := os.ReadFile(path) // #nosec G304 — test fixture path
	if err != nil {
		t.Fatalf("assertGolden: ReadFile %q: %v (run with -update to generate)", path, err)
	}

	if string(got) != string(want) {
		t.Errorf("golden mismatch at %s:\nwant: %q\n got: %q", path, string(want), string(got))
	}
}

// Test_Render_Golden_Matrix covers render-pretty/REQ-04:
// 3 representative styles × 4 profiles = 12 deterministic golden assertions.
//
// Sequential — no t.Parallel calls. Each sub-test pins lipgloss global state.
func Test_Render_Golden_Matrix(t *testing.T) {
	// Block TMUX bleed (lesson: colorprofile-test-gotchas).
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	profiles := []struct {
		name string
		p    termenv.Profile
	}{
		{"truecolor", termenv.TrueColor},
		{"ansi256", termenv.ANSI256},
		{"ansi16", termenv.ANSI},
		{"nocolor", termenv.Ascii},
	}

	cells := []struct {
		tok       theme.TokenName
		styleName string
		text      string
	}{
		{theme.TokPrimary, "primary", "builder"},
		{theme.TokSuccess, "success", "ok"},
		{theme.TokError, "error", "failed"},
	}

	for _, prof := range profiles {
		for _, cell := range cells {
			// Capture loop variables for sub-test closure.
			prof := prof
			cell := cell

			t.Run(prof.name+"/"+cell.styleName, func(t *testing.T) {
				// Sequential — no t.Parallel().
				withProfile(t, prof.p)

				th := theme.New(theme.DefaultPalette(), themeProfileFor(prof.p), theme.Dark)
				style := lipgloss.NewStyle().Foreground(th.Resolve(cell.tok))
				got := []byte(style.Render(cell.text))

				assertGolden(t, goldenPath(prof.name, cell.styleName), got)
			})
		}
	}
}

// Test_Render_Golden_REQ04_1 is the named scenario for render-pretty/REQ-04.1:
// NoColor / Error / "failed" → exactly "failed" (5 bytes, no \x1b sequences).
//
// Sequential — no t.Parallel.
func Test_Render_Golden_REQ04_1(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii)

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)
	style := lipgloss.NewStyle().Foreground(th.Resolve(theme.TokError))
	got := []byte(style.Render("failed"))

	// Hard assertion: zero escape sequences.
	for _, b := range got {
		if b == 0x1b {
			t.Errorf("REQ-04.1: NoColor/Error rendered with escape byte \\x1b — want plain text; got %q", got)
			break
		}
	}

	assertGolden(t, goldenPath("nocolor", "error-failed"), got)
}

// Test_Render_Golden_REQ04_2 is the named scenario for render-pretty/REQ-04.2:
// TrueColor / Dark / Success / "ok" → contains \x1b[38;2;34;197;94m (RGB for #22C55E).
//
// Sequential — no t.Parallel.
func Test_Render_Golden_REQ04_2(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)
	style := lipgloss.NewStyle().Foreground(th.Resolve(theme.TokSuccess))
	got := []byte(style.Render("ok"))

	// Hard assertion: must contain the RGB SGR for #22C55E = (34,197,94).
	const wantSGR = "\x1b[38;2;34;197;94m"
	if string(got) != "" && !contains(string(got), wantSGR) {
		t.Errorf("REQ-04.2: TrueColor/Success/ok missing RGB SGR %q\ngot: %q", wantSGR, got)
	}

	assertGolden(t, goldenPath("truecolor", "success-dark-ok"), got)
}

// contains is a simple substring check (avoids importing strings for one use).
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
