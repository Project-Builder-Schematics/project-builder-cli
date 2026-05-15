// Package initialise — handler.go contains the RunE handler for `builder init`.
//
// Responsibilities (kept tight to respect ADR-011 handler ≤ 100 SLOC ceiling):
//  1. Parse flags and positional argument
//  2. Canonicalise the target directory (delegated to dir.canonicaliseDir)
//  3. Resolve --mcp mode (delegated to mcp.parseMCPFlag + mcp.defaultMCPMode)
//  4. Build InitRequest and call svc.Init
//  5. Render the result (delegated to render.renderResult)
//
// Helpers live in dedicated files within the same package:
//   - mcp.go     — parseMCPFlag, resolveMCPMode, defaultMCPMode, isStdinTTY
//   - dir.go     — canonicaliseDir
//   - render.go  — renderResult
package initialise

import (
	"os"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/pathutil"
)

// newRunE returns the RunE closure wired with svc for use by NewCommand.
func newRunE(svc *Service) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		force, _ := flags.GetBool("force")
		dryRun, _ := flags.GetBool("dry-run")
		jsonOut, _ := flags.GetBool("json")
		nonInteractive, _ := flags.GetBool("non-interactive")
		pmFlag, _ := flags.GetString("package-manager")
		noInstall, _ := flags.GetBool("no-install")
		noSkill, _ := flags.GetBool("no-skill")
		publishable, _ := flags.GetBool("publishable")
		mcpFlag, _ := flags.GetString("mcp")

		// Canonicalise directory: positional arg or cwd.
		var rawDir string
		if len(args) > 0 {
			rawDir = args[0]
		} else {
			var err error
			rawDir, err = os.Getwd()
			if err != nil {
				return &errs.Error{
					Code:    errs.ErrCodeInvalidInput,
					Op:      "init.handler",
					Message: "could not determine current working directory",
					Cause:   err,
				}
			}
		}

		dir, err := pathutil.Canonicalise(rawDir)
		if err != nil {
			return err
		}

		// Resolve MCP mode.
		var mcpMode MCPMode
		if flags.Changed("mcp") {
			parsed, parseErr := parseMCPFlag(mcpFlag)
			if parseErr != nil {
				return parseErr
			}
			if resolveErr := resolveMCPMode(string(parsed), nonInteractive); resolveErr != nil {
				return resolveErr
			}
			mcpMode = parsed
		} else {
			mcpMode = defaultMCPMode(nonInteractive, isStdinTTY())
		}

		// Resolve MCPPrompt: in non-dry-run mode, actually prompt the user
		// (REQ-MCP-01). Dry-run skips the prompt entirely (REQ-DR-01).
		if !dryRun && mcpMode == MCPPrompt {
			mcpMode = promptMCP(cmd.InOrStdin(), cmd.OutOrStdout())
		}

		// Swap to dryRunFS at request time so PlannedOps are collected fresh
		// per invocation without affecting the osFS-backed production service
		// (REQ-DR-01, ADR-020).
		activeSvc := svc
		if dryRun {
			activeSvc = NewService(fswriter.NewDryRunWriter(), svc.pm, svc.skill)
		}

		req := InitRequest{
			Directory:          dir,
			Force:              force,
			DryRun:             dryRun,
			JSON:               jsonOut,
			NonInteractive:     nonInteractive,
			PackageManagerFlag: PackageManager(pmFlag),
			NoInstall:          noInstall,
			NoSkill:            noSkill,
			Publishable:        publishable,
			MCP:                mcpMode,
		}

		result, initErr := activeSvc.Init(cmd.Context(), req)
		if initErr != nil {
			return initErr
		}

		return renderResult(cmd, result, jsonOut)
	}
}
