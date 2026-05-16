package theme

import "io"

// Appearance represents the terminal background tone (light or dark).
type Appearance uint8

const (
	// Light indicates a light terminal background (default).
	Light Appearance = iota

	// Dark indicates a dark terminal background.
	Dark
)

// DetectAppearance inspects w to determine the terminal background tone.
// Returns Light for all writers in S-000; S-003 wires flag + env precedence.
//
// TODO(S-003): implement detection from terminal background color or env hints.
func DetectAppearance(_ io.Writer) Appearance {
	return Light
}

// ResolveAppearance applies the precedence chain:
//
//	flag (light|dark) > env (light|dark) > detected (from terminal)
//
// Both flag and env are ignored in S-000 (empty string passthrough);
// S-003 implements the full precedence logic including validation.
//
// TODO(S-003): implement flag > env > detected precedence and validate values.
func ResolveAppearance(_ string, _ string, detected Appearance) (Appearance, error) {
	return detected, nil
}
