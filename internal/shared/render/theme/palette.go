package theme

// Hex holds a light and dark hex color string for a single token.
// Values use the CSS hex format: "#RRGGBB".
// Hex literals MUST only appear in this package (theme-tokens/REQ-03).
type Hex struct {
	Light string
	Dark  string
}

// Palette is the 8-token semantic color vocabulary.
// Each field corresponds to a semantic role in builder CLI output.
// Canonical values come from the design/color-palette Engram entry (2026-05-16).
type Palette struct {
	Primary    Hex
	Accent     Hex
	Foreground Hex
	Muted      Hex
	Background Hex
	Success    Hex
	Warning    Hex
	Error      Hex
}

// DefaultPalette returns the canonical 8-token Palette with light and dark hex
// values matching the design/color-palette entry byte-for-byte (theme-tokens/REQ-01.1,
// REQ-02.1, REQ-04.1). Palette is a value type — callers may mutate their copy
// without affecting other calls.
func DefaultPalette() Palette {
	return Palette{
		Primary:    Hex{Light: "#8B5CF6", Dark: "#A78BFA"},
		Accent:     Hex{Light: "#0D9488", Dark: "#2DD4BF"},
		Foreground: Hex{Light: "#0F172A", Dark: "#F8FAFC"},
		Muted:      Hex{Light: "#64748B", Dark: "#94A3B8"},
		Background: Hex{Light: "#FFFFFF", Dark: "#0F172A"},
		Success:    Hex{Light: "#16A34A", Dark: "#22C55E"},
		Warning:    Hex{Light: "#D97706", Dark: "#F59E0B"},
		Error:      Hex{Light: "#E11D48", Dark: "#F43F5E"},
	}
}
