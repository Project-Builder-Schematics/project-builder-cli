// Package initialise — render.go contains the result-rendering helper used by
// handler.RunE. Pretty mode emits via output.Output; JSON mode encodes to a
// dedicated io.Writer (bypassing Output — JSON is structured data, not chrome).
//
// REQ-JO-01: --json selects the JSON renderer.
// REQ-DR-03: dry-run pretty output begins with "DRY RUN — no files written".
// REQ-MCP-02: MCP instructions are printed after install when MCP=yes (real mode).
// output-discipline/REQ-03.1: no direct fmt.Print* in internal/feature/* (FF-25).
package initialise

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
)

// renderResult writes the InitResult to the appropriate sink.
//
// JSON mode (jsonOut=true): encodes result as NDJSON to jsonWriter; does NOT
// invoke any Output methods. JSON is structured data, not user-facing chrome.
//
// Pretty mode (jsonOut=false): emits styled output via out.
// Dry-run → Heading + Newline + Body lines per planned op.
// Real-write → Heading(directory) + Path per created file + optional Success/Hint/Body.
//
// fmt.Fprintf/Fprintln errors are intentionally discarded in JSON mode —
// a failing write would surface elsewhere and we don't want it to mask the
// real Service.Init outcome.
func renderResult(out output.Output, jsonWriter io.Writer, result InitResult, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(jsonWriter)
		enc.SetEscapeHTML(false)
		return enc.Encode(result)
	}

	if result.DryRun {
		out.Heading("DRY RUN — no files written")
		out.Newline()
		for _, op := range result.PlannedOps {
			switch op.Op {
			case "create_file":
				out.Body(fmt.Sprintf("  Would create: %s", op.Path))
			case "append_marker":
				out.Body(fmt.Sprintf("  Would append: %s", op.Path))
			case "modify_devdep":
				out.Body(fmt.Sprintf("  Would modify: %s (%s)", op.Path, op.Details))
			case "install_package":
				out.Body(fmt.Sprintf("  Would install: %s", op.Details))
			case "mcp_setup_offered":
				out.Body("  Would offer:   MCP server setup instructions")
			}
		}
		return nil
	}

	// Real-write mode — announce directory, list created files, then results.
	out.Heading(fmt.Sprintf("Initialising Project Builder workspace in %s ...", result.Directory))
	for _, p := range result.OutputsCreated {
		out.Path(p)
	}
	if result.Installed {
		out.Success(fmt.Sprintf("Installing @pbuilder/sdk via %s ... done.", result.PackageManager))
	}
	out.Success("Project Builder is ready.")
	out.Hint("Try: builder add <name>")
	if result.MCPSetupOffered {
		out.Newline()
		out.Body(mcpInstructions)
	}
	return nil
}
