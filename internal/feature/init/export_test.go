// Package initialise — export_test.go exposes package-level injectable vars
// for test isolation (obs #181 pattern — CI host-env isolation for priority chains).
//
// Each var can be swapped per-test via the helper below; the original is
// restored via t.Cleanup so tests remain independent and race-clean.
package initialise

import (
	"testing"
)

// SetLookPathFn replaces the lookPath function (wrapping exec.LookPath) for
// the duration of t. Tests use this to simulate "PM binary not found" or to
// prevent real subprocess discovery.
func SetLookPathFn(t testing.TB, fn func(string) (string, error)) {
	t.Helper()
	orig := lookPathFn
	lookPathFn = fn
	t.Cleanup(func() { lookPathFn = orig })
}

// SetLockfileLookupFn replaces the lockfileLookup function for the duration
// of t. Tests use this to inject a fake filesystem check for lockfile detection
// without needing a real directory on disk.
func SetLockfileLookupFn(t testing.TB, fn func(dir, filename string) bool) {
	t.Helper()
	orig := lockfileLookupFn
	lockfileLookupFn = fn
	t.Cleanup(func() { lockfileLookupFn = orig })
}
