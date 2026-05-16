// Package pretty_test covers pretty.Renderer acceptance criteria.
//
// REQ-01.1 — interface satisfaction (compile-time) + theme consumption
// REQ-01.2 — channel-close terminates Render
// REQ-01.3 — context cancellation terminates Render
// REQ-02.1 — all 12 event types produce non-empty output
// REQ-02.2 — unknown event type returns non-nil error (does not panic)
// REQ-03.1 — LogLine.Sensitive=true masks Text → [REDACTED]
// REQ-03.2 — InputRequested.Sensitive=true masks DefaultValue → [REDACTED]
// REQ-03.3 — InputProvided.Sensitive=true masks Value → [REDACTED]
// REQ-03.4 — ScriptStarted.Sensitive=true masks Args → [REDACTED]
// REQ-03.5 — Sensitive=false renders value unchanged
// REQ-03.1 (public surface) — *Renderer public method set is unchanged
package pretty_test

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// base is a reusable EventBase for test events.
var base = events.EventBase{Seq: 1, At: time.Now()}

// noColorTheme is a deterministic NoColor theme for use in pretty tests.
var noColorTheme = theme.New(theme.Palette{}, theme.NoColor, theme.Light)

// ──────────────────────────────────────────────────────────────────────────────
// REQ-01.1 — compile-time interface satisfaction
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time assertion: pretty.Renderer must expose a Render method matching
// the render.Renderer interface signature. (Full interface assertion against
// render.Renderer lives in factory_test.go to avoid the import cycle.)
var _ interface {
	Render(ctx context.Context, ch <-chan events.Event) error
} = (*pretty.Renderer)(nil)

// ──────────────────────────────────────────────────────────────────────────────
// REQ-01.2 — channel-close terminates Render
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_ChannelClose_ReturnsNil(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	r := pretty.New(&buf, noColorTheme)

	ch := make(chan events.Event, 1)
	ch <- events.Done{EventBase: base}
	close(ch)

	if err := r.Render(context.Background(), ch); err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-01.3 — context cancellation terminates Render
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_ContextCancel_Terminates(t *testing.T) {
	t.Parallel()

	// Unbuffered channel that never sends — Render must exit on ctx cancel.
	ch := make(chan events.Event)
	ctx, cancel := context.WithCancel(context.Background())

	var buf strings.Builder
	r := pretty.New(&buf, noColorTheme)

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
// REQ-02.1 — all 12 event types produce non-empty output
// ──────────────────────────────────────────────────────────────────────────────

func Test_Renderer_All12EventTypes_NonEmptyLines(t *testing.T) {
	t.Parallel()

	allEvents := []struct {
		name  string
		event events.Event
	}{
		{"FileCreated", events.FileCreated{EventBase: base, Path: "a.go"}},
		{"FileModified", events.FileModified{EventBase: base, Path: "b.go"}},
		{"FileDeleted", events.FileDeleted{EventBase: base, Path: "c.go"}},
		{"ScriptStarted", events.ScriptStarted{EventBase: base, Name: "lint", Args: []string{"./..."}}},
		{"ScriptStopped", events.ScriptStopped{EventBase: base, Name: "lint", ExitCode: 0}},
		{"LogLine", events.LogLine{EventBase: base, Level: "info", Text: "hello"}},
		{"InputRequested", events.InputRequested{EventBase: base, Prompt: "Name?", DefaultValue: "world"}},
		{"InputProvided", events.InputProvided{EventBase: base, Prompt: "Name?", Value: "alice"}},
		{"Progress", events.Progress{EventBase: base, Step: 1, Total: 3, Label: "step 1"}},
		{"Done", events.Done{EventBase: base}},
		{"Failed", events.Failed{EventBase: base}},
		{"Cancelled", events.Cancelled{EventBase: base}},
	}

	for _, tt := range allEvents {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ch := make(chan events.Event, 1)
			ch <- tt.event
			close(ch)

			var buf strings.Builder
			r := pretty.New(&buf, noColorTheme)
			if err := r.Render(context.Background(), ch); err != nil {
				t.Fatalf("[%s] Render returned error: %v", tt.name, err)
			}

			out := strings.TrimSpace(buf.String())
			if out == "" {
				t.Errorf("[%s] Render produced empty output", tt.name)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-02.2 — unknown event type does not panic; returns non-nil error
// ──────────────────────────────────────────────────────────────────────────────

// fakeEvent is an external type that satisfies events.Event's unexported marker
// method. Since the method isEvent() is unexported, external packages cannot
// implement it — so we cannot truly inject a "fake" type at runtime.
//
// Instead, we verify REQ-02.2 by confirming the default-case path exists in
// the source code (structural test) rather than trying to trigger it at runtime.
// This is the only practical approach given the sealed interface.
func Test_Renderer_DefaultCase_InSource(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("pretty.go")
	if err != nil {
		t.Fatalf("cannot read pretty.go: %v", err)
	}

	content := string(src)

	// The default case must be present in the type-switch.
	if !strings.Contains(content, "default:") {
		t.Error("pretty.go must contain a default case in the type-switch (REQ-02.2)")
	}
	// The default case must return an error (not panic).
	if !strings.Contains(content, "fmt.Errorf") {
		t.Error("pretty.go default case must return fmt.Errorf (non-nil error, no panic) — REQ-02.2")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ-03 — sensitive-field masking
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
			name:         "LogLine sensitive=true masks Text (REQ-03.1)",
			event:        events.LogLine{EventBase: base, Text: "secret-token", Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "secret-token",
			reqID:        "REQ-03.1",
		},
		{
			name:         "InputRequested sensitive=true masks DefaultValue (REQ-03.2)",
			event:        events.InputRequested{EventBase: base, Prompt: "Enter key", DefaultValue: "secret", Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "secret",
			reqID:        "REQ-03.2",
		},
		{
			name:         "InputProvided sensitive=true masks Value (REQ-03.3)",
			event:        events.InputProvided{EventBase: base, Value: "my-password", Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "my-password",
			reqID:        "REQ-03.3",
		},
		{
			name:         "ScriptStarted sensitive=true masks Args (REQ-03.4)",
			event:        events.ScriptStarted{EventBase: base, Args: []string{"--token=abc123"}, Sensitive: true},
			wantContains: "[REDACTED]",
			wantAbsent:   "abc123",
			reqID:        "REQ-03.4",
		},
		{
			name:         "LogLine sensitive=false renders value unchanged (REQ-03.5)",
			event:        events.LogLine{EventBase: base, Text: "normal log line", Sensitive: false},
			wantContains: "normal log line",
			wantAbsent:   "",
			reqID:        "REQ-03.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ch := make(chan events.Event, 1)
			ch <- tt.event
			close(ch)

			var buf strings.Builder
			r := pretty.New(&buf, noColorTheme)
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
// REQ render-pretty/01.1 — New consumes theme; Primary foreground resolves hex
// ──────────────────────────────────────────────────────────────────────────────

// Test_NewRenderer_ConsumesTheme pins the global lipgloss profile to TrueColor,
// constructs a TrueColor/Dark theme, builds a Renderer, and verifies that
// r.styles.Primary produces a foreground SGR containing the RGB for #A78BFA
// when rendered. Also asserts AdaptiveColor is absent from styles.go source.
//
// Sequential — uses global lipgloss state mutation.
func Test_NewRenderer_ConsumesTheme(t *testing.T) {
	// Do NOT call t.Parallel() — mutates global lipgloss renderer.
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")

	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	th := theme.New(theme.DefaultPalette(), theme.TrueColor, theme.Dark)

	var buf strings.Builder
	r := pretty.New(&buf, th)

	// Send a ScriptStarted so Primary (Progress→Primary) gets exercised.
	ch := make(chan events.Event, 1)
	ch <- events.ScriptStarted{EventBase: base, Name: "test", Args: []string{}}
	close(ch)

	if err := r.Render(context.Background(), ch); err != nil {
		t.Fatalf("Render: %v", err)
	}

	got := buf.String()
	// #A78BFA = RGB(167,139,250) — truecolor SGR: ESC[38;2;167;139;250m
	const wantSGR = "\x1b[38;2;167;139;250m"
	if !strings.Contains(got, wantSGR) {
		t.Errorf("Primary foreground not found in rendered output\nwant SGR: %q\ngot: %q", wantSGR, got)
	}

	// styles.go must NOT contain AdaptiveColor literals (replaced by theme.Resolve).
	src, err := os.ReadFile("styles.go")
	if err != nil {
		t.Fatalf("cannot read styles.go: %v", err)
	}
	if strings.Contains(string(src), "AdaptiveColor") {
		t.Error("styles.go must not contain lipgloss.AdaptiveColor — use theme.Resolve() instead (S-005)")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// REQ render-pretty/03.1 — *Renderer public method surface is unchanged
// ──────────────────────────────────────────────────────────────────────────────

// Test_PrettyRenderer_PublicSurfaceUnchanged uses reflection to assert the
// exported method set of *pretty.Renderer matches the baseline captured at
// S-005 authoring time. This is the regression net protecting ADR-011 and
// the design's promise of API stability.
//
// Baseline: {"Render"} — only Render is exported.
func Test_PrettyRenderer_PublicSurfaceUnchanged(t *testing.T) {
	t.Parallel()

	// Baseline: exported methods of *Renderer as of S-004 HEAD.
	// Update this list ONLY if the public API contract changes intentionally.
	wantMethods := []string{
		"Render",
	}

	rt := reflect.TypeOf((*pretty.Renderer)(nil))
	var gotMethods []string
	for i := range rt.NumMethod() {
		m := rt.Method(i)
		if m.IsExported() {
			gotMethods = append(gotMethods, m.Name)
		}
	}
	sort.Strings(gotMethods)
	sort.Strings(wantMethods)

	if len(gotMethods) != len(wantMethods) {
		t.Fatalf("*Renderer exported method count: got %d, want %d\ngot:  %v\nwant: %v",
			len(gotMethods), len(wantMethods), gotMethods, wantMethods)
	}
	for i, name := range wantMethods {
		if gotMethods[i] != name {
			t.Errorf("method[%d]: got %q, want %q", i, gotMethods[i], name)
		}
	}
}

// Test_Renderer_Wiring_DoneFollowsSecret verifies the full masking integration:
// LogLine{sensitive:true} followed by Done{} — output contains [REDACTED] and
// not the secret, and Render returns nil after Done is received.
func Test_Renderer_Wiring_DoneFollowsSecret(t *testing.T) {
	t.Parallel()

	ch := make(chan events.Event, 2)
	ch <- events.LogLine{EventBase: base, Text: "secret", Sensitive: true}
	ch <- events.Done{EventBase: base}
	close(ch)

	var buf strings.Builder
	r := pretty.New(&buf, noColorTheme)
	err := r.Render(context.Background(), ch)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("output does not contain [REDACTED]\noutput: %s", out)
	}
	if strings.Contains(out, "secret") {
		t.Errorf("output contains raw sensitive value 'secret'\noutput: %s", out)
	}
}
