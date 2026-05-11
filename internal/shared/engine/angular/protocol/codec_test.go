// Package protocol_test covers codec.go.
//
// S-003 scope:
//   - REQ-08.1: all 12 event types decoded correctly
//   - REQ-08.2: malformed JSON line emits Failed event
//   - REQ-08.3: unknown type field emits Failed event
//   - REQ-08.4: Seq field passed through unchanged (no resequencing)
//   - REQ-12.1: ScriptStarted.Sensitive=true preserved
//   - REQ-12.2: ScriptStarted.Sensitive=false default when field absent
//   - REQ-13.1: LogLine.Sensitive=true preserved
package protocol_test

import (
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular/protocol"
	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// Test_DecodeEvent_AllTwelveTypes covers REQ-08.1:
// all 12 event types are decoded into the correct Go types.
func Test_DecodeEvent_AllTwelveTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		wantType interface{}
	}{
		{
			name:     "file_created",
			input:    `{"type":"file_created","seq":1,"path":"src/app.ts","is_dir":false}`,
			wantType: events.FileCreated{},
		},
		{
			name:     "file_modified",
			input:    `{"type":"file_modified","seq":2,"path":"src/app.ts"}`,
			wantType: events.FileModified{},
		},
		{
			name:     "file_deleted",
			input:    `{"type":"file_deleted","seq":3,"path":"src/old.ts"}`,
			wantType: events.FileDeleted{},
		},
		{
			name:     "script_started",
			input:    `{"type":"script_started","seq":4,"name":"install","args":["--save"]}`,
			wantType: events.ScriptStarted{},
		},
		{
			name:     "script_stopped",
			input:    `{"type":"script_stopped","seq":5,"name":"install","exit_code":0}`,
			wantType: events.ScriptStopped{},
		},
		{
			name:     "log_line",
			input:    `{"type":"log_line","seq":6,"level":"info","source":"stdout","text":"hello"}`,
			wantType: events.LogLine{},
		},
		{
			name:     "input_requested",
			input:    `{"type":"input_requested","seq":7,"prompt":"Name?","sensitive":false}`,
			wantType: events.InputRequested{},
		},
		{
			name:     "input_provided",
			input:    `{"type":"input_provided","seq":8,"prompt":"Name?","value":"foo"}`,
			wantType: events.InputProvided{},
		},
		{
			name:     "progress",
			input:    `{"type":"progress","seq":9,"step":1,"total":5,"label":"creating files"}`,
			wantType: events.Progress{},
		},
		{
			name:     "done",
			input:    `{"type":"done","seq":10}`,
			wantType: events.Done{},
		},
		{
			name:     "failed",
			input:    `{"type":"failed","seq":11,"message":"something broke"}`,
			wantType: events.Failed{},
		},
		{
			name:     "cancelled",
			input:    `{"type":"cancelled","seq":12}`,
			wantType: events.Cancelled{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := protocol.DecodeEvent([]byte(tc.input))
			if got == nil {
				t.Fatalf("DecodeEvent(%q) = nil, want %T", tc.input, tc.wantType)
			}
			// Check type matches using type switch on zero-value sentinel.
			gotType := typeName(got)
			wantTypeName := typeName(tc.wantType)
			if gotType != wantTypeName {
				t.Errorf("DecodeEvent(%q) type = %s, want %s — REQ-08.1 violated", tc.input, gotType, wantTypeName)
			}
		})
	}
}

// Test_DecodeEvent_MalformedJSON covers REQ-08.2:
// malformed JSON → Failed{Err: *errors.Error{Code: ErrCodeExecutionFailed}}.
func Test_DecodeEvent_MalformedJSON(t *testing.T) {
	t.Parallel()

	got := protocol.DecodeEvent([]byte("not valid json"))

	failed, ok := got.(events.Failed)
	if !ok {
		t.Fatalf("DecodeEvent(malformed) = %T, want events.Failed — REQ-08.2 violated", got)
	}
	if failed.Err == nil {
		t.Fatal("Failed.Err is nil — REQ-08.2 violated")
	}
	appErr := extractError(t, failed.Err)
	if appErr.Code != apperrors.ErrCodeExecutionFailed {
		t.Errorf("Failed.Err.Code = %q, want %q — REQ-08.2", appErr.Code, apperrors.ErrCodeExecutionFailed)
	}
}

// Test_DecodeEvent_UnknownType covers REQ-08.3:
// unknown type field → Failed event.
func Test_DecodeEvent_UnknownType(t *testing.T) {
	t.Parallel()

	got := protocol.DecodeEvent([]byte(`{"type":"unknown_future_type","seq":1}`))

	if _, ok := got.(events.Failed); !ok {
		t.Errorf("DecodeEvent(unknown type) = %T, want events.Failed — REQ-08.3 violated", got)
	}
}

// Test_DecodeEvent_SeqPassthrough covers REQ-08.4:
// Seq field is passed through unchanged; codec does NOT resequence.
func Test_DecodeEvent_SeqPassthrough(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		wantSeq uint64
	}{
		{`{"type":"done","seq":1}`, 1},
		{`{"type":"done","seq":42}`, 42},
		{`{"type":"done","seq":99999}`, 99999},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := protocol.DecodeEvent([]byte(tc.input))
			done, ok := got.(events.Done)
			if !ok {
				t.Fatalf("DecodeEvent(%q) = %T, want events.Done", tc.input, got)
			}
			if done.Seq != tc.wantSeq {
				t.Errorf("Done.Seq = %d, want %d — REQ-08.4 violated", done.Seq, tc.wantSeq)
			}
		})
	}
}

// Test_DecodeEvent_FileCreated_Fields covers REQ-08.1 (field fidelity — FileCreated).
func Test_DecodeEvent_FileCreated_Fields(t *testing.T) {
	t.Parallel()

	got := protocol.DecodeEvent([]byte(`{"type":"file_created","seq":3,"path":"src/hello.ts","is_dir":false}`))
	fc, ok := got.(events.FileCreated)
	if !ok {
		t.Fatalf("got %T, want events.FileCreated", got)
	}
	if fc.Seq != 3 {
		t.Errorf("Seq = %d, want 3", fc.Seq)
	}
	if fc.Path != "src/hello.ts" {
		t.Errorf("Path = %q, want %q", fc.Path, "src/hello.ts")
	}
	if fc.IsDir {
		t.Error("IsDir = true, want false")
	}
}

// Test_DecodeEvent_ScriptStarted_SensitiveTrue covers REQ-12.1:
// Sensitive=true preserved through codec.
func Test_DecodeEvent_ScriptStarted_SensitiveTrue(t *testing.T) {
	t.Parallel()

	input := `{"type":"script_started","seq":1,"name":"install","args":["--token=abc"],"sensitive":true}`
	got := protocol.DecodeEvent([]byte(input))

	ss, ok := got.(events.ScriptStarted)
	if !ok {
		t.Fatalf("got %T, want events.ScriptStarted — REQ-12.1 violated", got)
	}
	if !ss.Sensitive {
		t.Error("ScriptStarted.Sensitive = false, want true — REQ-12.1 violated")
	}
	if ss.Name != "install" {
		t.Errorf("Name = %q, want %q", ss.Name, "install")
	}
}

// Test_DecodeEvent_ScriptStarted_SensitiveFalseDefault covers REQ-12.2:
// Sensitive=false (default) when field is absent from JSON.
func Test_DecodeEvent_ScriptStarted_SensitiveFalseDefault(t *testing.T) {
	t.Parallel()

	input := `{"type":"script_started","seq":1,"name":"install","args":[]}`
	got := protocol.DecodeEvent([]byte(input))

	ss, ok := got.(events.ScriptStarted)
	if !ok {
		t.Fatalf("got %T, want events.ScriptStarted — REQ-12.2 violated", got)
	}
	if ss.Sensitive {
		t.Error("ScriptStarted.Sensitive = true when field absent — REQ-12.2 violated")
	}
}

// Test_DecodeEvent_LogLine_SensitiveTrue covers REQ-13.1:
// LogLine.Sensitive=true preserved from child JSON.
func Test_DecodeEvent_LogLine_SensitiveTrue(t *testing.T) {
	t.Parallel()

	input := `{"type":"log_line","seq":1,"level":"info","source":"stdout","text":"secret-value","sensitive":true}`
	got := protocol.DecodeEvent([]byte(input))

	ll, ok := got.(events.LogLine)
	if !ok {
		t.Fatalf("got %T, want events.LogLine — REQ-13.1 violated", got)
	}
	if !ll.Sensitive {
		t.Error("LogLine.Sensitive = false, want true — REQ-13.1 violated")
	}
	if ll.Text != "secret-value" {
		t.Errorf("LogLine.Text = %q, want %q", ll.Text, "secret-value")
	}
}

// Test_DecodeEvent_InputRequested_Fields covers REQ-08.1 and REQ-09 (field fidelity).
func Test_DecodeEvent_InputRequested_Fields(t *testing.T) {
	t.Parallel()

	input := `{"type":"input_requested","seq":5,"prompt":"Password?","sensitive":true}`
	got := protocol.DecodeEvent([]byte(input))

	ir, ok := got.(events.InputRequested)
	if !ok {
		t.Fatalf("got %T, want events.InputRequested", got)
	}
	if ir.Seq != 5 {
		t.Errorf("Seq = %d, want 5", ir.Seq)
	}
	if ir.Prompt != "Password?" {
		t.Errorf("Prompt = %q, want %q", ir.Prompt, "Password?")
	}
	if !ir.Sensitive {
		t.Error("Sensitive = false, want true — REQ-09.2 / REQ-13.1")
	}
	// Reply channel is nil from the codec — it is wired by the adapter's
	// handleInputRequested before forwarding the event to Go consumers.
	// The adapter_test.go covers the end-to-end Reply channel flow (REQ-09).
}

// Test_DecodeEvent_Progress_Fields covers REQ-08.1 (field fidelity — Progress).
func Test_DecodeEvent_Progress_Fields(t *testing.T) {
	t.Parallel()

	input := `{"type":"progress","seq":9,"step":2,"total":5,"label":"writing files"}`
	got := protocol.DecodeEvent([]byte(input))

	p, ok := got.(events.Progress)
	if !ok {
		t.Fatalf("got %T, want events.Progress", got)
	}
	if p.Step != 2 || p.Total != 5 {
		t.Errorf("Step=%d Total=%d, want 2/5", p.Step, p.Total)
	}
	if p.Label != "writing files" {
		t.Errorf("Label = %q, want %q", p.Label, "writing files")
	}
}

// --- helpers ---

// typeName returns a short type identifier for comparison.
func typeName(v interface{}) string {
	switch v.(type) {
	case events.FileCreated:
		return "FileCreated"
	case events.FileModified:
		return "FileModified"
	case events.FileDeleted:
		return "FileDeleted"
	case events.ScriptStarted:
		return "ScriptStarted"
	case events.ScriptStopped:
		return "ScriptStopped"
	case events.LogLine:
		return "LogLine"
	case events.InputRequested:
		return "InputRequested"
	case events.InputProvided:
		return "InputProvided"
	case events.Progress:
		return "Progress"
	case events.Done:
		return "Done"
	case events.Failed:
		return "Failed"
	case events.Cancelled:
		return "Cancelled"
	default:
		return "unknown"
	}
}

func extractError(t *testing.T, err error) *apperrors.Error {
	t.Helper()
	for err != nil {
		if e, ok := err.(*apperrors.Error); ok {
			return e
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	t.Fatalf("error chain does not contain *errors.Error: %v", err)
	return nil
}
