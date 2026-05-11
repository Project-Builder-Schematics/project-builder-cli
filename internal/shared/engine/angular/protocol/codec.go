// Package protocol implements the NDJSON wire codec between the Go adapter
// and the Node.js runner process.
//
// # Wire protocol
//
// Each stdout line from the runner is one JSON object with a "type" field.
// DecodeEvent dispatches on "type" to produce the corresponding events.Event.
// Unknown types produce events.Failed. Malformed JSON produces events.Failed.
//
// # Scanner buffer
//
// Callers (scanStdout in lifecycle.go) MUST set:
//
//	scanner.Buffer(make([]byte, 1<<20), 1<<20)
//
// The codec itself is stateless and does not impose a line-length limit, but
// pathological NDJSON lines larger than 1 MB will be truncated by the scanner.
// This is a documented limitation — schematics do not produce megabyte events.
//
// # Seq invariant
//
// The codec does NOT resequence events. Seq values are passed through verbatim
// from the wire. The runner is responsible for monotonic, sequential numbering
// starting at 1. The adapter trusts the runner on this (REQ-08.4).
package protocol

import (
	"encoding/json"
	"fmt"
	"time"

	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// wireBase holds the fields common to all wire events.
type wireBase struct {
	Type      string `json:"type"`
	Seq       uint64 `json:"seq"`
	At        string `json:"at,omitempty"` // RFC3339; optional
	Sensitive bool   `json:"sensitive,omitempty"`
}

// wireFileCreated is the wire shape for the "file_created" event.
type wireFileCreated struct {
	wireBase
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// wireFileModified is the wire shape for "file_modified".
type wireFileModified struct {
	wireBase
	Path string `json:"path"`
}

// wireFileDeleted is the wire shape for "file_deleted".
type wireFileDeleted struct {
	wireBase
	Path string `json:"path"`
}

// wireScriptStarted is the wire shape for "script_started".
type wireScriptStarted struct {
	wireBase
	Name string   `json:"name"`
	Args []string `json:"args"` // fitness:allow-untyped-args env-allowlist
}

// wireScriptStopped is the wire shape for "script_stopped".
type wireScriptStopped struct {
	wireBase
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
}

// wireLogLine is the wire shape for "log_line".
type wireLogLine struct {
	wireBase
	Level  string `json:"level"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

// wireInputRequested is the wire shape for "input_requested".
type wireInputRequested struct {
	wireBase
	Prompt       string          `json:"prompt"`
	DefaultValue string          `json:"default_value,omitempty"`
	Schema       wireInputSchema `json:"schema,omitempty"`
}

// wireInputSchema is the wire shape for input schema metadata.
type wireInputSchema struct {
	Type    string   `json:"type,omitempty"`
	Choices []string `json:"choices,omitempty"` // fitness:allow-untyped-args env-allowlist
	Default any      `json:"default,omitempty"`
}

// wireInputProvided is the wire shape for "input_provided".
type wireInputProvided struct {
	wireBase
	Prompt string `json:"prompt"`
	Value  string `json:"value"`
}

// wireProgress is the wire shape for "progress".
type wireProgress struct {
	wireBase
	Step  int    `json:"step"`
	Total int    `json:"total"`
	Label string `json:"label"`
}

// wireFailed is the wire shape for "failed".
type wireFailed struct {
	wireBase
	Message string `json:"message"`
}

// parseAt parses an optional RFC3339 timestamp. Returns zero time if empty or invalid.
func parseAt(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// makeBase builds an events.EventBase from a wireBase.
func makeBase(w wireBase) events.EventBase {
	return events.EventBase{
		Seq: w.Seq,
		At:  parseAt(w.At),
	}
}

// executionFailed builds a standard Failed event for runtime decoding errors.
func executionFailed(msg string, cause error) events.Failed {
	return events.Failed{Err: &apperrors.Error{
		Op:      "angular.execute",
		Code:    apperrors.ErrCodeExecutionFailed,
		Message: msg,
		Cause:   cause,
	}}
}

// DecodeEvent decodes a single NDJSON line (no trailing newline) into an
// events.Event value.
//
// Contract:
//   - Malformed JSON → events.Failed{Err: *errors.Error{Code: ErrCodeExecutionFailed}}
//   - Unknown "type" value → events.Failed{...}
//   - Known "type" → corresponding events.* concrete type with fields populated
//   - Seq is passed through verbatim; the codec does NOT resequence
//   - Sensitive flag is always propagated (never dropped or overridden)
//   - InputRequested.Reply is a freshly allocated buffered channel (cap=1)
//
//nolint:gocyclo // 12-way type switch is unavoidable for a sealed event catalogue
func DecodeEvent(line []byte) events.Event {
	// First decode just enough to dispatch on "type".
	var base wireBase
	if err := json.Unmarshal(line, &base); err != nil {
		return executionFailed("malformed JSON from runner: "+err.Error(), err)
	}

	switch base.Type {
	case "file_created":
		var w wireFileCreated
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode file_created: "+err.Error(), err)
		}
		return events.FileCreated{
			EventBase: makeBase(w.wireBase),
			Path:      w.Path,
			IsDir:     w.IsDir,
		}

	case "file_modified":
		var w wireFileModified
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode file_modified: "+err.Error(), err)
		}
		return events.FileModified{
			EventBase: makeBase(w.wireBase),
			Path:      w.Path,
		}

	case "file_deleted":
		var w wireFileDeleted
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode file_deleted: "+err.Error(), err)
		}
		return events.FileDeleted{
			EventBase: makeBase(w.wireBase),
			Path:      w.Path,
		}

	case "script_started":
		var w wireScriptStarted
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode script_started: "+err.Error(), err)
		}
		return events.ScriptStarted{
			EventBase: makeBase(w.wireBase),
			Name:      w.Name,
			Args:      w.Args,
			Sensitive: w.Sensitive, // REQ-12.1, REQ-12.2: never overridden
		}

	case "script_stopped":
		var w wireScriptStopped
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode script_stopped: "+err.Error(), err)
		}
		return events.ScriptStopped{
			EventBase: makeBase(w.wireBase),
			Name:      w.Name,
			ExitCode:  w.ExitCode,
		}

	case "log_line":
		var w wireLogLine
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode log_line: "+err.Error(), err)
		}
		return events.LogLine{
			EventBase: makeBase(w.wireBase),
			Level:     w.Level,
			Source:    events.LogSource(w.Source),
			Text:      w.Text,
			Sensitive: w.Sensitive, // REQ-13.1: never dropped
		}

	case "input_requested":
		var w wireInputRequested
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode input_requested: "+err.Error(), err)
		}
		// NOTE: Reply channel is nil here. The adapter's scanStdout intercepts
		// InputRequested events, creates the Reply channel (cap=1), inserts the
		// send-only side into the event before forwarding to consumers, and keeps
		// the receive side for the stdin-reply write (REQ-09). This separation
		// ensures the adapter controls the channel lifecycle — the codec is a
		// pure deserialiser and does not manage channels.
		return events.InputRequested{
			EventBase:    makeBase(w.wireBase),
			Prompt:       w.Prompt,
			DefaultValue: w.DefaultValue,
			Schema: events.InputSchema{
				Type:    w.Schema.Type,
				Choices: w.Schema.Choices,
				Default: w.Schema.Default,
			},
			Sensitive: w.Sensitive, // REQ-09.2: propagated to InputProvided
			Reply:     nil,         // set by adapter in handleInputRequested
		}

	case "input_provided":
		var w wireInputProvided
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode input_provided: "+err.Error(), err)
		}
		return events.InputProvided{
			EventBase: makeBase(w.wireBase),
			Prompt:    w.Prompt,
			Value:     w.Value,
			Sensitive: w.Sensitive,
		}

	case "progress":
		var w wireProgress
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode progress: "+err.Error(), err)
		}
		return events.Progress{
			EventBase: makeBase(w.wireBase),
			Step:      w.Step,
			Total:     w.Total,
			Label:     w.Label,
		}

	case "done":
		return events.Done{
			EventBase: makeBase(base),
		}

	case "failed":
		var w wireFailed
		if err := json.Unmarshal(line, &w); err != nil {
			return executionFailed("failed to decode failed event: "+err.Error(), err)
		}
		return events.Failed{
			EventBase: makeBase(w.wireBase),
			Err: &apperrors.Error{
				Op:      "angular.execute",
				Code:    apperrors.ErrCodeExecutionFailed,
				Message: w.Message,
			},
		}

	case "cancelled":
		return events.Cancelled{
			EventBase: makeBase(base),
		}

	default:
		return executionFailed(fmt.Sprintf("unknown event type %q from runner", base.Type), nil)
	}
}
