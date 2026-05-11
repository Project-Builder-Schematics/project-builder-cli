// Package engine defines the Engine port for project-builder-cli.
//
// # Anti-script invariant
//
// Implementations MUST NOT accept command strings derived from event payloads
// or user input. The os/exec boundary is gated by ExecuteRequest typed fields;
// no implementation may construct shell strings from arbitrary input. Scripts
// submitted via lifecycle hooks are resolved by name from the schematic
// manifest — never executed directly by the Engine from user-provided strings.
//
// # Cancellation ceiling
//
// Implementations MUST honour ctx.Done() within a bounded window. The
// end-to-end ceiling — from cancel signal to channel close — is 5 seconds.
// Concrete SIGTERM/SIGKILL timing windows are defined at /plan #4 within
// this 5s envelope.
//
// CONTRACT:STUB — behaviour-deferred to /plan #3+
package engine

import (
	"context"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// Engine spawns and controls a schematic execution.
//
// # Anti-script invariant
//
// Implementations MUST NOT accept command strings derived from event payloads
// or user input. The os/exec boundary is gated by ExecuteRequest typed fields;
// no implementation may construct shell strings from arbitrary input.
//
// # Cancellation ceiling
//
// Implementations MUST honour ctx.Done() within a bounded window. The
// end-to-end ceiling from cancel signal to channel close is 5 seconds.
// Callers MUST NOT assume faster cancellation. Concrete SIGTERM/SIGKILL
// windows are defined at /plan #4, within this 5-second envelope.
//
// # Usage
//
//	ch, err := engine.Execute(ctx, req)
//	if err != nil {
//	    // setup-time failure
//	}
//	for ev := range ch {
//	    // handle event; channel closes on Done, Failed, or Cancelled
//	}
type Engine interface {
	// Execute begins schematic execution and returns a read-only event channel.
	// The channel is closed when execution completes (Done, Failed, or Cancelled).
	// Cancelling ctx triggers a Cancelled terminal event followed by channel close
	// within the 5-second ceiling.
	Execute(ctx context.Context, req ExecuteRequest) (<-chan events.Event, error)
}

// ExecuteRequest is the typed request for Engine.Execute.
//
// All fields are typed — no raw string injection of schematic identity or
// execution arguments is permitted (see anti-script invariant above).
type ExecuteRequest struct {
	// Workspace is the absolute path to the project root directory.
	Workspace string

	// Schematic is the typed reference to the schematic to execute.
	// Must never be constructed from unvalidated user input.
	Schematic SchematicRef

	// Inputs contains schema-validated key/value pairs for the schematic.
	Inputs map[string]any

	// EnvAllowlist contains environment variable names permitted to be
	// inherited by the subprocess. Only listed names are forwarded.
	EnvAllowlist []string // fitness:allow-untyped-args env-allowlist
}

// SchematicRef is a typed reference to a schematic within a collection.
//
// This is a struct (not a string alias) to prevent raw string injection
// into the engine execution path (security.REQ-01.2). Callers must
// explicitly name Collection, Name, and Version — they cannot pass an
// opaque string that might embed shell metacharacters.
type SchematicRef struct {
	// Collection is the schematic collection identifier.
	// e.g. "@schematics/angular" or "./local".
	Collection string

	// Name is the schematic name within the collection.
	// e.g. "component", "module", "service".
	Name string

	// Version is the semver pin or "latest".
	Version string
}

// FakeEngine is a test/dev stub implementation of the Engine interface.
//
// FakeEngine honours context cancellation within the 5-second ceiling
// mandated by the Engine interface contract. On ctx.Done(), it emits a
// Cancelled terminal event and closes the channel promptly.
//
// # Sensitive propagation (CONTRACT:STUB)
//
// When Inbox is non-empty, FakeEngine emits each InputRequested event,
// waits for a reply on the paired InboxReplies receive channel (ctx-aware),
// then emits a paired InputProvided with Sensitive propagated from the
// request. After the Inbox is exhausted the goroutine blocks on ctx.Done()
// and emits Cancelled before closing.
//
// Inbox and InboxReplies must have the same length; index i of InboxReplies
// is the receive side of the channel stored in Inbox[i].Reply.
//
// CONTRACT:STUB — FakeEngine is for testing and composition-root wiring
// during the skeleton phase only. Replace at /plan #4 with
// AngularSubprocessAdapter or equivalent real implementation.
type FakeEngine struct {
	// Inbox is a test-only list of InputRequested events that FakeEngine will
	// emit (in order) before blocking on ctx.Done(). For each entry, FakeEngine
	// waits for a reply on the paired InboxReplies channel and emits a paired
	// InputProvided{Sensitive: req.Sensitive, Prompt: req.Prompt, Value: reply}.
	//
	// CONTRACT:STUB — production engines do not use this field.
	Inbox []events.InputRequested

	// InboxReplies is the receive side of each InputRequested.Reply channel,
	// paired by index with Inbox. Must be the same length as Inbox (or nil
	// when Inbox is empty).
	//
	// CONTRACT:STUB — production engines do not use this field.
	InboxReplies []<-chan string
}

// Execute implements Engine. Returns a channel that processes Inbox entries
// (if any), then blocks until ctx is cancelled, emitting a Cancelled terminal
// event and closing the channel.
//
// Sensitive propagation: for each InputRequested in Inbox, FakeEngine emits
// the request, receives the reply via InboxReplies[i] (or cancels if ctx
// fires first), and emits a paired InputProvided with Sensitive propagated
// from the request. CONTRACT:STUB — see FakeEngine doc.
func (f *FakeEngine) Execute(ctx context.Context, _ ExecuteRequest) (<-chan events.Event, error) {
	// Buffer large enough to hold all Inbox events plus their paired responses
	// plus the terminal Cancelled event; avoids blocking the goroutine on send.
	bufSize := len(f.Inbox)*2 + 1
	if bufSize < 1 {
		bufSize = 1
	}
	ch := make(chan events.Event, bufSize)

	go func() {
		defer close(ch)

		// Process each pre-loaded InputRequested in order.
		for i, req := range f.Inbox {
			// Emit the request so consumers can observe it.
			ch <- req

			// Receive the reply from the paired channel (ctx-aware).
			var value string
			select {
			case v := <-f.InboxReplies[i]:
				value = v
			case <-ctx.Done():
				ch <- events.Cancelled{}
				return
			}

			// Emit paired InputProvided with Sensitive propagated.
			ch <- events.InputProvided{
				EventBase: events.EventBase{Seq: req.Seq + 1, At: req.At},
				Prompt:    req.Prompt,
				Value:     value,
				Sensitive: req.Sensitive, // CONTRACT: propagate from request
			}
		}

		// Inbox exhausted.
		// If the Inbox was non-empty, emit Done to signal clean completion so
		// consumers and tests don't have to cancel to drain the channel.
		// If the Inbox was empty (zero items), block on ctx.Done() as before —
		// this preserves the original FakeEngine behaviour for callers that
		// don't use Inbox and control termination via context cancellation.
		if len(f.Inbox) > 0 {
			ch <- events.Done{}
			return
		}
		<-ctx.Done()
		ch <- events.Cancelled{}
	}()

	return ch, nil
}
