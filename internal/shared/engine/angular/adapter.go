// Package angular implements AngularSubprocessAdapter — the concrete
// implementation of engine.Engine that spawns Node.js via os/exec.
//
// # Security invariants
//
//   - NO shell invocation. exec.CommandContext only. cmd.Path is always the
//     Node.js binary (REQ-02.1).
//   - SchematicRef fields are validated before reaching cmd.Args (REQ-05).
//   - cmd.Env is built by buildEnv (default-deny allowlist; PATH always present).
//   - The embedded runner.js is written to a temp file and deleted after exit
//     (REQ-19.2).
//   - All errors returned as *errors.Error with Op matching
//     ^angular\.[a-z][a-z0-9_]*$ (REQ-16.1).
//
// # Composition
//
// Only cmd/builder/main.go imports this package. Feature packages MUST NOT
// import concrete adapters (FF-03).
package angular

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/discoverer"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular/runner"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// AngularSubprocessAdapter implements engine.Engine by spawning a Node.js
// subprocess that runs the embedded runner.js script.
//
// compile-time assertion lives in adapter_test.go:
//
//	var _ engine.Engine = (*AngularSubprocessAdapter)(nil)
//
//nolint:revive // stutter intentional: AngularSubprocessAdapter is the spec-mandated name (ADR-04).
type AngularSubprocessAdapter struct {
	// discovererFunc returns the discovered Node.js binary path.
	// Injected at construction time — enables unit-test substitution without
	// requiring an interface (ADR-04).
	discovererFunc func() (string, error)

	// tempFileSpy is called with the runner temp file path just before the
	// process is started. Used only in tests (via NewAdapterWithSpy).
	// nil in production.
	tempFileSpy func(path string)

	// cmdSpy is called with the fully-constructed *exec.Cmd just before
	// cmd.Start() is called. Used only in tests (via NewAdapterWithCmdSpy)
	// to assert cmd.Path and cmd.Args without a shell. nil in production.
	// REQ-02.1 test coverage.
	cmdSpy func(cmd *exec.Cmd)
}

// NewAdapter returns an AngularSubprocessAdapter wired with the real Discoverer.
// This is the constructor used by composeApp.
func NewAdapter() *AngularSubprocessAdapter {
	d := discoverer.New()
	return &AngularSubprocessAdapter{
		discovererFunc: d.FindNode,
	}
}

// NewAdapterWithSpy returns an AngularSubprocessAdapter wired with the real
// Discoverer plus a spy callback that is called with the runner temp file path
// just before the process is started. Used only in tests.
func NewAdapterWithSpy(spy func(path string)) *AngularSubprocessAdapter {
	d := discoverer.New()
	return &AngularSubprocessAdapter{
		discovererFunc: d.FindNode,
		tempFileSpy:    spy,
	}
}

// NewAdapterWithCmdSpy returns an AngularSubprocessAdapter wired with the real
// Discoverer plus a spy callback that is called with the fully-constructed
// *exec.Cmd just before cmd.Start() is called. Used only in tests to assert
// cmd.Path and cmd.Args without shell involvement (REQ-02.1).
func NewAdapterWithCmdSpy(spy func(cmd *exec.Cmd)) *AngularSubprocessAdapter {
	d := discoverer.New()
	return &AngularSubprocessAdapter{
		discovererFunc: d.FindNode,
		cmdSpy:         spy,
	}
}

// Execute begins schematic execution and returns a read-only event channel.
//
// Pre-execution steps (return error, channel is nil):
//  1. Validate SchematicRef (REQ-05) — S-000 skeleton: no-op validation.
//  2. Discover Node.js binary via discovererFunc (REQ-10).
//  3. Write embedded runner.js to a temp file (REQ-19.2).
//
// Post-start steps (terminal events arrive on channel):
//  4. exec.CommandContext(ctx, nodeBin, runnerTempPath) — no shell (REQ-02.1).
//  5. Goroutine fan-out: stdout→NDJSON decode, stderr→LogLine, ctx→kill.
//  6. Temp file deferred-deleted after cmd.Wait() (REQ-19.2).
func (a *AngularSubprocessAdapter) Execute(
	ctx context.Context,
	req engine.ExecuteRequest,
) (<-chan events.Event, error) {
	// Step 1: Validate SchematicRef (skeleton: no-op; full validation in S-002).
	if err := validateRef(req.Schematic); err != nil {
		return nil, err
	}

	// Step 2: Discover Node.js binary.
	nodeBin, err := a.discovererFunc()
	if err != nil {
		return nil, err
	}

	// Step 3: Write embedded runner.js to a temp file.
	runnerTempPath, err := writeRunnerTemp()
	if err != nil {
		return nil, err
	}

	// Notify spy (test only) before process start.
	if a.tempFileSpy != nil {
		a.tempFileSpy(runnerTempPath)
	}

	// Step 4+5: Start process and launch goroutine fan-out.
	// cmdSpy (test-only) is invoked inside startProcess before cmd.Start().
	ch, _, err := startProcess(ctx, nodeBin, runnerTempPath, req.EnvAllowlist, a.cmdSpy)
	if err != nil {
		// Clean up temp file if process failed to start.
		_ = os.Remove(runnerTempPath)
		return nil, err
	}

	return ch, nil
}

// writeRunnerTemp writes the embedded runner.js bytes to a temporary file and
// returns its path. The caller is responsible for deleting the file.
func writeRunnerTemp() (string, error) {
	f, err := os.CreateTemp("", "pb-runner-*.js")
	if err != nil {
		return "", &errors.Error{
			Op:      "angular.write_runner_temp",
			Code:    errors.ErrCodeExecutionFailed,
			Message: "failed to create temp file for runner script",
			Cause:   err,
		}
	}

	name := f.Name()

	if _, err := f.Write(runner.Script); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", &errors.Error{
			Op:      "angular.write_runner_temp",
			Code:    errors.ErrCodeExecutionFailed,
			Message: fmt.Sprintf("failed to write runner script to temp file: %v", err),
			Cause:   err,
		}
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", &errors.Error{
			Op:      "angular.write_runner_temp",
			Code:    errors.ErrCodeExecutionFailed,
			Message: "failed to close runner temp file",
			Cause:   err,
		}
	}

	return name, nil
}
