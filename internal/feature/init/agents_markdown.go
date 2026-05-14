// Package initialise — agents_markdown.go appends the locked pbuilder
// skill-reference marker block to AGENTS.md (preferred) or CLAUDE.md.
//
// # Output 4 — AGENTS.md / CLAUDE.md marker (REQ-AR-01..05, ADR-021)
//
// # Locked marker block (REQ-AR-01, durable post-v1.0.0 contract)
//
// The marker text is immutable once v1.0.0 ships. Changing it breaks
// idempotency for all existing users (they get a second block on re-run).
//
// # File selection precedence (REQ-AR-03)
//
//  1. Both AGENTS.md and CLAUDE.md exist → write to AGENTS.md (preferred)
//  2. Only AGENTS.md exists → write to it
//  3. Only CLAUDE.md exists → write to it
//  4. Neither exists → create AGENTS.md (default)
//
// # Idempotency (REQ-AR-02) — line-exact
//
// The check splits the target file on "\n" and looks for a line whose
// trimmed-right content equals exactly "<!-- pbuilder:skill:begin -->".
// Substring matches (e.g. "code: <!-- pbuilder:skill:begin --> bar") do NOT
// trigger the skip — adversarial defence.
//
// # Ambiguity (REQ-AR-04)
//
// When BOTH files already contain the marker (line-exact), return
// ErrCodeInitAgentFileAmbiguous unless force=true. With force=true, append a
// second copy to AGENTS.md.
//
// # Symlink safety (REQ-AR-05)
//
// Before reading/writing a candidate file, its path is passed through
// FSWriter.Lstat + FSWriter.EvalSymlinks. If the resolved target lies outside
// projectDir, the function rejects with ErrCodeInvalidInput.
//
// # No direct os.* calls (FF-init-02)
//
// All I/O goes through the FSWriter port.
package initialise

import (
	"os"
	"path/filepath"
	"strings"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// markerBeginLine is the exact line that triggers idempotency skip (REQ-AR-02).
// Must remain stable post-v1.0.0.
const markerBeginLine = "<!-- pbuilder:skill:begin -->"

// agentMarkerBlock is the full locked marker block (REQ-AR-01).
// Changing this string post-v1.0.0 breaks idempotency for existing users.
const agentMarkerBlock = "<!-- pbuilder:skill:begin -->\n## Project Builder Skill\n\nThis project uses [Project Builder](https://github.com/Project-Builder-Schematics/project-builder-cli).\nLoad the skill at `.claude/skills/pbuilder/SKILL.md` for command reference and authoring heuristics.\n<!-- pbuilder:skill:end -->\n"

// appendAgentsMarker appends the locked pbuilder skill-reference marker block
// to the appropriate agent file (AGENTS.md preferred; CLAUDE.md fallback;
// create AGENTS.md if neither exists).
//
// Returns the absolute path of the file written (or skipped for idempotency)
// and nil on success. Returns a structured *errs.Error on any failure.
//
// Parameters:
//   - projectDir: absolute path of the project root (req.Directory)
//   - force: when true, bypasses the double-marker ambiguity guard (REQ-AR-04)
//   - fs: the FSWriter port (all I/O goes through this; FF-init-02)
//
// Signature matches the call site in service.go output-4.
func appendAgentsMarker(projectDir string, force bool, fs FSWriter) (string, error) {
	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	claudePath := filepath.Join(projectDir, "CLAUDE.md")

	// Symlink safety for both candidate paths (REQ-AR-05) — check both regardless
	// of existence so an adversarial symlink for either path is caught early.
	for _, candidate := range []string{agentsPath, claudePath} {
		if err := checkSymlinkSafety(candidate, projectDir, fs); err != nil {
			return "", err
		}
	}

	state, err := readAgentFileState(fs, agentsPath, claudePath)
	if err != nil {
		return "", err
	}

	// Ambiguity check (REQ-AR-04): both files marked → error unless --force.
	if state.bothMarkered() {
		if !force {
			return "", ambiguousAgentFilesError()
		}
		// force=true: bypass idempotency, append second copy to AGENTS.md.
		return agentsPath, appendMarkerToFile(fs, agentsPath)
	}

	target := selectMarkerTarget(agentsPath, claudePath, state)

	// Idempotency (REQ-AR-02): target already markered → no-op.
	if (target == agentsPath && state.agentsHasMarker) ||
		(target == claudePath && state.claudeHasMarker) {
		return target, nil
	}

	return target, appendMarkerToFile(fs, target)
}

// agentFileState captures the existence + marker presence of AGENTS.md and
// CLAUDE.md in a single read pass. Pure data — no I/O.
type agentFileState struct {
	agentsExists    bool
	claudeExists    bool
	agentsHasMarker bool
	claudeHasMarker bool
}

// bothMarkered reports the REQ-AR-04 ambiguity condition.
func (s agentFileState) bothMarkered() bool {
	return s.agentsExists && s.claudeExists && s.agentsHasMarker && s.claudeHasMarker
}

// readAgentFileState stats both candidates and reads marker presence in one pass.
// Read failures (file exists but cannot be opened) surface as ErrCodeInvalidInput.
func readAgentFileState(fs FSWriter, agentsPath, claudePath string) (agentFileState, error) {
	state := agentFileState{
		agentsExists: fileExists(fs, agentsPath),
		claudeExists: fileExists(fs, claudePath),
	}

	if state.agentsExists {
		markered, err := readMarkerPresence(fs, agentsPath, "AGENTS.md")
		if err != nil {
			return state, err
		}
		state.agentsHasMarker = markered
	}

	if state.claudeExists {
		markered, err := readMarkerPresence(fs, claudePath, "CLAUDE.md")
		if err != nil {
			return state, err
		}
		state.claudeHasMarker = markered
	}

	return state, nil
}

// readMarkerPresence reads path and returns true iff the locked marker
// begin-line appears (REQ-AR-02 line-exact match).
func readMarkerPresence(fs FSWriter, path, label string) (bool, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return false, &errs.Error{
			Code:        errs.ErrCodeInvalidInput,
			Op:          "init.appendAgentsMarker",
			Message:     "failed to read " + label + ": " + err.Error(),
			Cause:       err,
			Suggestions: []string{"check file permissions for " + label},
		}
	}
	return containsMarkerLine(string(data)), nil
}

// ambiguousAgentFilesError builds the REQ-AR-04 error for both files markered.
func ambiguousAgentFilesError() error {
	return &errs.Error{
		Code:    errs.ErrCodeInitAgentFileAmbiguous,
		Op:      "init.appendAgentsMarker",
		Message: "both AGENTS.md and CLAUDE.md already contain the pbuilder skill marker — ambiguous which file to update",
		Suggestions: []string{
			"run with --force to append a second copy to AGENTS.md",
			"remove the marker from CLAUDE.md manually and re-run",
			"remove the marker from AGENTS.md manually and re-run",
		},
	}
}

// selectMarkerTarget implements REQ-AR-03 selection precedence:
//  1. AGENTS.md preferred when both exist
//  2. only AGENTS.md → AGENTS.md
//  3. only CLAUDE.md → CLAUDE.md
//  4. neither exists → create AGENTS.md (default)
//
// Pure logic — no I/O — so coverage by table-driven tests is straightforward.
func selectMarkerTarget(agentsPath, claudePath string, s agentFileState) string {
	if !s.agentsExists && s.claudeExists {
		return claudePath
	}
	// All other cases (agents exists, or neither exists) → AGENTS.md.
	return agentsPath
}

// appendMarkerToFile reads the current content of target (empty if not
// exists), prepends a blank-line separator when the file is non-empty, and
// then writes the combined content back atomically via FSWriter.WriteFile.
//
// Semantics (REQ-AR-01):
//   - non-empty target → "\n" + agentMarkerBlock appended
//   - empty / new target → agentMarkerBlock only (no leading blank)
func appendMarkerToFile(fs FSWriter, target string) error {
	// Read existing content (may be empty or non-existent).
	existing, err := fs.ReadFile(target)
	if err != nil {
		// File does not yet exist — start with empty content.
		existing = []byte{}
	}

	var newContent []byte
	if len(existing) > 0 {
		// Non-empty file: prepend one blank line separator.
		newContent = append(existing, '\n')
		newContent = append(newContent, []byte(agentMarkerBlock)...)
	} else {
		// Empty or new file: write marker only.
		newContent = []byte(agentMarkerBlock)
	}

	// Atomic write via FSWriter port (FF-init-02).
	return fs.WriteFile(target, newContent, 0o644)
}

// containsMarkerLine returns true iff content contains a line whose
// right-trimmed value equals exactly markerBeginLine (REQ-AR-02 line-exact
// idempotency check). Substring matches (mid-line occurrences) return false.
func containsMarkerLine(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimRight(line, " \t\r") == markerBeginLine {
			return true
		}
	}
	return false
}

// fileExists returns true iff FSWriter.Stat succeeds for path.
func fileExists(fs FSWriter, path string) bool {
	_, err := fs.Stat(path)
	return err == nil
}

// checkSymlinkSafety resolves symlinks on path using the FSWriter port.
// If path does not exist (os.ErrNotExist), no check is needed — return nil.
// If path is a symlink whose resolved target lies outside projectDir,
// return ErrCodeInvalidInput. (REQ-AR-05)
func checkSymlinkSafety(path, projectDir string, fs FSWriter) error {
	// Use Lstat to check whether path itself is a symlink.
	info, err := fs.Lstat(path)
	if err != nil {
		// Path does not exist or cannot be stat'd — no symlink to check.
		return nil
	}

	if info.Mode()&os.ModeSymlink == 0 {
		// Regular file or directory — not a symlink.
		return nil
	}

	// Path is a symlink — resolve it.
	resolved, err := fs.EvalSymlinks(path)
	if err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.appendAgentsMarker",
			Message: "failed to resolve symlink for " + path + ": " + err.Error(),
			Cause:   err,
			Suggestions: []string{
				"ensure AGENTS.md / CLAUDE.md are not broken symlinks",
			},
		}
	}

	// Verify resolved target is inside projectDir.
	// filepath.Rel returns a path without ".." iff resolved is inside projectDir.
	rel, err := filepath.Rel(projectDir, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.appendAgentsMarker",
			Message: path + " is a symlink pointing outside the project directory (" + resolved + ") — rejected for security",
			Suggestions: []string{
				"ensure AGENTS.md and CLAUDE.md are regular files within the project directory",
				"if intentional, remove the symlink and replace with a regular file",
			},
		}
	}

	return nil
}
