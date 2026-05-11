package angular

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular/protocol"
	angularErrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// eventChannelCap is the buffer capacity for the event channel returned by Execute.
// A buffer of 32 prevents the goroutine from blocking on slow consumers.
const eventChannelCap = 32

// inputReplyMsg is the JSON wire format for stdin replies to InputRequested.
//
// REQ-09.1: {"type":"input_reply","prompt_seq":<uint64>,"value":"<string>"}\n
type inputReplyMsg struct {
	Type      string `json:"type"`
	PromptSeq uint64 `json:"prompt_seq"`
	Value     string `json:"value"`
}

// startProcess constructs the exec.Cmd (NO shell invocation), starts the
// process, and launches goroutine fan-out for stdout/stderr plus the
// platform-specific kill goroutine.
//
// SECURITY: exec.CommandContext with explicit args only. No shell binary.
// cmd.Path is always nodeBin — REQ-02.1.
// cmd.Args = [nodeBin, runnerTempPath, "--collection", ref.Collection, "--schematic", ref.Name]
// Inputs are written to child stdin as JSON — NOT in cmd.Args (REQ-06).
//
// Temp file lifecycle: if isEmbedded is true, runnerTempPath is cleaned up
// by runFanOut after cmd.Wait() returns — REQ-19.2. When isEmbedded is false
// (test override), no cleanup is performed by the adapter.
//
// cmdSpy (test-only, may be nil): called with the constructed cmd immediately
// before cmd.Start(). Enables assertions on cmd.Path and cmd.Args (REQ-02.1).
func startProcess(
	ctx context.Context,
	nodeBin string,
	runnerTempPath string,
	req engine.ExecuteRequest,
	cmdSpy func(*exec.Cmd), // nil in production
	isEmbedded bool, // true = adapter owns cleanup; false = caller owns the file
) (<-chan events.Event, *exec.Cmd, error) {
	ch := make(chan events.Event, eventChannelCap)

	// cmd.Args: [nodeBin, runnerTempPath, "--collection", ref.Collection, "--schematic", ref.Name]
	// SECURITY: No shell binary. All args are typed, not interpolated (REQ-02.1).
	//nolint:gosec // G204: nodeBin is from Discoverer (NODE_BINARY / LookPath), not user input.
	cmd := exec.CommandContext(
		ctx, nodeBin,
		runnerTempPath,
		"--collection", req.Schematic.Collection,
		"--schematic", req.Schematic.Name,
	)
	cmd.Env = buildEnv(req.EnvAllowlist) // fitness:allow-untyped-args env-allowlist

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, &angularErrors.Error{
			Op:      "angular.start_process",
			Code:    angularErrors.ErrCodeExecutionFailed,
			Message: "failed to create stdin pipe",
			Cause:   err,
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, &angularErrors.Error{
			Op:      "angular.start_process",
			Code:    angularErrors.ErrCodeExecutionFailed,
			Message: "failed to create stdout pipe",
			Cause:   err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, &angularErrors.Error{
			Op:      "angular.start_process",
			Code:    angularErrors.ErrCodeExecutionFailed,
			Message: "failed to create stderr pipe",
			Cause:   err,
		}
	}

	// Invoke cmd spy (test-only) after pipes are wired, before Start.
	// This allows test assertions on cmd.Path / cmd.Args (REQ-02.1).
	if cmdSpy != nil {
		cmdSpy(cmd)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, &angularErrors.Error{
			Op:      "angular.start_process",
			Code:    angularErrors.ErrCodeExecutionFailed,
			Message: "failed to start Node.js process",
			Cause:   err,
		}
	}

	// Write Inputs to child stdin as JSON immediately after start (REQ-06).
	// Inputs MUST NOT appear in cmd.Args.
	// We write synchronously — the pipe buffer is large enough for the inputs
	// JSON. We do NOT close stdin here; it remains open for input_reply writes.
	writeInputsJSON(stdin, req.Inputs)

	// killDone is closed by killProcess after the kill sequence completes.
	// runFanOut waits on it before emitting Cancelled and closing ch.
	killDone := make(chan struct{})

	// pipesDrained is closed by runFanOut after stdout+stderr have drained.
	// killProcess selects on it to exit early (process died naturally).
	pipesDrained := make(chan struct{})

	go runFanOut(ctx, cmd, stdin, stdout, stderr, runnerTempPath, isEmbedded, ch, killDone, pipesDrained)
	go killProcess(ctx, cmd, killDone, pipesDrained)

	return ch, cmd, nil
}

// writeInputsJSON serialises inputs as a single JSON line to the child's stdin
// (REQ-06). stdin is NOT closed after this write — it stays open for
// subsequent input_reply messages (REQ-09.1).
//
// SECURITY: inputs contain raw values from the caller (e.g. user-entered text).
// They are sent as JSON data, not as CLI arguments — the runner treats them as
// data, never as code (REQ-06.2).
func writeInputsJSON(w io.Writer, inputs map[string]any) {
	if len(inputs) == 0 {
		// Write an empty JSON object so the runner doesn't block on its first read.
		_, _ = fmt.Fprintln(w, "{}")
		return
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // REQ-06.2: raw values unchanged — no HTML escaping
	_ = enc.Encode(inputs)   //nolint:errcheck // best-effort; runner handles missing inputs
}

// runFanOut drives the stdout/stderr goroutines, calls cmd.Wait() (REQ-03.2),
// and emits exactly one terminal event before closing ch (REQ-15.2).
//
// Terminal event decision (guarded by terminalSent atomic):
//   - Normal exit with Done from stdout → Done emitted by stdout scanner.
//   - Malformed JSON or unknown type → Failed emitted by stdout scanner.
//   - Abnormal exit (wait error) + no cancellation → Failed emitted here
//     (only if stdout scanner did not already send a terminal).
//   - Context cancelled → wait for killProcess (killDone), then emit Cancelled
//     (only if no terminal already sent by scanner).
//
// isEmbedded controls temp file cleanup: when true, the file is removed after
// cmd.Wait() — REQ-19.2. When false (test override), cleanup is skipped.
func runFanOut(
	ctx context.Context,
	cmd *exec.Cmd,
	stdin io.WriteCloser, // both write (for input reply) and close
	stdout io.Reader,
	stderr io.Reader,
	runnerTempPath string,
	isEmbedded bool,
	ch chan<- events.Event,
	killDone <-chan struct{},
	pipesDrained chan<- struct{},
) {
	defer close(ch)
	if isEmbedded {
		defer os.Remove(runnerTempPath) //nolint:errcheck // REQ-19.2: best-effort cleanup
	}

	// terminalSent is incremented (via CAS) by the first goroutine that emits
	// a terminal event (Done, Failed, or Cancelled). This ensures exactly one
	// terminal event per execution (REQ-15.2), preventing the double-event race
	// where scanStdout emits Done and runFanOut also emits Failed for non-zero
	// exit code.
	var terminalSent atomic.Int32

	// sendTerminal sends ev on ch only if no terminal has been sent yet.
	// Returns true if ev was sent.
	sendTerminal := func(ev events.Event) bool {
		if terminalSent.CompareAndSwap(0, 1) {
			select {
			case ch <- ev:
			default:
			}
			return true
		}
		return false
	}

	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})

	// Goroutine: scan stdout, decode NDJSON events, handle InputRequested reply loop.
	// stdin is closed after scanStdout returns — signals to the child that no
	// more input_reply writes are coming (Node's readline 'close' event).
	go func() {
		defer close(stdoutDone)
		defer stdin.Close() //nolint:errcheck // best-effort; child may already have exited
		scanStdout(ctx, stdout, stdin, ch, sendTerminal)
	}()

	// Goroutine: scan stderr, emit LogLine{Source: LogSourceStderr} events.
	// SECURITY: stderr is captured and mapped to LogLine; it is NEVER written
	// to the host process's os.Stderr (REQ-04.2).
	go func() {
		defer close(stderrDone)
		scanStderr(stderr, ch)
	}()

	// Wait for both pipe scanners to drain.
	<-stdoutDone
	<-stderrDone

	// Signal killProcess that the pipes have drained (process has exited or
	// its stdout/stderr were closed). This allows killProcess to exit early
	// rather than waiting the full sigTermGracePeriod.
	close(pipesDrained)

	// cmd.Wait() cleans up OS process resources. ALWAYS called (REQ-03.2).
	waitErr := cmd.Wait()

	if ctx.Err() != nil {
		// Cancellation path: wait for killProcess to complete its kill sequence.
		<-killDone
		// Emit Cancelled terminal event — only if scanner didn't already emit one.
		sendTerminal(events.Cancelled{})
		return
	}

	// Normal or abnormal exit (no cancellation).
	if waitErr != nil {
		// Abnormal exit — emit Failed terminal event if scanner did not already.
		sendTerminal(events.Failed{Err: &angularErrors.Error{
			Op:      "angular.execute",
			Code:    angularErrors.ErrCodeExecutionFailed,
			Message: fmt.Sprintf("Node.js process exited unexpectedly: %v", waitErr),
			Cause:   waitErr,
		}})
	}
	// Clean exit: stdout scanner already emitted Done{}; nothing more to emit.
}

// scanStdout reads lines from the child's stdout, decodes each NDJSON line,
// and sends to ch. Terminal events (Done, Failed, Cancelled) are sent via
// sendTerminal to enforce the exactly-one-terminal-event invariant (REQ-15.2).
//
// InputRequested handling (REQ-09):
//  1. Emit the InputRequested event (with its Reply channel) to the Go consumer.
//  2. Select on Reply or ctx.Done().
//  3. Write {"type":"input_reply","prompt_seq":<seq>,"value":<value>} to stdin.
//  4. Emit InputProvided{Sensitive: req.Sensitive} — Sensitive propagated (REQ-09.2).
//
// Uses the full 12-type codec (protocol.DecodeEvent) from S-003.
// Scanner buffer is 1 MB — matches codec's documented limit.
func scanStdout(
	ctx context.Context,
	r io.Reader,
	stdin io.Writer,
	ch chan<- events.Event,
	sendTerminal func(events.Event) bool,
) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB — matches codec limit (REQ-08 GoDoc)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		ev := protocol.DecodeEvent(line)
		switch e := ev.(type) {
		case events.Done, events.Failed, events.Cancelled:
			// Terminal events go through the sentinel to prevent duplicates.
			sendTerminal(ev)
			// Stop scanning after the first terminal event.
			return

		case events.InputRequested:
			// Handle the stdin-reply loop (REQ-09).
			handleInputRequested(ctx, e, stdin, ch)
			// Do NOT return — continue scanning for more events.

		default:
			ch <- ev
		}
	}
}

// handleInputRequested implements the stdin-reply protocol (REQ-09):
//
//  1. Create a buffered reply channel (cap=1); insert send-only view into the event.
//  2. Emit the InputRequested event (with its Reply channel) to the Go consumer.
//  3. Select on Reply channel (consumer answer) or ctx.Done() (cancellation).
//  4. Write the input_reply JSON to child stdin (REQ-09.1).
//  5. Emit InputProvided with Sensitive propagated from the original InputRequested (REQ-09.2).
//
// FF-07 compliance: the select statement below uses both Reply and ctx.Done().
func handleInputRequested(
	ctx context.Context,
	req events.InputRequested,
	stdin io.Writer,
	ch chan<- events.Event,
) {
	// Create the Reply channel (cap=1 buffer). The adapter is the producer
	// of this channel — the consumer (Go caller) sends their answer via the
	// send-only view; we receive via the bidirectional channel.
	replyCh := make(chan string, 1)
	req.Reply = replyCh // send-only view assigned to event field

	// Emit the InputRequested so the consumer can provide a value via Reply.
	ch <- req

	// FF-07: ctx-guarded select — BOTH branches mandatory.
	var value string
	select { // FF-07: ctx.Done() guard present
	case v := <-replyCh: // receive from bidirectional channel (adapter side)
		value = v
	case <-ctx.Done():
		// Context cancelled while waiting for reply — caller's goroutine will
		// emit Cancelled via the terminal path; we return without writing stdin.
		return
	}

	// Write the reply to child stdin (REQ-09.1).
	reply := inputReplyMsg{
		Type:      "input_reply",
		PromptSeq: req.Seq,
		Value:     value,
	}
	replyBytes, err := json.Marshal(reply)
	if err == nil {
		// Append newline (NDJSON framing).
		replyBytes = append(replyBytes, '\n')
		_, _ = stdin.Write(replyBytes) //nolint:errcheck // best-effort; runner handles missing reply
	}

	// Emit InputProvided with Sensitive propagated from InputRequested (REQ-09.2).
	ch <- events.InputProvided{
		EventBase: events.EventBase{Seq: req.Seq + 1},
		Prompt:    req.Prompt,
		Value:     value,
		Sensitive: req.Sensitive, // REQ-09.2: never overridden
	}
}

// scanStderr reads lines from the child's stderr and emits LogLine events
// with Source=LogSourceStderr (REQ-04.1, REQ-04.2).
//
// SECURITY: stderr lines MUST NOT be forwarded to host os.Stderr (REQ-04.2).
// They are captured and emitted as structured LogLine events only.
func scanStderr(r io.Reader, ch chan<- events.Event) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		ch <- events.LogLine{
			Source: events.LogSourceStderr,
			Text:   text,
		}
	}
}
