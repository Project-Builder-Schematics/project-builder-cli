package angular

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// eventChannelCap is the buffer capacity for the event channel returned by Execute.
// A buffer of 32 prevents the goroutine from blocking on slow consumers.
const eventChannelCap = 32

// startProcess constructs the exec.Cmd (NO shell invocation), starts the
// process, and launches goroutine fan-out for stdout/stderr plus the
// platform-specific kill goroutine.
//
// SECURITY: exec.CommandContext with explicit args only. No shell binary.
// cmd.Path is always nodeBin — REQ-02.1.
//
// Temp file lifecycle: runnerTempPath is cleaned up by runFanOut after
// cmd.Wait() returns — REQ-19.2.
//
// cmdSpy (test-only, may be nil): called with the constructed cmd immediately
// before cmd.Start(). Enables assertions on cmd.Path and cmd.Args (REQ-02.1).
func startProcess(
	ctx context.Context,
	nodeBin string,
	runnerTempPath string,
	allowlist []string, // fitness:allow-untyped-args env-allowlist
	cmdSpy func(*exec.Cmd), // nil in production
) (<-chan events.Event, *exec.Cmd, error) {
	ch := make(chan events.Event, eventChannelCap)

	//nolint:gosec // G204: nodeBin is from Discoverer (NODE_BINARY / LookPath), not user input.
	cmd := exec.CommandContext(ctx, nodeBin, runnerTempPath)
	cmd.Env = buildEnv(allowlist) // fitness:allow-untyped-args env-allowlist

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, &errors.Error{
			Op:      "angular.start_process",
			Code:    errors.ErrCodeExecutionFailed,
			Message: "failed to create stdout pipe",
			Cause:   err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, &errors.Error{
			Op:      "angular.start_process",
			Code:    errors.ErrCodeExecutionFailed,
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
		return nil, nil, &errors.Error{
			Op:      "angular.start_process",
			Code:    errors.ErrCodeExecutionFailed,
			Message: "failed to start Node.js process",
			Cause:   err,
		}
	}

	// killDone is closed by killProcess after the kill sequence completes.
	// runFanOut waits on it before emitting Cancelled and closing ch.
	killDone := make(chan struct{})

	// pipesDrained is closed by runFanOut after stdout+stderr have drained.
	// killProcess selects on it to exit early (process died naturally).
	pipesDrained := make(chan struct{})

	go runFanOut(ctx, cmd, stdout, stderr, runnerTempPath, ch, killDone, pipesDrained)
	go killProcess(ctx, cmd, killDone, pipesDrained)

	return ch, cmd, nil
}

// runFanOut drives the stdout/stderr goroutines, calls cmd.Wait() (REQ-03.2),
// and emits the terminal event before closing ch.
//
// Terminal event decision:
//   - Normal exit with Done from stdout → Done already sent by stdout scanner.
//   - Abnormal exit (wait error) + no cancellation → emit Failed.
//   - Context cancelled → wait for killProcess (killDone), then emit Cancelled.
func runFanOut(
	ctx context.Context,
	cmd *exec.Cmd,
	stdout io.Reader,
	stderr io.Reader,
	runnerTempPath string,
	ch chan<- events.Event,
	killDone <-chan struct{},
	pipesDrained chan<- struct{},
) {
	defer close(ch)
	defer os.Remove(runnerTempPath) //nolint:errcheck // REQ-19.2: best-effort cleanup

	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})

	// Goroutine: scan stdout, decode NDJSON events.
	go func() {
		defer close(stdoutDone)
		scanStdout(ctx, stdout, ch)
	}()

	// Goroutine: scan stderr, emit LogLine{Source: LogSourceStderr} events.
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
		// Cancellation path: wait for killProcess to complete its sequence.
		<-killDone
		// Emit Cancelled terminal event (REQ-03.1).
		select {
		case ch <- events.Cancelled{}:
		default:
		}
		return
	}

	// Normal or abnormal exit (no cancellation).
	if waitErr != nil {
		// Abnormal exit — emit Failed terminal event (REQ-15.1).
		select {
		case ch <- events.Failed{Err: &errors.Error{
			Op:      "angular.execute",
			Code:    errors.ErrCodeExecutionFailed,
			Message: fmt.Sprintf("Node.js process exited unexpectedly: %v", waitErr),
			Cause:   waitErr,
		}}:
		default:
			// Stdout scanner already sent a Failed terminal event.
		}
	}
	// Clean exit: stdout scanner emitted Done{} already; nothing more to emit.
}

// scanStdout reads lines from the child's stdout, decodes each NDJSON line,
// and sends to ch.
//
// S-000 skeleton: recognises "done" only; everything else → Failed.
// S-003 replaces this with the full DecodeEvent codec (12 types).
func scanStdout(_ context.Context, r io.Reader, ch chan<- events.Event) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		ch <- decodeWireEvent(line)
	}
}

// wireMsg is the minimal struct for S-000 NDJSON decoding.
type wireMsg struct {
	Type string `json:"type"`
	Seq  uint64 `json:"seq"`
}

// decodeWireEvent decodes a single NDJSON line into an Event.
//
// S-000 skeleton: recognises "done" only; everything else → Failed.
// S-003 replaces this with the full 12-type codec (DecodeEvent).
func decodeWireEvent(line []byte) events.Event {
	var msg wireMsg
	if err := json.Unmarshal(line, &msg); err != nil {
		return events.Failed{Err: &errors.Error{
			Op:      "angular.execute",
			Code:    errors.ErrCodeExecutionFailed,
			Message: "malformed JSON from runner",
			Cause:   err,
		}}
	}
	switch msg.Type {
	case "done":
		return events.Done{EventBase: events.EventBase{Seq: msg.Seq}}
	default:
		return events.Failed{Err: &errors.Error{
			Op:      "angular.execute",
			Code:    errors.ErrCodeExecutionFailed,
			Message: fmt.Sprintf("unknown event type %q from runner", msg.Type),
		}}
	}
}

// scanStderr reads lines from the child's stderr and emits LogLine events
// with Source=LogSourceStderr (REQ-04.1, REQ-04.2).
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
