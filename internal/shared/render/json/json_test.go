// Package json_test covers json.Renderer acceptance criteria.
//
// REQ-05.1 — interface satisfaction (compile-time)
// REQ-05.2 — channel-close terminates Render
// REQ-06.1 — NDJSON line-count invariant: N events → N newlines
// REQ-06.2 — each line is valid JSON
// REQ-06.3 — every line has a stable "type" field
// REQ-07.1 — LogLine.Sensitive=true → text field "[REDACTED]"
// REQ-07.2 — InputProvided.Sensitive=true → value field "[REDACTED]"
// REQ-07.3 — ScriptStarted.Sensitive=true → args field "[REDACTED]"
// REQ-07.4 — Sensitive=false → value unchanged
// REQ-08.1 — all 12 event types produce valid JSON output
package json_test

import (
	"context"
	stdjson "encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	renderjson "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/json"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// base is a reusable EventBase for test events.
var base = events.EventBase{Seq: 1, At: time.Now()}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-05.1 — compile-time interface satisfaction
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time assertion: json.Renderer must expose a Render method matching
// the render.Renderer interface signature. (Full interface assertion against
// render.Renderer lives in factory_test.go to avoid the import cycle.)
var _ interface {
	Render(ctx context.Context, ch <-chan events.Event) error
} = (*renderjson.Renderer)(nil)

// ──────────────────────────────────────────────────────────────────────────────
// REQ-05.2 — channel-close terminates Render
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_ChannelClose_ReturnsNil(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	r := renderjson.New(&buf)

	ch := make(chan events.Event, 1)
	ch <- events.Done{EventBase: base}
	close(ch)

	if err := r.Render(context.Background(), ch); err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-06.1 — NDJSON line-count invariant
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_NDJSONLineCount(t *testing.T) {
	t.Parallel()

	const n = 5
	ch := make(chan events.Event, n)
	for i := 0; i < n; i++ {
		ch <- events.LogLine{EventBase: base, Level: "info", Text: "hello"}
	}
	close(ch)

	var buf strings.Builder
	r := renderjson.New(&buf)
	if err := r.Render(context.Background(), ch); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	got := strings.Count(buf.String(), "\n")
	if got != n {
		t.Errorf("line count = %d; want %d\noutput:\n%s", got, n, buf.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-06.2 + REQ-06.3 — each line is valid JSON with a "type" field
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_EachLineIsValidJSON_WithTypeField(t *testing.T) {
	t.Parallel()

	ch := make(chan events.Event, 3)
	ch <- events.FileCreated{EventBase: base, Path: "foo.go"}
	ch <- events.LogLine{EventBase: base, Level: "info", Text: "msg"}
	ch <- events.Done{EventBase: base}
	close(ch)

	var buf strings.Builder
	r := renderjson.New(&buf)
	if err := r.Render(context.Background(), ch); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	lines := nonEmptyLines(buf.String())
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), buf.String())
	}

	for _, line := range lines {
		var obj map[string]any
		if err := stdjson.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("invalid JSON line %q: %v", line, err)
			continue
		}
		typ, ok := obj["type"].(string)
		if !ok || typ == "" {
			t.Errorf("line missing non-empty \"type\" field: %q", line)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-07 — sensitive-field redaction
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_SensitiveFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		event        events.Event
		wantContains string
		wantAbsent   string
		reqID        string
	}{
		{
			name:         "LogLine sensitive=true redacts text (REQ-07.1)",
			event:        events.LogLine{EventBase: base, Text: "secret-token", Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "secret-token",
			reqID:        "REQ-07.1",
		},
		{
			name:         "InputProvided sensitive=true redacts value (REQ-07.2)",
			event:        events.InputProvided{EventBase: base, Value: "my-password", Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "my-password",
			reqID:        "REQ-07.2",
		},
		{
			name:         "ScriptStarted sensitive=true redacts args (REQ-07.3)",
			event:        events.ScriptStarted{EventBase: base, Args: []string{"--token=abc123"}, Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "abc123",
			reqID:        "REQ-07.3",
		},
		{
			name:         "LogLine sensitive=false preserves value (REQ-07.4)",
			event:        events.LogLine{EventBase: base, Text: "normal log", Sensitive: false},
			wantContains: "normal log",
			wantAbsent:   "",
			reqID:        "REQ-07.4",
		},
		{
			name:         "InputRequested sensitive=true redacts default_value",
			event:        events.InputRequested{EventBase: base, DefaultValue: "secret-default", Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "secret-default",
			reqID:        "REQ-07 (InputRequested)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ch := make(chan events.Event, 1)
			ch <- tt.event
			close(ch)

			var buf strings.Builder
			r := renderjson.New(&buf)
			if err := r.Render(context.Background(), ch); err != nil {
				t.Fatalf("[%s] Render error: %v", tt.reqID, err)
			}

			out := buf.String()
			if !strings.Contains(out, tt.wantContains) {
				t.Errorf("[%s] output does not contain %q\noutput: %s", tt.reqID, tt.wantContains, out)
			}
			if tt.wantAbsent != "" && strings.Contains(out, tt.wantAbsent) {
				t.Errorf("[%s] output contains raw sensitive value %q\noutput: %s", tt.reqID, tt.wantAbsent, out)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-08.1 — all 12 event types produce valid JSON output
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_All12EventTypes(t *testing.T) {
	t.Parallel()

	// One instance of each of the 12 sealed event types.
	allEvents := []events.Event{
		events.FileCreated{EventBase: base, Path: "a.go"},
		events.FileModified{EventBase: base, Path: "b.go"},
		events.FileDeleted{EventBase: base, Path: "c.go"},
		events.ScriptStarted{EventBase: base, Name: "lint", Args: []string{"./..."}},
		events.ScriptStopped{EventBase: base, Name: "lint", ExitCode: 0},
		events.LogLine{EventBase: base, Level: "info", Text: "hello"},
		events.InputRequested{EventBase: base, Prompt: "Name?", DefaultValue: "world"},
		events.InputProvided{EventBase: base, Prompt: "Name?", Value: "alice"},
		events.Progress{EventBase: base, Step: 1, Total: 3, Label: "step 1"},
		events.Done{EventBase: base},
		events.Failed{EventBase: base},
		events.Cancelled{EventBase: base},
	}

	ch := make(chan events.Event, len(allEvents))
	for _, ev := range allEvents {
		ch <- ev
	}
	close(ch)

	var buf strings.Builder
	r := renderjson.New(&buf)
	if err := r.Render(context.Background(), ch); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	lines := nonEmptyLines(buf.String())
	if len(lines) != len(allEvents) {
		t.Fatalf("expected %d lines, got %d\noutput:\n%s", len(allEvents), len(lines), buf.String())
	}

	// ADR-04: stable snake_case type discriminators — one per event type.
	stableTypes := map[string]bool{
		"file_created":    false,
		"file_modified":   false,
		"file_deleted":    false,
		"script_started":  false,
		"script_stopped":  false,
		"log_line":        false,
		"input_requested": false,
		"input_provided":  false,
		"progress":        false,
		"done":            false,
		"failed":          false,
		"cancelled":       false,
	}

	for i, line := range lines {
		var obj map[string]any
		if err := stdjson.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d invalid JSON %q: %v", i+1, line, err)
			continue
		}
		typ, ok := obj["type"].(string)
		if !ok || typ == "" {
			t.Errorf("line %d missing non-empty \"type\" field: %q", i+1, line)
			continue
		}
		if _, known := stableTypes[typ]; !known {
			t.Errorf("line %d has unknown type %q (not in stable catalogue)", i+1, typ)
		} else {
			stableTypes[typ] = true
		}
		// Verify seq and at envelope fields are present.
		if _, ok := obj["seq"]; !ok {
			t.Errorf("line %d missing \"seq\" field: %q", i+1, line)
		}
		if _, ok := obj["at"]; !ok {
			t.Errorf("line %d missing \"at\" field: %q", i+1, line)
		}
	}

	for typ, seen := range stableTypes {
		if !seen {
			t.Errorf("event type %q was never emitted in 12-event sweep", typ)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Context cancellation — Render must honour ctx.Done()
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_ContextCancel_Terminates(t *testing.T) {
	t.Parallel()

	// Unbuffered channel that never sends — Render must exit on ctx cancel.
	ch := make(chan events.Event)
	ctx, cancel := context.WithCancel(context.Background())

	var buf strings.Builder
	r := renderjson.New(&buf)

	done := make(chan error, 1)
	go func() {
		done <- r.Render(ctx, ch)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Render returned non-nil error after ctx cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Render did not return after ctx cancel (2s timeout)")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// schema.go CONTRACT comment check (ADR-04 + tech-writer note)
// ──────────────────────────────────────────────────────────────────────────────

// Test_Schema_ContractCommentPresent reads schema.go and verifies the
// mandatory "// CONTRACT: stable" comment is present on the Type field.
// Mutation: remove the comment → this test fails.
func Test_Schema_ContractCommentPresent(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("schema.go")
	if err != nil {
		t.Fatalf("cannot read schema.go: %v", err)
	}
	if !strings.Contains(string(src), "// CONTRACT: stable") {
		t.Error("schema.go must contain \"// CONTRACT: stable\" comment on the Type field (ADR-04)")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
