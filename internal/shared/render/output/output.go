// Package output defines the Output port for project-builder-cli.
//
// All user-facing emission in builder commands flows through the Output
// interface. This keeps feature handlers decoupled from rendering concerns
// and enables deterministic testing via outputtest.Spy.
//
// See doc.go for the full port + adapter convention and the Prompt-vs-Stream rule.
package output

import (
	"context"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// Call records a single invocation of any Output method.
// Used by outputtest.Spy to capture ordered call history for assertions.
type Call struct {
	// Method is the name of the Output method called (e.g. "Heading").
	Method string

	// Args holds the text arguments passed to the method (if any).
	// For Newline and Stream, Args is empty.
	Args []string

	// Bytes holds the raw bytes written to the underlying writer, if any.
	// For adapter implementations this is populated; for spy implementations
	// it may be nil or empty (spies record intent, not bytes).
	Bytes []byte
}

// Output is the unified user-facing emission port.
//
// All feature handlers that produce terminal output MUST accept Output as a
// dependency — never write directly to os.Stdout or fmt.Print*.
//
// Implementations:
//   - output/themed.Adapter — production; styled bytes via theme.Theme + lipgloss.
//   - output/outputtest.Spy — test peer; records calls for assertion.
type Output interface {
	// Heading emits a primary-styled heading line (token: Primary).
	Heading(text string)

	// Body emits a regular body line (token: Foreground).
	Body(text string)

	// Hint emits a muted hint line (token: Muted).
	Hint(text string)

	// Success emits a success line (token: Success).
	Success(text string)

	// Warning emits a warning line (token: Warning).
	Warning(text string)

	// Error emits an error line (token: Error).
	Error(text string)

	// Path emits a filesystem path line (token: Accent).
	Path(p string)

	// Prompt writes a styled prompt prefix and reads a single line of user input.
	// Returns the user's reply trimmed of the trailing newline.
	// See doc.go — Prompt is synchronous; engine-driven prompts go through Stream.
	Prompt(text string) (string, error)

	// Newline emits a blank line for visual spacing.
	Newline()

	// Stream consumes an events channel until closed, rendering each event.
	// Returns nil on clean channel close; a non-nil error on write or unknown-event failures.
	// Respects ctx cancellation as an advisory termination signal.
	Stream(ctx context.Context, ch <-chan events.Event) error
}
