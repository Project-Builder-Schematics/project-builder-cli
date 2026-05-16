// Package themed_test covers the themed.Adapter production implementation.
//
// Test discipline (ADR-04):
//   - Golden cells: truecolor-dark/{heading,body,success,error} + nocolor/heading
//   - Writer-agnostic: bytes.Buffer injection (REQ-03.1)
//   - Light vs dark byte inequality (REQ-06.1)
//   - Stream byte parity with pretty.Renderer (render-pretty/REQ-06.1)
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
//	output-port/REQ-02.1  — all 9 chrome methods use correct tokens, TrueColor/Dark bytes match
//	output-port/REQ-03.1  — Adapter accepts any io.Writer (bytes.Buffer injection)
//	output-port/REQ-06.1  — Light vs Dark themes produce byte-different output
//	render-pretty/REQ-06.1 — Stream produces identical bytes to pretty.Renderer.Render
package themed_test

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output/themed"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"
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
	a.Body("test")

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

// ── S-001: Chrome method golden tests ─────────────────────────────────────────
//
// Group A: ANSI chrome methods — Body, Hint, Success, Warning, Error
// Group B: Path + Newline
// Group C: Prompt (functional option + sync read)
// Group D: Stream (delegates to pretty.Renderer byte-for-byte)
//
// Golden matrix cells per ADR-04:
//   truecolor-dark/body.golden
//   truecolor-dark/success.golden
//   truecolor-dark/error.golden
//   nocolor/heading.golden  (profile: NoColor — text only, no SGR)

// Test_Themed_Body_TrueColor_Dark_GoldenBytes — output-port/REQ-02.1 (Body→Foreground token).
//
// GIVEN TrueColor/Dark adapter
// WHEN adapter.Body("world") is called
// THEN bytes contain the Foreground.Dark SGR (#F8FAFC = RGB(248,250,252))
// AND the visible text equals "world"
// AND bytes match the committed golden file.
func Test_Themed_Body_TrueColor_Dark_GoldenBytes(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Body("world")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("world")) {
		t.Errorf("Body output does not contain text %q; got: %q", "world", got)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Body output does not end with newline; got: %q", got)
	}
	// Foreground.Dark = #F8FAFC = RGB(248,250,252)
	const wantSGR = "\x1b[38;2;248;250;252m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Body output missing Foreground/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}

	assertGolden(t, goldenPath("truecolor-dark", "body"), got)
}

// Test_Themed_Hint_TrueColor_Dark — output-port/REQ-02.1 (Hint→Muted token).
//
// GIVEN TrueColor/Dark adapter
// WHEN adapter.Hint("tip") is called
// THEN bytes contain the Muted.Dark SGR (#94A3B8 = RGB(148,163,184))
func Test_Themed_Hint_TrueColor_Dark(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Hint("tip")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("tip")) {
		t.Errorf("Hint output does not contain text %q; got: %q", "tip", got)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Hint output does not end with newline; got: %q", got)
	}
	// Muted.Dark = #94A3B8 = RGB(148,163,184); lipgloss quantizes to (147,163,184)
	const wantSGR = "\x1b[38;2;147;163;184m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Hint output missing Muted/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}
}

// Test_Themed_Success_TrueColor_Dark_GoldenBytes — output-port/REQ-02.1 (Success→Success token).
//
// GIVEN TrueColor/Dark adapter
// WHEN adapter.Success("done") is called
// THEN bytes contain the Success.Dark SGR (#22C55E = RGB(34,197,94))
// AND bytes match the committed golden file.
func Test_Themed_Success_TrueColor_Dark_GoldenBytes(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Success("done")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("done")) {
		t.Errorf("Success output does not contain text %q; got: %q", "done", got)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Success output does not end with newline; got: %q", got)
	}
	// Success.Dark = #22C55E = RGB(34,197,94)
	const wantSGR = "\x1b[38;2;34;197;94m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Success output missing Success/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}

	assertGolden(t, goldenPath("truecolor-dark", "success"), got)
}

// Test_Themed_Warning_TrueColor_Dark — output-port/REQ-02.1 (Warning→Warning token).
//
// GIVEN TrueColor/Dark adapter
// WHEN adapter.Warning("careful") is called
// THEN bytes contain the Warning.Dark SGR (#F59E0B = RGB(245,158,11))
func Test_Themed_Warning_TrueColor_Dark(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Warning("careful")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("careful")) {
		t.Errorf("Warning output does not contain text %q; got: %q", "careful", got)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Warning output does not end with newline; got: %q", got)
	}
	// Warning.Dark = #F59E0B = RGB(245,158,11)
	const wantSGR = "\x1b[38;2;245;158;11m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Warning output missing Warning/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}
}

// Test_Themed_Error_TrueColor_Dark_GoldenBytes — output-port/REQ-02.1 (Error→Error token).
//
// GIVEN TrueColor/Dark adapter
// WHEN adapter.Error("bad") is called
// THEN bytes contain the Error.Dark SGR (#F43F5E = RGB(244,63,94))
// AND bytes match the committed golden file.
func Test_Themed_Error_TrueColor_Dark_GoldenBytes(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Error("bad")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("bad")) {
		t.Errorf("Error output does not contain text %q; got: %q", "bad", got)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Error output does not end with newline; got: %q", got)
	}
	// Error.Dark = #F43F5E = RGB(244,63,94); lipgloss quantizes to (243,63,94)
	const wantSGR = "\x1b[38;2;243;63;94m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Error output missing Error/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}

	assertGolden(t, goldenPath("truecolor-dark", "error"), got)
}

// Test_Themed_Heading_NoColor_GoldenBytes — nocolor/heading golden cell.
//
// GIVEN NoColor adapter
// WHEN adapter.Heading("hello") is called
// THEN output contains "hello" with no escape bytes
// AND bytes match the committed nocolor/heading.golden file.
func Test_Themed_Heading_NoColor_GoldenBytes(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii)

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Heading("hello")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("hello")) {
		t.Errorf("NoColor Heading output does not contain text %q; got: %q", "hello", got)
	}
	// No escape bytes.
	for _, b := range got {
		if b == 0x1b {
			t.Errorf("NoColor Heading output contains escape byte \\x1b — want plain text; got %q", got)
			break
		}
	}

	assertGolden(t, goldenPath("nocolor", "heading"), got)
}

// Test_Themed_Path_TrueColor_Dark — output-port/REQ-02.1 (Path→Accent token).
//
// GIVEN TrueColor/Dark adapter
// WHEN adapter.Path("/a/b") is called
// THEN bytes contain the Accent.Dark SGR (#2DD4BF = RGB(45,212,191))
func Test_Themed_Path_TrueColor_Dark(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Path("/a/b")

	got := buf.Bytes()

	if !bytes.Contains(got, []byte("/a/b")) {
		t.Errorf("Path output does not contain path %q; got: %q", "/a/b", got)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Errorf("Path output does not end with newline; got: %q", got)
	}
	// Accent.Dark = #2DD4BF = RGB(45,212,191); lipgloss quantizes to (44,211,191)
	const wantSGR = "\x1b[38;2;44;211;191m"
	if !bytes.Contains(got, []byte(wantSGR)) {
		t.Errorf("Path output missing Accent/Dark RGB SGR %q\ngot: %q", wantSGR, got)
	}
}

// Test_Themed_Newline — Newline() emits exactly one blank line.
//
// GIVEN any adapter
// WHEN adapter.Newline() is called
// THEN output is exactly "\n" (one byte).
func Test_Themed_Newline(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii)

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	var buf bytes.Buffer
	a := themed.New(&buf, th)
	a.Newline()

	got := buf.Bytes()
	if string(got) != "\n" {
		t.Errorf("Newline() output: want %q, got %q", "\n", got)
	}
}

// Test_Themed_Prompt_WritesHintAndReadsLine — output-port/REQ-02.1 (Prompt→Primary prefix "? ").
//
// GIVEN an adapter with a reader returning "myinput\n"
// WHEN adapter.Prompt("Name?") is called
// THEN the writer gets a styled "? Name? " prefix (no trailing newline)
// AND the return value is "myinput" (trimmed of newline).
func Test_Themed_Prompt_WritesHintAndReadsLine(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii) // NoColor — we care about structure, not SGR

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	// Inject a fake reader that returns a single line.
	fakeReader := strings.NewReader("myinput\n")

	var buf bytes.Buffer
	a := themed.New(&buf, th, themed.WithReader(fakeReader))

	reply, err := a.Prompt("Name?")
	if err != nil {
		t.Fatalf("Prompt returned unexpected error: %v", err)
	}
	if reply != "myinput" {
		t.Errorf("Prompt reply: want %q, got %q", "myinput", reply)
	}
	// Writer got the styled prefix. Check it contains "?" and "Name?".
	written := buf.String()
	if !strings.Contains(written, "?") {
		t.Errorf("Prompt written prefix does not contain '?': %q", written)
	}
	if !strings.Contains(written, "Name?") {
		t.Errorf("Prompt written prefix does not contain 'Name?': %q", written)
	}
	// No trailing newline on the prompt prefix — cursor stays on same line.
	if strings.HasSuffix(written, "\n") {
		t.Errorf("Prompt written prefix must not end with newline; got: %q", written)
	}
}

// Test_Themed_Prompt_ReaderError — Prompt propagates reader errors verbatim.
//
// GIVEN an adapter with an always-error reader
// WHEN adapter.Prompt("X") is called
// THEN Prompt returns a non-nil error.
func Test_Themed_Prompt_ReaderError(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii)

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	// errReader always returns an error.
	errReader := &alwaysErrReader{}

	var buf bytes.Buffer
	a := themed.New(&buf, th, themed.WithReader(errReader))

	_, err := a.Prompt("X")
	if err == nil {
		t.Error("Prompt with failing reader: want non-nil error, got nil")
	}
}

// alwaysErrReader is a test helper that always returns an error from Read.
type alwaysErrReader struct{}

func (r *alwaysErrReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

// Test_Themed_LightVsDark_BytesDiffer — output-port/REQ-06.1.
//
// GIVEN two Themes: (TrueColor, Light) and (TrueColor, Dark)
// WHEN each calls Heading("title")
// THEN light.Bytes() != dark.Bytes() — different SGR sequences
// AND both contain "title".
func Test_Themed_LightVsDark_BytesDiffer(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	thLight := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Light)
	thDark := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var bufLight, bufDark bytes.Buffer
	aLight := themed.New(&bufLight, thLight)
	aDark := themed.New(&bufDark, thDark)

	aLight.Heading("title")
	aDark.Heading("title")

	light := bufLight.Bytes()
	dark := bufDark.Bytes()

	if !bytes.Contains(light, []byte("title")) {
		t.Errorf("Light Heading does not contain 'title'; got: %q", light)
	}
	if !bytes.Contains(dark, []byte("title")) {
		t.Errorf("Dark Heading does not contain 'title'; got: %q", dark)
	}
	if bytes.Equal(light, dark) {
		t.Errorf("Light and Dark Heading produced identical bytes (REQ-06.1 violated)\nlight: %q\ndark: %q", light, dark)
	}
}

// Test_Themed_Stream_ByteParityWith_PrettyRender — render-pretty/REQ-06.1.
//
// GIVEN a themed.Adapter and a pretty.Renderer constructed from the same (w, theme)
// AND a channel containing one events.FileCreated{Path: "x.ts"} event then closed
// WHEN adapter.Stream(ctx, ch1) and pretty.Renderer.Render(ctx, ch2) are run
// THEN both buffers contain byte-identical output.
func Test_Themed_Stream_ByteParityWith_PrettyRender(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.TrueColor)

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	ctx := context.Background()

	// Run via Adapter.Stream.
	ch1 := make(chan events.Event, 1)
	ch1 <- events.FileCreated{Path: "x.ts"}
	close(ch1)

	var bufAdapter bytes.Buffer
	a := themed.New(&bufAdapter, th)
	if err := a.Stream(ctx, ch1); err != nil {
		t.Fatalf("Adapter.Stream returned error: %v", err)
	}

	// Run via pretty.Renderer.Render directly.
	ch2 := make(chan events.Event, 1)
	ch2 <- events.FileCreated{Path: "x.ts"}
	close(ch2)

	var bufPretty bytes.Buffer
	r := pretty.New(&bufPretty, th)
	if err := r.Render(ctx, ch2); err != nil {
		t.Fatalf("pretty.Renderer.Render returned error: %v", err)
	}

	adapterBytes := bufAdapter.Bytes()
	prettyBytes := bufPretty.Bytes()

	if !bytes.Equal(adapterBytes, prettyBytes) {
		t.Errorf("Stream byte parity failed (render-pretty/REQ-06.1)\nadapter: %q\npretty:  %q", adapterBytes, prettyBytes)
	}
}

// ── S-001 functional option test ──────────────────────────────────────────────

// Test_Themed_WithReader_DefaultIsStdin verifies that New without options
// uses os.Stdin as the default reader (structural check — we test it doesn't panic).
//
// We can't interactively test stdin reads in unit tests, but we can confirm the
// option is honoured by using WithReader and verifying it overrides the default.
// The previous Test_Themed_Prompt_WritesHintAndReadsLine covers this path.
func Test_Themed_WithReader_OptionIsHonoured(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	withProfile(t, termenv.Ascii)

	th := theme.New(theme.DefaultPalette(), theme.NoColor, theme.Dark)

	// Two adapters: one with custom reader, one using default (stdin).
	customReader := strings.NewReader("custom\n")

	var buf bytes.Buffer
	a := themed.New(&buf, th, themed.WithReader(customReader))

	reply, err := a.Prompt("Q")
	if err != nil {
		t.Fatalf("Prompt with custom reader: unexpected error: %v", err)
	}
	if reply != "custom" {
		t.Errorf("WithReader option not honoured: want %q, got %q", "custom", reply)
	}
}
