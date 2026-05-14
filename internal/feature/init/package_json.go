// Package initialise — package_json.go mutates the user's package.json to add
// @pbuilder/sdk to devDependencies.
//
// # Output 5 — package.json mutation (REQ-PM-01..04)
//
// # Algorithm
//
//  1. Build path = filepath.Join(req.Directory, "package.json")
//  2. ReadFile via FSWriter; if os.ErrNotExist → treat as empty object "{}"
//  3. Unmarshal into map[string]json.RawMessage; on JSON error → ErrCodeInvalidInput
//  4. Extract devDependencies (if present) → unmarshal into map[string]string
//  5. Check if @pbuilder/sdk already present; if YES leave unchanged (REQ-PM-02)
//  6. Otherwise add "@pbuilder/sdk": "^1.0.0" (REQ-PM-01)
//  7. Re-marshal inner devDeps map → store back in outer map
//  8. MarshalIndent with 2-space prefix + final newline (REQ-PM-03)
//  9. WriteFile (atomic via FSWriter — REQ-PM-04)
//
// 10. Return absolute path
//
// # Field ordering
//
// The outer map[string]json.RawMessage produces alphabetically sorted keys
// (Go 1.12+ sorts map keys during JSON marshal). This is deterministic and
// diff-stable. Formatting preservation was explicitly rejected in the locked
// policy (see explore obs #234 "package.json mutation policy").
//
// # No direct os.* calls (FF-init-02)
//
// All I/O goes through the FSWriter port.
package initialise

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// sdkPackageName is the npm package name for the Project Builder SDK.
// Locked post-v1.0.0 — changing it breaks all existing users' package.json files.
const sdkPackageName = "@pbuilder/sdk"

// sdkVersionRange is the exact version range added to devDependencies (REQ-PM-01).
// Locked literal: never "latest", ">=1.0.0", or bare "1.0.0".
const sdkVersionRange = "^1.0.0"

// mutatePackageJSON reads the project's package.json (or creates a minimal one if
// absent), adds @pbuilder/sdk to devDependencies if not already present, and
// writes the result back atomically via FSWriter.
//
// Returns the absolute path of the written file on success.
//
// Error cases:
//   - File exists but contains malformed JSON → ErrCodeInvalidInput (file NOT written)
//   - Any FSWriter I/O error → propagated as-is
func mutatePackageJSON(fs FSWriter, req InitRequest) (string, error) {
	pkgPath := filepath.Join(req.Directory, "package.json")

	// Step 1: Read existing content, or start from an empty object.
	raw, err := fs.ReadFile(pkgPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// Unexpected read error — propagate.
			return "", err
		}
		// File does not exist → create minimal package.json (REQ-PM-01).
		raw = []byte("{}")
	}

	// Step 2: Unmarshal into a generic map so that all top-level keys (known
	// and unknown) are preserved. Go's encoding/json marshals map[string]X with
	// alphabetically sorted keys (Go 1.12+), giving deterministic output even
	// though the original ordering may differ. Formatting preservation is OUT
	// of scope (policy: reformat unconditionally, REQ-PM-03).
	var outer map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(raw, &outer); unmarshalErr != nil {
		return "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.mutatePackageJSON",
			Message: pkgPath + " is not valid JSON: " + unmarshalErr.Error(),
			Cause:   unmarshalErr,
			Suggestions: []string{
				"fix the syntax errors in package.json and re-run",
				"use --dry-run to preview what builder init would write",
			},
		}
	}
	if outer == nil {
		outer = make(map[string]json.RawMessage)
	}

	// Step 3: Extract and unmarshal devDependencies (if present).
	devDeps := make(map[string]string)
	if raw, ok := outer["devDependencies"]; ok {
		if unmarshalErr := json.Unmarshal(raw, &devDeps); unmarshalErr != nil {
			// devDependencies exists but is not a string map — treat as invalid input.
			return "", &errs.Error{
				Code:    errs.ErrCodeInvalidInput,
				Op:      "init.mutatePackageJSON",
				Message: pkgPath + " is not valid JSON: devDependencies is not a string map: " + unmarshalErr.Error(),
				Cause:   unmarshalErr,
				Suggestions: []string{
					"fix the devDependencies field in package.json and re-run",
				},
			}
		}
	}

	// Step 4: Add @pbuilder/sdk if not already present (REQ-PM-02 — additive only).
	if _, exists := devDeps[sdkPackageName]; !exists {
		devDeps[sdkPackageName] = sdkVersionRange
	}

	// Step 5: Re-marshal devDeps and store back into the outer map.
	devDepsRaw, marshalErr := json.Marshal(devDeps)
	if marshalErr != nil {
		// This should not occur for map[string]string, but handle defensively.
		return "", marshalErr
	}
	outer["devDependencies"] = json.RawMessage(devDepsRaw)

	// Step 6: Marshal the full outer map with 2-space indentation (REQ-PM-03).
	// SetEscapeHTML(false) preserves characters like '>' and '<' as-is, which
	// is correct for package.json version ranges (e.g. ">=18"). The default
	// encoding/json behaviour of escaping these as > etc. is undesirable
	// in a human-readable file.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(outer); encErr != nil {
		return "", encErr
	}
	// json.Encoder.Encode always appends a trailing newline — no explicit append needed.
	indented := buf.Bytes()

	// Step 7: Ensure the parent directory exists, then write atomically (REQ-PM-04).
	if err := fs.MkdirAll(filepath.Dir(pkgPath), 0o755); err != nil {
		return "", err
	}
	if err := fs.WriteFile(pkgPath, indented, 0o644); err != nil {
		return "", err
	}

	return pkgPath, nil
}
