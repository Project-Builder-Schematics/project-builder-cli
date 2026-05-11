// Package main is the entry point for the builder CLI.
//
// composeApp is the sole adapter-aware site in the codebase (ADR-011).
// It wires all port interfaces — Engine, Renderer, and the Cobra command tree —
// and returns a fully-initialised *App. No other file under internal/ imports
// concrete adapter types.
//
// CONTRACT:STUB — wires FakeEngine + NoopRenderer during the skeleton phase.
// Real adapters land at /plan #3 (renderer) and /plan #4 (engine).
package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/add"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/execute"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/info"
	initialise "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/remove"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/skill"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/sync"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/validate"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render"
)

// Config holds application-wide configuration.
//
// At the skeleton phase this is empty; fields will be added at /plan #3+
// when Viper is wired for config-file and environment-variable loading.
type Config struct{}

// validate checks the Config for obvious misconfiguration.
// At the skeleton phase Config is empty so validate always returns nil;
// when fields are added at /plan #3, this becomes the validation site.
func (c Config) validate() error {
	// Skeleton: no fields to validate.
	// /plan #3: add workspace path, log level, and env-allowlist validation here.
	_ = c
	return nil
}

// App is the composed application root returned by composeApp.
//
// All interface fields are guaranteed non-nil after a successful composeApp
// call (composition-root.REQ-01.1).
type App struct {
	// Engine is the schematic execution port. Wired to FakeEngine at skeleton.
	Engine engine.Engine

	// Renderer is the event-stream output port. Wired to NoopRenderer at skeleton.
	Renderer render.Renderer

	// Root is the Cobra root command with all feature sub-commands registered.
	Root *cobra.Command
}

// composeApp wires all dependencies and returns the composed *App.
//
// This is the SOLE site in the codebase where concrete adapter types are
// imported and instantiated (ADR-011). The function body ceiling is ≤120 SLOC
// (composition-root.REQ-01.2, enforced by Test_ComposeApp_LOC_Within120).
//
// CONTRACT:STUB — FakeEngine and NoopRenderer are skeleton-phase stubs.
func composeApp(cfg Config) (*App, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Concrete adapter construction — the only place in the codebase where
	// FakeEngine and NoopRenderer are instantiated. Swap here at /plan #3 + #4.
	eng := &engine.FakeEngine{}
	ren := &render.NoopRenderer{}

	// Cobra root command — metadata used by `builder --help`.
	root := &cobra.Command{
		Use:   "builder",
		Short: "Project Builder CLI — schematic-driven project scaffolding",
		Long: `builder is a schematic-driven CLI for initialising, scaffolding, and
maintaining software projects.

Run 'builder --help' to see all available commands.
Run 'builder <command> --help' for command-specific usage.`,
		// SilenceErrors prevents Cobra from printing errors before main() does.
		SilenceErrors: true,
		// SilenceUsage prevents Cobra from printing usage on RunE errors.
		SilenceUsage: true,
	}

	// Register all 8 leaf commands (cobra-command-tree.REQ-01.1).
	// Order matches alphabetical --help listing; skill is last as it is a group.
	root.AddCommand(initialise.NewCommand()) // init
	root.AddCommand(execute.NewCommand())    // execute
	root.AddCommand(add.NewCommand())        // add
	root.AddCommand(info.NewCommand())       // info
	root.AddCommand(sync.NewCommand())       // sync
	root.AddCommand(validate.NewCommand())   // validate
	root.AddCommand(remove.NewCommand())     // remove
	root.AddCommand(skill.NewCommand())      // skill (parent; skill update is its leaf)

	return &App{
		Engine:   eng,
		Renderer: ren,
		Root:     root,
	}, nil
}

func main() {
	app, err := composeApp(Config{})
	if err != nil {
		slog.Error("failed to initialise application", "error", err)
		os.Exit(1)
	}

	if err := app.Root.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
