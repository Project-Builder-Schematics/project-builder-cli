// Package theme — profile.go wraps charmbracelet/colorprofile.Detect and maps
// the upstream Profile enum to our vendored theme.Profile (ADR-02: insulates
// callers from colorprofile v0.x churn; our enum is the public contract).
package theme

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
)

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

// DetectProfile inspects w and the current environment to determine the active
// terminal color profile. It wraps colorprofile.Detect and maps the upstream
// enum to theme.Profile so callers are insulated from upstream v0.x churn (ADR-02).
//
// The environment is read at call time via os.Environ(), which means t.Setenv
// overrides in tests are visible to the implementation.
func DetectProfile(w io.Writer) Profile {
	return mapColorProfile(colorprofile.Detect(w, os.Environ()))
}

// mapColorProfile maps a colorprofile.Profile to our vendored theme.Profile.
// Conservative: any unknown/unhandled upstream value maps to NoColor.
func mapColorProfile(p colorprofile.Profile) Profile {
	switch p {
	case colorprofile.TrueColor:
		return TrueColor
	case colorprofile.ANSI256:
		return ANSI256
	case colorprofile.ANSI:
		return ANSI16
	default:
		// colorprofile.NoTTY, colorprofile.ASCII, colorprofile.Unknown → NoColor
		return NoColor
	}
}
