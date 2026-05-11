// Package runner holds the embedded Node.js runner script.
//
// The runner script is embedded at compile time via //go:embed; it is NEVER
// read from the filesystem at runtime. The adapter writes the embedded bytes
// to a temp file, executes it, and deletes the file after cmd.Wait() returns.
//
// This prevents post-installation tampering with the runner (ADR-02).
package runner

import _ "embed"

// Script holds the embedded runner.js bytes.
// Non-empty after compile (REQ-19.1). The adapter writes these bytes to a
// temp file before executing Node.js; the temp file is deleted after exit (REQ-19.2).
//
//go:embed runner.js
var Script []byte
