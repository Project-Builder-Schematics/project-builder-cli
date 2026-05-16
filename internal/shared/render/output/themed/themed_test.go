// Package themed_test covers the themed.Adapter production implementation.
//
// Test discipline (ADR-04):
//   - Golden cell: truecolor-dark/heading — pins ANSI bytes for Primary token
//   - Writer-agnostic: bytes.Buffer injection (REQ-03.1)
//
// Sequential by design — lipgloss uses global renderer state (termenv.Profile).
// Go 1.26 forbids t.Setenv + t.Parallel in the same test. No t.Parallel() here.
//
// Regenerate golden files when palette hex values change intentionally:
//
//	go test ./internal/shared/render/output/themed/... -run Test_Themed -update
//
// REQ coverage:
//
//	output-port/REQ-02.1 — Heading uses Primary token, TrueColor/Dark bytes match
//	output-port/REQ-03.1 — Adapter accepts any io.Writer (bytes.Buffer injection)
package themed_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output/themed"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// update is the -update flag for golden regeneration.
var update = flag.Bool("update", false, "overwrite golden files with current output")

// withProfile pins the global lipgloss color profile for the duration of the
// test and restores it via t.Cleanup. Sequential tests only.
func withProfile(t *testing.T, p termenv.Profile) {
	t.Helper()
	prior := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(p)
	t.Cleanup(func() { lipgloss.SetColorProfile(prior) })
}

// goldenPath returns the path for a given profile + method golden file.
// Layout: testdata/golden/output/<profile>/<method>.golden
func goldenPath(profile, method string) string {
	return filepath.Join("testdata", "golden", "output", profile, method+".golden")
}

// assertGolden compares got against the golden file at path.
// When -update is set, it writes got to path (creating parent dirs as needed).
func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("assertGolden: MkdirAll %q: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil { //nolint:gosec // test fixture
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

// Test_Themed_Heading_TrueColor_Dark_GoldenBytes covers output-port/REQ-02.1.
//
// GIVEN themed.Adapter with TrueColor/Dark theme
// WHEN adapter.Heading("hello") is called
// THEN bytes contain the truecolor SGR for Primary.Dark (#A78BFA)
// AND the visible text equals "hello"
// AND bytes match the committed golden file.
//
// Sequential — no t.Parallel().
func Test_Themed_Heading_TrueColor_Dark_GoldenBytes(t *testing.T) {
	// Isolate from tmux bleed (lessons-learned: colorprofile-test-gotchas).
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Heading("hello")

	got := buf.Bytes()

	// Hard assertion: must contain visible text "hello".
	if !bytes.Contains(got, []byte("hello")) {
		t.Errorf("Heading output does not contain text %q; got: %q", "hello", got)
	}

	// Hard assertion: must end with newline.
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Heading output does not end with newline; got: %q", got)
	}

	// Hard assertion: TrueColor/Dark Primary = #A78BFA = RGB(167,139,250).
	// SGR sequence: \x1b[38;2;167;139;250m
	const wantSGR = "\x1b[38;2;167;139;250m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Heading output missing Primary/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}

	assertGolden(t, goldenPath("truecolor-dark", "heading"), got)
}

// Test_Themed_AcceptsAnyWriter covers output-port/REQ-03.1.
//
// GIVEN var buf bytes.Buffer and a NoColor Theme
// WHEN themed.New(&buf, theme).Body("test") runs
// THEN buf.Bytes() is non-empty AND ends with newline AND contains "test".
//
// Uses NoColor so the test is profile-independent (no need to pin lipgloss global).
// Sequential — no t.Parallel() (Heading impl calls lipgloss, shares global state).
func Test_Themed_AcceptsAnyWriter(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii) // NoColor profile

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Heading("test")

	got := buf.Bytes()

	if len(got) == 0 {
		t.Error("Adapter with bytes.Buffer: output is empty (REQ-03.1)")
	}
	if got[len(got)-1] != '\n' {
		t.Errorf("Adapter output does not end with newline; got: %q (REQ-03.1)", got)
	}
	if !bytes.Contains(got, []byte("test")) {
		t.Errorf("Adapter output does not contain text %q; got: %q (REQ-03.1)", "test", got)
	}
}
