// Package newfeature — language_test.go covers --language flag auto-detection
// and explicit override logic.
//
// REQ coverage:
//   - REQ-LG-01: typescript in devDependencies → .ts factory (no warning)
//   - REQ-LG-02: tsconfig.json present → .ts factory (no warning)
//   - REQ-LG-03: neither detected → .ts default + WARN via Renderer
//   - REQ-LG-04: --language=ts explicit → .ts (no detection, no warning)
//   - REQ-LG-05: --language=js explicit → .js (detection skipped)
//   - REQ-LG-06: --language=python → ErrCodeInvalidLanguage
package newfeature_test

import (
	"errors"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// packageJSONWithTypescript is a minimal package.json that lists typescript
// in devDependencies (REQ-LG-01).
const packageJSONWithTypescript = `{
  "name": "my-project",
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}`

// packageJSONNoTypescript is a minimal package.json without typescript.
const packageJSONNoTypescript = `{
  "name": "my-project",
  "devDependencies": {}
}`

// tsconfigJSON is a minimal tsconfig.json (REQ-LG-02).
const tsconfigJSON = `{
  "compilerOptions": {
    "target": "ES2020"
  }
}`

// Test_DetectLanguage_TypescriptInDevDeps verifies that typescript in
// devDependencies triggers .ts detection with no warning (REQ-LG-01).
func Test_DetectLanguage_TypescriptInDevDeps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// Write package.json with typescript in devDependencies.
	if err := fs.WriteFile(dir+"/package.json", []byte(packageJSONWithTypescript), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	lang, warn, err := newfeature.DetectLanguage(dir, fs)
	if err != nil {
		t.Fatalf("DetectLanguage: unexpected error: %v", err)
	}
	if lang != "ts" {
		t.Errorf("DetectLanguage: lang = %q; want %q (REQ-LG-01)", lang, "ts")
	}
	if warn != "" {
		t.Errorf("DetectLanguage: warn = %q; want empty (REQ-LG-01: no warning when typescript found)", warn)
	}
}

// Test_DetectLanguage_TsconfigPresent verifies that tsconfig.json presence
// triggers .ts detection with no warning (REQ-LG-02).
func Test_DetectLanguage_TsconfigPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// No package.json, but tsconfig.json present.
	if err := fs.WriteFile(dir+"/tsconfig.json", []byte(tsconfigJSON), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	lang, warn, err := newfeature.DetectLanguage(dir, fs)
	if err != nil {
		t.Fatalf("DetectLanguage: unexpected error: %v", err)
	}
	if lang != "ts" {
		t.Errorf("DetectLanguage: lang = %q; want %q (REQ-LG-02)", lang, "ts")
	}
	if warn != "" {
		t.Errorf("DetectLanguage: warn = %q; want empty (REQ-LG-02: no warning when tsconfig found)", warn)
	}
}

// Test_DetectLanguage_BothNegative verifies that when neither typescript
// in devDeps nor tsconfig.json is present, the detector returns "ts" with a
// warning (REQ-LG-03).
func Test_DetectLanguage_BothNegative(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// No package.json, no tsconfig.json.
	lang, warn, err := newfeature.DetectLanguage(dir, fs)
	if err != nil {
		t.Fatalf("DetectLanguage: unexpected error: %v", err)
	}
	if lang != "ts" {
		t.Errorf("DetectLanguage: lang = %q; want %q (REQ-LG-03 default)", lang, "ts")
	}
	if warn == "" {
		t.Error("DetectLanguage: warn is empty; want non-empty WARN (REQ-LG-03 both-negative case)")
	}
	// Warning must mention TypeScript and suggest --language=js.
	if !containsAll(warn, "TypeScript", "--language=js") {
		t.Errorf("DetectLanguage: warn = %q; must mention TypeScript and --language=js", warn)
	}
}

// Test_DetectLanguage_PackageJSONNoTypescript verifies that a package.json
// without typescript triggers the tsconfig.json fallback check (REQ-LG-02).
func Test_DetectLanguage_PackageJSONNoTypescript(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// package.json exists but no typescript devDep + tsconfig.json also present.
	if err := fs.WriteFile(dir+"/package.json", []byte(packageJSONNoTypescript), 0o644); err != nil {
		t.Fatalf("setup package.json: %v", err)
	}
	if err := fs.WriteFile(dir+"/tsconfig.json", []byte(tsconfigJSON), 0o644); err != nil {
		t.Fatalf("setup tsconfig.json: %v", err)
	}

	lang, warn, err := newfeature.DetectLanguage(dir, fs)
	if err != nil {
		t.Fatalf("DetectLanguage: unexpected error: %v", err)
	}
	if lang != "ts" {
		t.Errorf("DetectLanguage: lang = %q; want %q (tsconfig fallback, REQ-LG-02)", lang, "ts")
	}
	if warn != "" {
		t.Errorf("DetectLanguage: warn = %q; want empty (tsconfig found, no warning)", warn)
	}
}

// Test_ResolveLanguageExplicit_TS verifies that --language=ts explicit returns
// "ts" with no warning and no detection (REQ-LG-04).
func Test_ResolveLanguageExplicit_TS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// No TS signals in workspace — explicit flag wins, no detection performed.
	lang, warn, err := newfeature.ResolveLanguage("ts", dir, fs)
	if err != nil {
		t.Fatalf("ResolveLanguage(ts): unexpected error: %v", err)
	}
	if lang != "ts" {
		t.Errorf("ResolveLanguage(ts): lang = %q; want %q", lang, "ts")
	}
	if warn != "" {
		t.Errorf("ResolveLanguage(ts): warn = %q; want empty (explicit flag, REQ-LG-04)", warn)
	}
}

// Test_ResolveLanguageExplicit_JS verifies that --language=js explicit returns
// "js" with no warning and no detection (REQ-LG-05).
func Test_ResolveLanguageExplicit_JS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// Workspace has typescript signals — explicit js flag overrides detection.
	if err := fs.WriteFile(dir+"/tsconfig.json", []byte(tsconfigJSON), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	lang, warn, err := newfeature.ResolveLanguage("js", dir, fs)
	if err != nil {
		t.Fatalf("ResolveLanguage(js): unexpected error: %v", err)
	}
	if lang != "js" {
		t.Errorf("ResolveLanguage(js): lang = %q; want %q", lang, "js")
	}
	if warn != "" {
		t.Errorf("ResolveLanguage(js): warn = %q; want empty (explicit flag, REQ-LG-05)", warn)
	}
}

// Test_ResolveLanguage_InvalidValue verifies that an unsupported --language value
// returns ErrCodeInvalidLanguage (REQ-LG-06).
func Test_ResolveLanguage_InvalidValue(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	cases := []struct {
		value string
	}{
		{"python"},
		{"ruby"},
		{"Java"},
		{"Typescript"},
		{"TS"},
		{"JS"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.value, func(t *testing.T) {
			t.Parallel()

			_, _, err := newfeature.ResolveLanguage(tc.value, dir, fs)
			if err == nil {
				t.Fatalf("ResolveLanguage(%q): expected ErrCodeInvalidLanguage; got nil", tc.value)
			}

			var e *errs.Error
			if !errors.As(err, &e) {
				t.Fatalf("ResolveLanguage(%q): error not *errs.Error; got: %T %v", tc.value, err, err)
			}
			if e.Code != errs.ErrCodeInvalidLanguage {
				t.Errorf("ResolveLanguage(%q): code = %q; want %q", tc.value, e.Code, errs.ErrCodeInvalidLanguage)
			}
			// Message must mention the invalid value and valid options (REQ-EC-06).
			if !containsAll(e.Message, tc.value, "ts", "js") {
				t.Errorf("ResolveLanguage(%q): message = %q; must mention value and valid options", tc.value, e.Message)
			}
		})
	}
}

// Test_ResolveLanguage_AutoDetect_NeitherSignal verifies that auto-detect with
// no TS signals returns "ts" with a warning (REQ-LG-03 via ResolveLanguage).
func Test_ResolveLanguage_AutoDetect_NeitherSignal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fswriter.NewFakeFS()

	// Empty language = auto-detect, no TS signals = default + warn.
	lang, warn, err := newfeature.ResolveLanguage("", dir, fs)
	if err != nil {
		t.Fatalf("ResolveLanguage(''): unexpected error: %v", err)
	}
	if lang != "ts" {
		t.Errorf("ResolveLanguage(''): lang = %q; want %q (REQ-LG-03)", lang, "ts")
	}
	if warn == "" {
		t.Error("ResolveLanguage(''): want non-empty WARN for both-negative case (REQ-LG-03)")
	}
}

// containsAll returns true iff all substrings are present in s.
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
