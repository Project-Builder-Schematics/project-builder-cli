// Package main — help_test.go covers REQ-AL-03: alias visibility in --help output.
//
// REQ-AL-03 mandates that `builder new schematic --help` and
// `builder new collection --help` surfaces the alias identifiers so users
// can discover the short forms without consulting external docs.
//
// Approach: Option A (per S-006 task description).
// Cobra natively renders the "Aliases:" section when cmd.Aliases is populated.
// Tests drive app.Root.Execute() directly (bypassing fang) for deterministic
// output. The parent `new` command's Long description also advertises aliases
// explicitly for discoverability at the parent level.
package main

import (
	"bytes"
	"strings"
	"testing"
)

// Test_REQ_AL03_Schematic_HelpShowsAlias verifies that running
// `builder new schematic --help` surfaces the alias "s" to the user.
// Cobra renders "Aliases:  schematic, s" automatically when Aliases: []string{"s"}
// is set on the command.
func Test_REQ_AL03_Schematic_HelpShowsAlias(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{OutputMode: "pretty"})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"new", "schematic", "--help"})
	_ = app.Root.Execute()

	helpText := buf.String()

	// Cobra renders: "Aliases:\n  schematic, s" when Aliases: []string{"s"}.
	// We assert the alias short form "s" appears in the aliases section.
	if !strings.Contains(helpText, "Aliases:") {
		t.Errorf("REQ-AL-03: 'builder new schematic --help' did not render Aliases section; got:\n%s", helpText)
	}
	if !strings.Contains(helpText, " s") {
		t.Errorf("REQ-AL-03: alias 's' not visible in 'builder new schematic --help'; got:\n%s", helpText)
	}
}

// Test_REQ_AL03_Collection_HelpShowsAlias verifies that running
// `builder new collection --help` surfaces the alias "c" to the user.
func Test_REQ_AL03_Collection_HelpShowsAlias(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{OutputMode: "pretty"})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"new", "collection", "--help"})
	_ = app.Root.Execute()

	helpText := buf.String()

	if !strings.Contains(helpText, "Aliases:") {
		t.Errorf("REQ-AL-03: 'builder new collection --help' did not render Aliases section; got:\n%s", helpText)
	}
	if !strings.Contains(helpText, " c") {
		t.Errorf("REQ-AL-03: alias 'c' not visible in 'builder new collection --help'; got:\n%s", helpText)
	}
}

// Test_REQ_AL03_New_ParentHelp_MentionsAliases verifies that the parent
// `builder new --help` also conveys alias information via the Long description.
// The Long field explicitly states "(alias: s)" and "(alias: c)" for discoverability.
func Test_REQ_AL03_New_ParentHelp_MentionsAliases(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{OutputMode: "pretty"})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"new", "--help"})
	_ = app.Root.Execute()

	helpText := buf.String()

	// The parent 'new' command's Long description contains "alias: s" and "alias: c".
	if !strings.Contains(helpText, "alias: s") {
		t.Errorf("REQ-AL-03: 'builder new --help' Long description does not mention 'alias: s'; got:\n%s", helpText)
	}
	if !strings.Contains(helpText, "alias: c") {
		t.Errorf("REQ-AL-03: 'builder new --help' Long description does not mention 'alias: c'; got:\n%s", helpText)
	}
}
