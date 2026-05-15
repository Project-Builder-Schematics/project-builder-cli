// Package newfeature — render.go contains result-rendering helpers used by
// handler RunE closures.
//
// ADR-019: ALL user-facing output goes through Renderer / render helpers;
// NEVER fmt.Println / fmt.Fprintf direct in handlers.
// L-builder-init-03: use json.NewEncoder + SetEscapeHTML(false); NEVER MarshalIndent.
package newfeature

import (
	"encoding/json"
	"fmt"
	"io"
)

// InlineSchematicThreshold is the count at which a soft warning is emitted
// for inline schematics in a collection (REQ-NSI-04).
// Used by registerSchematicInline to decide whether to call WarnApproachingSchematicLimit.
const InlineSchematicThreshold = 10

// FileSizeThresholdBytes is the project-builder.json byte size at which a soft
// warning is emitted (REQ-NSI-05). 20 KB = 20 * 1024.
// Used by registerSchematicInline to decide whether to call WarnApproachingFileSize.
const FileSizeThresholdBytes = 20 * 1024

// WarnApproachingSchematicLimit returns a soft warning message when the inline
// schematic count in a collection reaches or exceeds the threshold (REQ-NSI-04).
//
// The warning is routed through NewResult.Warnings → RenderPretty (ADR-019).
// The caller (registerSchematicInline) calls this AFTER the write, appending
// the result to NewResult.Warnings if the count is at or above the threshold.
func WarnApproachingSchematicLimit(collection string, count int) string {
	return fmt.Sprintf(
		"collection '%s' now has %d inline schematics; consider --path mode for large collections",
		collection, count,
	)
}

// WarnApproachingFileSize returns a soft warning message when project-builder.json
// exceeds the 20KB size limit after an inline write (REQ-NSI-05).
//
// bytes is the size of the file after write.
func WarnApproachingFileSize(bytes int) string {
	kb := bytes / 1024
	return fmt.Sprintf(
		"project-builder.json is now >%dKB; consider migrating schematics to path mode",
		kb,
	)
}

// RenderPretty writes a human-readable representation of result to w.
// Dry-run mode emits a "DRY RUN" header followed by planned operations.
// Real-write mode emits created files. Warnings are always emitted last.
//
// ADR-019: ALL user-facing output goes through Renderer — never fmt.Println.
// fmt.Fprintf/Fprintln errors are intentionally discarded — a failing write
// would surface elsewhere and we don't want it to mask the service outcome.
func RenderPretty(w io.Writer, result NewResult) {
	if result.DryRun {
		_, _ = fmt.Fprintln(w, "DRY RUN — no files written")
		_, _ = fmt.Fprintln(w)
		for _, op := range result.PlannedOps {
			switch op.Op {
			case "create_file":
				_, _ = fmt.Fprintf(w, "  Would create: %s\n", op.Path)
			default:
				_, _ = fmt.Fprintf(w, "  Would %s: %s\n", op.Op, op.Path)
			}
		}
	} else {
		// Real-write mode — list created files.
		for _, p := range result.FilesCreated {
			_, _ = fmt.Fprintf(w, "  Created: %s\n", p)
		}
	}

	// Warnings are always rendered, regardless of dry-run mode (ADR-019).
	for _, warn := range result.Warnings {
		_, _ = fmt.Fprintf(w, "warning: %s\n", warn)
	}
}

// RenderJSON encodes result as a single JSON object to w.
// Uses SetEscapeHTML(false) per L-builder-init-03 to avoid mangling paths
// and names that contain angle brackets or other HTML-special characters.
func RenderJSON(w io.Writer, result NewResult) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}
