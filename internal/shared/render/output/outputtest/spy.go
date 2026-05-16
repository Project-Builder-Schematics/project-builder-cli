// Package outputtest provides a test peer for the output.Output port.
//
// The Spy records all Output method invocations in call order so that
// feature handler tests can assert on which methods were called with which
// arguments — without caring about the rendered bytes (ADR-04: hybrid test
// discipline: goldens for adapter, spy for handler tests).
//
// Usage in tests:
//
//	spy := outputtest.New()
//	handler := myfeature.New(spy)
//	handler.Run(ctx, args)
//	spy.AssertCalledWith(t, "Heading", "Project initialised")
//
// The compile-time assertion below ensures Spy structurally satisfies
// output.Output. If the interface changes, this file fails to build.
package outputtest

import (
	"context"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
)

// Compile-time assertion: Spy must satisfy output.Output.
// Build failure here means the interface changed and Spy needs updating.
var _ output.Output = (*Spy)(nil)

// Spy is a test-only implementation of output.Output that records every
// invocation in call order. Construct via New(); do not create directly.
//
// Spy is NOT goroutine-safe. Tests must drive it sequentially.
type Spy struct {
	calls []output.Call
}

// New constructs a fresh Spy with an empty call history.
func New() *Spy {
	return &Spy{}
}

// Calls returns a copy of the recorded call slice in invocation order.
// The returned slice is independent of the spy's internal state —
// mutations do not affect the spy.
func (s *Spy) Calls() []output.Call {
	result := make([]output.Call, len(s.calls))
	copy(result, s.calls)
	return result
}

// AssertCalled fails t if method was never called on this Spy.
func (s *Spy) AssertCalled(t *testing.T, method string) {
	t.Helper()
	for _, c := range s.calls {
		if c.Method == method {
			return
		}
	}
	t.Errorf("outputtest.Spy: expected %q to be called, but it was not\nall calls: %v", method, s.methodNames())
}

// AssertCalledWith fails t if method was never called with the given args.
// args are compared positionally against each Call's Args slice.
func (s *Spy) AssertCalledWith(t *testing.T, method string, args ...string) {
	t.Helper()
	for _, c := range s.calls {
		if c.Method != method {
			continue
		}
		if stringSlicesEqual(c.Args, args) {
			return
		}
	}
	t.Errorf("outputtest.Spy: expected %q to be called with args %v, but it was not\nall calls: %v", method, args, s.calls)
}

// record appends a Call to the spy's internal history.
func (s *Spy) record(method string, args ...string) {
	s.calls = append(s.calls, output.Call{
		Method: method,
		Args:   args,
	})
}

// methodNames returns a slice of method names in call order (for error messages).
func (s *Spy) methodNames() []string {
	names := make([]string, len(s.calls))
	for i, c := range s.calls {
		names[i] = c.Method
	}
	return names
}

// stringSlicesEqual returns true if a and b have identical length and elements.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ── output.Output implementation ──────────────────────────────────────────────
//
// Every method records its invocation and args, then returns without producing
// any bytes. Spies record intent; they do not render.

// Heading records a "Heading" call with text.
func (s *Spy) Heading(text string) { s.record("Heading", text) }

// Body records a "Body" call with text.
func (s *Spy) Body(text string) { s.record("Body", text) }

// Hint records a "Hint" call with text.
func (s *Spy) Hint(text string) { s.record("Hint", text) }

// Success records a "Success" call with text.
func (s *Spy) Success(text string) { s.record("Success", text) }

// Warning records a "Warning" call with text.
func (s *Spy) Warning(text string) { s.record("Warning", text) }

// Error records an "Error" call with text.
func (s *Spy) Error(text string) { s.record("Error", text) }

// Path records a "Path" call with p.
func (s *Spy) Path(p string) { s.record("Path", p) }

// Prompt records a "Prompt" call with text and returns ("", nil).
// Feature handler tests that call Prompt should inject a real or stubbed
// reply by coupling Prompt to a fake reader at a higher level.
func (s *Spy) Prompt(text string) (string, error) {
	s.record("Prompt", text)
	return "", nil
}

// Newline records a "Newline" call with no args.
func (s *Spy) Newline() { s.record("Newline") }

// Stream records a "Stream" call and drains the channel until closed.
// Returns nil on clean drain; does NOT render events.
func (s *Spy) Stream(ctx context.Context, ch <-chan events.Event) error {
	s.record("Stream")
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ch:
			if !ok {
				return nil
			}
		}
	}
}
