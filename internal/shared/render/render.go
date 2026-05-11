// Package render defines the Renderer port for project-builder-cli.
//
// CONTRACT:STUB — behaviour-deferred to /plan #3+
package render

import (
	"context"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// Renderer consumes an event stream and produces output.
//
// CONTRACT: implementations MUST return when the events channel is closed.
//
// # Security — Sensitive field masking
//
// When a field is marked Sensitive=true, Pretty renderers MUST NOT display
// the sensitive value; JSON renderers MUST emit "value":"[REDACTED]".
// This mandate applies to the following four fields:
//   - InputRequested.Sensitive (masks the default value display)
//   - InputProvided.Sensitive  (masks Value — the user's reply)
//   - LogLine.Sensitive        (masks Text — the log line content)
//   - ScriptStarted.Sensitive  (masks Args — the script arguments)
//
// Concrete renderer implementations (Pretty, JSON) are defined at /plan #3.
// The masking contract is enforced there; this interface declares the obligation.
type Renderer interface {
	// Render consumes the events channel until it is closed, producing output
	// appropriate to the implementation. Returns nil on clean channel close.
	// Cancelling ctx is advisory — implementations should respect it but the
	// primary termination signal is channel close.
	Render(ctx context.Context, events <-chan events.Event) error
}

// NoopRenderer is a test/dev stub implementation of Renderer.
//
// NoopRenderer drains the event channel without producing any output and
// without panicking — including when events carry Sensitive=true fields.
// It does NOT mask sensitive values (masking is the responsibility of
// concrete renderer implementations at /plan #3).
//
// CONTRACT:STUB — NoopRenderer is for testing and composition-root wiring
// during the skeleton phase only. Replace at /plan #3 with PrettyRenderer
// or JSONRenderer.
type NoopRenderer struct{}

// Render implements Renderer by draining the channel and discarding all events.
// Returns nil on channel close. Does not panic on Sensitive=true events.
func (n *NoopRenderer) Render(_ context.Context, ch <-chan events.Event) error {
	for range ch { //nolint:revive // noop drain is intentional
	}
	return nil
}
