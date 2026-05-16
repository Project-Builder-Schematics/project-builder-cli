package theme

import "io"

// Profile represents the color capability tier of the active terminal.
// It is a vendored enum wrapping the upstream colorprofile detection
// (ADR-02: insulate callers from colorprofile v0.x churn).
type Profile uint8

const (
	// NoColor indicates no ANSI color support (e.g. piped output, dumb terminal).
	NoColor Profile = iota

	// ANSI16 indicates support for the basic 16 ANSI colors.
	ANSI16

	// ANSI256 indicates support for the 256-color xterm palette.
	ANSI256

	// TrueColor indicates support for 24-bit RGB hex colors.
	TrueColor
)

// DetectProfile inspects w to determine the active terminal color profile.
// Returns NoColor for all writers in S-000; S-002 implements real detection
// via colorprofile.Detect.
//
// TODO(S-002): implement real detection wrapping colorprofile.Detect(w, os.Environ()).
func DetectProfile(_ io.Writer) Profile {
	return NoColor
}
