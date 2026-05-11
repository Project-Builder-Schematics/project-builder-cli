//go:build !windows

package angular

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// sigTermGracePeriod is the time the adapter waits after sending SIGTERM before
// escalating to SIGKILL (REQ-17.1, REQ-17.2).
const sigTermGracePeriod = 3 * time.Second

// killProcess implements the POSIX cancellation path (REQ-17):
// SIGTERM → wait sigTermGracePeriod → SIGKILL.
//
// pipesDrained is closed by runFanOut after stdout+stderr drain, signalling
// that the process has exited. If it fires before the grace period elapses,
// SIGKILL is skipped (process died voluntarily after SIGTERM).
//
// killDone is closed by defer when this function returns, signalling runFanOut
// that it may now emit Cancelled and close the event channel.
//
// Build tag: //go:build !windows — REQ-18.1.
func killProcess(ctx context.Context, cmd *exec.Cmd, killDone chan<- struct{}, pipesDrained <-chan struct{}) {
	defer close(killDone)

	// Wait for cancellation or normal exit (pipesDrained fires if process exits
	// before ctx is cancelled, e.g. completed successfully while we wait here).
	select {
	case <-ctx.Done():
		// Context cancelled — begin kill sequence.
	case <-pipesDrained:
		// Process exited naturally before cancellation — nothing to kill.
		return
	}

	if cmd.Process == nil {
		return
	}

	// Send SIGTERM first (REQ-17.1).
	_ = cmd.Process.Signal(syscall.SIGTERM)

	// Wait up to sigTermGracePeriod for the process to exit voluntarily.
	timer := time.NewTimer(sigTermGracePeriod)
	defer timer.Stop()

	select {
	case <-pipesDrained:
		// Process exited within grace period — SIGKILL not needed (REQ-17.1).
	case <-timer.C:
		// Grace period elapsed — escalate to SIGKILL (REQ-17.2).
		_ = cmd.Process.Kill()
	}
	// killDone closed by defer; runFanOut will emit Cancelled.
}
