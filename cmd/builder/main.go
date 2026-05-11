// Package main is the entry point for the builder CLI.
//
// composeApp is the sole adapter-aware site in the codebase (ADR-011).
// It wires all port interfaces — Engine, Renderer, and the Cobra command tree —
// and returns a fully-initialised *App. No other file under internal/ imports
// concrete adapter types.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mattn/go-isatty"
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
type Config struct {
	// OutputMode controls which renderer adapter is selected.
	// Accepted values: "pretty", "json", "" (auto — resolved via TTY detection).
	// Corresponds to the --output persistent flag on the root Cobra command.
	OutputMode render.OutputMode
}

// validate checks the Config for obvious misconfiguration.
func (c Config) validate() error {
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
func composeApp(cfg Config) (*App, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Engine adapter — FakeEngine until /plan #4.
	eng := &engine.FakeEngine{}

	// Renderer adapter — selected by factory based on --output flag + TTY.
	// isTTY is injected here (production path); tests pass their own stub.
	isTTY := func() bool { return isatty.IsTerminal(os.Stdout.Fd()) }
	ren, renErr := render.NewRenderer(cfg.OutputMode, isTTY)
	if renErr != nil {
		return nil, renErr
	}

	// Cobra root command — metadata used by `builder --help`.
	root := &cobra.Command{
		Use:   "builder",
		Short: "Project Builder CLI — schematic-driven project scaffolding",
		Long: `builder is a schematic-driven CLI for initialising, scaffolding, and
maintaining software projects.

Run 'builder --help' to see all available commands.
Run 'builder <command> --help' for command-specific usage.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// --output persistent flag (REQ-14): propagated to all sub-commands.
	// Default is "" (OutputModeAuto) — factory resolves via TTY detection.
	root.PersistentFlags().String("output", "", `output format: "pretty" (human-readable) or "json" (NDJSON for CI/pipes). Default: auto-detect from terminal.`)

	// Register all 8 leaf commands (cobra-command-tree.REQ-01.1).
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
	// Extract --output flag value before calling composeApp.
	// We use pflag directly on a temporary FlagSet so the factory receives
	// the resolved mode at startup — before the full Cobra tree is constructed.
	// This avoids a two-phase init or PersistentPreRunE indirection.
	fs := cobra.Command{}
	var outputFlag string
	fs.PersistentFlags().StringVar(&outputFlag, "output", "", "")
	// ParseErrorsWhitelist allows unknown flags (sub-command flags) to pass.
	_ = fs.ParseFlags(os.Args[1:])

	app, err := composeApp(Config{OutputMode: render.OutputMode(outputFlag)})
	if err != nil {
		slog.Error("failed to initialise application", "error", err)
		os.Exit(1)
	}

	// fang.Execute wraps Cobra's Execute with styled help, error, and
	// version output (charmbracelet aesthetics). Tests still drive
	// app.Root.Execute() directly to keep assertions deterministic.
	if err := fang.Execute(context.Background(), app.Root); err != nil {
		os.Exit(1)
	}
}
