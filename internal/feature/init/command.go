// Package initialise wires the `builder init` leaf command.
//
// This file registers the Cobra command with all flags defined by the spec
// (REQ-CS-01..05, REQ-JO-01..02, REQ-MCP-01). The handler RunE is provided
// by handler.go via newRunE(svc).
package initialise

import "github.com/spf13/cobra"

// NewCommand returns the Cobra leaf command for `builder init`.
//
// svc is the wired Service instance provided by composeApp (REQ-FW-03).
// All flag defaults and usage strings use UK English (REQ-JO-01).
func NewCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialise a new project workspace",
		Long: `Initialise bootstraps a new project workspace from a schematic collection.

The command creates five artefacts in the target directory (or the current
working directory if no argument is supplied):

  1. project-builder.json   — project configuration
  2. schematics/.gitkeep    — schematics collection skeleton
  3. .claude/skills/pbuilder/SKILL.md — AI skill artefact
  4. AGENTS.md (or CLAUDE.md) — pbuilder skill reference marker
  5. package.json           — @pbuilder/sdk added to devDependencies

Use --dry-run to preview the planned operations without writing any files.
Use --json to receive machine-readable NDJSON output (suitable for CI/AI).`,
		Args: cobra.MaximumNArgs(1),
		RunE: newRunE(svc),
	}

	flags := cmd.Flags()

	// REQ-CS-03: --force allows overwriting an existing project-builder.json.
	flags.Bool("force", false, "overwrite existing project-builder.json if present")

	// REQ-JO-02: --dry-run previews operations without writing files.
	flags.Bool("dry-run", false, "preview planned operations without writing any files")

	// REQ-JO-01: --json selects NDJSON output format.
	flags.Bool("json", false, "output machine-readable JSON (NDJSON)")

	// REQ-CS-04: --non-interactive disables all interactive prompts.
	flags.Bool("non-interactive", false, "disable all interactive prompts (suitable for CI)")

	// --package-manager selects the package manager explicitly.
	flags.String("package-manager", "", "package manager to use: npm, pnpm, yarn, or bun (default: auto-detect)")

	// REQ-PD-03: --no-install skips the install subprocess.
	flags.Bool("no-install", false, "skip the package manager install step")

	// REQ-SA-03: --no-skill skips SKILL.md, AGENTS.md marker, and SDK dev-dep atomically.
	flags.Bool("no-skill", false, "skip SKILL.md, agent file marker, and @pbuilder/sdk dev-dep")

	// REQ-CS-05: --publishable selects the publishable template (returns ErrCodeInitNotImplemented).
	flags.Bool("publishable", false, "use the publishable schematic template (not yet available)")

	// REQ-MCP-01: --mcp controls MCP server setup prompt.
	flags.String("mcp", "", "MCP setup: yes, no, or prompt (default: prompt in TTY, no in --non-interactive)")

	return cmd
}
