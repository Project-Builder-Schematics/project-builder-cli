package pretty

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// Styles groups the lipgloss style definitions for PrettyRenderer output.
// Each field corresponds to a semantic token in the 8-token vocabulary
// (theme-tokens/REQ-01); colors are sourced from theme.Resolve() — no
// hard-coded ANSI indices or adaptive-color literals (render-pretty/REQ-02.1).
//
// Migration note (S-005): the previous 4 fields (Progress, FileOp, LogLevel,
// Terminal) are replaced by the canonical 8-token names per the fixed mapping:
//
//	Progress → Primary
//	FileOp   → Accent
//	LogLevel → Muted
//	Terminal → Foreground
//
// Success, Warning, Error, and Background exist in the struct but are not yet
// wired to call sites in pretty.go — future slices/commands consume them.
type Styles struct {
	Primary    lipgloss.Style
	Accent     lipgloss.Style
	Foreground lipgloss.Style
	Muted      lipgloss.Style
	Background lipgloss.Style
	Success    lipgloss.Style
	Warning    lipgloss.Style
	Error      lipgloss.Style
}

// NewStyles constructs a Styles from a theme.Theme. Each field is a
// lipgloss.Style with the foreground color sourced from theme.Resolve(tok),
// precomputed once at construction time (O(1) at render time).
func NewStyles(t theme.Theme) Styles {
	color := func(tok theme.TokenName) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(t.Resolve(tok))
	}
	return Styles{
		Primary:    color(theme.TokPrimary),
		Accent:     color(theme.TokAccent),
		Foreground: color(theme.TokForeground),
		Muted:      color(theme.TokMuted),
		Background: color(theme.TokBackground),
		Success:    color(theme.TokSuccess),
		Warning:    color(theme.TokWarning),
		Error:      color(theme.TokError),
	}
}
