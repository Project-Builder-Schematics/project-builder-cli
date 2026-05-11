//go:build tools

// Package main anchors module dependencies that do not yet have call sites
// in production code. Without these blank imports, `go mod tidy` would remove
// them from go.mod.
//
// Cobra is now imported directly by main.go and does NOT need anchoring here.
//
// Remaining anchors:
//   - charmbracelet/log: structured logger, wired at /plan #3 (pretty renderer)
//   - viper: config-file + env-var loading, wired at /plan #3 when Config gains fields
package main

import (
	_ "github.com/charmbracelet/log"
	_ "github.com/spf13/viper"
)
