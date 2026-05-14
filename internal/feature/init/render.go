// Package initialise — render.go contains the result-rendering helper used by
// handler.RunE. Pretty mode emits human-readable text; JSON mode emits NDJSON.
//
// REQ-JO-01: --json selects the JSON renderer.
// REQ-DR-03: dry-run pretty output begins with "DRY RUN — no files written".
// REQ-MCP-02: MCP instructions are printed after install when MCP=yes (real mode).
package initialise

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// renderResult writes the InitResult to the command's output in the requested format.
// JSON mode emits a single NDJSON line; pretty mode emits human-readable text.
//
// fmt.Fprintf/Fprintln errors are intentionally discarded — pretty output goes to
// the renderer (stdout by default) and a failing write would surface elsewhere; we
// don't want a write error to mask the real Service.Init outcome.
func renderResult(cmd *cobra.Command, result InitResult, jsonOut bool) error {
	out := cmd.OutOrStdout()
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		return enc.Encode(result)
	}

	if result.DryRun {
		_, _ = fmt.Fprintln(out, "DRY RUN — no files written")
		_, _ = fmt.Fprintln(out)
		for _, op := range result.PlannedOps {
			switch op.Op {
			case "create_file":
				_, _ = fmt.Fprintf(out, "  Would create: %s\n", op.Path)
			case "append_marker":
				_, _ = fmt.Fprintf(out, "  Would append: %s\n", op.Path)
			case "modify_devdep":
				_, _ = fmt.Fprintf(out, "  Would modify: %s (%s)\n", op.Path, op.Details)
			case "install_package":
				_, _ = fmt.Fprintf(out, "  Would install: %s\n", op.Details)
			case "mcp_setup_offered":
				_, _ = fmt.Fprintln(out, "  Would offer:   MCP server setup instructions")
			}
		}
		return nil
	}

	// Real-write mode — list created files.
	_, _ = fmt.Fprintf(out, "Initialising Project Builder workspace in %s ...\n\n", result.Directory)
	for _, p := range result.OutputsCreated {
		_, _ = fmt.Fprintf(out, "  Created: %s\n", p)
	}
	if result.Installed {
		_, _ = fmt.Fprintf(out, "\nInstalling @pbuilder/sdk via %s ... done.\n", result.PackageManager)
	}
	_, _ = fmt.Fprintln(out, "\nProject Builder is ready. Try: builder add <name>")
	if result.MCPSetupOffered {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, mcpInstructions)
	}
	return nil
}
