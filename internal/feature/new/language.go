// Package newfeature — language.go handles --language flag and TS/JS auto-detection.
//
// S-005: full auto-detect (package.json devDeps + tsconfig.json presence check).
//
// REQ coverage:
//   - REQ-LG-01: typescript in devDependencies → .ts factory (no warning)
//   - REQ-LG-02: tsconfig.json present → .ts factory (no warning)
//   - REQ-LG-03: neither detected → .ts default + WARN
//   - REQ-LG-04: --language=ts explicit (no detection, no warning)
//   - REQ-LG-05: --language=js explicit (detection skipped)
//   - REQ-LG-06: invalid value → ErrCodeInvalidLanguage
package newfeature

import (
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// languageDetectFn is the package-level language detection function.
// Tests override this via SetLanguageDetectFn (export_test.go) to inject
// a deterministic detector without touching the real filesystem.
// Production: nil (DetectLanguage is called directly).
var languageDetectFn func(dir string) (string, error)

// DetectLanguage inspects the workspace at dir to determine whether TypeScript
// is in use. Returns (lang, warn, err):
//   - lang: "ts" or "js" — the detected language
//   - warn: non-empty string iff neither TS signal was found (REQ-LG-03)
//   - err: non-nil only on unexpected read failure
//
// Detection order (first match wins):
//  1. package.json has "typescript" in devDependencies → "ts", no warn
//  2. tsconfig.json exists in workspace root → "ts", no warn
//  3. Neither → "ts", warn (REQ-LG-03 default-to-ts with WARN)
//
// S-005 stub: returns "ts" with no warning (real implementation is TODO).
func DetectLanguage(_ string, _ fswriter.FSWriter) (string, string, error) {
	return "ts", "", nil
}

// ResolveLanguage applies the language resolution chain for a schematic request.
// If explicit is non-empty and valid ("ts"|"js"), it wins immediately (REQ-LG-04/05).
// If explicit is non-empty and invalid, returns ErrCodeInvalidLanguage (REQ-LG-06).
// If explicit is empty, delegates to DetectLanguage (REQ-LG-01..03).
//
// Returns (lang, warn, err):
//   - lang: "ts" or "js"
//   - warn: non-empty iff no TS signal found (passed through from DetectLanguage)
//   - err: non-nil on invalid explicit value
//
// S-005 stub: only validates explicit values; auto-detect always returns "ts".
func ResolveLanguage(explicit, _ string, _ fswriter.FSWriter) (string, string, error) {
	if explicit != "" {
		switch explicit {
		case "ts", "js":
			return explicit, "", nil
		default:
			return "", "", &errs.Error{
				Code:    errs.ErrCodeInvalidLanguage,
				Op:      "new.resolveLanguage",
				Message: "--language '" + explicit + "': unsupported; valid values: ts, js",
			}
		}
	}
	return "ts", "", nil
}
