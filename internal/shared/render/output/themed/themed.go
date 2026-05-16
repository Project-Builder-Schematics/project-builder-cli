// Package themed is the production adapter for the output.Output port.
//
// themed.Adapter writes styled bytes to any io.Writer, deriving all colors
// from the provided theme.Theme via the 8-token vocabulary. It does NOT hold
// a hard reference to os.Stdout (output-port/REQ-03).
//
// S-000 (walking skeleton): only Heading is implemented. The remaining 9 methods
// (Body, Hint, Success, Warning, Error, Path, Prompt, Newline, Stream) are added
// in S-001. They panic with a clear message if called prematurely so tests fail
// loudly rather than silently passing with no output.
package themed

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// Adapter is the production implementation of output.Output.
// Construct via New; do not create directly.
type Adapter struct {
	w      io.Writer
	styles styles
}

// styles groups precomputed lipgloss styles for each semantic token.
// Derived once at construction time (O(1) at render time per ADR-04).
type styles struct {
	primary lipgloss.Style
}

// New constructs an Adapter writing to w, deriving all colors from t.
// w may be any io.Writer (bytes.Buffer, os.Stdout, etc.) — REQ-03.
func New(w io.Writer, t theme.Theme) *Adapter {
	return &Adapter{
		w: w,
		styles: styles{
			primary: lipgloss.NewStyle().Foreground(t.Resolve(theme.TokPrimary)),
		},
	}
}

// emit writes the styled line to a.w, appending a trailing newline.
func (a *Adapter) emit(s lipgloss.Style, text string) {
	line := s.Render(text)
	if len(line) == 0 || line[len(line)-1] != '\n' {
		line += "\n"
	}
	_, _ = fmt.Fprint(a.w, line)
}

// Heading emits a primary-styled heading line (token: Primary).
// Implements output.Output.Heading [REQ-02: token mapping].
func (a *Adapter) Heading(text string) {
	a.emit(a.styles.primary, text)
}

// ── S-001 stubs — panic loudly so tests fail clearly if called before S-001 ──

// Body is not yet implemented (S-001).
func (a *Adapter) Body(_ string) { panic("themed.Adapter.Body: not implemented until S-001") }

// Hint is not yet implemented (S-001).
func (a *Adapter) Hint(_ string) { panic("themed.Adapter.Hint: not implemented until S-001") }

// Success is not yet implemented (S-001).
func (a *Adapter) Success(_ string) { panic("themed.Adapter.Success: not implemented until S-001") }

// Warning is not yet implemented (S-001).
func (a *Adapter) Warning(_ string) { panic("themed.Adapter.Warning: not implemented until S-001") }

// Error is not yet implemented (S-001).
func (a *Adapter) Error(_ string) { panic("themed.Adapter.Error: not implemented until S-001") }

// Path is not yet implemented (S-001).
func (a *Adapter) Path(_ string) { panic("themed.Adapter.Path: not implemented until S-001") }

// Prompt is not yet implemented (S-001).
func (a *Adapter) Prompt(_ string) (string, error) {
	panic("themed.Adapter.Prompt: not implemented until S-001")
}

// Newline is not yet implemented (S-001).
func (a *Adapter) Newline() { panic("themed.Adapter.Newline: not implemented until S-001") }

// Stream is not yet implemented (S-001).
func (a *Adapter) Stream(_ context.Context, _ <-chan events.Event) error {
	panic("themed.Adapter.Stream: not implemented until S-001")
}
