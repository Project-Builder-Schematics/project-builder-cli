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

// RenderPretty writes a human-readable representation of result to w.
// Dry-run mode emits a "DRY RUN" header followed by planned operations.
// Real-write mode emits created files (stub: empty in S-000b).
//
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
		return
	}

	// Real-write mode — list created files (populated in S-001/S-004).
	for _, p := range result.FilesCreated {
		_, _ = fmt.Fprintf(w, "  Created: %s\n", p)
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
