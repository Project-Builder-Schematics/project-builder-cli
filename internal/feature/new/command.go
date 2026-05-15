// Package newfeature — command.go wires the `builder new` parent command and
// its two leaf subcommands: `schematic` (alias: s) and `collection` (alias: c).
//
// RESERVED aliases: "s" (schematic), "c" (collection). Do not use these as
// subcommand names. (REQ-AL-04)
//
// All flags match spec literals exactly (L-builder-init-01 cross-check).
// The parent command has no RunE — it delegates entirely to subcommands.
package newfeature

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the Cobra parent command for `builder new` with the
// schematic and collection leaf subcommands registered.
//
// svc is the wired Service instance provided by composeApp.
func NewCommand(svc *Service) *cobra.Command {
	parent := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new schematic or collection",
		Long: `new scaffolds a schematic or collection in an existing workspace.

Subcommands:
  schematic (alias: s)  — create a new schematic
  collection (alias: c) — create a new collection`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	parent.AddCommand(newSchematicCommand(svc))
	parent.AddCommand(newCollectionCommand(svc))

	return parent
}

// newSchematicCommand returns the Cobra leaf command for `builder new schematic`.
// Alias "s" is registered per REQ-AL-01.
func newSchematicCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schematic [name]",
		Aliases: []string{"s"},
		Short:   "Create a new schematic in the workspace",
		Long: `Scaffold a new schematic with factory and schema files.

Use --dry-run to preview the planned operations without writing any files.
Use --inline to embed the schematic definition in project-builder.json instead
of creating standalone files.`,
		Args: cobra.MaximumNArgs(1),
	}

	flags := cmd.Flags()

	// REQ-NS-03: --force allows overwriting an existing schematic.
	flags.Bool("force", false, "overwrite existing schematic if present")

	// REQ-NS-05: --dry-run previews operations without writing files.
	flags.Bool("dry-run", false, "preview planned operations without writing any files")

	// REQ-NSI-01: --inline embeds the schematic in project-builder.json.
	flags.Bool("inline", false, "embed schematic definition in project-builder.json (no files created)")

	// REQ-LG-01: --language selects TypeScript or JavaScript factory.
	flags.String("language", "", "factory language: ts or js (default: auto-detect)")

	// REQ-EX-01: --extends selects a base schematic to inherit from.
	flags.String("extends", "", "base schematic in @scope/pkg:collection grammar (e.g. @my-org/pkg:base)")

	// RunE adapter: extracts dryRun + jsonOut from persistent parent flags,
	// then delegates to handleSchematic.
	handler := handleSchematic(svc)
	cmd.RunE = func(c *cobra.Command, args []string) error {
		dryRun, _ := c.Flags().GetBool("dry-run")
		jsonOut := false // --output flag lives on root; injected via PersistentPreRunE in S-006
		return handler(c, args, dryRun, jsonOut)
	}

	return cmd
}

// newCollectionCommand returns the Cobra leaf command for `builder new collection`.
// Alias "c" is registered per REQ-AL-02.
func newCollectionCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "collection [name]",
		Aliases: []string{"c"},
		Short:   "Create a new collection in the workspace",
		Long: `Scaffold a new schematic collection with a skeleton collection.json.

Use --publishable to also generate add/ and remove/ lifecycle stubs.
Use --dry-run to preview the planned operations without writing any files.`,
		Args: cobra.MaximumNArgs(1),
	}

	flags := cmd.Flags()

	// REQ-NC-03: --force allows overwriting an existing collection.
	flags.Bool("force", false, "overwrite existing collection if present")

	// REQ-NC-06: --dry-run previews operations without writing files.
	flags.Bool("dry-run", false, "preview planned operations without writing any files")

	// REQ-NCP-01: --publishable generates lifecycle stubs (add/ + remove/).
	flags.Bool("publishable", false, "generate add/ and remove/ lifecycle stubs")

	// RunE adapter: extracts dryRun + jsonOut from flags, delegates to handleCollection.
	handler := handleCollection(svc)
	cmd.RunE = func(c *cobra.Command, args []string) error {
		dryRun, _ := c.Flags().GetBool("dry-run")
		jsonOut := false
		return handler(c, args, dryRun, jsonOut)
	}

	return cmd
}
