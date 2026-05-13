// Package main_test (handlers_smoke_test.go) covers inert-stub-contract and
// cobra-command-tree REQs via a table-driven smoke test over the remaining stub handlers.
//
// Tests:
//   - inert-stub-contract.REQ-01.1 — every remaining stub handler returns *errors.Error with ErrCodeNotImplemented
//   - inert-stub-contract.REQ-01.2 — every handler's Op matches the OpRegex pattern
//   - cobra-command-tree.REQ-02.1 — `skill` parent command with no args exits 0
//   - structured-error.REQ-01.3 — Op format is "^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$"
//
// Note: the `init` handler row was removed from this smoke test.
// It was asserting ErrCodeNotImplemented for the stub. Now that `builder init`
// has a real handler (S-000 walking skeleton), its behaviour is covered by
// internal/feature/init/handler_test.go. (REQ-EC-03)
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/spf13/cobra"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/add"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/execute"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/info"
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

// handlerRows lists the 7 remaining stub handler RunE functions with their
// expected Op values. The `init` row is omitted — it now has a real handler
// covered by internal/feature/init/handler_test.go.
// inert-stub-contract.REQ-01.1 + .REQ-01.2 + structured-error.REQ-01.3.
var handlerRows = []handlerRow{
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

// Test_Init_DryRun_JSON_EndToEnd covers the S-000 walking skeleton acceptance criterion:
// builder init --dry-run --json --mcp=yes <dir> → valid JSON envelope with
// mcp_setup_offered:true and 6 planned_ops.
// REQ-JO-03, REQ-DR-01, REQ-MCP-03.
func Test_Init_DryRun_JSON_EndToEnd(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	dir := t.TempDir()
	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"init", "--dry-run", "--json", "--mcp=yes", dir})

	if execErr := app.Root.Execute(); execErr != nil {
		t.Fatalf("init --dry-run --json --mcp=yes: %v — stderr: %s", execErr, buf.String())
	}

	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v — raw output: %s", err, buf.String())
	}

	if result["directory"] != dir {
		t.Errorf("directory = %v, want %q", result["directory"], dir)
	}
	if dryRun, _ := result["dry_run"].(bool); !dryRun {
		t.Errorf("dry_run = %v, want true", result["dry_run"])
	}
	if mcp, _ := result["mcp_setup_offered"].(bool); !mcp {
		t.Errorf("mcp_setup_offered = %v, want true (REQ-MCP-03)", result["mcp_setup_offered"])
	}
	ops, ok := result["planned_ops"].([]any)
	if !ok {
		t.Fatalf("planned_ops is not an array; got: %s", buf.String())
	}
	if len(ops) != 6 {
		t.Errorf("planned_ops count = %d, want 6 (5 outputs + mcp_setup_offered) (REQ-DR-02)", len(ops))
	}
}

// Test_Init_Publishable_ReturnsInitNotImplemented verifies the smoke-level
// behaviour of --publishable via the real Cobra command tree.
// REQ-CS-05, REQ-EC-03.
func Test_Init_Publishable_ReturnsInitNotImplemented(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"init", "--dry-run", "--publishable", t.TempDir()})

	execErr := app.Root.Execute()
	if execErr == nil {
		t.Fatal("init --publishable: expected error, got nil (REQ-CS-05)")
	}

	if !errors.Is(execErr, &errs.Error{Code: errs.ErrCodeInitNotImplemented}) {
		t.Errorf("errors.Is(ErrCodeInitNotImplemented) = false; got: %v", execErr)
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
