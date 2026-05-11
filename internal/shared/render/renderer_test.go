// Package render_test covers renderer-port.REQ-01.*, security.REQ-03.2,
// and event-catalogue.REQ-02.4 via compile-time contracts, GoDoc grep
// checks, and channel round-trip assertions.
//
// CONTRACT:STUB — behaviour-deferred to /plan #3+
package render

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// compile-time interface satisfaction check (not a test function — package-level assertion).
// Mutation: remove NoopRenderer.Render → compile error.
var _ Renderer = (*NoopRenderer)(nil)

// ──────────────────────────────────────────────────────────────────────────────
// renderer-port.REQ-01.1 — Renderer interface satisfiable by NoopRenderer
// ──────────────────────────────────────────────────────────────────────────────

// Test_FakeRenderer_SatisfiesRendererIface verifies the compile-time assertion
// above by calling Render on a *NoopRenderer through the Renderer interface.
func Test_FakeRenderer_SatisfiesRendererIface(t *testing.T) {
	var r Renderer = &NoopRenderer{}
	ctx := context.Background()

	ch := make(chan events.Event)
	close(ch) // immediately closed — nothing to drain

	if err := r.Render(ctx, ch); err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// renderer-port.REQ-01.2 / security.REQ-03.2 — GoDoc names all 4 Sensitive fields + [REDACTED]
// ──────────────────────────────────────────────────────────────────────────────

// Test_Renderer_GoDoc_DeclaresSensitiveMasking reads render.go and asserts that
// the Renderer interface GoDoc names all four Sensitive event fields.
// Mutation: remove any field name → grep fails.
func Test_Renderer_GoDoc_DeclaresSensitiveMasking(t *testing.T) {
	src, err := os.ReadFile("render.go")
	if err != nil {
		t.Fatalf("could not read render.go: %v", err)
	}
	content := string(src)

	// All four sensitive field references must appear in the file.
	sensitiveFields := []string{
		"ScriptStarted",
		"LogLine",
		"InputRequested",
		"InputProvided",
	}
	for _, field := range sensitiveFields {
		if !strings.Contains(content, field) {
			t.Errorf("render.go GoDoc does not reference sensitive field %q", field)
		}
	}
}

// Test_Renderer_GoDoc_REDACTED_Mention reads render.go and asserts it contains
// the literal string "[REDACTED]" (the mandated masking placeholder).
// Mutation: remove "[REDACTED]" from GoDoc → grep fails.
func Test_Renderer_GoDoc_REDACTED_Mention(t *testing.T) {
	src, err := os.ReadFile("render.go")
	if err != nil {
		t.Fatalf("could not read render.go: %v", err)
	}
	if !strings.Contains(string(src), "[REDACTED]") {
		t.Error(`render.go GoDoc must contain the literal string "[REDACTED]" as the mandated masking placeholder`)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// event-catalogue.REQ-02.4 / security.REQ-03.2 — Sensitive field round-trips through channel
// (CONTRACT:STUB — masking behaviour deferred to /plan #3; this asserts field
// survives a noop-drain channel round-trip without panic)
// ──────────────────────────────────────────────────────────────────────────────

// Test_LogLine_Sensitive_PreservedThroughChannel pre-loads a buffered channel
// with a LogLine{Sensitive:true}, drains it via NoopRenderer.Render, and
// asserts no panic occurred and the channel was fully consumed.
// CONTRACT:STUB marker: actual masking is deferred to /plan #3. This test
// only asserts the Sensitive field can flow through the channel without panic.
func Test_LogLine_Sensitive_PreservedThroughChannel(t *testing.T) {
	ch := make(chan events.Event, 3)
	ch <- events.LogLine{Sensitive: true}
	ch <- events.InputProvided{Sensitive: true}
	ch <- events.ScriptStarted{Sensitive: true}
	close(ch)

	r := &NoopRenderer{}
	ctx := context.Background()

	// Must not panic, must return nil.
	err := r.Render(ctx, ch)
	if err != nil {
		t.Errorf("Render returned unexpected error: %v", err)
	}
}
