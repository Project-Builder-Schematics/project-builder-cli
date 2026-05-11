//go:build windows

package angular

import (
	"context"
	"os/exec"
)

// killProcess implements the Windows cancellation path (REQ-18.2):
// cmd.Process.Kill() directly (maps to TerminateProcess).
//
// pipesDrained is closed by runFanOut after stdout+stderr drain. If it fires
// before ctx.Done(), the process has already exited and no kill is needed.
//
// killDone is closed by defer when this function returns, signalling runFanOut
// that it may now emit Cancelled and close the event channel.
//
// Build tag: //go:build windows — REQ-18.2.
func killProcess(ctx context.Context, cmd *exec.Cmd, killDone chan<- struct{}, pipesDrained <-chan struct{}) {
	defer close(killDone)

	select {
	case <-ctx.Done():
		// Context cancelled — kill the process.
	case <-pipesDrained:
		// Process exited naturally before cancellation — nothing to kill.
		return
	}

	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
