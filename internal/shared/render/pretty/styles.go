package pretty

import "github.com/charmbracelet/lipgloss"

// Styles groups the lipgloss style definitions for PrettyRenderer output.
// Each field corresponds to a distinct event category, providing visual
// hierarchy in the terminal.
//
// Colours use lipgloss.AdaptiveColor so the renderer renders correctly on
// both light and dark terminal backgrounds (UX design note).
type Styles struct {
	// Progress is used for script execution, progress counters, and input prompts.
	Progress lipgloss.Style

	// FileOp is used for file-system events (created, modified, deleted).
	// ASCII-safe glyphs: + created, ~ modified, - deleted.
	FileOp lipgloss.Style

	// LogLevel is used for log lines, colour-coded by severity level.
	LogLevel lipgloss.Style

	// Terminal is used for terminal/final events (Done, Failed, Cancelled).
	Terminal lipgloss.Style
}

// DefaultStyles returns the default Styles with lipgloss adaptive colours.
// The colour choices follow conventional terminal conventions:
//
//   - Progress: cyan (neutral, forward-looking)
//   - FileOp:   green (creation/modification) — overridden per glyph at render time
//   - LogLevel: white/grey (neutral log baseline) — overridden per level at render time
//   - Terminal: bold (emphasis for final state)
func DefaultStyles() Styles {
	return Styles{
		Progress: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "6", Dark: "14"}), // cyan

		FileOp: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "2", Dark: "10"}), // green

		LogLevel: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "8", Dark: "7"}), // grey/white

		Terminal: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "15"}), // black/white (bold)
	}
}
