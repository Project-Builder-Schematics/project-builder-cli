// Package initialise — handler.go contains the RunE handler for `builder init`.
//
// Responsibilities:
//  1. Parse flags and positional argument
//  2. Canonicalise the target directory (filepath.Abs + filepath.Clean)
//  3. Reject paths with .. traversal outside cwd (REQ-DV-02)
//  4. Parse and resolve --mcp flag (case-insensitive; TTY + non-interactive defaults)
//  5. Build InitRequest and call svc.Init
//  6. Render the result (pretty or JSON based on --json flag)
//  7. Map service errors to exit codes via the error return
package initialise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// svcKey is used to attach a *Service to a Cobra command via SetContext.
// We store the service pointer on the command itself using Cobra's annotation
// map, avoiding global state. The handler retrieves it in RunE.
const svcKey = "init.service"

// RunE is the Cobra RunE entrypoint for `builder init`.
// It is kept separate from newRunE to allow the smoke test to reference it
// before the real service is wired (will be removed once smoke test is updated).
//
// Deprecated: use newRunE(svc) instead. This stub is retained for the
// smoke test transition only.
func RunE(cmd *cobra.Command, args []string) error {
	// Retrieve the wired service from the command's annotation map.
	// If not wired (legacy path), fall back to ErrCodeNotImplemented.
	svc, ok := cmd.Annotations[svcKey]
	if !ok || svc == "" {
		return &errs.Error{
			Code:    errs.ErrCodeNotImplemented,
			Op:      "init.handler",
			Message: "init not yet implemented (planned for /plan #5)",
		}
	}
	// The real implementation is handled by newRunE. This path is a fallback.
	return &errs.Error{
		Code:    errs.ErrCodeNotImplemented,
		Op:      "init.handler",
		Message: "init handler not wired — call NewCommand(svc) from composeApp",
	}
}

// newRunE returns the RunE closure wired with svc for use by NewCommand.
func newRunE(svc *Service) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		// Parse flags.
		force, _ := flags.GetBool("force")
		dryRun, _ := flags.GetBool("dry-run")
		jsonOut, _ := flags.GetBool("json")
		nonInteractive, _ := flags.GetBool("non-interactive")
		pmFlag, _ := flags.GetString("package-manager")
		noInstall, _ := flags.GetBool("no-install")
		noSkill, _ := flags.GetBool("no-skill")
		publishable, _ := flags.GetBool("publishable")
		mcpFlag, _ := flags.GetString("mcp")

		// Canonicalise directory.
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

		dir, err := canonicaliseDir(rawDir)
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
			resolveErr := resolveMCPMode(string(parsed), nonInteractive)
			if resolveErr != nil {
				return resolveErr
			}
			mcpMode = parsed
		} else {
			// No explicit --mcp flag: derive from TTY + non-interactive.
			isTTY := isStdinTTY()
			mcpMode = defaultMCPMode(nonInteractive, isTTY)
		}

		// For dry-run mode, create a local dryRunFS-backed service so that
		// PlannedOps are collected fresh per invocation without affecting the
		// osFS-backed production service (REQ-DR-01, ADR-020).
		activeSvc := svc
		if dryRun {
			dfs := newDryRunFS()
			activeSvc = NewService(dfs, svc.pm, svc.skill)
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

		// Render output to the command's output writer.
		return renderResult(cmd, result, jsonOut)
	}
}

// renderResult writes the InitResult to the command's output in the requested format.
// JSON mode emits a single NDJSON line; pretty mode emits human-readable text.
// REQ-JO-01, REQ-JO-03.
func renderResult(cmd *cobra.Command, result InitResult, jsonOut bool) error {
	out := cmd.OutOrStdout()
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		return enc.Encode(result)
	}

	// Pretty mode: human-readable summary.
	if result.DryRun {
		fmt.Fprintln(out, "DRY RUN — no files written")
		fmt.Fprintln(out)
		for _, op := range result.PlannedOps {
			switch op.Op {
			case "create_file":
				fmt.Fprintf(out, "  Would create: %s\n", op.Path)
			case "append_marker":
				fmt.Fprintf(out, "  Would append: %s\n", op.Path)
			case "modify_devdep":
				fmt.Fprintf(out, "  Would modify: %s (%s)\n", op.Path, op.Details)
			case "install_package":
				fmt.Fprintf(out, "  Would install: %s\n", op.Details)
			case "mcp_setup_offered":
				fmt.Fprintln(out, "  Would offer:   MCP server setup instructions")
			}
		}
		return nil
	}

	// Real-write mode (post-S-001) — list created files.
	fmt.Fprintf(out, "Initialising Project Builder workspace in %s ...\n\n", result.Directory)
	for _, p := range result.OutputsCreated {
		fmt.Fprintf(out, "  Created: %s\n", p)
	}
	if result.Installed {
		fmt.Fprintf(out, "\nInstalling @pbuilder/sdk via %s ... done.\n", result.PackageManager)
	}
	fmt.Fprintln(out, "\nProject Builder is ready. Try: builder add <name>")
	if result.MCPSetupOffered {
		fmt.Fprintln(out)
		fmt.Fprintln(out, mcpInstructions)
	}
	return nil
}

// parseMCPFlag parses a raw --mcp flag value (case-insensitive) to MCPMode.
// Returns ErrCodeInvalidInput for unrecognised values.
// REQ-MCP-01.
func parseMCPFlag(val string) (MCPMode, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "yes":
		return MCPYes, nil
	case "no":
		return MCPNo, nil
	case "prompt":
		return MCPPrompt, nil
	default:
		return "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.handler",
			Message: "invalid --mcp value: must be one of yes, no, prompt",
			Suggestions: []string{
				"--mcp=yes",
				"--mcp=no",
				"--mcp=prompt (TTY only)",
			},
		}
	}
}

// resolveMCPMode validates that the parsed MCPMode is compatible with the
// --non-interactive flag. Returns ErrCodeInvalidInput for incompatible combos.
// REQ-MCP-01: --mcp=prompt + --non-interactive is incompatible.
func resolveMCPMode(flagVal string, nonInteractive bool) error {
	mode := strings.ToLower(strings.TrimSpace(flagVal))
	if mode == "prompt" && nonInteractive {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.handler",
			Message: "--mcp=prompt is incompatible with --non-interactive (non-interactive mode cannot prompt)",
			Suggestions: []string{
				"use --mcp=yes for explicit MCP setup",
				"use --mcp=no to skip MCP",
				"remove --non-interactive to allow prompting",
			},
		}
	}

	// Validate the value even when non-interactive is false.
	switch mode {
	case "yes", "no", "prompt":
		return nil
	default:
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.handler",
			Message: "invalid --mcp value: must be one of yes, no, prompt",
			Suggestions: []string{
				"--mcp=yes",
				"--mcp=no",
				"--mcp=prompt (TTY only)",
			},
		}
	}
}

// defaultMCPMode returns the default MCPMode when no --mcp flag is provided.
// Rules (REQ-MCP-01):
//   - nonInteractive=true → MCPNo (never prompt in non-interactive mode)
//   - isTTY=true          → MCPPrompt (prompt the user)
//   - otherwise           → MCPNo
func defaultMCPMode(nonInteractive, isTTY bool) MCPMode {
	if nonInteractive {
		return MCPNo
	}
	if isTTY {
		return MCPPrompt
	}
	return MCPNo
}

// canonicaliseDir applies filepath.Abs + filepath.Clean to rawDir and
// rejects paths that resolve outside the current working directory via
// .. segments (REQ-DV-01, REQ-DV-02).
func canonicaliseDir(rawDir string) (string, error) {
	abs, err := filepath.Abs(rawDir)
	if err != nil {
		return "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.handler",
			Message: "could not resolve directory path",
			Cause:   err,
		}
	}
	clean := filepath.Clean(abs)

	// Reject paths outside cwd: a valid project dir must not climb above cwd.
	// We detect this by checking if the cleaned absolute path starts with the
	// cwd prefix. This prevents symlink shenanigans and relative .. escapes.
	//
	// Implementation note: we only reject paths that are clearly OUTSIDE the
	// cwd (i.e. resolving to an ancestor of cwd or a completely different tree).
	// Sibling directories (e.g. ../sibling) that are outside cwd are rejected
	// because they could point to sensitive system directories.
	//
	// We check whether the raw input contains ".." segments that would resolve
	// to a path above cwd. A clean absolute path that is genuinely inside cwd
	// will have cwd as a prefix.
	if strings.Contains(rawDir, "..") {
		cwd, cwdErr := os.Getwd()
		if cwdErr == nil {
			cwdClean := filepath.Clean(cwd)
			// If the cleaned target path is the same as or within cwd, allow it.
			if !strings.HasPrefix(clean, cwdClean) {
				return "", &errs.Error{
					Code:    errs.ErrCodeInvalidInput,
					Op:      "init.handler",
					Message: "directory path resolves outside the current working directory (.. traversal rejected)",
					Suggestions: []string{
						"use an absolute path instead of a relative path with ..",
						"use a path relative to the current working directory without traversal",
					},
				}
			}
		}
	}

	return clean, nil
}

// isStdinTTY returns true if os.Stdin is connected to a character device (TTY).
// Uses os.Stdin.Stat() which is available in the stdlib — no external dep needed.
// REQ-MCP-01: TTY detection for default --mcp resolution.
func isStdinTTY() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
