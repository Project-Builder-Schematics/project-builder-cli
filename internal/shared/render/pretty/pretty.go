// Package pretty provides Renderer, a human-facing output adapter that
// emits colour-coded, structured terminal lines using charmbracelet/lipgloss.
//
// Import note: this package intentionally does NOT import
// internal/shared/render to avoid an import cycle (factory.go in render/
// imports render/pretty). Interface satisfaction is asserted in factory_test.go.
package pretty

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// mask returns "[REDACTED]" when sensitive is true, otherwise returns v unchanged.
// Inlined per ADR-03 amendment (2026-05-11): import-cycle prevents sharing the
// render-level helper; each adapter owns an identical copy.
func mask(v string, sensitive bool) string {
	if sensitive {
		return "[REDACTED]"
	}
	return v
}

// Renderer is the human-facing output adapter for project-builder-cli.
// It selects lipgloss styles per event category and masks sensitive fields
// with the "[REDACTED]" placeholder.
//
// Renderer structurally satisfies render.Renderer via its Render method.
// The compile-time assertion lives in factory_test.go (cycle-free).
type Renderer struct {
	w      io.Writer
	styles Styles
}

// New constructs a Renderer writing to w, deriving all style colors from the
// provided theme (S-005: theme is now consumed; replaces the previous
// lipgloss.AdaptiveColor hard-coded styles).
func New(w io.Writer, t theme.Theme) *Renderer {
	return &Renderer{w: w, styles: NewStyles(t)}
}

// Render satisfies the render.Renderer interface. It emits one human-readable
// line per event, using lipgloss styles for visual hierarchy. Sensitive fields
// are replaced with "[REDACTED]". Returns nil on channel close; a non-nil error
// on unknown event type. Respects ctx cancellation.
func (r *Renderer) Render(ctx context.Context, ch <-chan events.Event) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			line, err := r.format(ev)
			if err != nil {
				return err
			}
			// Ensure each line ends with exactly one newline.
			if !strings.HasSuffix(line, "\n") {
				line += "\n"
			}
			if _, err := fmt.Fprint(r.w, line); err != nil {
				return fmt.Errorf("pretty.Renderer.Render: write: %w", err)
			}
		}
	}
}

// format converts a single event to a styled string. Returns a non-nil error
// for unknown event types (REQ-02.2 — default case must not panic).
func (r *Renderer) format(ev events.Event) (string, error) {
	switch e := ev.(type) {
	case events.FileCreated:
		glyph := "+"
		path := e.Path
		if e.IsDir {
			path += "/"
		}
		return r.styles.Accent.Render(glyph + " " + path), nil

	case events.FileModified:
		return r.styles.Accent.Render("~ " + e.Path), nil

	case events.FileDeleted:
		return r.styles.Accent.Render("- " + e.Path), nil

	case events.ScriptStarted:
		args := mask(strings.Join(e.Args, " "), e.Sensitive)
		return r.styles.Primary.Render(fmt.Sprintf("▶ %s %s", e.Name, args)), nil

	case events.ScriptStopped:
		return r.styles.Primary.Render(fmt.Sprintf("■ %s (exit %d)", e.Name, e.ExitCode)), nil

	case events.LogLine:
		text := mask(e.Text, e.Sensitive)
		return r.styles.Muted.Render(fmt.Sprintf("[%s] %s", e.Level, text)), nil

	case events.InputRequested:
		def := mask(e.DefaultValue, e.Sensitive)
		return r.styles.Primary.Render(fmt.Sprintf("? %s [%s]", e.Prompt, def)), nil

	case events.InputProvided:
		val := mask(e.Value, e.Sensitive)
		return r.styles.Primary.Render(fmt.Sprintf("> %s", val)), nil

	case events.Progress:
		return r.styles.Primary.Render(fmt.Sprintf("[%d/%d] %s", e.Step, e.Total, e.Label)), nil

	case events.Done:
		return r.styles.Foreground.Render("✓ done"), nil

	case events.Failed:
		msg := ""
		if e.Err != nil {
			msg = ": " + e.Err.Error()
		}
		return r.styles.Foreground.Render("✗ failed" + msg), nil

	case events.Cancelled:
		return r.styles.Foreground.Render("⊘ cancelled"), nil

	default:
		return "", fmt.Errorf("pretty.Renderer.format: unknown event type %T", ev)
	}
}
