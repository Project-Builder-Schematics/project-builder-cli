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
// Zero values are intentional placeholders for S-000; S-001 populates
// the canonical hex values byte-for-byte from the design/color-palette entry.
//
// TODO(S-001): populate all 8 fields with canonical hex values.
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

// DefaultPalette returns the canonical Palette.
// Zero values are placeholders for S-000; S-001 populates them.
//
// TODO(S-001): replace zero-value Hex structs with canonical light/dark hex.
func DefaultPalette() Palette {
	return Palette{}
}
