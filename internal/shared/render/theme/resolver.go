package theme

import "github.com/charmbracelet/lipgloss"

// TokenName identifies a semantic color token in the palette vocabulary.
// Callers reference tokens by these typed constants, never by raw hex strings.
type TokenName string

// The 8 canonical token names (theme-tokens/REQ-01).
// These constants are the vocabulary; hex values live in Palette.
const (
	TokPrimary    TokenName = "primary"
	TokAccent     TokenName = "accent"
	TokForeground TokenName = "foreground"
	TokMuted      TokenName = "muted"
	TokBackground TokenName = "background"
	TokSuccess    TokenName = "success"
	TokWarning    TokenName = "warning"
	TokError      TokenName = "error"
)

// Resolver maps token names to concrete lipgloss.TerminalColor values for a
// given (Palette, Profile, Appearance) combination.
//
// In S-000 the resolver returns lipgloss.NoColor{} for every token, producing
// zero SGR escape sequences regardless of terminal capability.
//
// TODO(S-004): implement precomputed map[TokenName]lipgloss.TerminalColor
// keyed on (Profile, Appearance) with proper TrueColor / ANSI256 / ANSI16 mapping.
type Resolver struct{}

// Resolve returns the terminal color for the given token under the active
// profile and appearance. Returns lipgloss.NoColor{} in S-000 (NoColor-only
// skeleton); S-004 replaces this with the full mapping.
func (r Resolver) Resolve(_ TokenName) lipgloss.TerminalColor {
	return lipgloss.NoColor{}
}
