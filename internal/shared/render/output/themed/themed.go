// Package themed is the production adapter for the output.Output port.
//
// themed.Adapter writes styled bytes to any io.Writer, deriving all colors
// from the provided theme.Theme via the 8-token vocabulary. It does NOT hold
// a hard reference to os.Stdout (output-port/REQ-03).
//
// S-001 completes all 9 remaining methods (Body, Hint, Success, Warning, Error,
// Path, Prompt, Newline, Stream). Prompt uses a functional option pattern for
// reader injection; Stream delegates to pretty.Renderer byte-for-byte.
package themed

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// Option configures an Adapter at construction time.
type Option func(*Adapter)

// WithReader overrides the default reader (os.Stdin) used by Prompt.
// Primarily useful in tests to inject a fake input source.
func WithReader(r io.Reader) Option {
	return func(a *Adapter) {
		a.reader = r
	}
}

// Adapter is the production implementation of output.Output.
// Construct via New; do not create directly.
type Adapter struct {
	w        io.Writer
	styles   adapterStyles
	renderer *pretty.Renderer
	reader   io.Reader
}

// adapterStyles groups precomputed lipgloss styles for each semantic token.
// Derived once at construction time (O(1) at render time per ADR-04).
type adapterStyles struct {
	primary    lipgloss.Style
	foreground lipgloss.Style
	muted      lipgloss.Style
	success    lipgloss.Style
	warning    lipgloss.Style
	errStyle   lipgloss.Style
	accent     lipgloss.Style
}

// New constructs an Adapter writing to w, deriving all colors from t.
// w may be any io.Writer (bytes.Buffer, os.Stdout, etc.) — REQ-03.
// opts may include WithReader to override the default os.Stdin for Prompt.
func New(w io.Writer, t theme.Theme, opts ...Option) *Adapter {
	a := &Adapter{
		w:        w,
		renderer: pretty.New(w, t),
		reader:   os.Stdin,
		styles: adapterStyles{
			primary:    lipgloss.NewStyle().Foreground(t.Resolve(theme.TokPrimary)),
			foreground: lipgloss.NewStyle().Foreground(t.Resolve(theme.TokForeground)),
			muted:      lipgloss.NewStyle().Foreground(t.Resolve(theme.TokMuted)),
			success:    lipgloss.NewStyle().Foreground(t.Resolve(theme.TokSuccess)),
			warning:    lipgloss.NewStyle().Foreground(t.Resolve(theme.TokWarning)),
			errStyle:   lipgloss.NewStyle().Foreground(t.Resolve(theme.TokError)),
			accent:     lipgloss.NewStyle().Foreground(t.Resolve(theme.TokAccent)),
		},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
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

// Body emits a regular body line (token: Foreground).
func (a *Adapter) Body(text string) {
	a.emit(a.styles.foreground, text)
}

// Hint emits a muted hint line (token: Muted).
func (a *Adapter) Hint(text string) {
	a.emit(a.styles.muted, text)
}

// Success emits a success line (token: Success).
func (a *Adapter) Success(text string) {
	a.emit(a.styles.success, text)
}

// Warning emits a warning line (token: Warning).
func (a *Adapter) Warning(text string) {
	a.emit(a.styles.warning, text)
}

// Error emits an error line (token: Error).
func (a *Adapter) Error(text string) {
	a.emit(a.styles.errStyle, text)
}

// Path emits a filesystem path line (token: Accent).
func (a *Adapter) Path(p string) {
	a.emit(a.styles.accent, p)
}

// Prompt writes a styled prompt prefix (Primary token, prefix "? ") followed by
// a space to a.w (no trailing newline — cursor stays on the same line), then
// reads a single line from a.reader. Returns the line trimmed of trailing \n/\r\n.
// Any reader error is returned verbatim.
func (a *Adapter) Prompt(text string) (string, error) {
	prefix := a.styles.primary.Render("? " + text + " ")
	_, _ = fmt.Fprint(a.w, prefix)

	line, err := bufio.NewReader(a.reader).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// Newline emits a blank line for visual spacing.
func (a *Adapter) Newline() {
	_, _ = fmt.Fprint(a.w, "\n")
}

// Stream consumes an events channel until closed, delegating to the
// pretty.Renderer constructed at New time — produces byte-identical output
// to a direct pretty.Renderer.Render call (render-pretty/REQ-06.1).
func (a *Adapter) Stream(ctx context.Context, ch <-chan events.Event) error {
	return a.renderer.Render(ctx, ch)
}
