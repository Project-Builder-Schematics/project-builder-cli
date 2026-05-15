// Package newfeature — export_test.go exposes package-level injectable vars
// for test isolation (per L-builder-init-01 pattern).
//
// Each injection fn can be swapped per-test via the Set* helpers below;
// the original is restored via t.Cleanup so tests remain independent and race-clean.
//
// S-000b: The seam exists with at minimum SetLanguageDetectFn for downstream
// slices (S-005 language auto-detection). Additional seams are added as needed.
package newfeature

import (
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

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

// SetPromptExtendsFn replaces the promptExtendsFn for the duration of t.
// promptExtendsFn is declared in extends.go (production file).
//
// Use this in tests that simulate an interactive TUI extends prompt
// (REQ-EX-04) without spawning a real Bubble Tea UI.
//
// fn signature: func(externals []string) (selected string, skipped bool, err error)
func SetPromptExtendsFn(t testing.TB, fn func(externals []string) (string, bool, error)) {
	t.Helper()
	orig := promptExtendsFn
	promptExtendsFn = fn
	t.Cleanup(func() { promptExtendsFn = orig })
}

// NewOSWriterForTest returns a real-OS FSWriter for use in tests that require
// actual filesystem mutations (e.g. ADV-09 read-only filesystem test).
// Tests MUST use t.TempDir() and restore permissions via t.Cleanup.
func NewOSWriterForTest() fswriter.FSWriter {
	return fswriter.NewOSWriter()
}
