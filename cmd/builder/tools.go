//go:build tools

// Package main anchors module dependencies in go.mod for slices S-001..S-003,
// where Cobra, Viper, and charmbracelet/log gain real call sites. This file
// is excluded from the production build via the `tools` build tag; `go mod
// tidy` still sees the imports and keeps the entries in go.mod.
package main

import (
	_ "github.com/charmbracelet/log"
	_ "github.com/spf13/cobra"
	_ "github.com/spf13/viper"
)
