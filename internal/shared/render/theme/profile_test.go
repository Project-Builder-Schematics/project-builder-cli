package theme_test

import (
	"bytes"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// nonTTYWriter returns a writer that is not a TTY (bytes.Buffer — no fd, no isatty).
// colorprofile.Detect uses isatty; a bytes.Buffer will never be a TTY.
func nonTTYWriter() *bytes.Buffer {
	return &bytes.Buffer{}
}

// Test_DetectProfile covers the three canonical terminal capability paths
// (theme-profile-detection/REQ-01.1, 01.2, 01.3).
//
// t.Setenv controls os.Environ; DetectProfile calls os.Environ() at call time
// so the overrides are visible to colorprofile.Detect inside the implementation.
//
// Design notes (colorprofile v0.4.1 behavior — observed 2026-05-16):
//   - TTY_FORCE=1 simulates a TTY fd without requiring an actual pty; this is
//     the colorprofile-internal mechanism (isTTYForced). COLORTERM/TERM upgrades
//     only apply when the writer is (or is forced to be) a TTY.
//   - TMUX must be cleared: colorprofile.Detect calls `tmux info` when TMUX is
//     set, which can return TrueColor regardless of TERM/COLORTERM — breaking
//     the ANSI256 and NoColor test paths.
//   - Tests are sequential (not parallel) because t.Setenv mutates os.Environ.
func Test_DetectProfile(t *testing.T) {
	t.Run("REQ-01.1_TrueColor_via_COLORTERM", func(t *testing.T) {
		// TTY_FORCE=1 simulates a TTY fd; COLORTERM=truecolor upgrades to TrueColor.
		// TERM must be non-dumb so colorProfile doesn't short-circuit to NoTTY.
		// TMUX must be empty to prevent tmux(env) from running `tmux info`.
		t.Setenv("TTY_FORCE", "1")
		t.Setenv("TERM", "xterm")
		t.Setenv("COLORTERM", "truecolor")
		t.Setenv("NO_COLOR", "")
		t.Setenv("TMUX", "")
		w := nonTTYWriter()
		got := theme.DetectProfile(w)
		if got != theme.TrueColor {
			t.Errorf("DetectProfile with TTY_FORCE=1 COLORTERM=truecolor: got %v, want TrueColor", got)
		}
	})

	t.Run("REQ-01.2_ANSI256_via_TERM_xterm256color", func(t *testing.T) {
		// TTY_FORCE=1 simulates TTY; xterm-256color suffix triggers ANSI256 path.
		// COLORTERM must be empty; TMUX must be empty (tmux info can report TrueColor).
		t.Setenv("TTY_FORCE", "1")
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("COLORTERM", "")
		t.Setenv("NO_COLOR", "")
		t.Setenv("TMUX", "")
		w := nonTTYWriter()
		got := theme.DetectProfile(w)
		if got != theme.ANSI256 {
			t.Errorf("DetectProfile with TTY_FORCE=1 TERM=xterm-256color: got %v, want ANSI256", got)
		}
	})

	t.Run("REQ-01.3_NoColor_when_non_TTY_no_color_hints", func(t *testing.T) {
		// No TTY_FORCE, no COLORTERM, TERM unset — bytes.Buffer is not a TTY.
		// colorprofile returns NoTTY → mapped to NoColor.
		// TMUX must be empty to prevent tmux detection path.
		t.Setenv("TTY_FORCE", "")
		t.Setenv("COLORTERM", "")
		t.Setenv("TERM", "")
		t.Setenv("NO_COLOR", "")
		t.Setenv("TMUX", "")
		t.Setenv("CLICOLOR_FORCE", "")
		w := nonTTYWriter()
		got := theme.DetectProfile(w)
		if got != theme.NoColor {
			t.Errorf("DetectProfile with non-TTY, no env hints: got %v, want NoColor", got)
		}
	})
}
