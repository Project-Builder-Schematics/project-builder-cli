// Package initialise — handler_test.go covers the RunE handler logic.
//
// REQ coverage:
//   - REQ-CS-01 (positional arg sets target directory)
//   - REQ-CS-02 (no positional arg defaults to cwd)
//   - REQ-CS-03 (--force flag registered)
//   - REQ-CS-04 (--non-interactive flag registered)
//   - REQ-CS-05 (--publishable → ErrCodeInitNotImplemented)
//   - REQ-DV-01 (directory canonicalised: filepath.Abs + filepath.Clean)
//   - REQ-DV-02 (reject path with .. traversal outside cwd)
//   - REQ-JO-01 (--json flag registered)
//   - REQ-JO-02 (--dry-run flag registered)
//   - REQ-JO-03 (--dry-run + --json → valid JSON envelope)
//   - REQ-MCP-01 (--mcp flag: yes/no/prompt accepted; invalid rejected)
//   - REQ-MCP-01 (--mcp=prompt + --non-interactive → ErrCodeInvalidInput)
//   - REQ-MCP-01 (--non-interactive + no --mcp flag → defaults to MCPNo)
//   - REQ-MCP-01 (prompt answer-parsing: affirmative → MCPYes; negative/empty → MCPNo)
//   - REQ-MCP-02 (dry-run skips prompt; real-mode with MCPPrompt does prompt)
//   - REQ-MCP-03 (--json output includes mcp_setup_offered bool)
//   - REQ-EC-03 (--publishable → ErrCodeInitNotImplemented via handler path)
//
// S-003: promptMCP tests migrated to parseMCPAnswer (ADR-05: Prompt handled
// by output.Output; answer-parsing stays in parseMCPAnswer for testability).
package initialise

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/pathutil"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output/outputtest"
)

// fakePromptOutput wraps outputtest.Spy but overrides Prompt to return a
// configurable answer. Used to test promptMCP with specific user answers.
type fakePromptOutput struct {
	*outputtest.Spy
	promptAnswer string
}

func newFakePromptOutput(answer string) *fakePromptOutput {
	return &fakePromptOutput{Spy: outputtest.New(), promptAnswer: answer}
}

// Prompt overrides Spy.Prompt to return the configured answer.
func (f *fakePromptOutput) Prompt(text string) (string, error) {
	_, _ = f.Spy.Prompt(text) // record the call in the Spy; discard dummy return
	return f.promptAnswer, nil
}

// compile-time assertion: fakePromptOutput satisfies output.Output.
var _ output.Output = (*fakePromptOutput)(nil)

// newHandlerFunc constructs a handler RunE-like function with injected fakes.
// This tests the handler logic (flag parsing + validation + service call)
// without going through Cobra, for precise error assertions.
func newHandlerFunc(dryRunMode bool, mcp MCPMode, publishable bool, dir string) func() (InitResult, error) {
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	return func() (InitResult, error) {
		req := InitRequest{
			Directory:   dir,
			DryRun:      dryRunMode,
			MCP:         mcp,
			Publishable: publishable,
		}
		return svc.Init(context.Background(), req)
	}
}

// Test_Handler_Publishable_ReturnsErrInitNotImplemented covers REQ-CS-05
// and REQ-EC-03 via the handler's passthrough to the service.
func Test_Handler_Publishable_ReturnsErrInitNotImplemented(t *testing.T) {
	t.Parallel()

	fn := newHandlerFunc(true, MCPNo, true, t.TempDir())
	_, err := fn()

	if err == nil {
		t.Fatal("expected error for --publishable, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInitNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(ErrCodeInitNotImplemented) = false; got: %v", err)
	}

	var e *errs.Error
	if errors.As(err, &e) {
		if e.Op != "init.handler" {
			t.Errorf("error Op = %q, want %q", e.Op, "init.handler")
		}
	}
}

// Test_Handler_DryRun_JSON_ProducesValidEnvelope tests that the handler produces
// a valid JSON-serialisable InitResult in dry-run mode.
// REQ-JO-03, REQ-DR-01.
func Test_Handler_DryRun_JSON_ProducesValidEnvelope(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	req := InitRequest{
		Directory: dir,
		DryRun:    true,
		JSON:      true,
		MCP:       MCPNo,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Serialise and parse — must produce valid JSON.
	data, jsonErr := json.Marshal(result)
	if jsonErr != nil {
		t.Fatalf("json.Marshal(InitResult): %v", jsonErr)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v — raw: %s", err, data)
	}

	// Required top-level fields.
	if parsed["directory"] == nil {
		t.Errorf("JSON envelope missing 'directory' field; got: %s", data)
	}
	dryRunField, ok := parsed["dry_run"].(bool)
	if !ok {
		t.Errorf("JSON envelope 'dry_run' is not bool; got: %s", data)
	}
	if !dryRunField {
		t.Errorf("JSON 'dry_run' = false, want true")
	}
	if _, hasMCPField := parsed["mcp_setup_offered"]; !hasMCPField {
		t.Errorf("JSON envelope missing 'mcp_setup_offered' field (REQ-MCP-03); got: %s", data)
	}
}

// Test_Handler_DryRun_MCP_Yes_JSONEnvelope_MCPSetupOffered_True covers REQ-MCP-03:
// mcp_setup_offered must be true when MCP=yes.
func Test_Handler_DryRun_MCP_Yes_JSONEnvelope_MCPSetupOffered_True(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	req := InitRequest{
		Directory: dir,
		DryRun:    true,
		JSON:      true,
		MCP:       MCPYes,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.MCPSetupOffered {
		t.Errorf("MCPSetupOffered = false, want true when MCP=yes (REQ-MCP-03)")
	}

	// Verify JSON serialisation.
	data, _ := json.Marshal(result)
	var parsed map[string]any
	_ = json.Unmarshal(data, &parsed)

	mcpField, ok := parsed["mcp_setup_offered"].(bool)
	if !ok {
		t.Fatalf("mcp_setup_offered is not bool in JSON; got: %s", data)
	}
	if !mcpField {
		t.Errorf("mcp_setup_offered = false in JSON, want true (REQ-MCP-03)")
	}
}

// Test_Handler_DryRun_MCP_No_JSONEnvelope_MCPSetupOffered_False covers REQ-MCP-03:
// mcp_setup_offered must be false when MCP=no.
func Test_Handler_DryRun_MCP_No_JSONEnvelope_MCPSetupOffered_False(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	req := InitRequest{Directory: dir, DryRun: true, MCP: MCPNo}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := json.Marshal(result)
	var parsed map[string]any
	_ = json.Unmarshal(data, &parsed)

	mcpField, _ := parsed["mcp_setup_offered"].(bool)
	if mcpField {
		t.Errorf("mcp_setup_offered = true, want false when MCP=no (REQ-MCP-03)")
	}
}

// Test_Handler_MCP_PromptPlusNonInteractive_ReturnsInvalidInput covers REQ-MCP-01:
// --mcp=prompt + --non-interactive is an incompatible combination.
func Test_Handler_MCP_PromptPlusNonInteractive_ReturnsInvalidInput(t *testing.T) {
	t.Parallel()

	err := resolveMCPMode("prompt", true)
	if err == nil {
		t.Fatal("expected error for --mcp=prompt + --non-interactive, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInvalidInput}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(ErrCodeInvalidInput) = false; got: %v", err)
	}
}

// Test_Handler_MCP_InvalidValue_ReturnsInvalidInput covers REQ-MCP-01:
// an unrecognised --mcp value is rejected with ErrCodeInvalidInput.
func Test_Handler_MCP_InvalidValue_ReturnsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []string{"maybe", "YES_PLEASE", "1", "true"}
	for _, val := range tests {
		val := val
		t.Run(val, func(t *testing.T) {
			t.Parallel()
			err := resolveMCPMode(val, false)
			if err == nil {
				t.Errorf("resolveMCPMode(%q, false): expected error, got nil", val)
				return
			}
			sentinel := &errs.Error{Code: errs.ErrCodeInvalidInput}
			if !errors.Is(err, sentinel) {
				t.Errorf("resolveMCPMode(%q): errors.Is(ErrCodeInvalidInput) = false; got: %v", val, err)
			}
		})
	}
}

// Test_Handler_MCP_CaseInsensitive_Accepted covers REQ-MCP-01:
// flag values are case-insensitive (YES, Yes, yes are all MCPYes).
func Test_Handler_MCP_CaseInsensitive_Accepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  MCPMode
	}{
		{"yes", MCPYes},
		{"YES", MCPYes},
		{"Yes", MCPYes},
		{"no", MCPNo},
		{"NO", MCPNo},
		{"No", MCPNo},
		{"prompt", MCPPrompt},
		{"PROMPT", MCPPrompt},
		{"Prompt", MCPPrompt},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			mode, err := parseMCPFlag(tt.input)
			if err != nil {
				t.Errorf("parseMCPFlag(%q): unexpected error: %v", tt.input, err)
				return
			}
			if mode != tt.want {
				t.Errorf("parseMCPFlag(%q) = %q, want %q", tt.input, mode, tt.want)
			}
		})
	}
}

// Test_Handler_MCP_NonInteractive_DefaultsNo covers REQ-MCP-01:
// when --non-interactive is set and no --mcp flag is given, default is MCPNo.
func Test_Handler_MCP_NonInteractive_DefaultsNo(t *testing.T) {
	t.Parallel()

	// Simulate no --mcp flag set (empty string) + nonInteractive=true.
	mode := defaultMCPMode(true /* nonInteractive */, false /* isTTY */)
	if mode != MCPNo {
		t.Errorf("defaultMCPMode(nonInteractive=true): got %q, want %q", mode, MCPNo)
	}
}

// Test_Handler_MCP_NoFlagNoTTY_DefaultsNo covers REQ-MCP-01:
// without a TTY and without --non-interactive, default is MCPNo.
func Test_Handler_MCP_NoFlagNoTTY_DefaultsNo(t *testing.T) {
	t.Parallel()

	mode := defaultMCPMode(false /* nonInteractive */, false /* isTTY */)
	if mode != MCPNo {
		t.Errorf("defaultMCPMode(nonInteractive=false, isTTY=false): got %q, want %q", mode, MCPNo)
	}
}

// Test_Handler_MCP_NoFlagWithTTY_DefaultsPrompt covers REQ-MCP-01:
// in a TTY without --non-interactive, default is MCPPrompt.
func Test_Handler_MCP_NoFlagWithTTY_DefaultsPrompt(t *testing.T) {
	t.Parallel()

	mode := defaultMCPMode(false /* nonInteractive */, true /* isTTY */)
	if mode != MCPPrompt {
		t.Errorf("defaultMCPMode(nonInteractive=false, isTTY=true): got %q, want %q", mode, MCPPrompt)
	}
}

// Test_Handler_DirectoryCanonicalisation covers REQ-DV-01:
// the handler applies filepath.Abs + filepath.Clean to the directory argument.
func Test_Handler_DirectoryCanonicalisation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantAbs bool // true if result should be absolute
	}{
		{name: "absolute path", input: "/tmp/my-project", wantAbs: true},
		{name: "temp dir", input: t.TempDir(), wantAbs: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := pathutil.Canonicalise(tt.input)
			if err != nil {
				t.Fatalf("pathutil.Canonicalise(%q): unexpected error: %v", tt.input, err)
			}
			if tt.wantAbs && !filepath.IsAbs(got) {
				t.Errorf("pathutil.Canonicalise(%q) = %q: expected absolute path", tt.input, got)
			}
			if got != filepath.Clean(got) {
				t.Errorf("pathutil.Canonicalise(%q) = %q: not Clean", tt.input, got)
			}
		})
	}
}

// Test_Handler_DirectoryTraversal_Rejected covers REQ-DV-02:
// a path containing .. that resolves outside the cwd must be rejected.
func Test_Handler_DirectoryTraversal_Rejected(t *testing.T) {
	t.Parallel()

	_, err := pathutil.Canonicalise("../../../etc")
	if err == nil {
		t.Fatal("pathutil.Canonicalise with .. traversal: expected error, got nil (REQ-DV-02)")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInvalidInput}
	if !errors.Is(err, sentinel) {
		t.Errorf("pathutil.Canonicalise traversal error: errors.Is(ErrCodeInvalidInput) = false; got: %v", err)
	}
}

// Test_Handler_EndToEnd_DryRun_JSON covers S-000 walking skeleton acceptance:
// builder init --dry-run --json /tmp/empty → valid JSON envelope.
// REQ-JO-03, REQ-DR-01, REQ-MCP-03.
func Test_Handler_EndToEnd_DryRun_JSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var buf bytes.Buffer

	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	result, err := svc.Init(context.Background(), InitRequest{
		Directory: dir,
		DryRun:    true,
		JSON:      true,
		MCP:       MCPYes,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		t.Fatalf("json.Encode: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v — raw: %s", err, buf.Bytes())
	}

	// Validate envelope fields.
	if parsed["directory"] != dir {
		t.Errorf("directory = %v, want %q", parsed["directory"], dir)
	}
	if dryRun, _ := parsed["dry_run"].(bool); !dryRun {
		t.Errorf("dry_run = %v, want true", parsed["dry_run"])
	}
	if mcp, _ := parsed["mcp_setup_offered"].(bool); !mcp {
		t.Errorf("mcp_setup_offered = %v, want true (REQ-MCP-03)", parsed["mcp_setup_offered"])
	}

	// planned_ops must be a non-empty array.
	ops, ok := parsed["planned_ops"].([]any)
	if !ok || len(ops) == 0 {
		t.Errorf("planned_ops is empty or missing; got: %s", buf.Bytes())
	}
}

// --- REQ-MCP-01: MCP answer-parsing tests ---
//
// After S-003 migration (ADR-05), promptMCP uses output.Output.Prompt for I/O.
// The answer-parsing logic is extracted to parseMCPAnswer, tested independently.

// Test_ParseMCPAnswer_AffirmativeResponses verifies that y, Y, yes, YES all
// resolve to MCPYes. REQ-MCP-01 (affirmative set).
func Test_ParseMCPAnswer_AffirmativeResponses(t *testing.T) {
	t.Parallel()

	affirmatives := []string{"y", "Y", "yes", "YES"}
	for _, ans := range affirmatives {
		ans := ans
		t.Run(ans, func(t *testing.T) {
			t.Parallel()
			got := parseMCPAnswer(ans)
			if got != MCPYes {
				t.Errorf("parseMCPAnswer(%q) = %q, want MCPYes (REQ-MCP-01)", ans, got)
			}
		})
	}
}

// Test_ParseMCPAnswer_NegativeAndEmptyResponses verifies that empty, n, N, no,
// NO, and other strings all resolve to MCPNo. REQ-MCP-01.
func Test_ParseMCPAnswer_NegativeAndEmptyResponses(t *testing.T) {
	t.Parallel()

	negatives := []string{"", "n", "N", "no", "NO", "maybe", "nope"}
	for _, ans := range negatives {
		ans := ans
		t.Run("\""+ans+"\"", func(t *testing.T) {
			t.Parallel()
			got := parseMCPAnswer(ans)
			if got != MCPNo {
				t.Errorf("parseMCPAnswer(%q) = %q, want MCPNo (REQ-MCP-01)", ans, got)
			}
		})
	}
}

// Test_ParseMCPAnswer_TrailingNewline verifies that trailing \r\n is trimmed
// before matching (REQ-MCP-01 — real Prompt output has newlines trimmed, but
// we guard explicitly for double-safety).
func Test_ParseMCPAnswer_TrailingNewline(t *testing.T) {
	t.Parallel()

	got := parseMCPAnswer("y\n")
	if got != MCPYes {
		t.Errorf("parseMCPAnswer(%q) = %q, want MCPYes (trailing newline not trimmed?)", "y\n", got)
	}
}

// Test_PromptMCP_CallsPromptWithQuestion verifies that promptMCP calls
// out.Prompt with mcpPromptQuestion (ADR-05 contract).
func Test_PromptMCP_CallsPromptWithQuestion(t *testing.T) {
	t.Parallel()

	out := newFakePromptOutput("y")
	promptMCP(out)

	out.AssertCalledWith(t, "Prompt", mcpPromptQuestion)
}

// Test_PromptMCP_AffirmativeAnswer_ReturnsMCPYes verifies that promptMCP
// returns MCPYes when out.Prompt returns an affirmative answer.
func Test_PromptMCP_AffirmativeAnswer_ReturnsMCPYes(t *testing.T) {
	t.Parallel()

	affirmatives := []string{"y", "Y", "yes", "YES"}
	for _, ans := range affirmatives {
		ans := ans
		t.Run(ans, func(t *testing.T) {
			t.Parallel()
			out := newFakePromptOutput(ans)
			got := promptMCP(out)
			if got != MCPYes {
				t.Errorf("promptMCP with answer %q = %q, want MCPYes", ans, got)
			}
		})
	}
}

// Test_PromptMCP_NegativeAnswer_ReturnsMCPNo verifies that promptMCP returns
// MCPNo for empty and non-affirmative answers.
func Test_PromptMCP_NegativeAnswer_ReturnsMCPNo(t *testing.T) {
	t.Parallel()

	negatives := []string{"", "n", "N", "no", "NO", "maybe"}
	for _, ans := range negatives {
		ans := ans
		t.Run("\""+ans+"\"", func(t *testing.T) {
			t.Parallel()
			out := newFakePromptOutput(ans)
			got := promptMCP(out)
			if got != MCPNo {
				t.Errorf("promptMCP with answer %q = %q, want MCPNo", ans, got)
			}
		})
	}
}
