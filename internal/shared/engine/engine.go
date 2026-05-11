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
// CONTRACT:STUB — FakeEngine is for testing and composition-root wiring
// during the skeleton phase only. Replace at /plan #4 with
// AngularSubprocessAdapter or equivalent real implementation.
type FakeEngine struct{}

// Execute implements Engine. Returns a channel that emits a single Cancelled
// event and closes when ctx is cancelled, or emits Done and closes immediately
// if ctx is already done.
//
// Sensitive propagation: if an InputRequested event were emitted (future
// extension), FakeEngine would emit a paired InputProvided with
// Sensitive propagated from the request. Current stub honours the channel
// close contract only.
func (f *FakeEngine) Execute(ctx context.Context, _ ExecuteRequest) (<-chan events.Event, error) {
	ch := make(chan events.Event, 1)

	go func() {
		defer close(ch)

		// Block until ctx is cancelled, then emit Cancelled terminal event.
		// Honouring the 5-second cancellation ceiling mandated by the interface.
		<-ctx.Done()
		ch <- events.Cancelled{}
	}()

	return ch, nil
}
