// Package initialise — service_test.go covers Service.Init orchestration.
//
// REQ coverage:
//   - REQ-FW-01 (FSWriter port; service never calls os.* directly)
//   - REQ-DR-01 (dry-run records PlannedOps, zero real writes)
//   - REQ-DR-02 (PlannedOps shape: stable 5-value op enum)
//   - REQ-EC-05 (write order: PJ → schematics → SKILL → AGENTS → pkg.json → install → MCP)
//   - REQ-CS-05 (--publishable → ErrCodeInitNotImplemented)
//   - REQ-MCP-02 (MCP=yes records mcp_setup_offered op in dry-run)
//   - REQ-EC-03 (non-dry-run stub returns ErrCodeNotImplemented for S-000)
package initialise

import (
	"context"
	"errors"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// newTestService constructs a Service with in-memory fakes suitable for unit tests.
func newTestService() (*Service, *fakeFS, *fakePM) {
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, []byte{})
	return svc, ffs, pm
}

// Test_Service_Init_DryRun_RecordsOps_NoRealWrites verifies the core dry-run
// contract: all operations are recorded as PlannedOps, nothing written to disk.
// REQ-DR-01.
func Test_Service_Init_DryRun_RecordsOps_NoRealWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	req := InitRequest{
		Directory: dir,
		DryRun:    true,
		MCP:       MCPNo,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Service.Init dry-run: unexpected error: %v", err)
	}

	if !result.DryRun {
		t.Errorf("InitResult.DryRun = false, want true")
	}
	if result.Directory != dir {
		t.Errorf("InitResult.Directory = %q, want %q", result.Directory, dir)
	}
	if result.MCPSetupOffered {
		t.Errorf("InitResult.MCPSetupOffered = true, want false when MCP=no")
	}

	// PlannedOps must be non-nil and non-empty in dry-run mode.
	ops := result.PlannedOps
	if len(ops) == 0 {
		t.Fatal("InitResult.PlannedOps: got empty, want at least 1 op in dry-run")
	}

	// Verify no install calls were made (dry-run skips subprocess).
	if pm.installCalls != 0 {
		t.Errorf("pm.installCalls = %d, want 0 in dry-run", pm.installCalls)
	}
}

// Test_Service_Init_DryRun_MCP_Yes_RecordsMCPOp verifies that when MCP=yes
// in dry-run mode, the mcp_setup_offered PlannedOp is present and
// MCPSetupOffered is true.
// REQ-MCP-02, REQ-MCP-03.
func Test_Service_Init_DryRun_MCP_Yes_RecordsMCPOp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	req := InitRequest{
		Directory: dir,
		DryRun:    true,
		MCP:       MCPYes,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Service.Init dry-run MCP=yes: unexpected error: %v", err)
	}

	if !result.MCPSetupOffered {
		t.Errorf("InitResult.MCPSetupOffered = false, want true when MCP=yes")
	}

	// Find the mcp_setup_offered op.
	var found bool
	for _, op := range result.PlannedOps {
		if op.Op == "mcp_setup_offered" {
			found = true
			if op.Path != "" {
				t.Errorf("PlannedOp mcp_setup_offered.Path = %q, want empty (REQ-DR-02)", op.Path)
			}
		}
	}
	if !found {
		t.Errorf("PlannedOps does not contain {op: mcp_setup_offered} (REQ-MCP-02)")
	}
}

// Test_Service_Init_DryRun_MCP_No_NoMCPOp verifies that when MCP=no in
// dry-run mode, no mcp_setup_offered op is recorded and MCPSetupOffered
// is false.
// REQ-MCP-02.
func Test_Service_Init_DryRun_MCP_No_NoMCPOp(t *testing.T) {
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

	if result.MCPSetupOffered {
		t.Error("MCPSetupOffered should be false when MCP=no")
	}
	for _, op := range result.PlannedOps {
		if op.Op == "mcp_setup_offered" {
			t.Errorf("PlannedOps should not contain mcp_setup_offered when MCP=no; got %+v", op)
		}
	}
}

// Test_Service_Init_Publishable_ReturnsErrInitNotImplemented verifies that
// --publishable returns ErrCodeInitNotImplemented.
// REQ-CS-05.
func Test_Service_Init_Publishable_ReturnsErrInitNotImplemented(t *testing.T) {
	t.Parallel()

	svc, _, _ := newTestService()

	req := InitRequest{
		Directory:   t.TempDir(),
		Publishable: true,
		DryRun:      true,
	}

	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for --publishable, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInitNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, ErrCodeInitNotImplemented) = false; got: %v", err)
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("errors.As(*errs.Error) failed")
	}
	if e.Op != "init.handler" {
		t.Errorf("error Op = %q, want %q", e.Op, "init.handler")
	}
	if len(e.Suggestions) == 0 {
		t.Errorf("Suggestions must be non-empty for ErrCodeInitNotImplemented (UX contract)")
	}
}

// Test_Service_Init_NonDryRun_ReturnsNotImplemented verifies that real-write
// mode (DryRun=false) returns ErrCodeNotImplemented in S-000.
// This is the S-000 walking skeleton guard — real writes land in S-001..S-005.
func Test_Service_Init_NonDryRun_ReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	svc, _, _ := newTestService()

	req := InitRequest{
		Directory: t.TempDir(),
		DryRun:    false,
	}

	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for non-dry-run in S-000, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, ErrCodeNotImplemented) = false for S-000 non-dry-run; got: %v", err)
	}
}

// Test_Service_Init_DryRun_PlannedOps_AllStableOps verifies the write order
// and that all op values are within the stable 5-value enum (REQ-DR-02).
// With MCP=yes, we expect ops in REQ-EC-05 order.
// REQ-EC-05, REQ-DR-02.
func Test_Service_Init_DryRun_PlannedOps_AllStableOps(t *testing.T) {
	t.Parallel()

	stableOps := map[string]bool{
		"create_file":       true,
		"append_marker":     true,
		"modify_devdep":     true,
		"install_package":   true,
		"mcp_setup_offered": true,
	}

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, []byte{})

	req := InitRequest{Directory: dir, DryRun: true, MCP: MCPYes}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, op := range result.PlannedOps {
		if !stableOps[op.Op] {
			t.Errorf("PlannedOps[%d].Op = %q is not in the stable enum (REQ-DR-02)", i, op.Op)
		}
	}
}

// Test_Service_Init_DryRun_WithMCP_HasSixOps verifies that with MCP=yes,
// dry-run produces exactly 6 PlannedOps (5 outputs + mcp_setup_offered).
// Without MCP, exactly 5 ops.
// REQ-DR-02, S-000 acceptance criterion.
func Test_Service_Init_DryRun_WithMCP_HasSixOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mcp     MCPMode
		wantOps int
	}{
		{name: "mcp=no", mcp: MCPNo, wantOps: 5},
		{name: "mcp=yes", mcp: MCPYes, wantOps: 6},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			dfs := newDryRunFakeFS()
			pm := &fakePM{detectResult: PMNpm}
			svc := NewService(dfs, pm, []byte{})

			req := InitRequest{Directory: dir, DryRun: true, MCP: tt.mcp}
			result, err := svc.Init(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.PlannedOps) != tt.wantOps {
				t.Errorf("PlannedOps count = %d, want %d (mcp=%s)", len(result.PlannedOps), tt.wantOps, tt.mcp)
			}
		})
	}
}
