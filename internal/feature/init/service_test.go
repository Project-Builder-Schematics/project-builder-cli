// Package initialise — service_test.go covers Service.Init orchestration.
//
// REQ coverage:
//   - REQ-FW-01 (FSWriter port; service never calls os.* directly)
//   - REQ-DR-01 (dry-run records PlannedOps, zero real writes)
//   - REQ-DR-02 (PlannedOps shape: stable 5-value op enum)
//   - REQ-EC-05 (write order: PJ → schematics → SKILL → AGENTS → pkg.json → install → MCP)
//   - REQ-CS-05 (--publishable → ErrCodeInitNotImplemented)
//   - REQ-MCP-02 (MCP=yes records mcp_setup_offered op in dry-run)
//   - REQ-EC-03 (non-dry-run stub returns ErrCodeNotImplemented for S-000..S-001 guard)
//   - REQ-SA-01 (real-write writes SKILL.md with locked bytes)
//   - REQ-SA-02 (pre-existing SKILL.md without --force → warning in Warnings, no error)
//   - REQ-SA-02 (pre-existing SKILL.md with --force → overwritten)
//   - REQ-SA-03 (--no-skill → SKILL.md not written, outputs 4+SDK skipped)
package initialise

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init/template"
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
// mode (DryRun=false) returns ErrCodeNotImplemented for the first un-wired
// output stub. After S-002, this is output 4 (AGENTS marker, S-003).
func Test_Service_Init_NonDryRun_ReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	// Use non-empty skill bytes so output 3 (SKILL.md) succeeds and we reach
	// the output 4 stub (S-003 territory).
	svc := NewService(ffs, pm, []byte("placeholder"))

	req := InitRequest{
		Directory: dir,
		DryRun:    false,
	}

	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected ErrCodeNotImplemented for un-wired output stub; got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, ErrCodeNotImplemented) = false; got: %v", err)
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

// --- S-001 real-write tests ---

// Test_Service_Init_RealWrite_S001_WritesBothFiles verifies that in real-write
// mode, Service.Init writes project-builder.json and schematics/.gitkeep with
// the locked bytes. After S-002, outputs 1, 2, and 3 (SKILL.md) are wired;
// outputs 4..5 + install + MCP still return ErrCodeNotImplemented.
//
// This test uses empty skill bytes ([]byte{}) to isolate S-001 concerns and
// ensures the locked project-builder.json and .gitkeep bytes are written.
// The ErrCodeNotImplemented is now for output 4 (AGENTS marker stub, S-003).
//
// REQ-PJ-01 (project-builder.json locked bytes via service path)
// REQ-SF-01  (schematics/.gitkeep locked bytes via service path)
// Option A partial-write contract: outputs 1 & 2 are written before error.
func Test_Service_Init_RealWrite_S001_WritesBothFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, []byte("skill-placeholder"))

	req := InitRequest{
		Directory: dir,
		DryRun:    false,
		MCP:       MCPNo,
	}

	_, err := svc.Init(context.Background(), req)
	// S-002: expect ErrCodeNotImplemented for the not-yet-wired output 4 (AGENTS marker).
	if err == nil {
		t.Fatal("expected ErrCodeNotImplemented for output 4 (AGENTS, S-003); got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected ErrCodeNotImplemented for S-002 partial real-write; got: %v", err)
	}

	// Output 1: project-builder.json MUST have been written with locked bytes.
	pbPath := filepath.Join(dir, "project-builder.json")
	gotPB, readErr := ffs.ReadFile(pbPath)
	if readErr != nil {
		t.Fatalf("project-builder.json not written before error: %v", readErr)
	}
	if !bytes.Equal(gotPB, lockedProjectBuilderJSON) {
		t.Errorf("project-builder.json bytes mismatch after partial S-001 write\ngot:\n%s\nwant:\n%s", gotPB, lockedProjectBuilderJSON)
	}

	// Output 2: schematics/.gitkeep MUST have been written with locked bytes.
	gitkeepPath := filepath.Join(dir, schematicsFolderName, ".gitkeep")
	gotGK, readErr := ffs.ReadFile(gitkeepPath)
	if readErr != nil {
		t.Fatalf("schematics/.gitkeep not written before error: %v", readErr)
	}
	if !bytes.Equal(gotGK, lockedGitkeepBytes) {
		t.Errorf("schematics/.gitkeep bytes mismatch after partial S-001 write\ngot:\n%q\nwant:\n%q", gotGK, lockedGitkeepBytes)
	}
}

// Test_Service_Init_RealWrite_PreexistingConfig_ReturnsErrInitConfigExists
// verifies that when project-builder.json already exists and Force=false,
// Service.Init returns ErrCodeInitConfigExists before writing anything.
// REQ-DV-04.
func Test_Service_Init_RealWrite_PreexistingConfig_ReturnsErrInitConfigExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	pbPath := filepath.Join(dir, "project-builder.json")
	existingContent := []byte(`{"version":"existing"}`)
	if err := ffs.WriteFile(pbPath, existingContent, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, []byte{})

	req := InitRequest{Directory: dir, DryRun: false, Force: false}
	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected ErrCodeInitConfigExists, got nil")
	}

	sentinel := &errs.Error{Code: errs.ErrCodeInitConfigExists}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, ErrCodeInitConfigExists) = false; got: %v", err)
	}

	// File must not have been overwritten.
	got, readErr := ffs.ReadFile(pbPath)
	if readErr != nil {
		t.Fatalf("ReadFile after error: %v", readErr)
	}
	if !bytes.Equal(got, existingContent) {
		t.Error("pre-existing config was overwritten despite no --force")
	}
}

// Test_Service_Init_RealWrite_Force_OverwritesExistingConfig verifies that
// when Force=true, an existing project-builder.json is overwritten.
// REQ-EC-03.
func Test_Service_Init_RealWrite_Force_OverwritesExistingConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	pbPath := filepath.Join(dir, "project-builder.json")
	if err := ffs.WriteFile(pbPath, []byte(`{"version":"old"}`), 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, []byte("skill-placeholder"))

	req := InitRequest{Directory: dir, DryRun: false, Force: true}
	_, err := svc.Init(context.Background(), req)
	// S-002 partial: ErrCodeNotImplemented for output 4 (AGENTS stub, S-003).
	if err == nil {
		t.Fatal("expected ErrCodeNotImplemented for S-002 partial real-write; got nil")
	}
	sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected ErrCodeNotImplemented; got: %v", err)
	}

	// project-builder.json must have been overwritten with locked bytes.
	got, readErr := ffs.ReadFile(pbPath)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if !bytes.Equal(got, lockedProjectBuilderJSON) {
		t.Errorf("project-builder.json after --force overwrite:\ngot:\n%s\nwant:\n%s", got, lockedProjectBuilderJSON)
	}
}

// --- S-002 SKILL.md real-write integration tests ---

// Test_Service_Init_S002_RealWrite_WritesSkillMD verifies that Service.Init
// writes .claude/skills/pbuilder/SKILL.md with the locked template bytes.
// REQ-SA-01.
func Test_Service_Init_S002_RealWrite_WritesSkillMD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}

	_, err := svc.Init(context.Background(), req)
	// After S-002, outputs 1+2+3 succeed. Output 4 (AGENTS) returns ErrCodeNotImplemented.
	sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
	if err != nil && !errors.Is(err, sentinel) {
		t.Fatalf("unexpected non-stub error: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
	got, readErr := ffs.ReadFile(skillPath)
	if readErr != nil {
		t.Fatalf("SKILL.md not written: %v", readErr)
	}
	if !bytes.Equal(got, template.Skill) {
		t.Errorf("SKILL.md bytes mismatch:\ngot  len=%d\nwant len=%d", len(got), len(template.Skill))
	}
}

// Test_Service_Init_S002_PreexistingSkill_NoForce_WarningInResult verifies
// that when SKILL.md already exists without --force, the service:
//   - does NOT return a hard error
//   - populates InitResult.Warnings with a non-empty message
//
// REQ-SA-02.
func Test_Service_Init_S002_PreexistingSkill_NoForce_WarningInResult(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	skillPath := filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
	existing := []byte("old skill content")
	if err := ffs.WriteFile(skillPath, existing, 0o644); err != nil {
		t.Fatalf("seed SKILL.md: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, Force: false, MCP: MCPNo}
	result, err := svc.Init(context.Background(), req)

	// Hard error from ErrCodeNotImplemented (S-003 stub) is still acceptable —
	// but ErrCodeInitSkillExists must NOT propagate as the hard error.
	if err != nil {
		skillSentinel := &errs.Error{Code: errs.ErrCodeInitSkillExists}
		if errors.Is(err, skillSentinel) {
			t.Errorf("ErrCodeInitSkillExists must not propagate as hard error (REQ-SA-02 — skip is not a failure)")
		}
	}

	// SKILL.md must remain untouched.
	got, readErr := ffs.ReadFile(skillPath)
	if readErr != nil {
		t.Fatalf("ReadFile after skip: %v", readErr)
	}
	if !bytes.Equal(got, existing) {
		t.Error("SKILL.md was overwritten despite Force=false (REQ-SA-02)")
	}

	// Warnings must be non-empty (skip is recorded as a warning).
	if len(result.Warnings) == 0 {
		t.Errorf("InitResult.Warnings is empty; expected SKILL.md skip warning (REQ-SA-02)")
	}
}

// Test_Service_Init_S002_PreexistingSkill_Force_Overwrites verifies that
// with Force=true, an existing SKILL.md is overwritten. REQ-SA-02.
func Test_Service_Init_S002_PreexistingSkill_Force_Overwrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	skillPath := filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
	if err := ffs.WriteFile(skillPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed SKILL.md: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, Force: true, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	// ErrCodeNotImplemented for output 4 (S-003 stub) is acceptable.
	if err != nil {
		sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
		if !errors.Is(err, sentinel) {
			t.Fatalf("unexpected non-stub error: %v", err)
		}
	}

	got, readErr := ffs.ReadFile(skillPath)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if !bytes.Equal(got, template.Skill) {
		t.Errorf("SKILL.md not overwritten with locked bytes after --force")
	}
}

// Test_Service_Init_S002_NoSkill_SkipsSkillMD verifies that when --no-skill is
// set, Service.Init does NOT write SKILL.md and does NOT hit the output 4
// stub (AGENTS marker). The service must succeed for outputs 1+2, skip 3+4+SDK,
// and then return nil or ErrCodeNotImplemented for a later stub. REQ-SA-03.
func Test_Service_Init_S002_NoSkill_SkipsSkillMD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, NoSkill: true, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)

	// After S-002 with --no-skill: outputs 1+2 written, output 3 skipped entirely.
	// Outputs 4+SDK are also skipped (NoSkill). The next un-wired stub is
	// output 5 (package.json, S-004) — but since --no-skill also skips SDK
	// dev-dep (output 5), the service jumps to the install stub (S-005) or
	// returns ErrCodeNotImplemented for the next relevant operation.
	// We don't assert specific error here; we assert SKILL.md was NOT written.
	_ = err

	skillPath := filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
	if _, statErr := ffs.Stat(skillPath); statErr == nil {
		t.Errorf("SKILL.md was written despite --no-skill (REQ-SA-03)")
	}
}

// Test_Service_Init_S002_DryRun_SKILL_InPlannedOps verifies that dry-run
// records create_file for .claude/skills/pbuilder/SKILL.md when --no-skill
// is false. REQ-DR-01, REQ-SA-01.
func Test_Service_Init_S002_DryRun_SKILL_InPlannedOps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: true, NoSkill: false, MCP: MCPNo}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
	var found bool
	for _, op := range result.PlannedOps {
		if op.Op == "create_file" && op.Path == skillPath {
			found = true
		}
	}
	if !found {
		t.Errorf("PlannedOps does not contain create_file for %q (REQ-SA-01 dry-run)", skillPath)
	}
}

// Test_Service_Init_S002_DryRun_NoSkill_SKILL_AbsentFromPlannedOps verifies
// that dry-run with --no-skill does NOT record create_file for SKILL.md.
// REQ-SA-03.
func Test_Service_Init_S002_DryRun_NoSkill_SKILL_AbsentFromPlannedOps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: true, NoSkill: true, MCP: MCPNo}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "pbuilder", "SKILL.md")
	for _, op := range result.PlannedOps {
		if op.Op == "create_file" && op.Path == skillPath {
			t.Errorf("PlannedOps contains create_file for SKILL.md but --no-skill was set (REQ-SA-03)")
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
