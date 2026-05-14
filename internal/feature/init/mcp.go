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
package initialise

import (
	"bufio"
	"io"
	"os"
	"strings"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
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

// promptMCP writes mcpPromptQuestion to w and reads one line from r.
// Returns MCPYes if the answer is affirmative (y, Y, yes, YES after trim).
// Returns MCPNo for anything else, including empty Enter (capital-N default).
// REQ-MCP-01.
//
// The handler calls this with os.Stdin and os.Stdout. Tests inject fake
// io.Reader and io.Writer via the handler's stdin/stdout coupling.
func promptMCP(r io.Reader, w io.Writer) MCPMode {
	_, _ = io.WriteString(w, mcpPromptQuestion)

	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		answer := strings.TrimRight(scanner.Text(), "\r\n")
		switch answer {
		case "y", "Y", "yes", "YES":
			return MCPYes
		}
	}
	return MCPNo
}
