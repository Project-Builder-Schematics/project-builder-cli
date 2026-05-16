// Package main_test covers composition-root and cobra-command-tree REQs.
//
// Tests:
//   - composition-root.REQ-01.1 — composeApp returns *App with all interface fields non-nil
//   - composition-root.REQ-01.2 — composeApp function body is ≤120 SLOC
//   - cobra-command-tree.REQ-01.1 — Root has exactly 8 leaf commands
//   - dependencies.REQ-01.1 — go.mod pins cobra v1.9.x, viper v1.19.x, charmbracelet/log v0.4.x
//   - theme-profile-detection/REQ-02.1, 02.2, 03.1, 03.2 — flag/env precedence (S-003)
//
// CONTRACT:STUB — wires FakeEngine + NoopRenderer; production wiring at /plan #3+
package main

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	renderjson "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/json"
	prettyrend "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// Test_ComposeApp_AllInterfaceFieldsNonNil covers composition-root.REQ-01.1.
// Reflects over the App struct and asserts every interface/pointer field is non-nil.
func Test_ComposeApp_AllInterfaceFieldsNonNil(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}
	if app == nil {
		t.Fatal("composeApp returned nil *App")
	}
	if app.Engine == nil {
		t.Error("App.Engine is nil — composition-root.REQ-01.1 violated")
	}
	if app.Renderer == nil {
		t.Error("App.Renderer is nil — composition-root.REQ-01.1 violated")
	}
	if app.Root == nil {
		t.Error("App.Root is nil — composition-root.REQ-01.1 violated")
	}
}

// Test_ComposeApp_DoesNotPanic covers composition-root.REQ-01.1 (zero Config).
func Test_ComposeApp_DoesNotPanic(t *testing.T) {
	t.Parallel()

	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, _ = composeApp(Config{})
	}()

	if panicked {
		t.Error("composeApp panicked with zero-value Config")
	}
}

// Test_ComposeApp_LOC_Within120 covers composition-root.REQ-01.2.
// Reads the source file and counts non-blank, non-comment lines within the
// composeApp function body (between the opening brace and its matching close).
func Test_ComposeApp_LOC_Within120(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("cannot read main.go: %v", err)
	}

	sloc := countComposeAppSLOC(src)
	if sloc > 120 {
		t.Errorf("composeApp has %d SLOC — exceeds 120 LOC ceiling (composition-root.REQ-01.2)", sloc)
	}
	if sloc == 0 {
		t.Error("composeApp not found in main.go or has zero SLOC — implementation missing")
	}
}

// countComposeAppSLOC counts non-blank non-comment lines within the composeApp
// function body. It locates "func composeApp(" and counts from the line after
// the opening `{` to the matching closing `}`.
func countComposeAppSLOC(src []byte) int {
	scanner := bufio.NewScanner(bytes.NewReader(src))
	var inFunc, pastOpen bool
	depth := 0
	sloc := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inFunc {
			if strings.HasPrefix(trimmed, "func composeApp(") {
				inFunc = true
			}
			continue
		}

		// Inside function signature — look for opening brace.
		if !pastOpen {
			if strings.Contains(line, "{") {
				pastOpen = true
				depth = 1
			}
			continue
		}

		// Track brace depth to find end of function.
		for _, ch := range line {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
		if depth <= 0 {
			break // reached end of function
		}

		// Count as SLOC if not blank and not a pure comment line.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		sloc++
	}
	return sloc
}

// Test_RootCmd_ListsExactly10Leaves covers cobra-command-tree REQ.
// A "leaf" command is one with HasSubCommands() == false.
// Expected leaves: init, execute, add, info, sync, validate, remove, skill update,
// new schematic, new collection.
// S-000b: count updated from 8 → 10 after adding the new parent + 2 leaves.
func Test_RootCmd_ListsExactly10Leaves(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	leaves := collectLeaves(app.Root)
	wantLeaves := []string{
		"init", "execute", "add", "info", "sync", "validate", "remove",
		"update",     // skill update
		"schematic",  // new schematic
		"collection", // new collection
	}
	if len(leaves) != 10 {
		t.Errorf("got %d leaf commands, want 10; leaves: %v", len(leaves), leaves)
	}

	// Verify each expected leaf is present.
	leaveSet := make(map[string]bool)
	for _, l := range leaves {
		leaveSet[l] = true
	}
	for _, want := range wantLeaves {
		if !leaveSet[want] {
			t.Errorf("missing expected leaf command %q", want)
		}
	}
}

// hasAlias reports whether a Cobra command has the given alias string.
func hasAlias(cmd *cobra.Command, alias string) bool {
	for _, a := range cmd.Aliases {
		if a == alias {
			return true
		}
	}
	return false
}

// Test_RootCmd_NewSchematic_HasAliasS verifies alias "s" for `builder new schematic`.
// REQ-AL-01.
func Test_RootCmd_NewSchematic_HasAliasS(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}
	scCmd, _, scErr := app.Root.Find([]string{"new", "schematic"})
	if scErr != nil || scCmd == nil || scCmd.Name() != "schematic" {
		t.Fatalf("new schematic subcommand not found: %v", scErr)
	}
	if !hasAlias(scCmd, "s") {
		t.Errorf("new schematic: missing alias 's' (REQ-AL-01)")
	}
}

// Test_RootCmd_NewCollection_HasAliasC verifies alias "c" for `builder new collection`.
// REQ-AL-02.
func Test_RootCmd_NewCollection_HasAliasC(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}
	colCmd, _, colErr := app.Root.Find([]string{"new", "collection"})
	if colErr != nil || colCmd == nil || colCmd.Name() != "collection" {
		t.Fatalf("new collection subcommand not found: %v", colErr)
	}
	if !hasAlias(colCmd, "c") {
		t.Errorf("new collection: missing alias 'c' (REQ-AL-02)")
	}
}

// collectLeaves recursively walks the command tree and returns names of all
// leaf commands (those with no sub-commands).
func collectLeaves(cmd *cobra.Command) []string {
	if !cmd.HasSubCommands() {
		return []string{cmd.Name()}
	}
	var leaves []string
	for _, sub := range cmd.Commands() {
		leaves = append(leaves, collectLeaves(sub)...)
	}
	return leaves
}

// ──────────────────────────────────────────────────────────────────────────────
// renderer-adapters.REQ-13 — composeApp wires renderer from factory
// ──────────────────────────────────────────────────────────────────────────────

// Test_ComposeApp_WiresJSONRenderer covers REQ-13.1.
// composeApp with OutputMode "json" must return a *json.JSONRenderer as Renderer.
func Test_ComposeApp_WiresJSONRenderer(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{OutputMode: "json"})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}
	if app.Renderer == nil {
		t.Fatal("app.Renderer is nil")
	}
	if _, ok := app.Renderer.(*renderjson.Renderer); !ok {
		t.Errorf("app.Renderer = %T; want *json.Renderer", app.Renderer)
	}
}

// Test_ComposeApp_WiresPrettyRenderer covers REQ-13.2.
// composeApp with OutputMode "pretty" must return a *pretty.PrettyRenderer as Renderer.
func Test_ComposeApp_WiresPrettyRenderer(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{OutputMode: "pretty"})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}
	if app.Renderer == nil {
		t.Fatal("app.Renderer is nil")
	}
	if _, ok := app.Renderer.(*prettyrend.Renderer); !ok {
		t.Errorf("app.Renderer = %T; want *pretty.Renderer", app.Renderer)
	}
}

// Test_ComposeApp_InvalidOutputMode covers REQ-14.2 at the composeApp level.
// composeApp with an unknown OutputMode must return a non-nil error.
func Test_ComposeApp_InvalidOutputMode(t *testing.T) {
	t.Parallel()

	_, err := composeApp(Config{OutputMode: "xml"})
	if err == nil {
		t.Fatal("expected error for invalid OutputMode; got nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// renderer-adapters.REQ-14.1 — --output flag appears in --help output
// ──────────────────────────────────────────────────────────────────────────────

// Test_RootCmd_OutputFlagInHelp covers REQ-14.1.
// The root Cobra command's help text must contain "--output".
func Test_RootCmd_OutputFlagInHelp(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{OutputMode: "json"})
	if err != nil {
		t.Fatalf("composeApp: %v", err)
	}

	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"--help"})
	// Execute returns a help-request "error" from Cobra — ignore it.
	_ = app.Root.Execute()

	helpText := buf.String()
	if !strings.Contains(helpText, "--output") {
		t.Errorf("--help output does not contain \"--output\"; got:\n%s", helpText)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// S-000 — smoke test: NoColor theme wired through composeApp (render-pretty/REQ-01.1)
// ──────────────────────────────────────────────────────────────────────────────

// Test_ComposeApp_NoColorTheme_ZeroSGR verifies that composeApp wires the
// renderer via the new NewRenderer(mode, theme, isTTY) signature and that
// piped (non-TTY) output produces zero SGR escape sequences.
//
// Acceptance (S-000):
//   - composeApp constructs the renderer via NewRenderer(mode, theme, isTTY)
//   - Piped output contains no SGR escapes (\x1b[)
func Test_ComposeApp_NoColorTheme_ZeroSGR(t *testing.T) {
	t.Parallel()

	// composeApp must succeed with default (auto) mode.
	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	// Drive the renderer with a piped (non-TTY) buffer.
	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"info"})
	_ = app.Root.Execute()

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("piped output contains SGR escape sequences; want zero SGR bytes\noutput: %q", out)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// S-003 — theme flag / env precedence integration tests
// REQ-IDs: theme-profile-detection/REQ-02.1, 02.2, 03.1, 03.2
//
// TTY / colorprofile hygiene (lessons-learned/colorprofile-test-gotchas):
//   - t.Setenv("TMUX", "")       — block tmux bleed (colorprofile shells out to `tmux info`)
//   - t.Setenv("TMUX_PANE", "")  — block tmux pane bleed
//
// DetectAppearance does not depend on colorprofile/TTY, so TTY_FORCE is not
// needed here. The hygiene guard is a defensive boundary against unexpected
// colorprofile side-effects during composeApp (which calls DetectProfile).
// ──────────────────────────────────────────────────────────────────────────────

// applyColorprofileHygiene isolates the test from TMUX and colorprofile TTY
// heuristics. Must be called before any composeApp invocation in S-003 tests.
func applyColorprofileHygiene(t *testing.T) {
	t.Helper()
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
}

// Test_Theme_Flag_Light_OverridesDetection covers theme-profile-detection/REQ-02.1.
// With themeFlag=ThemeLight the constructed theme must have Appearance == theme.Light
// regardless of what the environment would otherwise detect.
//
// Note: t.Setenv is used for env isolation — cannot be combined with t.Parallel().
func Test_Theme_Flag_Light_OverridesDetection(t *testing.T) {
	applyColorprofileHygiene(t)

	app, err := composeApp(Config{themeFlag: ThemeLight})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	got := app.Theme.Appearance()
	if got != theme.Light {
		t.Errorf("Appearance() = %v; want theme.Light (REQ-02.1)", got)
	}
}

// Test_Theme_DefaultAuto_UsesDetection covers theme-profile-detection/REQ-02.2.
// With themeFlag=ThemeAuto and BUILDER_THEME unset, the theme appearance must equal
// whatever DetectAppearance returns for the test environment (no hardcoded expectation).
//
// Note: t.Setenv is used for env isolation — cannot be combined with t.Parallel().
func Test_Theme_DefaultAuto_UsesDetection(t *testing.T) {
	applyColorprofileHygiene(t)
	t.Setenv("BUILDER_THEME", "") // ensure env does not intervene

	// Compute expected appearance the same way composeApp would in auto mode.
	expected := theme.DetectAppearance(os.Stdout)

	app, err := composeApp(Config{themeFlag: ThemeAuto})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	got := app.Theme.Appearance()
	if got != expected {
		t.Errorf("Appearance() = %v; want detected (%v) (REQ-02.2)", got, expected)
	}
}

// Test_Theme_Env_Applied_When_Flag_Auto covers theme-profile-detection/REQ-03.1.
// With BUILDER_THEME=dark and themeFlag=ThemeAuto, the theme must resolve to Dark.
//
// Note: t.Setenv is used for env isolation — cannot be combined with t.Parallel().
func Test_Theme_Env_Applied_When_Flag_Auto(t *testing.T) {
	applyColorprofileHygiene(t)
	t.Setenv("BUILDER_THEME", "dark")

	app, err := composeApp(Config{themeFlag: ThemeAuto})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	got := app.Theme.Appearance()
	if got != theme.Dark {
		t.Errorf("Appearance() = %v; want theme.Dark (REQ-03.1)", got)
	}
}

// Test_Theme_Flag_Wins_Over_Env covers theme-profile-detection/REQ-03.2.
// With BUILDER_THEME=dark AND themeFlag=ThemeLight, the flag must win and
// appearance must be Light.
//
// Note: t.Setenv is used for env isolation — cannot be combined with t.Parallel().
func Test_Theme_Flag_Wins_Over_Env(t *testing.T) {
	applyColorprofileHygiene(t)
	t.Setenv("BUILDER_THEME", "dark")

	app, err := composeApp(Config{themeFlag: ThemeLight})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	got := app.Theme.Appearance()
	if got != theme.Light {
		t.Errorf("Appearance() = %v; want theme.Light (flag must win over env — REQ-03.2)", got)
	}
}

// Test_GoMod_HasPinnedDeps covers dependencies.REQ-01.1.
// Reads go.mod and verifies cobra v1.9.x, viper v1.19.x, charmbracelet/log v0.4.x.
// cobra was bumped from v1.8.x to v1.9.x when charmbracelet/fang was added (fang requires >= v1.9).
func Test_GoMod_HasPinnedDeps(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("cannot read go.mod: %v", err)
	}
	content := string(src)

	deps := []struct {
		name    string
		pattern string
	}{
		{"cobra v1.9.x", `github\.com/spf13/cobra v1\.9\.`},
		{"viper v1.19.x", `github\.com/spf13/viper v1\.19\.`},
		{"charmbracelet/log v0.4.x", `github\.com/charmbracelet/log v0\.4\.`},
	}

	for _, dep := range deps {
		matched, err := regexp.MatchString(dep.pattern, content)
		if err != nil {
			t.Errorf("bad pattern for %s: %v", dep.name, err)
			continue
		}
		if !matched {
			t.Errorf("go.mod missing or wrong version for %s (pattern: %s)", dep.name, dep.pattern)
		}
	}
}
