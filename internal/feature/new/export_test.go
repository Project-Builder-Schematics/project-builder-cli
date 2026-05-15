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

// SetLanguageDetectFn replaces the languageDetectFn for the duration of t.
// languageDetectFn is declared in language.go (production file) so it is
// accessible to both production code and tests.
//
// Use this in tests that need deterministic language detection without
// a real workspace on disk (CI runners may have tsconfig.json installed).
func SetLanguageDetectFn(t testing.TB, fn func(dir string) (string, error)) {
	t.Helper()
	orig := languageDetectFn
	languageDetectFn = fn
	t.Cleanup(func() { languageDetectFn = orig })
}

// SetTTYCheckFn replaces the ttyCheckFn for the duration of t.
// ttyCheckFn is declared in extends.go (production file).
//
// Use this in tests that need deterministic TTY detection without depending
// on the real stdin state (which varies by shell, CI, WSL2 environment).
func SetTTYCheckFn(t testing.TB, fn func() bool) {
	t.Helper()
	orig := ttyCheckFn
	ttyCheckFn = fn
	t.Cleanup(func() { ttyCheckFn = orig })
}
