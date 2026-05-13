// Package template holds the embedded SKILL.md bytes for the init feature.
//
// The SKILL.md file is embedded at compile time via //go:embed; it is NEVER
// read from the filesystem at runtime. The init service writes the embedded
// bytes verbatim to .claude/skills/pbuilder/SKILL.md in the user's project.
//
// This provides supply-chain auditability: the exact bytes committed to the
// repository are the bytes that land in the user's project. ADR-022 mirrors
// ADR-017 (runner.js pattern).
//
// FF-init-04: a golden-bytes test in template_test.go enforces byte-for-byte
// stability of the locked v0 placeholder. Any unintentional change to SKILL.md
// causes that test to fail.
package template

import _ "embed"

// Skill holds the locked v0 SKILL.md placeholder bytes bundled at compile time.
// Non-empty after compilation (REQ-SA-01, FF-init-04).
//
//go:embed SKILL.md
var Skill []byte
