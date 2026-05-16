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
// The color map is precomputed at construction time (newResolver) so that
// Resolve is an O(1) map lookup at render time (ADR-04 — performance).
//
// Quantization strategy:
//   - TrueColor: lipgloss.Color(hex) — stored as-is; lipgloss quantizes at
//     render time via the global renderer's color profile (termenv.TrueColor).
//   - ANSI256:   lipgloss.Color(hex) — stored as-is; lipgloss quantizes to
//     nearest 256-color at render time via termenv.ANSI256.Convert(RGBColor).
//   - ANSI16:    lipgloss.Color(hex) — stored as-is; lipgloss quantizes to
//     nearest 16-color at render time via termenv.ANSI.Convert(RGBColor).
//   - NoColor:   lipgloss.NoColor{} — emits zero SGR bytes regardless of profile.
//
// Quantization is performed by lipgloss's internal renderer at Render() time,
// not at construction. This keeps the resolver profile-neutral: storing the
// hex literal means the exact same value works across profiles when the global
// renderer profile matches the theme profile.
type Resolver struct {
	resolved map[TokenName]lipgloss.TerminalColor
}

// newResolver precomputes the token→TerminalColor map for the given
// (Palette, Profile, Appearance) triple. Called once at theme.New().
func newResolver(p Palette, prof Profile, app Appearance) Resolver {
	if prof == NoColor {
		// NoColor: all tokens map to lipgloss.NoColor{} — zero SGR, always.
		m := make(map[TokenName]lipgloss.TerminalColor, 8)
		for _, tok := range allTokens {
			m[tok] = lipgloss.NoColor{}
		}
		return Resolver{resolved: m}
	}

	// For all color-capable profiles, pick hex by appearance then store as
	// lipgloss.Color. The global lipgloss renderer quantizes at render time.
	pick := func(h Hex) lipgloss.Color {
		if app == Dark {
			return lipgloss.Color(h.Dark)
		}
		return lipgloss.Color(h.Light)
	}

	return Resolver{
		resolved: map[TokenName]lipgloss.TerminalColor{
			TokPrimary:    pick(p.Primary),
			TokAccent:     pick(p.Accent),
			TokForeground: pick(p.Foreground),
			TokMuted:      pick(p.Muted),
			TokBackground: pick(p.Background),
			TokSuccess:    pick(p.Success),
			TokWarning:    pick(p.Warning),
			TokError:      pick(p.Error),
		},
	}
}

// allTokens is the ordered set of all 8 canonical token names.
// Used internally by newResolver to populate the NoColor map without
// manually listing every token in two places.
var allTokens = [8]TokenName{
	TokPrimary, TokAccent, TokForeground, TokMuted,
	TokBackground, TokSuccess, TokWarning, TokError,
}

// Resolve returns the terminal color for the given token under the active
// profile and appearance. Returns lipgloss.NoColor{} for unknown tokens
// (defensive: the vocabulary is fixed, but guards against future extension bugs).
func (r Resolver) Resolve(tok TokenName) lipgloss.TerminalColor {
	if c, ok := r.resolved[tok]; ok {
		return c
	}
	return lipgloss.NoColor{}
}
