// Package newfeature — handler_test.go covers handler smoke behaviour.
//
// REQ coverage:
//   - REQ-AL-01..03: aliases s/c wired + help shows aliases (covered via command_test.go)
//   - REQ-NS-05 (partial): --dry-run flag is registered and recognised
//   - REQ-NC-06 (partial): collection --dry-run returns result (handler smoke)
//
// NOTE: Full command alias and --help tests live in command_test.go (Task F).
// This file covers handler RunE behaviour in isolation.
package newfeature

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// newTestService returns a Service wired with a dryRun FSWriter for handler tests.
func newTestService() *Service {
	return NewService(fswriter.NewDryRunWriter())
}

// newSchematicCmd returns a bare schematic subcommand for handler tests.
// Flags must match the spec literals (same as newSchematicCommand in command.go).
func newSchematicCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "schematic [name]", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("inline", false, "")
	cmd.Flags().String("language", "", "")
	cmd.Flags().String("extends", "", "")
	return cmd
}

// newCollectionCmd returns a bare collection subcommand for handler tests.
func newCollectionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "collection [name]", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("publishable", false, "")
	return cmd
}

// Test_HandleSchematic_DryRun_ReturnsDryRunResult verifies the schematic handler
// returns a valid result in dry-run mode (REQ-NS-05).
// S-001: handler now dispatches to real service; dry-run returns planned ops.
func Test_HandleSchematic_DryRun_ReturnsDryRunResult(t *testing.T) {
	t.Parallel()

	svc := newTestService()
	cmd := newSchematicCmd()
	// Pass dryRun=true so no project-builder.json read is attempted.
	err := handleSchematic(svc)(cmd, []string{"my-schematic"}, true, false)
	if err != nil {
		t.Errorf("handleSchematic(dry-run): unexpected error: %v", err)
	}
}

// Test_HandleCollection_DryRun_ReturnsResult verifies the collection handler
// returns a valid result in dry-run mode after S-004 implements RegisterCollection.
// Replaces the S-000b stub sentinel test (REQ-EC-07: stub removed in S-004).
func Test_HandleCollection_DryRun_ReturnsResult(t *testing.T) {
	t.Parallel()

	svc := newTestService()
	cmd := newCollectionCmd()
	// Pass dryRun=true so no project-builder.json read is attempted.
	err := handleCollection(svc)(cmd, []string{"my-collection"}, true, false)
	if err != nil {
		t.Errorf("handleCollection(dry-run): unexpected error: %v", err)
	}
}

// Test_FlagNames_Schematic_MatchSpec verifies the schematic subcommand has
// all the flag names mandated by the spec (L-builder-init-01 cross-check).
// Any flag name drift from the spec would be caught here.
func Test_FlagNames_Schematic_MatchSpec(t *testing.T) {
	t.Parallel()

	svc := newTestService()
	root := NewCommand(svc)

	// Find the schematic subcommand.
	scCmd, _, err := root.Find([]string{"schematic"})
	if err != nil || scCmd == nil || scCmd.Name() != "schematic" {
		t.Fatalf("schematic subcommand not found: %v", err)
	}

	requiredFlags := []string{"force", "dry-run", "language", "extends", "inline"}
	for _, name := range requiredFlags {
		if f := scCmd.Flags().Lookup(name); f == nil {
			t.Errorf("schematic subcommand missing flag --%s (spec literal)", name)
		}
	}
}

// Test_FlagNames_Collection_MatchSpec verifies the collection subcommand has
// all the flag names mandated by the spec.
func Test_FlagNames_Collection_MatchSpec(t *testing.T) {
	t.Parallel()

	svc := newTestService()
	root := NewCommand(svc)

	colCmd, _, err := root.Find([]string{"collection"})
	if err != nil || colCmd == nil || colCmd.Name() != "collection" {
		t.Fatalf("collection subcommand not found: %v", err)
	}

	requiredFlags := []string{"force", "dry-run", "publishable"}
	for _, name := range requiredFlags {
		if f := colCmd.Flags().Lookup(name); f == nil {
			t.Errorf("collection subcommand missing flag --%s (spec literal)", name)
		}
	}
}
