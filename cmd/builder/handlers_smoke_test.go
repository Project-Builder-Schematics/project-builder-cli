// Package main_test (handlers_smoke_test.go) covers inert-stub-contract and
// cobra-command-tree REQs via a table-driven smoke test over all 8 handlers.
//
// Tests:
//   - inert-stub-contract.REQ-01.1 — every handler returns *errors.Error with ErrCodeNotImplemented
//   - inert-stub-contract.REQ-01.2 — every handler's Op matches the OpRegex pattern
//   - cobra-command-tree.REQ-02.1 — `skill` parent command with no args exits 0
//   - structured-error.REQ-01.3 — Op format is "^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$"
//
// CONTRACT:STUB — all handlers return ErrCodeNotImplemented at the skeleton phase.
// Real handler implementations land at /plan #5+.
package main

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/add"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/execute"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/info"
	initialise "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/remove"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/skill"
	skillupdate "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/skill/update"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/sync"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/validate"
)

// handlerRow describes a single command under test.
type handlerRow struct {
	name   string
	runE   func(*cobra.Command, []string) error
	wantOp string
}

// handlerRows lists all 8 leaf handler RunE functions with their expected Op values.
// inert-stub-contract.REQ-01.1 + .REQ-01.2 + structured-error.REQ-01.3.
var handlerRows = []handlerRow{
	{name: "init", runE: initialise.RunE, wantOp: "init.handler"},
	{name: "execute", runE: execute.RunE, wantOp: "execute.handler"},
	{name: "add", runE: add.RunE, wantOp: "add.handler"},
	{name: "info", runE: info.RunE, wantOp: "info.handler"},
	{name: "sync", runE: sync.RunE, wantOp: "sync.handler"},
	{name: "validate", runE: validate.RunE, wantOp: "validate.handler"},
	{name: "remove", runE: remove.RunE, wantOp: "remove.handler"},
	{name: "skill/update", runE: skillupdate.RunE, wantOp: "skill_update.handler"},
}

// opRegex is the compiled OpRegex for assertion.
var opRegex = regexp.MustCompile(errs.OpRegex)

// Test_AllHandlers_ReturnNotImplemented covers inert-stub-contract.REQ-01.1.
// Each handler must return an error satisfying errors.Is(err, &errs.Error{Code: ErrCodeNotImplemented}).
func Test_AllHandlers_ReturnNotImplemented(t *testing.T) {
	t.Parallel()

	sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
	dummyCmd := &cobra.Command{Use: "dummy"}

	for _, row := range handlerRows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			err := row.runE(dummyCmd, nil)
			if err == nil {
				t.Errorf("handler %q returned nil error — expected ErrCodeNotImplemented", row.name)
				return
			}
			if !errors.Is(err, sentinel) {
				t.Errorf("handler %q: errors.Is(err, ErrCodeNotImplemented) = false; got %v", row.name, err)
			}
		})
	}
}

// Test_AllHandlers_ErrorsAs_OpField covers inert-stub-contract.REQ-01.2 + structured-error.REQ-01.3.
// Each handler must return an *errs.Error whose Op matches OpRegex and equals the expected Op.
func Test_AllHandlers_ErrorsAs_OpField(t *testing.T) {
	t.Parallel()

	dummyCmd := &cobra.Command{Use: "dummy"}

	for _, row := range handlerRows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			err := row.runE(dummyCmd, nil)
			if err == nil {
				t.Errorf("handler %q returned nil error", row.name)
				return
			}

			var e *errs.Error
			if !errors.As(err, &e) {
				t.Errorf("handler %q: errors.As(*errs.Error) failed; got type %T", row.name, err)
				return
			}

			// Op must match regex.
			if !opRegex.MatchString(e.Op) {
				t.Errorf("handler %q: Op %q does not match regex %s (structured-error.REQ-01.3)", row.name, e.Op, errs.OpRegex)
			}

			// Op must equal expected.
			if e.Op != row.wantOp {
				t.Errorf("handler %q: Op = %q, want %q", row.name, e.Op, row.wantOp)
			}
		})
	}
}

// Test_Skill_NoArgs_ExitsZero covers cobra-command-tree.REQ-02.1.
// The `skill` parent command invoked with no positional args must return nil (exit 0).
func Test_Skill_NoArgs_ExitsZero(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	// Redirect output to discard — help text is informational, not under test here.
	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)

	app.Root.SetArgs([]string{"skill", "--help"})
	execErr := app.Root.Execute()
	if execErr != nil {
		t.Errorf("skill --help returned error: %v (cobra-command-tree.REQ-02.1)", execErr)
	}
}

// Test_SkillNewCommand_RegistersUpdateChild verifies skill command has update child.
func Test_SkillNewCommand_RegistersUpdateChild(t *testing.T) {
	t.Parallel()

	skillCmd := skill.NewCommand()
	if skillCmd == nil {
		t.Fatal("skill.NewCommand() returned nil")
	}

	var found bool
	for _, sub := range skillCmd.Commands() {
		if sub.Name() == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Error("skill command does not have 'update' subcommand (cobra-command-tree.REQ-01.1)")
	}
}
