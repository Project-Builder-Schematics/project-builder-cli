// Package newfeature — render.go contains result-rendering helpers used by
// handler RunE closures.
//
// ADR-019: ALL user-facing output goes through Output port; NEVER fmt.Println
// or fmt.Fprintf directly in handlers (FF-25 gate).
// L-builder-init-03: use json.NewEncoder + SetEscapeHTML(false); NEVER MarshalIndent.
//
// S-004: RenderPretty migrated from io.Writer to output.Output (ADR-03/ADR-04).
// output-discipline/REQ-03.1: no fmt.Print* in internal/feature/* production code.
package newfeature

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
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

// RenderPretty writes a human-readable representation of result via out.
// Dry-run mode emits a "DRY RUN" header followed by planned operations.
// Real-write mode emits created files as Path calls. Warnings are emitted last.
//
// ADR-019: ALL user-facing output goes through Output — never fmt.Println.
// ADR-04: handler tests inject outputtest.Spy and assert (method, args) only.
func RenderPretty(out output.Output, result NewResult) {
	if result.DryRun {
		out.Heading("DRY RUN — no files written")
		out.Newline()
		for _, op := range result.PlannedOps {
			switch op.Op {
			case "create_file":
				out.Body(fmt.Sprintf("  Would create: %s", op.Path))
			default:
				out.Body(fmt.Sprintf("  Would %s: %s", op.Op, op.Path))
			}
		}
	} else {
		// Real-write mode — list created files.
		for _, p := range result.FilesCreated {
			out.Path(p)
		}
	}

	// Warnings are always rendered, regardless of dry-run mode (ADR-019).
	for _, warn := range result.Warnings {
		out.Warning(warn)
	}
}

// RenderJSON encodes result as a single JSON object to w.
// Uses SetEscapeHTML(false) per L-builder-init-03 to avoid mangling paths
// and names that contain angle brackets or other HTML-special characters.
// JSON is structured data — it bypasses output.Output entirely.
func RenderJSON(w io.Writer, result NewResult) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}
