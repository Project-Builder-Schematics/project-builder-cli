// Package initialise — render_test.go covers renderResult via outputtest.Spy.
//
// ADR-04: feature handler tests inject outputtest.Spy and assert (method, args)
// — NOT bytes. Adapter golden tests live in output/themed/themed_test.go.
//
// REQ coverage:
//   - output-port/REQ-05.2 (theme flag drives real output)
//   - output-discipline/REQ-03.1 (clean tree — init side: no fmt.Fprint* in render.go)
//   - REQ-DR-03: dry-run pretty output begins with "DRY RUN — no files written"
//   - REQ-MCP-02: MCP instructions printed after install when MCP=yes
//   - REQ-JO-01: --json selects the JSON renderer (bypasses Output)
package initialise

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output/outputtest"
)

// noopWriter is a minimal io.Writer that discards all bytes (for JSON path tests).
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }

// Test_RenderResult_DryRun_EmitsHeadingAndNewline verifies that renderResult
// in dry-run mode calls Heading("DRY RUN — no files written") then Newline.
// REQ-DR-03.
func Test_RenderResult_DryRun_EmitsHeadingAndNewline(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun: true,
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Heading", "DRY RUN — no files written")
	spy.AssertCalled(t, "Newline")
}

// Test_RenderResult_DryRun_CreateOp_EmitsBody verifies create_file planned op
// is emitted as a Body call with the formatted path string.
func Test_RenderResult_DryRun_CreateOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun: true,
		PlannedOps: []PlannedOp{
			{Op: "create_file", Path: "/tmp/foo/project-builder.json"},
		},
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Body", "  Would create: /tmp/foo/project-builder.json")
}

// Test_RenderResult_DryRun_AppendOp_EmitsBody verifies append_marker planned
// op is emitted as a Body call.
func Test_RenderResult_DryRun_AppendOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun: true,
		PlannedOps: []PlannedOp{
			{Op: "append_marker", Path: "/tmp/foo/AGENTS.md"},
		},
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Body", "  Would append: /tmp/foo/AGENTS.md")
}

// Test_RenderResult_DryRun_ModifyOp_EmitsBody verifies modify_devdep planned
// op is emitted as a Body call with path + details.
func Test_RenderResult_DryRun_ModifyOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun: true,
		PlannedOps: []PlannedOp{
			{Op: "modify_devdep", Path: "/tmp/foo/package.json", Details: "@pbuilder/sdk"},
		},
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Body", "  Would modify: /tmp/foo/package.json (@pbuilder/sdk)")
}

// Test_RenderResult_DryRun_InstallOp_EmitsBody verifies install_package planned
// op is emitted as a Body call.
func Test_RenderResult_DryRun_InstallOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun: true,
		PlannedOps: []PlannedOp{
			{Op: "install_package", Details: "@pbuilder/sdk"},
		},
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Body", "  Would install: @pbuilder/sdk")
}

// Test_RenderResult_DryRun_MCPOp_EmitsBody verifies mcp_setup_offered planned
// op is emitted as a Body call.
func Test_RenderResult_DryRun_MCPOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun: true,
		PlannedOps: []PlannedOp{
			{Op: "mcp_setup_offered"},
		},
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Body", "  Would offer:   MCP server setup instructions")
}

// Test_RenderResult_RealWrite_EmitsHeadingAndPaths verifies that renderResult
// in real-write mode calls Heading with the directory announcement and Path
// for each created file.
func Test_RenderResult_RealWrite_EmitsHeadingAndPaths(t *testing.T) {
	t.Parallel()

	dir := "/tmp/my-workspace"
	spy := outputtest.New()
	result := InitResult{
		DryRun:    false,
		Directory: dir,
		OutputsCreated: []string{
			"/tmp/my-workspace/project-builder.json",
			"/tmp/my-workspace/schematics/.gitkeep",
		},
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Heading", "Initialising Project Builder workspace in "+dir+" ...")
	spy.AssertCalledWith(t, "Path", "/tmp/my-workspace/project-builder.json")
	spy.AssertCalledWith(t, "Path", "/tmp/my-workspace/schematics/.gitkeep")
}

// Test_RenderResult_RealWrite_Installed_EmitsSuccess verifies that a completed
// install emits a Success call with package manager name.
func Test_RenderResult_RealWrite_Installed_EmitsSuccess(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun:         false,
		Directory:      "/tmp/x",
		PackageManager: PMNpm,
		Installed:      true,
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Success", "Installing @pbuilder/sdk via npm ... done.")
}

// Test_RenderResult_RealWrite_Ready_EmitsSuccessAndHint verifies that the
// "Project Builder is ready" message emits both Success and Hint.
func Test_RenderResult_RealWrite_Ready_EmitsSuccessAndHint(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun:    false,
		Directory: "/tmp/x",
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalledWith(t, "Success", "Project Builder is ready.")
	spy.AssertCalledWith(t, "Hint", "Try: builder add <name>")
}

// Test_RenderResult_RealWrite_MCPSetupOffered_EmitsNewlineAndBody verifies
// that MCPSetupOffered=true results in Newline + Body(mcpInstructions).
// REQ-MCP-02: MCP instructions are printed after install when MCP=yes (real mode).
func Test_RenderResult_RealWrite_MCPSetupOffered_EmitsNewlineAndBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := InitResult{
		DryRun:          false,
		Directory:       "/tmp/x",
		MCPSetupOffered: true,
	}
	if err := renderResult(spy, noopWriter{}, result, false); err != nil {
		t.Fatalf("renderResult: unexpected error: %v", err)
	}

	spy.AssertCalled(t, "Newline")
	spy.AssertCalledWith(t, "Body", mcpInstructions)
}

// Test_RenderResult_JSON_ProducesValidEnvelope verifies that JSON mode writes
// a valid JSON envelope and does NOT call any Output methods.
// REQ-JO-01: --json selects the JSON renderer.
func Test_RenderResult_JSON_ProducesValidEnvelope(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	var buf bytes.Buffer
	result := InitResult{
		DryRun:    true,
		Directory: "/tmp/x",
		PlannedOps: []fswriter.PlannedOp{
			{Op: "create_file", Path: "/tmp/x/project-builder.json"},
		},
	}

	if err := renderResult(spy, &buf, result, true); err != nil {
		t.Fatalf("renderResult(json=true): unexpected error: %v", err)
	}

	// JSON path must NOT call any Output methods.
	if calls := spy.Calls(); len(calls) > 0 {
		t.Errorf("renderResult(json=true): expected no Output calls; got %d: %v", len(calls), calls)
	}

	// JSON output must be a valid JSON object.
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v — raw: %s", err, buf.Bytes())
	}
	if parsed["directory"] == nil {
		t.Errorf("JSON envelope missing 'directory'; got: %s", buf.Bytes())
	}
	if _, ok := parsed["dry_run"]; !ok {
		t.Errorf("JSON envelope missing 'dry_run'; got: %s", buf.Bytes())
	}
}
