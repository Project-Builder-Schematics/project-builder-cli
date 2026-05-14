// Package initialise — pkgmanager.go provides the real PackageManagerRunner
// implementation for S-005 (REQ-PD-01..04, ADR-023).
//
// Priority chain for PM detection (REQ-PD-01):
//  1. --pm flag (non-empty PackageManagerFlag overrides all)
//  2. Lockfile sniff in req.Directory (pnpm > yarn > bun > npm)
//  3. Fallback: npm (no error from Detect; binary check deferred to Install)
//
// Install subprocess (REQ-PD-02, ADR-023):
//   - Binary resolved via exec.LookPath (wrapped in lookPathFn for test injection)
//   - Binary path validated with shell-metachar guard (defence-in-depth)
//   - exec.CommandContext with 120-second timeout wired inside Install
//   - Inherits caller's env (private registry support)
//   - No sh -c interpolation — typed args only
//   - On missing binary → ErrCodeInitPackageManagerNotFound
//   - On context deadline/cancel → wraps the original context error
//   - On non-zero exit → error wrapping stderr output
//
// Package-level vars (lookPathFn, lockfileLookupFn) are replaceable in tests
// via export_test.go helpers (obs #181 isolation pattern).
package initialise

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/validate"
)

// --- Package-level injectable vars (obs #181 pattern) ---

// lookPathFn wraps exec.LookPath. Tests replace this to simulate missing
// or pathological binary paths without touching the real PATH.
var lookPathFn = exec.LookPath

// lockfileLookupFn checks whether filename exists in dir.
// Tests replace this to inject a fake filesystem without real directories.
var lockfileLookupFn = osLockfileLookup

// pmInstallTimeout is the subprocess timeout mandated by ADR-023.
// Not replaceable via export_test — tests use context directly.
const pmInstallTimeout = 120 * time.Second

// --- Lockfile → PM mapping (REQ-PD-01 priority 2) ---

// lockfileOrder is the priority-ordered list of lockfile names and their
// associated PM. First match wins.
var lockfileOrder = []struct {
	filename string
	pm       PackageManager
}{
	{"pnpm-lock.yaml", PMPnpm},
	{"yarn.lock", PMYarn},
	{"bun.lockb", PMBun},
	{"package-lock.json", PMNpm},
}

// osLockfileLookup is the production lockfile checker — it uses os.Stat.
func osLockfileLookup(dir, filename string) bool {
	_, err := os.Stat(filepath.Join(dir, filename))
	return err == nil
}

// --- realPM: the concrete PackageManagerRunner ---

// realPM is the S-005 real implementation of PackageManagerRunner.
// It replaces pkgmanager_stub.go wholesale.
type realPM struct{}

// NewRealPM returns the PackageManagerRunner for use by composeApp.
func NewRealPM() PackageManagerRunner { return &realPM{} }

// Detect returns the resolved PackageManager for dir.
//
// Priority chain (REQ-PD-01):
//  1. flag != PMUnset → return flag immediately
//  2. lockfile sniff (pnpm-lock.yaml > yarn.lock > bun.lockb > package-lock.json)
//  3. fallback → PMNpm (binary check is deferred to Install time)
func (r *realPM) Detect(dir string, flag PackageManager) (PackageManager, error) {
	// Priority 1: explicit --pm flag.
	if flag != PMUnset {
		return flag, nil
	}

	// Priority 2: lockfile sniff.
	for _, entry := range lockfileOrder {
		if lockfileLookupFn(dir, entry.filename) {
			return entry.pm, nil
		}
	}

	// Priority 3: fallback.
	return PMNpm, nil
}

// Install runs `<pm> install --save-dev @pbuilder/sdk` in dir.
//
// ADR-023 rules:
//   - binary path validated via shell-metachar guard before exec
//   - 120-second internal timeout wrapped around the provided ctx
//   - inherits caller's environment (for NPM_TOKEN, registry config, proxy)
//   - no sh -c; typed args only
//
// Errors:
//   - PMUnset → ErrCodeInitPackageManagerNotFound
//   - binary not in PATH → ErrCodeInitPackageManagerNotFound
//   - metachar in binary path → ErrCodeInvalidInput
//   - context cancelled/deadline → wraps context.Canceled or context.DeadlineExceeded
//   - non-zero exit → wraps combined stderr output
func (r *realPM) Install(ctx context.Context, dir string, pm PackageManager) error {
	if pm == PMUnset {
		return &apperrors.Error{
			Code:    apperrors.ErrCodeInitPackageManagerNotFound,
			Op:      "init.handler",
			Message: "no package manager specified — run Detect first or set --pm flag",
		}
	}

	// Resolve the binary via PATH.
	binPath, err := lookPathFn(string(pm))
	if err != nil {
		return &apperrors.Error{
			Code:    apperrors.ErrCodeInitPackageManagerNotFound,
			Op:      "init.handler",
			Message: fmt.Sprintf("package manager %q not found in PATH — install it and re-run, or use --no-install", pm),
			Cause:   err,
			Suggestions: []string{
				fmt.Sprintf("install %s: https://nodejs.org (or visit the PM's official docs)", pm),
				"use --no-install to skip the install step",
			},
		}
	}

	// Defence-in-depth: reject metacharacters in the resolved binary path.
	// exec.CommandContext does not invoke a shell, but we reject them anyway
	// to surface any PATH manipulation attack early (ADR-023).
	if err := validate.RejectMetachars("init.handler", "package manager binary path", binPath); err != nil {
		return err
	}

	// Wrap the provided context with ADR-023's 120-second internal timeout.
	// If the caller already has a shorter deadline, that takes effect first.
	installCtx, cancel := context.WithTimeout(ctx, pmInstallTimeout)
	defer cancel()

	// Typed args — no sh -c, no shell interpolation (ADR-023 security invariant).
	// #nosec G204 — args are type-safe; binary path validated above.
	cmd := exec.CommandContext(installCtx, binPath, "install", "--save-dev", "@pbuilder/sdk")
	cmd.Dir = dir
	cmd.Env = os.Environ() // inherit caller's environment (NPM_TOKEN, registry, proxy)

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		// Unwrap context errors before wrapping in apperrors so callers can
		// use errors.Is(err, context.Canceled) / context.DeadlineExceeded.
		if installCtx.Err() != nil {
			return installCtx.Err()
		}
		return &apperrors.Error{
			Code:    apperrors.ErrCodeExecutionFailed,
			Op:      "init.handler",
			Message: fmt.Sprintf("package manager %q install failed: %s", pm, strings.TrimSpace(string(out))),
			Cause:   runErr,
			Suggestions: []string{
				"check your network connectivity",
				"verify your npm registry configuration",
				"use --no-install to skip the install step and run it manually later",
			},
		}
	}

	return nil
}
