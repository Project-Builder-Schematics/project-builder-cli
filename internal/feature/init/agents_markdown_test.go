// Package initialise — agents_markdown_test.go covers appendAgentsMarker.
//
// REQ coverage:
//   - REQ-AR-01 (locked marker block appended with correct bytes)
//   - REQ-AR-02 (line-exact idempotency; substring match does NOT trigger skip)
//   - REQ-AR-03 (file selection precedence: both→AGENTS.md, only AGENTS→AGENTS.md,
//     only CLAUDE→CLAUDE.md, neither→create AGENTS.md)
//   - REQ-AR-04 (both files already contain marker → ErrCodeInitAgentFileAmbiguous
//     unless --force; force appends second copy to AGENTS.md)
//   - REQ-AR-05 (symlink resolved outside project dir → ErrCodeInvalidInput)
package initialise

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// lockedMarkerBegin is the exact opening comment used in the marker block.
// Must match the durable post-v1.0.0 contract.
const lockedMarkerBegin = "<!-- pbuilder:skill:begin -->"

// expectedMarkerBlock is the full locked block that must be appended.
// REQ-AR-01.
const expectedMarkerBlock = "<!-- pbuilder:skill:begin -->\n## Project Builder Skill\n\nThis project uses [Project Builder](https://github.com/Project-Builder-Schematics/project-builder-cli).\nLoad the skill at `.claude/skills/pbuilder/SKILL.md` for command reference and authoring heuristics.\n<!-- pbuilder:skill:end -->\n"

// --- REQ-AR-01 + REQ-AR-03 (neither file exists → create AGENTS.md) ---

// Test_AppendAgentsMarker_WritesLockedBlock_WhenNeitherExists verifies that
// when neither AGENTS.md nor CLAUDE.md exists, appendAgentsMarker creates
// AGENTS.md with the exact locked marker block and no leading blank line
// (target is empty/new). REQ-AR-01, REQ-AR-03.
func Test_AppendAgentsMarker_WritesLockedBlock_WhenNeitherExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	targetPath, err := appendAgentsMarker(dir, false, ffs)
	if err != nil {
		t.Fatalf("appendAgentsMarker: unexpected error: %v", err)
	}

	wantPath := filepath.Join(dir, "AGENTS.md")
	if targetPath != wantPath {
		t.Errorf("returned path = %q, want %q", targetPath, wantPath)
	}

	got, readErr := ffs.ReadFile(wantPath)
	if readErr != nil {
		t.Fatalf("AGENTS.md not written: %v", readErr)
	}

	// When target is empty (new file), no leading blank line.
	if string(got) != expectedMarkerBlock {
		t.Errorf("AGENTS.md content mismatch\ngot:\n%q\nwant:\n%q", string(got), expectedMarkerBlock)
	}
}

// --- REQ-AR-03 (both exist → prefer AGENTS.md) ---

// Test_AppendAgentsMarker_WritesToAgents_WhenBothExist verifies that when
// both AGENTS.md and CLAUDE.md exist (neither contains the marker), the
// marker is appended to AGENTS.md and CLAUDE.md is untouched. REQ-AR-03.
func Test_AppendAgentsMarker_WritesToAgents_WhenBothExist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	claudePath := filepath.Join(dir, "CLAUDE.md")

	agentsContent := []byte("# Agents\n\nExisting content.\n")
	claudeContent := []byte("# Claude\n\nExisting content.\n")

	_ = ffs.WriteFile(agentsPath, agentsContent, 0o644)
	_ = ffs.WriteFile(claudePath, claudeContent, 0o644)

	targetPath, err := appendAgentsMarker(dir, false, ffs)
	if err != nil {
		t.Fatalf("appendAgentsMarker: unexpected error: %v", err)
	}

	if targetPath != agentsPath {
		t.Errorf("returned path = %q, want %q (AGENTS.md preferred)", targetPath, agentsPath)
	}

	got, _ := ffs.ReadFile(agentsPath)
	if !strings.Contains(string(got), lockedMarkerBegin) {
		t.Errorf("AGENTS.md does not contain marker begin tag after append")
	}

	// CLAUDE.md must be untouched.
	gotClaude, _ := ffs.ReadFile(claudePath)
	if string(gotClaude) != string(claudeContent) {
		t.Errorf("CLAUDE.md was modified (should be untouched when AGENTS.md is preferred)")
	}
}

// --- REQ-AR-03 (only CLAUDE.md exists → write to CLAUDE.md) ---

// Test_AppendAgentsMarker_WritesToClaude_WhenOnlyClaudeExists verifies that
// when only CLAUDE.md exists, the marker is appended to it. REQ-AR-03.
func Test_AppendAgentsMarker_WritesToClaude_WhenOnlyClaudeExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	claudePath := filepath.Join(dir, "CLAUDE.md")
	claudeContent := []byte("# Claude config\n")
	_ = ffs.WriteFile(claudePath, claudeContent, 0o644)

	targetPath, err := appendAgentsMarker(dir, false, ffs)
	if err != nil {
		t.Fatalf("appendAgentsMarker: unexpected error: %v", err)
	}

	if targetPath != claudePath {
		t.Errorf("returned path = %q, want %q", targetPath, claudePath)
	}

	got, _ := ffs.ReadFile(claudePath)
	if !strings.Contains(string(got), lockedMarkerBegin) {
		t.Errorf("CLAUDE.md does not contain marker begin tag after append")
	}
}

// --- REQ-AR-01 (non-empty file gets blank line separator) ---

// Test_AppendAgentsMarker_NonEmptyFile_HasBlankLineSeparator verifies that
// when the target file is non-empty, a single blank line is prepended before
// the marker block. REQ-AR-01 append semantics.
func Test_AppendAgentsMarker_NonEmptyFile_HasBlankLineSeparator(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	existing := []byte("# Existing content\n")
	_ = ffs.WriteFile(agentsPath, existing, 0o644)

	_, err := appendAgentsMarker(dir, false, ffs)
	if err != nil {
		t.Fatalf("appendAgentsMarker: unexpected error: %v", err)
	}

	got, _ := ffs.ReadFile(agentsPath)
	// Expected: existing content + blank line + marker block.
	want := string(existing) + "\n" + expectedMarkerBlock
	if string(got) != want {
		t.Errorf("content after append mismatch\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

// --- REQ-AR-02 (line-exact idempotency) ---

// Test_AppendAgentsMarker_IsIdempotent_LineExact verifies that when the
// target file already contains the marker begin tag on its own line,
// appendAgentsMarker returns the path without appending again. REQ-AR-02.
func Test_AppendAgentsMarker_IsIdempotent_LineExact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	// Pre-seed file with the marker already present (locked block).
	existing := []byte("# Agents\n\n" + expectedMarkerBlock)
	_ = ffs.WriteFile(agentsPath, existing, 0o644)

	targetPath, err := appendAgentsMarker(dir, false, ffs)
	if err != nil {
		t.Fatalf("appendAgentsMarker: unexpected error on idempotent re-run: %v", err)
	}
	if targetPath != agentsPath {
		t.Errorf("idempotent: returned path = %q, want %q", targetPath, agentsPath)
	}

	// Content must be unchanged — no second marker appended.
	got, _ := ffs.ReadFile(agentsPath)
	if string(got) != string(existing) {
		t.Errorf("idempotent re-run modified file content\nbefore:\n%q\nafter:\n%q", string(existing), string(got))
	}
}

// --- REQ-AR-02 (adversarial mid-line marker — substring match must NOT skip) ---

// Test_AppendAgentsMarker_RejectsAdversarialMidLineMarker verifies that when
// AGENTS.md contains the marker begin tag embedded within a longer line (e.g.
// `code: <!-- pbuilder:skill:begin -->`), the idempotency check does NOT
// trigger — the marker is still appended. REQ-AR-02 adversarial defence.
func Test_AppendAgentsMarker_RejectsAdversarialMidLineMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	// Line contains the marker text but NOT as the full trimmed line.
	adversarialContent := []byte("# Agents\n\ncode: <!-- pbuilder:skill:begin --> example\n")
	_ = ffs.WriteFile(agentsPath, adversarialContent, 0o644)

	_, err := appendAgentsMarker(dir, false, ffs)
	if err != nil {
		t.Fatalf("appendAgentsMarker: unexpected error: %v", err)
	}

	got, _ := ffs.ReadFile(agentsPath)
	// The marker block MUST have been appended (adversarial line is not a match).
	if !strings.Contains(string(got), expectedMarkerBlock) {
		t.Errorf("adversarial mid-line marker: marker block was not appended\ncontent:\n%q", string(got))
	}
}

// --- REQ-AR-04 (both files already have marker → ErrCodeInitAgentFileAmbiguous) ---

// Test_AppendAgentsMarker_BothMarkered_ReturnsErrAmbiguous_NoForce verifies
// that when both AGENTS.md and CLAUDE.md already contain the marker (line-exact)
// and force=false, appendAgentsMarker returns ErrCodeInitAgentFileAmbiguous.
// REQ-AR-04.
func Test_AppendAgentsMarker_BothMarkered_ReturnsErrAmbiguous_NoForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	claudePath := filepath.Join(dir, "CLAUDE.md")

	withMarker := []byte("# Existing\n\n" + expectedMarkerBlock)
	_ = ffs.WriteFile(agentsPath, withMarker, 0o644)
	_ = ffs.WriteFile(claudePath, withMarker, 0o644)

	_, err := appendAgentsMarker(dir, false, ffs)
	if err == nil {
		t.Fatal("expected ErrCodeInitAgentFileAmbiguous, got nil")
	}

	if !isErrCode(err, errs.ErrCodeInitAgentFileAmbiguous) {
		t.Errorf("expected ErrCodeInitAgentFileAmbiguous; got: %v", err)
	}
}

// Test_AppendAgentsMarker_BothMarkered_WithForce_AppendsToAgents verifies
// that when both files already contain the marker and force=true, a second
// copy is appended to AGENTS.md. CLAUDE.md is untouched. REQ-AR-04.
func Test_AppendAgentsMarker_BothMarkered_WithForce_AppendsToAgents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	claudePath := filepath.Join(dir, "CLAUDE.md")

	withMarker := []byte("# Existing\n\n" + expectedMarkerBlock)
	_ = ffs.WriteFile(agentsPath, withMarker, 0o644)
	_ = ffs.WriteFile(claudePath, withMarker, 0o644)

	targetPath, err := appendAgentsMarker(dir, true, ffs) // force=true
	if err != nil {
		t.Fatalf("appendAgentsMarker with force: unexpected error: %v", err)
	}

	if targetPath != agentsPath {
		t.Errorf("force: returned path = %q, want %q", targetPath, agentsPath)
	}

	got, _ := ffs.ReadFile(agentsPath)
	// Count occurrences of the marker begin tag — must be 2.
	count := strings.Count(string(got), lockedMarkerBegin)
	if count != 2 {
		t.Errorf("force: expected 2 occurrences of marker in AGENTS.md, got %d\ncontent:\n%q", count, string(got))
	}

	// CLAUDE.md must be unchanged.
	gotClaude, _ := ffs.ReadFile(claudePath)
	if string(gotClaude) != string(withMarker) {
		t.Errorf("force: CLAUDE.md was modified (should be untouched)")
	}
}

// --- REQ-AR-05 (symlink resolved outside project dir → ErrCodeInvalidInput) ---

// Test_AppendAgentsMarker_SymlinkOutOfProject_Rejected verifies that when
// AGENTS.md is a symlink whose resolved target lies outside the project
// directory, appendAgentsMarker returns ErrCodeInvalidInput. REQ-AR-05.
func Test_AppendAgentsMarker_SymlinkOutOfProject_Rejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	// Register AGENTS.md as a symlink pointing outside the project directory.
	outsideTarget := "/etc/hosts"
	ffs.addSymlink(agentsPath, outsideTarget)

	_, err := appendAgentsMarker(dir, false, ffs)
	if err == nil {
		t.Fatal("expected ErrCodeInvalidInput for out-of-project symlink, got nil")
	}

	if !isErrCode(err, errs.ErrCodeInvalidInput) {
		t.Errorf("expected ErrCodeInvalidInput for symlink rejection; got: %v", err)
	}
}

// --- helper ---

// isErrCode reports whether err (or any in its chain) is an *errs.Error with
// the given code. Used to avoid importing errors.Is sentinel construction in
// tests.
func isErrCode(err error, code errs.ErrCode) bool {
	var e *errs.Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Code == code
}
