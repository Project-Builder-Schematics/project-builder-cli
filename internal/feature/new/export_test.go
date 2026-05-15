// Package newfeature — export_test.go exposes package-level injectable vars
// for test isolation (per L-builder-init-01 pattern).
//
// Each injection fn can be swapped per-test via the Set* helpers below;
// the original is restored via t.Cleanup so tests remain independent and race-clean.
//
// S-000b: The seam exists with at minimum SetLanguageDetectFn for downstream
// slices (S-005 language auto-detection). Additional seams are added as needed.
package newfeature

import "testing"

// languageDetectFn is the package-level language detection function.
// Tests can override this to prevent TS/JS auto-detection from touching
// the real filesystem (e.g. scanning for tsconfig.json, package.json).
// Real implementation lands in S-005 (language.go).
var languageDetectFn func(dir string) (string, error)

// SetLanguageDetectFn replaces the languageDetectFn for the duration of t.
// Use this in tests that need deterministic language detection without
// a real workspace on disk (CI runners may have tsconfig.json installed).
func SetLanguageDetectFn(t testing.TB, fn func(dir string) (string, error)) {
	t.Helper()
	orig := languageDetectFn
	languageDetectFn = fn
	t.Cleanup(func() { languageDetectFn = orig })
}
