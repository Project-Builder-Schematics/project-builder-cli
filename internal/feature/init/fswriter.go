// Package initialise — fswriter.go re-exports NewOSWriter from the shared
// fswriter package so that cmd/builder/main.go (composeApp) can continue
// calling initialise.NewOSWriter() without changes in this slice.
//
// S-000b updates composeApp to call fswriter.NewOSWriter() directly, after
// which this file is deleted.
//
// ADR-020: all filesystem I/O in the init feature goes through FSWriter
// (now promoted to internal/shared/fswriter).
package initialise

import "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"

// NewOSWriter is the exported constructor used by composeApp.
// Delegates to the shared fswriter package (promoted in S-000a).
// This alias will be removed when composeApp is updated in S-000b.
func NewOSWriter() FSWriter { return fswriter.NewOSWriter() }
