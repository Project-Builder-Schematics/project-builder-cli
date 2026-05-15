// Package newfeature — language.go handles --language flag and TS/JS auto-detection.
//
// S-005: full auto-detect (package.json devDeps + tsconfig.json presence check).
//
// REQ coverage:
//   - REQ-LG-01: typescript in devDependencies → .ts factory (no warning)
//   - REQ-LG-02: tsconfig.json present → .ts factory (no warning)
//   - REQ-LG-03: neither detected → .ts default + WARN (ADR-019: via NewResult.Warnings)
//   - REQ-LG-04: --language=ts explicit (no detection, no warning)
//   - REQ-LG-05: --language=js explicit (detection skipped)
//   - REQ-LG-06: invalid value → ErrCodeInvalidLanguage
package newfeature

import (
	"encoding/json"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// languageDetectFn is the package-level language detection function.
// Tests override this via SetLanguageDetectFn (export_test.go) to inject
// a deterministic detector without touching the real filesystem.
// Production: nil (DetectLanguage is called directly).
var languageDetectFn func(dir string) (string, error)

// warnTSNotDetected is the standard WARN message for REQ-LG-03.
// Routed through NewResult.Warnings → RenderPretty (ADR-019).
const warnTSNotDetected = "TypeScript not detected; defaulting to .ts. Use --language=js to suppress."

// DetectLanguage inspects the workspace at dir to determine whether TypeScript
// is in use. Returns (lang, warn, err):
//   - lang: "ts" or "js" — the detected language
//   - warn: non-empty string iff neither TS signal was found (REQ-LG-03)
//   - err: non-nil only on unexpected structural failure (not ErrNotExist)
//
// Detection order (first match wins):
//  1. package.json has "typescript" in devDependencies → "ts", no warn (REQ-LG-01)
//  2. tsconfig.json exists in workspace root → "ts", no warn (REQ-LG-02)
//  3. Neither → "ts", warn (REQ-LG-03 default-to-ts with WARN)
func DetectLanguage(dir string, fs fswriter.FSWriter) (string, string, error) {
	// 1. Check package.json devDependencies (REQ-LG-01).
	pkgPath := filepath.Join(dir, "package.json")
	if pkgBytes, err := fs.ReadFile(pkgPath); err == nil {
		if hasTypescriptDevDep(pkgBytes) {
			return "ts", "", nil
		}
	}

	// 2. Check tsconfig.json existence (REQ-LG-02).
	tscPath := filepath.Join(dir, "tsconfig.json")
	if _, err := fs.Stat(tscPath); err == nil {
		return "ts", "", nil
	}

	// 3. Neither signal found — default to "ts" with WARN (REQ-LG-03).
	return "ts", warnTSNotDetected, nil
}

// hasTypescriptDevDep reports whether rawJSON is a valid package.json that
// contains "typescript" as a key under "devDependencies".
// Returns false (not an error) on any parse failure — malformed package.json
// is treated as "typescript not found" rather than a fatal error.
func hasTypescriptDevDep(rawJSON []byte) bool {
	var pkg struct {
		DevDependencies map[string]json.RawMessage `json:"devDependencies"`
	}
	if err := json.Unmarshal(rawJSON, &pkg); err != nil {
		return false
	}
	_, ok := pkg.DevDependencies["typescript"]
	return ok
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
func ResolveLanguage(explicit, dir string, fs fswriter.FSWriter) (string, string, error) {
	if explicit != "" {
		switch explicit {
		case "ts", "js":
			// Explicit flag wins — no detection, no warning (REQ-LG-04/05).
			return explicit, "", nil
		default:
			return "", "", &errs.Error{
				Code:    errs.ErrCodeInvalidLanguage,
				Op:      "new.resolveLanguage",
				Message: "--language '" + explicit + "': unsupported; valid values: ts, js",
			}
		}
	}

	// Empty flag — auto-detect (REQ-LG-01..03).
	return DetectLanguage(dir, fs)
}
