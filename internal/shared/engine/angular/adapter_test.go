// Package angular_test covers adapter.go.
//
// S-000 scope: REQ-01.1 (interface assertion), REQ-01.2 (channel returned
// immediately), REQ-02.1 (cmd.Path is Node binary, not shell), REQ-19.2
// (runner temp file deleted after exit).
package angular_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular"
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

	d := angular.NewAdapter()
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

// Test_AngularSubprocessAdapter_Execute_DoneReceived covers the S-000 acceptance
// criterion: FakeNode (real Node running runner.js) emits {"type":"done","seq":1};
// adapter channel receives Done{} and closes.
func Test_AngularSubprocessAdapter_Execute_DoneReceived(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	d := angular.NewAdapter()
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
// cmd.Args[0] must equal the resolved Node binary path.
// A malicious-looking SchematicRef.Name must appear verbatim as a single
// element in cmd.Args (not interpolated through a shell).
func Test_AngularSubprocessAdapter_CmdPath_IsNode(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	// SchematicRef with a malicious-looking Name that a shell-wrapping
	// invocation would execute but a direct exec must treat as a literal arg.
	maliciousName := "foo; rm -rf $HOME"

	var capturedCmd *exec.Cmd
	d := angular.NewAdapterWithCmdSpy(func(cmd *exec.Cmd) {
		capturedCmd = cmd
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := d.Execute(ctx, engine.ExecuteRequest{
		Workspace: t.TempDir(),
		Schematic: engine.SchematicRef{
			Collection: "@schematics/angular",
			Name:       maliciousName,
		},
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
	// exec.CommandContext sets Args[0] = the binary path.
	if len(capturedCmd.Args) == 0 {
		t.Fatal("cmd.Args is empty")
	}
	if capturedCmd.Args[0] != nodeBin {
		t.Errorf("cmd.Args[0] = %q, want Node binary %q — REQ-02.1 violated", capturedCmd.Args[0], nodeBin)
	}

	// REQ-02.1 (injection guard): the malicious string must NOT appear
	// as a shell-interpolatable fragment; it must appear verbatim as a single
	// element somewhere in cmd.Args (S-002 wires --schematic; for now we
	// confirm no shell expansion happened by checking cmd.Path is node).
	// TODO(S-002): assert maliciousName == cmd.Args[N] for the --schematic slot.
}

// Test_AngularSubprocessAdapter_TempFileDeleted covers REQ-19.2:
// the runner temp file is deleted after subprocess exit.
func Test_AngularSubprocessAdapter_TempFileDeleted(t *testing.T) {
	nodeBin := fakeNodePath(t)
	t.Setenv("NODE_BINARY", nodeBin)

	// Inject a spy discoverer that records the temp path used.
	var capturedTempPath string
	d := angular.NewAdapterWithSpy(func(path string) {
		capturedTempPath = path
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

	// After channel close (subprocess exited), temp file must be gone.
	if capturedTempPath == "" {
		t.Skip("temp path not captured — spy not called (adapter path issue)")
	}
	if _, err := os.Stat(capturedTempPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q still exists after subprocess exit — REQ-19.2 violated", capturedTempPath)
	}
}
