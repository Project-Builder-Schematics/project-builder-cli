// Package newfeature — language.go handles --language flag and TS/JS auto-detection.
//
// S-001: language resolution is minimal — explicit "ts" or "js" flags are
// respected; empty string defaults to "ts". Full auto-detect (package.json
// devDeps + tsconfig.json presence check) lands in S-005.
//
// REQ coverage (S-005 completes these):
//   - REQ-LG-01: typescript in devDependencies → .ts factory
//   - REQ-LG-02: tsconfig.json present → .ts factory
//   - REQ-LG-03: neither detected → .ts default + WARN
//   - REQ-LG-04: --language=ts explicit
//   - REQ-LG-05: --language=js explicit
//   - REQ-LG-06: invalid value → ErrCodeInvalidLanguage
package newfeature

// languageDetectFn is the package-level language detection function.
// In production this is nil (S-001 defaults to "ts").
// Tests override this via SetLanguageDetectFn (export_test.go) to inject
// a deterministic detector without touching the real filesystem.
// Real implementation (S-005) sets this to the auto-detect logic.
var languageDetectFn func(dir string) (string, error)
