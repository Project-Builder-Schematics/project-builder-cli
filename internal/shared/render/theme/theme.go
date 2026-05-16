// Package theme owns the token vocabulary, terminal-capability detection, and
// resolution of semantic tokens to lipgloss.TerminalColor values. It is a
// sibling of pretty/ and json/ under render/ so any renderer can consume the
// same vocabulary without crossing inward imports (ADR-01).
//
// Canonical hex values in palette.go are sourced from the design/color-palette
// Engram entry (project-builder-cli, 2026-05-16). Hex literals are confined to
// this package (theme-tokens/REQ-03 — enforced by just fitness-hex-leak).
package theme

import (
	"io"

	"github.com/charmbracelet/lipgloss"
)

// Theme is the aggregate value passed to renderers. It carries a resolved color
// for every token so renderer call sites need only call Resolve(tok) — they never
// perform detection or precedence logic themselves.
type Theme struct {
	palette    Palette
	profile    Profile
	appearance Appearance
	resolver   Resolver
}

// New constructs a Theme from an already-resolved (Palette, Profile, Appearance)
// triple. The Resolver is built at construction time; lookups are O(1).
func New(p Palette, prof Profile, app Appearance) Theme {
	return Theme{
		palette:    p,
		profile:    prof,
		appearance: app,
		resolver:   newResolver(p, prof, app),
	}
}

// Default constructs the canonical Theme from the active output writer and
// optional override strings for flag and env.
//
// Detection order (each placeholder in S-000; each wired in later slices):
//  1. DetectProfile(w)         — S-002 replaces stub
//  2. DetectAppearance(w)      — S-003 replaces stub
//  3. ResolveAppearance(flag, env, detected) — S-003 adds precedence chain
//
// Returns (Theme{NoColor, Light}, nil) in S-000 for all inputs.
func Default(w io.Writer, flag, env string) (Theme, error) {
	palette := DefaultPalette()
	profile := DetectProfile(w)
	detected := DetectAppearance(w)

	appearance, err := ResolveAppearance(flag, env, detected)
	if err != nil {
		return Theme{}, err
	}

	return New(palette, profile, appearance), nil
}

// Resolve returns the lipgloss.TerminalColor for the given token name under the
// active profile and appearance. In S-000 always returns lipgloss.NoColor{}.
func (t Theme) Resolve(token TokenName) lipgloss.TerminalColor {
	return t.resolver.Resolve(token)
}

// Profile returns the active terminal color profile.
func (t Theme) Profile() Profile {
	return t.profile
}

// Appearance returns the active terminal appearance (light or dark).
func (t Theme) Appearance() Appearance {
	return t.appearance
}
