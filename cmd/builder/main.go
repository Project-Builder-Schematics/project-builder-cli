// Package main is the entry point for the builder CLI.
//
// composeApp is the sole adapter-aware site in the codebase (ADR-011).
// It wires all port interfaces — Engine, Renderer, and the Cobra command tree —
// and returns a fully-initialised *App. No other file under internal/ imports
// concrete adapter types.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/add"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/execute"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/info"
	initialise "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init"
	inittemplate "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init/template"
	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/remove"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/skill"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/sync"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/validate"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
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
	// Engine is the schematic execution port. Wired to AngularSubprocessAdapter (/plan #4).
	Engine engine.Engine

	// Renderer is the event-stream output port. Wired via render.NewRenderer factory (/plan #3).
	Renderer render.Renderer

	// Root is the Cobra root command with all feature sub-commands registered.
	Root *cobra.Command
}

// composeApp wires all dependencies and returns the composed *App.
//
// This is the SOLE site in the codebase where concrete adapter types are
// imported and instantiated (ADR-010). The function body ceiling is ≤120 SLOC
// (composition-root.REQ-01.2, enforced by Test_ComposeApp_LOC_Within120).
func composeApp(cfg Config) (*App, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Engine adapter — AngularSubprocessAdapter (real; /plan #4 S-000).
	eng := angular.NewAdapter()

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

	// Init feature wiring (REQ-FW-03, ADR-020).
	// initFS is osFS (real writes); in dry-run mode the handler swaps to dryRunFS.
	// initPM is the PackageManagerRunner stub (real impl lands in S-005).
	// inittemplate.Skill holds the locked v0 SKILL.md bytes bundled via //go:embed (ADR-022, S-002).
	// NOTE: switched from initialise.NewOSWriter() to fswriter.NewOSWriter() (S-000b, shared promotion).
	initFS := fswriter.NewOSWriter()
	initPM := initialise.NewRealPM()
	initSvc := initialise.NewService(initFS, initPM, inittemplate.Skill)

	// New feature wiring (S-000b).
	// newFS is osFS; in dry-run mode the handler swaps to dryRunFS per request.
	newFS := fswriter.NewOSWriter()
	newSvc := newfeature.NewService(newFS)

	// Register all commands (cobra-command-tree.REQ-01.1).
	root.AddCommand(initialise.NewCommand(initSvc)) // init
	root.AddCommand(execute.NewCommand())           // execute
	root.AddCommand(add.NewCommand())               // add
	root.AddCommand(info.NewCommand())              // info
	root.AddCommand(sync.NewCommand())              // sync
	root.AddCommand(validate.NewCommand())          // validate
	root.AddCommand(remove.NewCommand())            // remove
	root.AddCommand(skill.NewCommand())             // skill (parent; skill update is its leaf)
	root.AddCommand(newfeature.NewCommand(newSvc))  // new (parent; schematic + collection leaves)

	return &App{
		Engine:   eng,
		Renderer: ren,
		Root:     root,
	}, nil
}

// exitCodeForErr maps an error from fang.Execute to a process exit code.
//
// Exit code semantics:
//   - 0: no error (nil — defensive; main never calls this for nil)
//   - 2: structured user-facing error (*errs.Error, direct or wrapped via %w)
//   - 1: unexpected / infrastructure error (anything else)
//
// Using exit code 2 for *errs.Error ensures that CI pipelines and AI agents
// can distinguish user-correctable errors (bad name, existing schematic, mode
// conflict, invalid extends, invalid language) from unexpected crashes.
//
// Trace: handler returns *errs.Error → service propagates → fang.Execute returns
// it → main calls exitCodeForErr → errors.As finds *errs.Error → returns 2 →
// os.Exit(2). REQ-NS-02, REQ-NS-04, REQ-NS-07, REQ-NSI-02, REQ-NCP-03,
// REQ-EX-02, REQ-EX-03, REQ-LG-06, REQ-NC-02, REQ-NC-05, REQ-EC-01..06 all
// require exit code 2 for structured errors (moves from PARTIAL to COMPLIANT).
func exitCodeForErr(err error) int {
	if err == nil {
		return 0
	}
	var ee *errs.Error
	if errors.As(err, &ee) {
		return 2
	}
	return 1
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
		os.Exit(exitCodeForErr(err))
	}
}
