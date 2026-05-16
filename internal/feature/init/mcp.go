// Package initialise — mcp.go contains MCP flag parsing, resolution, TTY
// detection, and the interactive prompt helper used by handler.RunE.
//
// REQ-MCP-01: --mcp flag values (yes/no/prompt) are case-insensitive.
// Default resolution:
//   - --non-interactive       → MCPNo
//   - TTY (no flag)           → MCPPrompt
//   - non-TTY (no flag)       → MCPNo
//   - --mcp=prompt + non-int  → ErrCodeInvalidInput (incompatible)
//
// REQ-MCP-01 prompt affirmatives: y, Y, yes, YES (trim trailing newline).
// Anything else (including empty Enter) → MCPNo.
//
// ADR-05: promptMCP delegates write+read to output.Output.Prompt, which is
// synchronous. The answer-parsing logic is extracted to parseMCPAnswer for
// independent testability.
package initialise

import (
	"os"
	"strings"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
)

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

// promptMCP writes mcpPromptQuestion via out.Prompt and reads the answer.
// Returns MCPYes if the answer is affirmative (y, Y, yes, YES after trim).
// Returns MCPNo for anything else, including empty Enter (capital-N default).
// REQ-MCP-01.
//
// ADR-05: output.Output.Prompt handles write+read synchronously. In tests,
// inject a fake Output whose Prompt returns the desired answer.
func promptMCP(out output.Output) MCPMode {
	answer, _ := out.Prompt(mcpPromptQuestion)
	return parseMCPAnswer(answer)
}

// parseMCPAnswer maps a raw answer string to MCPYes or MCPNo.
// Affirmatives: y, Y, yes, YES (trimmed). Anything else → MCPNo.
// Extracted for independent testability (REQ-MCP-01 answer-parsing).
func parseMCPAnswer(answer string) MCPMode {
	trimmed := strings.TrimRight(answer, "\r\n")
	switch trimmed {
	case "y", "Y", "yes", "YES":
		return MCPYes
	}
	return MCPNo
}
