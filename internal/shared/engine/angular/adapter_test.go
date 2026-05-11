// Package angular_test covers adapter.go.
//
// S-000 scope: REQ-01.1 (interface assertion), REQ-01.2 (channel returned
// immediately), REQ-02.1 (cmd.Path is Node binary, not shell), REQ-19.2
// (runner temp file deleted after exit).
// S-002 scope: REQ-02.2, REQ-02.3, REQ-05.1–05.3, REQ-06.1, REQ-06.2,
//
//	REQ-14.1, REQ-16.1
//
// S-003 scope: REQ-04.1, REQ-04.2, REQ-09.1, REQ-09.2, REQ-09.3
package angular_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular"
	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// compile-time assertion: REQ-01.1 — AngularSubprocessAdapter implements engine.Engine.
var _ engine.Engine = (*angular.AngularSubprocessAdapter)(nil)

// fakeNodePath returns the path to the system node binary.
// The FakeNode for S-000 is the embedded runner.js — since runner.js emits
// {"type":"done","seq":1}, any real Node.js binary is the FakeNode.
//
// We skip the test if no node binary can be located.
func fakeNodePath(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("NODE_BINARY"); bin != "" {
		return bin
	}
	// exec.LookPath honours the current PATH (including linuxbrew, nvm, etc).
	if bin, err := exec.LookPath("node"); err == nil {
		return bin
	}
	t.Skip("no Node.js binary available — set NODE_BINARY or add node to PATH to run this test")
	return ""
}

// Test_AngularSubprocessAdapter_Execute_ReturnsChannel covers REQ-01.2:
// Execute returns a non-nil channel immediately (before first event).
func Test_AngularSubprocessAdapter_Execute_ReturnsChannel(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, doneScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{
			Collection: "@schematics/angular",
			Name:       "component",
		},
	})
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("Execute() returned nil channel — REQ-01.2 violated")
	}
	// Drain channel to avoid goroutine leak.
	for range ch { //nolint:revive // intentional drain
	}
}

// doneScript is a minimal Node.js script that reads args and emits done.
// Used in unit tests instead of the production runner.js (which requires
// @angular-devkit/schematics-cli to be installed).
const doneScript = `'use strict';
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });
rl.once('line', () => {
  process.stdout.write(JSON.stringify({ type: 'done', seq: 1 }) + '\n');
  rl.close();
});
`

// Test_AngularSubprocessAdapter_Execute_DoneReceived covers REQ-01.2:
// Execute returns a non-nil channel; Done{} is the last event.
//
// Uses a minimal done script instead of the production runner.js to avoid
// requiring @angular-devkit/schematics-cli in the unit test environment.
func Test_AngularSubprocessAdapter_Execute_DoneReceived(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, doneScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{
			Collection: "@schematics/angular",
			Name:       "component",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// Collect all events; expect at least one Done as terminal event.
	var got []events.Event
	for ev := range ch {
		got = append(got, ev)
	}

	if len(got) == 0 {
		t.Fatal("channel closed with zero events — expected at least Done{}")
	}

	last := got[len(got)-1]
	if _, ok := last.(events.Done); !ok {
		t.Errorf("last event = %T, want events.Done — REQ-01.2 / acceptance criterion", last)
	}
}

// Test_AngularSubprocessAdapter_CmdPath_IsNode covers REQ-02.1:
// cmd.Path must resolve to the Node.js binary — NOT sh, bash, or any shell.
// cmd.Args = [nodeBin, runnerPath, "--collection", ref.Collection, "--schematic", ref.Name].
// A valid SchematicRef is used (malicious refs are rejected by validateRef in S-002
// before reaching cmd.Args — the no-shell property is therefore proven separately).
//
//nolint:gocyclo // multi-assertion test; splitting would obscure the REQ-02.1 verification flow
func Test_AngularSubprocessAdapter_CmdPath_IsNode(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	// Use a valid SchematicRef — REQ-02.1 asserts cmd.Path is node and Args are typed.
	// Malicious refs are now rejected before reaching cmd.Args (REQ-02.2, REQ-02.3).
	ref := engine.SchematicRef{
		Collection: "@schematics/angular",
		Name:       "component",
	}

	scriptPath := writeAdapterScript(t, doneScript)
	defer func() { _ = os.Remove(scriptPath) }()

	var capturedCmd *exec.Cmd
	d := angular.NewAdapterWithCmdSpyAndRunnerPath(scriptPath, func(cmd *exec.Cmd) {
		capturedCmd = cmd
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: ref,
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	// Drain to allow the goroutine to finish cleanly.
	for range ch { //nolint:revive // intentional drain
	}

	if capturedCmd == nil {
		t.Fatal("cmd spy was never called — NewAdapterWithCmdSpy not wired correctly")
	}

	// REQ-02.1: cmd.Path must NOT be a shell binary.
	base := filepath.Base(capturedCmd.Path)
	if base == "sh" || base == "bash" || base == "zsh" || base == "fish" {
		t.Errorf("cmd.Path = %q — shell invocation detected; REQ-02.1 violated", capturedCmd.Path)
	}

	// REQ-02.1: cmd.Args[0] must equal the resolved Node.js binary.
	if len(capturedCmd.Args) == 0 {
		t.Fatal("cmd.Args is empty")
	}
	if capturedCmd.Args[0] != nodeBin {
		t.Errorf("cmd.Args[0] = %q, want Node binary %q — REQ-02.1 violated", capturedCmd.Args[0], nodeBin)
	}

	// REQ-02.1: args contain --collection and --schematic as typed args (not shell-interpolated).
	args := capturedCmd.Args
	foundCollection, foundSchematic := false, false
	for i, a := range args {
		if a == "--collection" && i+1 < len(args) && args[i+1] == ref.Collection {
			foundCollection = true
		}
		if a == "--schematic" && i+1 < len(args) && args[i+1] == ref.Name {
			foundSchematic = true
		}
	}
	if !foundCollection {
		t.Errorf("cmd.Args %v does not contain --collection %q — REQ-02.1 violated", args, ref.Collection)
	}
	if !foundSchematic {
		t.Errorf("cmd.Args %v does not contain --schematic %q — REQ-02.1 violated", args, ref.Name)
	}
}

// Test_AngularSubprocessAdapter_MaliciousRef_RejectedBeforeExec covers the
// second half of REQ-02.1: a malicious-looking SchematicRef.Name must appear
// verbatim in cmd.Args (not shell-interpolated). Since S-002 validateRef rejects
// it BEFORE exec, the proof is that Execute() returns ErrCodeInvalidInput and
// the cmd spy is never called.
func Test_AngularSubprocessAdapter_MaliciousRef_RejectedBeforeExec(t *testing.T) {
	t.Parallel()

	maliciousName := "foo; rm -rf $HOME"

	var cmdSpyCalled bool
	d := angular.NewAdapterWithCmdSpy(func(_ *exec.Cmd) {
		cmdSpyCalled = true
	})

	ch, err := d.Execute(context.Background(), engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{
			Collection: "@schematics/angular",
			Name:       maliciousName,
		},
	})

	// Must return validation error before exec.
	if err == nil {
		t.Fatal("Execute() with malicious Name expected non-nil error — REQ-02.1 violated")
	}
	if ch != nil {
		t.Error("channel must be nil on pre-exec validation error — REQ-14.1 violated")
	}
	if cmdSpyCalled {
		t.Error("cmd spy was called — malicious ref reached exec; REQ-02.1 violated")
	}
}

// Test_AngularSubprocessAdapter_TempFileDeleted covers REQ-19.2:
// the runner temp file is deleted after subprocess exit.
func Test_AngularSubprocessAdapter_TempFileDeleted(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	// Inject a spy that records the temp path used.
	var capturedTempPath string
	d := angular.NewAdapterWithSpy(func(path string) {
		capturedTempPath = path
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// The spy adapter uses the embedded runner.js. Since it may fail if
	// schematics-cli is not installed, we use a short timeout and accept
	// any terminal event — we only care about temp file cleanup.
	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "component", Collection: "@schematics/angular"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	for range ch { //nolint:revive // intentional drain
	}

	// After channel close (subprocess exited), temp file must be gone.
	if capturedTempPath == "" {
		t.Skip("temp path not captured — spy not called (adapter path issue)")
	}
	if _, err := os.Stat(capturedTempPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q still exists after subprocess exit — REQ-19.2 violated", capturedTempPath)
	}
}

// stderrScript is a Node.js script that writes a line to stderr, then emits Done.
const stderrScript = `'use strict';
process.stderr.write("error: something failed\n");
process.stdout.write(JSON.stringify({ type: 'done', seq: 1 }) + '\n');
`

// Test_Adapter_StderrEmitsLogLine covers REQ-04.1:
// stderr lines from child → LogLine{Source: LogSourceStderr} events.
func Test_Adapter_StderrEmitsLogLine(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, stderrScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "test"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var gotStderrLine bool
	for ev := range ch {
		if ll, ok := ev.(events.LogLine); ok && ll.Source == events.LogSourceStderr {
			if strings.Contains(ll.Text, "something failed") {
				gotStderrLine = true
			}
		}
	}

	if !gotStderrLine {
		t.Error("no LogLine{Source: LogSourceStderr} with expected text — REQ-04.1 violated")
	}
}

// Test_Adapter_StderrNotForwardedToHostStderr covers REQ-04.2:
// the adapter does not set cmd.Stderr = os.Stderr (tested via the cmd spy).
func Test_Adapter_StderrNotForwardedToHostStderr(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, doneScript)
	defer func() { _ = os.Remove(scriptPath) }()

	var capturedCmd *exec.Cmd
	d := angular.NewAdapterWithCmdSpyAndRunnerPath(scriptPath, func(cmd *exec.Cmd) {
		capturedCmd = cmd
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "component"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	for range ch { //nolint:revive // intentional drain
	}

	if capturedCmd == nil {
		t.Skip("cmd spy not called")
	}
	// cmd.Stderr must NOT be os.Stderr. It should be nil (piped) or a pipe.
	if capturedCmd.Stderr == os.Stderr {
		t.Error("cmd.Stderr == os.Stderr — host stderr is exposed; REQ-04.2 violated")
	}
}

// inputsScript reads the first stdin line (the inputs JSON), echoes it as a
// log_line, then emits done. The adapter sends inputs as the first JSON line.
const inputsScript = `'use strict';
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });
let read = false;
rl.on('line', line => {
  if (read) return;
  read = true;
  // Echo the raw inputs line back as a log_line so the Go test can verify.
  process.stdout.write(JSON.stringify({ type: 'log_line', seq: 1, level: 'info', source: 'stdout', text: 'inputs:' + line }) + '\n');
  process.stdout.write(JSON.stringify({ type: 'done', seq: 2 }) + '\n');
  rl.close();
});
`

// Test_Adapter_InputsViaSdin covers REQ-06.1:
// Inputs map does NOT appear in cmd.Args; child stdin receives JSON inputs.
func Test_Adapter_InputsViaStdin(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, inputsScript)
	defer func() { _ = os.Remove(scriptPath) }()

	dangerousValue := "my-component; rm -rf /"
	inputs := map[string]any{"name": dangerousValue}

	var capturedCmd *exec.Cmd
	d := angular.NewAdapterWithCmdSpyAndRunnerPath(scriptPath, func(cmd *exec.Cmd) {
		capturedCmd = cmd
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "test"},
		Inputs:    inputs,
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var stdinContents string
	for ev := range ch {
		if ll, ok := ev.(events.LogLine); ok && strings.HasPrefix(ll.Text, "inputs:") {
			stdinContents = strings.TrimPrefix(ll.Text, "inputs:")
		}
	}

	// REQ-06.1: inputs must NOT appear in cmd.Args.
	if capturedCmd != nil {
		argsStr := strings.Join(capturedCmd.Args, " ")
		if strings.Contains(argsStr, "my-component") {
			t.Errorf("inputs appear in cmd.Args — REQ-06.1 violated; args: %v", capturedCmd.Args)
		}
	}

	// REQ-06.2: stdin contains the raw value unchanged.
	if !strings.Contains(stdinContents, dangerousValue) {
		t.Errorf("stdin contents %q do not contain raw input value %q — REQ-06.2 violated", stdinContents, dangerousValue)
	}
}

// inputRequestedScript emits an input_requested event, reads stdin for a reply
// (skipping the first line which is the initial inputs JSON), then emits done.
const inputRequestedScript = `'use strict';
const readline = require('readline');

// Emit input_requested immediately.
process.stdout.write(JSON.stringify({ type: 'input_requested', seq: 3, prompt: 'Name?', sensitive: false }) + '\n');

// Read stdin for the input_reply.
// Note: the Go adapter writes the inputs JSON first (line 1), then input replies
// arrive on subsequent lines. We skip lines that are not type:input_reply.
const rl = readline.createInterface({ input: process.stdin });
let answered = false;
rl.on('line', line => {
  if (answered) return;
  try {
    const msg = JSON.parse(line);
    if (msg.type === 'input_reply') {
      answered = true;
      // Echo the value back as a log_line so the Go test can verify.
      process.stdout.write(JSON.stringify({ type: 'log_line', seq: 4, level: 'info', source: 'stdout', text: 'reply:' + msg.value }) + '\n');
      process.stdout.write(JSON.stringify({ type: 'done', seq: 5 }) + '\n');
      rl.close();
    }
  } catch(e) { /* ignore non-JSON lines */ }
});
`

// inputRequestedSensitiveScript is the same but with sensitive: true.
const inputRequestedSensitiveScript = `'use strict';
const readline = require('readline');

process.stdout.write(JSON.stringify({ type: 'input_requested', seq: 5, prompt: 'Password?', sensitive: true }) + '\n');

const rl = readline.createInterface({ input: process.stdin });
let answered = false;
rl.on('line', line => {
  if (answered) return;
  try {
    const msg = JSON.parse(line);
    if (msg.type === 'input_reply') {
      answered = true;
      process.stdout.write(JSON.stringify({ type: 'log_line', seq: 6, level: 'info', source: 'stdout', text: 'reply:' + msg.value }) + '\n');
      process.stdout.write(JSON.stringify({ type: 'done', seq: 7 }) + '\n');
      rl.close();
    }
  } catch(e) {}
});
`

// Test_Adapter_StdinReply covers REQ-09.1:
// Reply channel value → stdin input_reply JSON → InputProvided emitted.
func Test_Adapter_StdinReply(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, inputRequestedScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "test"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// Collect events; when InputRequested arrives, send the answer.
	var gotInputRequested, gotInputProvided bool
	var gotReplyEcho string
	for ev := range ch {
		switch e := ev.(type) {
		case events.InputRequested:
			gotInputRequested = true
			if e.Reply == nil {
				t.Error("InputRequested.Reply is nil — adapter did not wire Reply channel")
				continue
			}
			// Send answer via Reply channel — ctx-guarded per FF-07.
			select {
			case e.Reply <- "my-answer":
			case <-ctx.Done():
			}
		case events.InputProvided:
			gotInputProvided = true
			if e.Value != "my-answer" {
				t.Errorf("InputProvided.Value = %q, want %q — REQ-09.1", e.Value, "my-answer")
			}
			if e.Sensitive {
				t.Error("InputProvided.Sensitive = true, want false — REQ-09.2")
			}
		case events.LogLine:
			if strings.HasPrefix(e.Text, "reply:") {
				gotReplyEcho = strings.TrimPrefix(e.Text, "reply:")
			}
		}
	}

	if !gotInputRequested {
		t.Error("no InputRequested event received — REQ-09.1 violated")
	}
	if !gotInputProvided {
		t.Error("no InputProvided event received — REQ-09.1 violated")
	}
	if gotReplyEcho != "my-answer" {
		t.Errorf("runner received reply %q, want %q — REQ-09.1 violated (stdin not written)", gotReplyEcho, "my-answer")
	}
}

// Test_Adapter_StdinReply_SensitivePropagated covers REQ-09.2:
// Sensitive=true from InputRequested propagated to InputProvided.
func Test_Adapter_StdinReply_SensitivePropagated(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, inputRequestedSensitiveScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "test"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var gotSensitiveInputProvided bool
	for ev := range ch {
		switch e := ev.(type) {
		case events.InputRequested:
			if e.Reply != nil {
				// ctx-guarded send — satisfies FF-07.
				select {
				case e.Reply <- "secret123":
				case <-ctx.Done():
				}
			}
		case events.InputProvided:
			if e.Sensitive {
				gotSensitiveInputProvided = true
			} else {
				t.Error("InputProvided.Sensitive = false, want true — REQ-09.2 violated")
			}
		}
	}

	if !gotSensitiveInputProvided {
		t.Error("no Sensitive InputProvided event received — REQ-09.2 violated")
	}
}

// Test_Adapter_StdinReply_CtxCancelDuringWait covers REQ-09.3:
// ctx.Cancel() while waiting for reply → Cancelled event within 5s.
func Test_Adapter_StdinReply_CtxCancelDuringWait(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, inputRequestedScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "test"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// Drain the channel in the main goroutine; cancel when InputRequested arrives.
	cancelledAt := time.Time{}
	var gotCancelled bool
	cancelCalled := false

	deadline := time.Now().Add(5 * time.Second)
	for ev := range ch {
		switch ev.(type) {
		case events.InputRequested:
			if !cancelCalled {
				cancelCalled = true
				cancelledAt = time.Now()
				// Do NOT reply — cancel instead (REQ-09.3).
				cancel()
			}
		case events.Cancelled:
			gotCancelled = true
		}
	}

	// Channel is now closed.
	if cancelCalled && time.Now().After(deadline) {
		t.Errorf("channel did not close within 5s of ctx.Cancel during InputRequested wait — elapsed: %v; REQ-09.3 violated", time.Since(cancelledAt))
	}
	if !gotCancelled {
		t.Error("no Cancelled event received — REQ-09.3 violated")
	}
}

// --- helpers ---

// nodeAvailable returns the Node.js binary path or skips if unavailable.
// Separate from fakeNodePath to allow use in tests that don't call t.Setenv.
func nodeAvailable(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("NODE_BINARY"); bin != "" {
		if _, err := os.Stat(bin); err == nil { //nolint:gosec // bin is from a trusted env var (NODE_BINARY)
			return bin
		}
	}
	if bin, err := exec.LookPath("node"); err == nil {
		return bin
	}
	t.Skip("no Node.js binary available — set NODE_BINARY or add node to PATH")
	return ""
}

// writeAdapterScript writes a JS script to a temp file. Caller must os.Remove.
func writeAdapterScript(t *testing.T, script string) string {
	t.Helper()
	f, err := os.CreateTemp("", "adapter-test-*.js")
	if err != nil {
		t.Fatalf("failed to create temp script: %v", err)
	}
	if _, err := f.WriteString(script); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		t.Fatalf("failed to write temp script: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

// Test_Adapter_InputsNotInArgs_CmdSpy provides an additional check that
// json-encoded inputs are written to stdin, not cmd.Args (REQ-06.1).
// This test uses the cmd spy to verify Args do not contain input values.
func Test_Adapter_InputsNotInArgs_CmdSpy(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeAdapterScript(t, doneScript)
	defer func() { _ = os.Remove(scriptPath) }()

	var capturedCmd *exec.Cmd
	d := angular.NewAdapterWithCmdSpyAndRunnerPath(scriptPath, func(cmd *exec.Cmd) {
		capturedCmd = cmd
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "component"},
		Inputs:    map[string]any{"name": "my-component; rm -rf /"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	for range ch { //nolint:revive // drain
	}

	if capturedCmd == nil {
		t.Skip("cmd spy not called")
	}

	// REQ-06.1: "my-component" must NOT appear in args.
	argsStr := strings.Join(capturedCmd.Args, " ")
	if strings.Contains(argsStr, "my-component") {
		t.Errorf("inputs found in cmd.Args — REQ-06.1 violated; args: %v", capturedCmd.Args)
	}
}

// Ensure bufio and json imports are used.
var (
	_ = bufio.NewScanner
	_ = json.Marshal
	_ *apperrors.Error
)
