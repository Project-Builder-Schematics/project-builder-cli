// Package json defines the stable NDJSON envelope format for the json.Renderer.
//
// The "type" field values in eventEnvelope are a public contract — changing them
// is a breaking change for AI agent and CI consumers. Any modification requires
// a re-spec of the json-renderer capability.
//
// # Stable type discriminators
//
// The toEnvelope function maps each of the 12 sealed event types to a
// snake_case string discriminator:
//
//	FileCreated    → "file_created"
//	FileModified   → "file_modified"
//	FileDeleted    → "file_deleted"
//	ScriptStarted  → "script_started"
//	ScriptStopped  → "script_stopped"
//	LogLine        → "log_line"
//	InputRequested → "input_requested"
//	InputProvided  → "input_provided"
//	Progress       → "progress"
//	Done           → "done"
//	Failed         → "failed"
//	Cancelled      → "cancelled"
//
// These values are hardcoded (not derived via reflection) per ADR-04 so that
// Go refactors cannot silently break downstream consumers.
package json

import (
	"fmt"
	"time"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// eventEnvelope is the top-level NDJSON object emitted for every event.
// All fields are always present; "data" may be omitted for terminal events
// that carry no additional payload.
type eventEnvelope struct {
	// Type identifies the event kind using a stable snake_case string.
	// CONTRACT: stable; breaking change (rename/remove) requires re-spec.
	// Values: file_created, file_modified, file_deleted, script_started,
	// script_stopped, log_line, input_requested, input_provided,
	// progress, done, failed, cancelled.
	Type string `json:"type"`

	// Seq is the monotonically increasing sequence number from the event.
	Seq uint64 `json:"seq"`

	// At is the event timestamp in RFC3339Nano format.
	At string `json:"at"`

	// Data carries event-specific fields. Omitted when empty (terminal events).
	Data any `json:"data,omitempty"`
}

// fileCreatedData is the "data" payload for file_created events.
type fileCreatedData struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir,omitempty"`
}

// fileModifiedData is the "data" payload for file_modified events.
type fileModifiedData struct {
	Path string `json:"path"`
}

// fileDeletedData is the "data" payload for file_deleted events.
type fileDeletedData struct {
	Path string `json:"path"`
}

// scriptStartedData is the "data" payload for script_started events.
// When the event is sensitive, Args is replaced with "[REDACTED]".
type scriptStartedData struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

// scriptStoppedData is the "data" payload for script_stopped events.
type scriptStoppedData struct {
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
}

// logLineData is the "data" payload for log_line events.
// When the event is sensitive, Text is replaced with "[REDACTED]".
type logLineData struct {
	Level     string `json:"level"`
	Source    string `json:"source,omitempty"`
	Text      string `json:"text"`
	Sensitive bool   `json:"sensitive"`
}

// inputRequestedData is the "data" payload for input_requested events.
// When the event is sensitive, DefaultValue is replaced with "[REDACTED]".
type inputRequestedData struct {
	Prompt       string `json:"prompt"`
	DefaultValue string `json:"default_value,omitempty"`
	Sensitive    bool   `json:"sensitive"`
}

// inputProvidedData is the "data" payload for input_provided events.
// When the event is sensitive, Value is replaced with "[REDACTED]".
type inputProvidedData struct {
	Prompt    string `json:"prompt,omitempty"`
	Value     string `json:"value"`
	Sensitive bool   `json:"sensitive"`
}

// progressData is the "data" payload for progress events.
type progressData struct {
	Step  int    `json:"step"`
	Total int    `json:"total"`
	Label string `json:"label,omitempty"`
}

// failedData is the "data" payload for failed events.
type failedData struct {
	Error string `json:"error,omitempty"`
}

// toEnvelope converts an Event into an eventEnvelope with the seq counter.
// Sensitive fields are masked via the package-local mask/maskSlice helpers.
// Returns an error only for unexpected (non-sealed) event types — in practice
// this cannot happen with the sealed event interface, but is defensive.
func toEnvelope(ev events.Event, seq uint64) (eventEnvelope, error) {
	at := formatTime(eventAt(ev))

	switch e := ev.(type) {
	case events.FileCreated:
		return eventEnvelope{
			Type: "file_created",
			Seq:  seq,
			At:   at,
			Data: fileCreatedData{Path: e.Path, IsDir: e.IsDir},
		}, nil

	case events.FileModified:
		return eventEnvelope{
			Type: "file_modified",
			Seq:  seq,
			At:   at,
			Data: fileModifiedData{Path: e.Path},
		}, nil

	case events.FileDeleted:
		return eventEnvelope{
			Type: "file_deleted",
			Seq:  seq,
			At:   at,
			Data: fileDeletedData{Path: e.Path},
		}, nil

	case events.ScriptStarted:
		return eventEnvelope{
			Type: "script_started",
			Seq:  seq,
			At:   at,
			Data: scriptStartedData{
				Name: e.Name,
				Args: maskSlice(e.Args, e.Sensitive),
			},
		}, nil

	case events.ScriptStopped:
		return eventEnvelope{
			Type: "script_stopped",
			Seq:  seq,
			At:   at,
			Data: scriptStoppedData{Name: e.Name, ExitCode: e.ExitCode},
		}, nil

	case events.LogLine:
		return eventEnvelope{
			Type: "log_line",
			Seq:  seq,
			At:   at,
			Data: logLineData{
				Level:     e.Level,
				Source:    string(e.Source),
				Text:      mask(e.Text, e.Sensitive),
				Sensitive: e.Sensitive,
			},
		}, nil

	case events.InputRequested:
		return eventEnvelope{
			Type: "input_requested",
			Seq:  seq,
			At:   at,
			Data: inputRequestedData{
				Prompt:       e.Prompt,
				DefaultValue: mask(e.DefaultValue, e.Sensitive),
				Sensitive:    e.Sensitive,
			},
		}, nil

	case events.InputProvided:
		return eventEnvelope{
			Type: "input_provided",
			Seq:  seq,
			At:   at,
			Data: inputProvidedData{
				Prompt:    e.Prompt,
				Value:     mask(e.Value, e.Sensitive),
				Sensitive: e.Sensitive,
			},
		}, nil

	case events.Progress:
		return eventEnvelope{
			Type: "progress",
			Seq:  seq,
			At:   at,
			Data: progressData{Step: e.Step, Total: e.Total, Label: e.Label},
		}, nil

	case events.Done:
		return eventEnvelope{Type: "done", Seq: seq, At: at}, nil

	case events.Failed:
		var errMsg string
		if e.Err != nil {
			errMsg = e.Err.Error()
		}
		return eventEnvelope{
			Type: "failed",
			Seq:  seq,
			At:   at,
			Data: failedData{Error: errMsg},
		}, nil

	case events.Cancelled:
		return eventEnvelope{Type: "cancelled", Seq: seq, At: at}, nil

	default:
		return eventEnvelope{}, fmt.Errorf("toEnvelope: unknown event type %T", ev)
	}
}

// eventAt extracts the At timestamp from an event's embedded EventBase.
// Since EventBase is embedded by value (not by pointer), we use a type-switch
// to reach the concrete type's At field. All 12 event types embed EventBase.
func eventAt(ev events.Event) time.Time {
	switch e := ev.(type) {
	case events.FileCreated:
		return e.At
	case events.FileModified:
		return e.At
	case events.FileDeleted:
		return e.At
	case events.ScriptStarted:
		return e.At
	case events.ScriptStopped:
		return e.At
	case events.LogLine:
		return e.At
	case events.InputRequested:
		return e.At
	case events.InputProvided:
		return e.At
	case events.Progress:
		return e.At
	case events.Done:
		return e.At
	case events.Failed:
		return e.At
	case events.Cancelled:
		return e.At
	default:
		return time.Time{}
	}
}

// formatTime formats t as RFC3339Nano, using UTC to ensure consistent output
// regardless of the system timezone.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
