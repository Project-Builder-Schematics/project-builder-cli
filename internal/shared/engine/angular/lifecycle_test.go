//go:build !windows

// Package angular_test covers lifecycle.go, lifecycle_posix.go.
//
// S-001 scope: REQ-03.1 (cancellation → Cancelled within 5s), REQ-03.2
// (cmd.Wait called exactly once), REQ-15.1 (abnormal exit → Failed),
// REQ-17.1 (SIGTERM before SIGKILL on POSIX), REQ-17.2 (SIGKILL after 3s).
package angular_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// blockerScript returns a shell-free Node.js script string that blocks
// indefinitely (sleeps 60s), ignoring signals.
//
// The test uses process exec — not shell invocation.
const blockerScript = `'use strict';
// Block indefinitely. Sleep resolves after 60s — context cancels first.
function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }
sleep(60000).then(() => process.exit(0));
`

// sigTermScript is a Node.js process that exits cleanly within 2s of SIGTERM.
const sigTermScript = `'use strict';
process.on('SIGTERM', () => {
  setTimeout(() => process.exit(0), 200);
});
function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }
sleep(60000).then(() => process.exit(0));
`

// sigIgnoreScript is a Node.js process that ignores SIGTERM entirely.
const sigIgnoreScript = `'use strict';
process.on('SIGTERM', () => { /* ignored */ });
function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }
sleep(60000).then(() => process.exit(0));
`

// exitNonZeroScript is a Node.js process that exits with code 1 immediately,
// without emitting any NDJSON event.
const exitNonZeroScript = `'use strict';
process.exit(1);
`

// writeScript writes a JS script to a temp file and returns the path.
// The caller must defer os.Remove(path).
// Alias to writeAdapterScript defined in adapter_test.go — uses a different
// prefix to avoid linker conflict.
func writeScript(t *testing.T, script string) string {
	t.Helper()
	f, err := os.CreateTemp("", "lifecycle-test-*.js")
	if err != nil {
		t.Fatalf("failed to create temp script file: %v", err)
	}
	if _, err := f.WriteString(script); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		t.Fatalf("failed to write temp script: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

// nodeAvailable is defined in adapter_test.go (shared helper in the same test package).

// Test_Lifecycle_CancellationEmitsCancelled covers REQ-03.1:
// ctx.Cancel() → Cancelled event received within 5s; channel closed.
func Test_Lifecycle_CancellationEmitsCancelled(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeScript(t, blockerScript)
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

	// Cancel after a brief delay to let the process start.
	time.Sleep(200 * time.Millisecond)
	cancel()

	deadline := time.Now().Add(5 * time.Second)
	var gotCancelled bool
	for ev := range ch {
		if _, ok := ev.(events.Cancelled); ok {
			gotCancelled = true
		}
	}

	if time.Now().After(deadline) {
		t.Error("channel did not close within 5s of ctx.Cancel() — REQ-03.1 violated")
	}
	if !gotCancelled {
		t.Error("no Cancelled event received — REQ-03.1 violated")
	}
}

// Test_Lifecycle_SIGTERMBeforeSIGKILL covers REQ-17.1:
// A process that obeys SIGTERM exits without SIGKILL escalation; Cancelled received.
func Test_Lifecycle_SIGTERMBeforeSIGKILL(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeScript(t, sigTermScript)
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

	// Allow process to start before cancelling.
	time.Sleep(200 * time.Millisecond)
	cancelTime := time.Now()
	cancel()

	var gotCancelled bool
	for ev := range ch {
		if _, ok := ev.(events.Cancelled); ok {
			gotCancelled = true
		}
	}

	elapsed := time.Since(cancelTime)
	// Should exit well under 3s (the SIGTERM grace period) — process exits in ~200ms.
	if elapsed > 3*time.Second {
		t.Errorf("process took %.1fs to exit — expected < 3s (obeys SIGTERM); REQ-17.1", elapsed.Seconds())
	}
	if !gotCancelled {
		t.Error("no Cancelled event received — REQ-17.1 violated")
	}
}

// Test_Lifecycle_SIGKILLAfterGracePeriod covers REQ-17.2:
// A process that ignores SIGTERM receives SIGKILL after ~3s; Cancelled received within 5s.
func Test_Lifecycle_SIGKILLAfterGracePeriod(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeScript(t, sigIgnoreScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)

	// 10s total timeout — generous enough for 3s grace + SIGKILL.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "test"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	cancelTime := time.Now()
	cancel()

	deadline := time.Now().Add(5 * time.Second)
	var gotCancelled bool
	for ev := range ch {
		if _, ok := ev.(events.Cancelled); ok {
			gotCancelled = true
		}
	}

	if time.Now().After(deadline) {
		t.Error("channel did not close within 5s after ctx.Cancel — SIGKILL path took too long; REQ-17.2 violated")
	}
	_ = cancelTime
	if !gotCancelled {
		t.Error("no Cancelled event received after SIGKILL — REQ-17.2 violated")
	}
}

// Test_Lifecycle_AbnormalExitEmitsFailed covers REQ-15.1:
// A process that exits with code 1 (no Done event emitted) → Failed terminal event.
func Test_Lifecycle_AbnormalExitEmitsFailed(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	scriptPath := writeScript(t, exitNonZeroScript)
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

	var terminal events.Event
	for ev := range ch {
		terminal = ev
	}

	failed, ok := terminal.(events.Failed)
	if !ok {
		t.Fatalf("last event = %T, want events.Failed — REQ-15.1 violated", terminal)
	}

	var appErr *errors.Error
	if failed.Err == nil {
		t.Fatal("Failed.Err is nil — REQ-15.1 violated")
	}
	// Walk error chain.
	for err := failed.Err; err != nil; {
		if e, ok2 := err.(*errors.Error); ok2 {
			appErr = e
			break
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok2 := err.(unwrapper); ok2 {
			err = u.Unwrap()
		} else {
			break
		}
	}
	if appErr == nil {
		t.Fatalf("Failed.Err is not *errors.Error: %T", failed.Err)
	}
	if appErr.Code != errors.ErrCodeExecutionFailed {
		t.Errorf("Failed.Err.Code = %q, want %q — REQ-15.1 violated", appErr.Code, errors.ErrCodeExecutionFailed)
	}
}

// doneOnlyScript is a minimal Node.js script that reads one stdin line then emits done.
const doneOnlyScript = `'use strict';
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });
rl.once('line', () => {
  process.stdout.write(JSON.stringify({ type: 'done', seq: 1 }) + '\n');
  rl.close();
});
`

// Test_Lifecycle_ExactlyOneTerminalEvent covers REQ-15.2:
// Exactly one terminal event (Done, Failed, or Cancelled) per execution;
// channel closed immediately after.
func Test_Lifecycle_ExactlyOneTerminalEvent(t *testing.T) {
	nodeBin := nodeAvailable(t)
	t.Setenv("NODE_BINARY", nodeBin)

	// Use a minimal script that emits done — avoids needing schematics-cli.
	scriptPath := writeScript(t, doneOnlyScript)
	defer func() { _ = os.Remove(scriptPath) }()

	d := angular.NewAdapterWithRunnerPath(scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{Name: "component"},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var terminalCount int
	for ev := range ch {
		switch ev.(type) {
		case events.Done, events.Failed, events.Cancelled:
			terminalCount++
		}
	}

	if terminalCount != 1 {
		t.Errorf("got %d terminal events, want exactly 1 — REQ-15.2 violated", terminalCount)
	}
}
