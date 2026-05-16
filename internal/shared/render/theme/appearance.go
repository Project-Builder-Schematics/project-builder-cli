package theme

import (
	"fmt"
	"io"
	"strings"
)

// Appearance represents the terminal background tone (light or dark).
type Appearance uint8

const (
	// Light indicates a light terminal background (default).
	Light Appearance = iota

	// Dark indicates a dark terminal background.
	Dark
)

// DetectAppearance inspects w to determine the terminal background tone.
//
// In the current implementation there is no reliable cross-platform mechanism
// to query the terminal background color without a PTY and OS-specific escape
// sequences, so this always returns Light. The flag (--theme) and env
// (BUILDER_THEME) override paths in ResolveAppearance are the primary means by
// which users can select Dark mode.
//
// TODO(future): implement xterm OSC 11 background-color query for interactive TTYs.
func DetectAppearance(_ io.Writer) Appearance {
	return Light
}

// ResolveAppearance applies the precedence chain:
//
//	flag (light|dark, non-auto) > env (BUILDER_THEME=light|dark) > detected
//
// flag is the raw --theme flag value; env is the BUILDER_THEME env var value.
// An empty flag or "auto" passes through to the env check.
// An empty env falls through to the detected value.
// An unrecognised non-empty flag value returns an error (REQ-05.1 — the flag
// layer rejects first, but this is a defensive second check for callers that
// bypass the flag layer).
//
// REQ-02.1 — flag=light/dark wins over everything.
// REQ-02.2 — flag=auto/empty → use detected.
// REQ-03.1 — flag=auto/empty, env=dark/light → use env.
// REQ-03.2 — flag=light/dark wins over env.
func ResolveAppearance(flag, env string, detected Appearance) (Appearance, error) {
	f := strings.ToLower(strings.TrimSpace(flag))
	e := strings.ToLower(strings.TrimSpace(env))

	// Flag takes highest precedence when it is explicitly set to a direction.
	switch f {
	case "light":
		return Light, nil
	case "dark":
		return Dark, nil
	case "", "auto":
		// Fall through to env check.
	default:
		return detected, fmt.Errorf(
			`invalid theme flag %q: must be light, dark, or auto`,
			flag,
		)
	}

	// Env is middle precedence.
	switch e {
	case "light":
		return Light, nil
	case "dark":
		return Dark, nil
	default:
		// Unrecognised or empty env → use detected value.
		return detected, nil
	}
}
