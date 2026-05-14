// Package template_test covers the compile-time //go:embed guarantee for SKILL.md.
//
// REQ coverage:
//   - REQ-SA-01 (SKILL.md bundled via //go:embed — non-empty, valid YAML frontmatter)
//   - FF-init-04 (golden bytes — any drift in the embed surfaces at test time)
package template_test

import (
	"bytes"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init/template"
)

// lockedSkillBytes is the verbatim content of the locked v0 SKILL.md placeholder.
// This golden literal is the source of truth for FF-init-04: if the embedded
// bytes change without a deliberate spec amendment, this test fails.
//
// IMPORTANT: keep this literal byte-for-byte identical to
// internal/feature/init/template/SKILL.md. Any divergence is a bug.
var lockedSkillBytes = []byte("---\n" +
	"name: pbuilder\n" +
	"description: AI agent skill for Project Builder CLI (preview)\n" +
	"---\n" +
	"\n" +
	"# Project Builder — AI Skill (v0 / preview)\n" +
	"\n" +
	"This is a placeholder skill artefact bundled with `builder init` in v1.0.\n" +
	"Full content design (decision heuristics, examples, when-to-formalise rules)\n" +
	"is tracked at: https://github.com/Project-Builder-Schematics/project-builder-cli (roadmap row 13)\n" +
	"\n" +
	"## CLI Operations (current command inventory)\n" +
	"\n" +
	"- `builder init` — initialise a Project Builder workspace\n" +
	"- `builder execute` (alias: `e`, `generate`, `g`) — run a schematic\n" +
	"- `builder add` — scaffold a new local schematic\n" +
	"- `builder info` — inspect a collection or schematic\n" +
	"- `builder sync` — fetch declared remote collections\n" +
	"- `builder validate` — lint mode for schematics\n" +
	"- `builder remove` — remove a local schematic\n" +
	"- `builder skill update` — regenerate this skill when the CLI version changes\n" +
	"\n" +
	"## Decision Heuristics\n" +
	"\n" +
	"TODO — content design deferred to roadmap row 13.\n" +
	"\n" +
	"## Update\n" +
	"\n" +
	"When the CLI version changes, run `builder skill update` to refresh this file.\n")

// Test_Template_Skill_NonEmpty verifies that the embedded Skill bytes are
// non-empty after compilation (supply-chain audit: the embed must not silently
// produce an empty slice). REQ-SA-01.
func Test_Template_Skill_NonEmpty(t *testing.T) {
	t.Parallel()

	if len(template.Skill) == 0 {
		t.Errorf("template.Skill is empty — //go:embed SKILL.md produced no bytes")
	}
}

// Test_Template_Skill_HasYAMLFrontmatter verifies the embedded content begins
// with "---\n", confirming the YAML frontmatter block is present (REQ-SA-01).
func Test_Template_Skill_HasYAMLFrontmatter(t *testing.T) {
	t.Parallel()

	const want = "---\n"
	if !bytes.HasPrefix(template.Skill, []byte(want)) {
		t.Errorf("template.Skill does not start with YAML frontmatter %q; got prefix: %q",
			want, firstN(template.Skill, 10))
	}
}

// Test_Template_Skill_GoldenBytes verifies the embedded bytes match the locked
// v0 placeholder exactly (FF-init-04). Any unintentional change to SKILL.md
// will surface here as a diff between the embed and this golden literal.
func Test_Template_Skill_GoldenBytes(t *testing.T) {
	t.Parallel()

	if !bytes.Equal(template.Skill, lockedSkillBytes) {
		t.Errorf("template.Skill bytes differ from locked golden literal (FF-init-04).\n"+
			"got  len=%d\nwant len=%d\n\n"+
			"First differing byte at index: %d",
			len(template.Skill), len(lockedSkillBytes),
			firstDiff(template.Skill, lockedSkillBytes))
	}
}

// firstN returns the first n bytes of b as a string (for error messages).
func firstN(b []byte, n int) string {
	if len(b) < n {
		return string(b)
	}
	return string(b[:n])
}

// firstDiff returns the index of the first byte that differs between a and b,
// or the length of the shorter slice if all matching bytes are equal.
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
