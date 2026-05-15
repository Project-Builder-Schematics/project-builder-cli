// Package newfeature — handler_test.go covers the stub handler smoke behaviour.
//
// REQ coverage:
//   - REQ-EC-07: handlers return ErrCodeNewNotImplemented via Renderer (stub sentinel)
//   - REQ-AL-01..03: aliases s/c wired + help shows aliases (covered via command_test.go)
//   - REQ-NS-05 (partial): --dry-run flag is registered and recognised
//
// NOTE: Full command alias and --help tests live in command_test.go (Task F).
// This file covers handler RunE behaviour in isolation.
package newfeature

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
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

// Test_HandleSchematic_ReturnsErrNewNotImplemented verifies the schematic handler
// returns ErrCodeNewNotImplemented when called (stub S-000b).
// REQ-EC-07.
func Test_HandleSchematic_ReturnsErrNewNotImplemented(t *testing.T) {
	t.Parallel()

	svc := newTestService()
	cmd := newSchematicCmd()
	err := handleSchematic(svc)(cmd, []string{"my-schematic"}, false, false)
	if err == nil {
		t.Fatal("handleSchematic: expected error, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNewNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("handleSchematic: errors.Is(ErrCodeNewNotImplemented) = false; got: %v", err)
	}
}

// Test_HandleCollection_ReturnsErrNewNotImplemented verifies the collection handler
// returns ErrCodeNewNotImplemented when called (stub S-000b).
// REQ-EC-07.
func Test_HandleCollection_ReturnsErrNewNotImplemented(t *testing.T) {
	t.Parallel()

	svc := newTestService()
	cmd := newCollectionCmd()
	err := handleCollection(svc)(cmd, []string{"my-collection"}, false, false)
	if err == nil {
		t.Fatal("handleCollection: expected error, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNewNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("handleCollection: errors.Is(ErrCodeNewNotImplemented) = false; got: %v", err)
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
