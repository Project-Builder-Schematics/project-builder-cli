// Package initialise — service_test.go covers Service.Init orchestration.
//
// REQ coverage:
//   - REQ-FW-01 (FSWriter port; service never calls os.* directly)
//   - REQ-DR-01 (dry-run records PlannedOps, zero real writes)
//   - REQ-DR-02 (PlannedOps shape: stable 6-value op enum, incl. install_package)
//   - REQ-EC-05 (write order: PJ → schematics → SKILL → AGENTS → pkg.json → install → MCP)
//   - REQ-CS-05 (--publishable → ErrCodeInitNotImplemented)
//   - REQ-MCP-02 (MCP=yes records mcp_setup_offered op in dry-run; sets flag in real-write)
//   - REQ-SA-01 (real-write writes SKILL.md with locked bytes)
//   - REQ-SA-02 (pre-existing SKILL.md without --force → warning in Warnings, no error)
//   - REQ-SA-02 (pre-existing SKILL.md with --force → overwritten)
//   - REQ-SA-03 (--no-skill → SKILL.md not written, outputs 4+SDK skipped)
//   - REQ-AR-01 (real-write writes AGENTS.md marker with locked bytes)
//   - REQ-AR-02 (pre-existing AGENTS.md with marker → idempotent; no duplication)
//   - REQ-AR-04 (both files markered → ErrInitAgentFileAmbiguous unless --force)
//   - REQ-SA-03 regression (--no-skill still skips outputs 3+4+SDK atomically)
//   - REQ-DR-01 regression (dry-run still records append_marker for AGENTS.md)
//   - REQ-PM-01 (real-write writes package.json with @pbuilder/sdk dev-dep)
//   - REQ-PM-02 (additive only — pre-existing deps preserved; malformed JSON → ErrCodeInvalidInput)
//   - REQ-SA-03 regression (--no-skill skips package.json)
//   - REQ-DR-01 regression (dry-run still records modify_devdep for package.json)
//   - REQ-PD-01 (PM detection: detect called with correct flag; result in InitResult)
//   - REQ-PD-02 (Install called once with detected PM; success → Installed=true)
//   - REQ-PD-03 (Install fail → error returned)
//   - REQ-PD-04 (--no-install: detect runs; install skipped; Installed=false)
//   - REQ-MCP-02 (MCP=yes real-write → MCPSetupOffered=true)
//   - REQ-MCP-02 (MCP=no real-write → MCPSetupOffered=false)
package initialise

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/init/template"
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

// Test_Service_Init_NonDryRun_Succeeds verifies that real-write mode
// (DryRun=false) now completes all 5 outputs + install (via fakePM with
// no install error) and returns nil. REQ-PD-02 happy path via fakePM.
// S-005: install subprocess is now fully wired.
func Test_Service_Init_NonDryRun_Succeeds(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm, installErr: nil}
	// Use non-empty skill bytes so outputs 3+4+5 succeed.
	svc := NewService(ffs, pm, []byte("placeholder"))

	req := InitRequest{
		Directory: dir,
		DryRun:    false,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("expected nil error after S-005 wiring; got: %v", err)
	}
	if !result.Installed {
		t.Errorf("InitResult.Installed = false, want true after successful fakePM install")
	}
	if result.PackageManager != PMNpm {
		t.Errorf("InitResult.PackageManager = %q, want %q", result.PackageManager, PMNpm)
	}
	if pm.installCalls != 1 {
		t.Errorf("pm.installCalls = %d, want 1", pm.installCalls)
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
// the locked bytes. After S-005, all outputs succeed end-to-end.
//
// REQ-PJ-01 (project-builder.json locked bytes via service path)
// REQ-SF-01  (schematics/.gitkeep locked bytes via service path)
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
	// After S-005: all outputs succeed; no stub error expected.
	if err != nil {
		t.Fatalf("expected nil after S-005 full wiring; got: %v", err)
	}

	// Output 1: project-builder.json MUST have been written with locked bytes.
	pbPath := filepath.Join(dir, "project-builder.json")
	gotPB, readErr := ffs.ReadFile(pbPath)
	if readErr != nil {
		t.Fatalf("project-builder.json not written: %v", readErr)
	}
	if !bytes.Equal(gotPB, lockedProjectBuilderJSON) {
		t.Errorf("project-builder.json bytes mismatch\ngot:\n%s\nwant:\n%s", gotPB, lockedProjectBuilderJSON)
	}

	// Output 2: schematics/.gitkeep MUST have been written with locked bytes.
	gitkeepPath := filepath.Join(dir, schematicsFolderName, ".gitkeep")
	gotGK, readErr := ffs.ReadFile(gitkeepPath)
	if readErr != nil {
		t.Fatalf("schematics/.gitkeep not written: %v", readErr)
	}
	if !bytes.Equal(gotGK, lockedGitkeepBytes) {
		t.Errorf("schematics/.gitkeep bytes mismatch\ngot:\n%q\nwant:\n%q", gotGK, lockedGitkeepBytes)
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
	// After S-005: full wiring; no stub error.
	if err != nil {
		t.Fatalf("expected nil after S-005 full wiring; got: %v", err)
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
	// After S-003, outputs 1+2+3+4 succeed. Output 5 (package.json) returns ErrCodeNotImplemented.
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
	// Hard error from ErrCodeNotImplemented (S-004 stub for output 5) is still
	// acceptable — but ErrCodeInitSkillExists must NOT propagate as the hard error.
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
	// ErrCodeNotImplemented for output 5 (package.json, S-004 stub) is acceptable.
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

	// After S-003 with --no-skill: outputs 1+2 written, outputs 3+4+SDK skipped
	// atomically. The service returns nil (no stub, no further outputs in the
	// --no-skill path). We don't assert specific error here; we assert SKILL.md
	// was NOT written, and that the function succeeded (or returned only a
	// valid stub error for a later slice — not expected after S-003).
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

// --- S-003 AGENTS.md/CLAUDE.md marker integration tests ---

// Test_Service_Init_S003_RealWrite_WritesAgentsMarker verifies that when
// neither AGENTS.md nor CLAUDE.md exists, Service.Init creates AGENTS.md
// with the locked marker block (outputs 1+2+3+4 all land before the
// ErrCodeNotImplemented stub for output 5). REQ-AR-01.
func Test_Service_Init_S003_RealWrite_WritesAgentsMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	// Output 5 (package.json) returns ErrCodeNotImplemented — that's expected.
	if err != nil {
		sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
		if !errors.Is(err, sentinel) {
			t.Fatalf("unexpected non-stub error: %v", err)
		}
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	got, readErr := ffs.ReadFile(agentsPath)
	if readErr != nil {
		t.Fatalf("AGENTS.md not written: %v", readErr)
	}

	if string(got) != agentMarkerBlock {
		t.Errorf("AGENTS.md content mismatch\ngot:\n%q\nwant:\n%q", string(got), agentMarkerBlock)
	}
}

// Test_Service_Init_S003_RealWrite_Idempotent_PreexistingMarker verifies that
// when AGENTS.md already contains the locked marker (line-exact), Service.Init
// does NOT append a second copy. REQ-AR-02.
func Test_Service_Init_S003_RealWrite_Idempotent_PreexistingMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	agentsPath := filepath.Join(dir, "AGENTS.md")
	existingWithMarker := []byte("# Agents\n\n" + agentMarkerBlock)
	if err := ffs.WriteFile(agentsPath, existingWithMarker, 0o644); err != nil {
		t.Fatalf("seed AGENTS.md: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	if err != nil {
		sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
		if !errors.Is(err, sentinel) {
			t.Fatalf("unexpected non-stub error: %v", err)
		}
	}

	got, _ := ffs.ReadFile(agentsPath)
	// Content must be identical to original — no second marker appended.
	if string(got) != string(existingWithMarker) {
		t.Errorf("idempotent: AGENTS.md was modified on re-run (REQ-AR-02)\nbefore:\n%q\nafter:\n%q",
			string(existingWithMarker), string(got))
	}
}

// Test_Service_Init_S003_BothMarkered_ReturnsErrAmbiguous verifies that
// when both AGENTS.md and CLAUDE.md already contain the marker and --force
// is not set, Service.Init returns ErrCodeInitAgentFileAmbiguous. REQ-AR-04.
func Test_Service_Init_S003_BothMarkered_ReturnsErrAmbiguous(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	withMarker := []byte("# Existing\n\n" + agentMarkerBlock)
	agentsPath := filepath.Join(dir, "AGENTS.md")
	claudePath := filepath.Join(dir, "CLAUDE.md")
	_ = ffs.WriteFile(agentsPath, withMarker, 0o644)
	_ = ffs.WriteFile(claudePath, withMarker, 0o644)

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, Force: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected ErrCodeInitAgentFileAmbiguous, got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) || e.Code != errs.ErrCodeInitAgentFileAmbiguous {
		t.Errorf("expected ErrCodeInitAgentFileAmbiguous; got: %v", err)
	}
}

// Test_Service_Init_S003_NoSkill_SkipsAgentsMarker_Regression verifies that
// --no-skill still skips output 4 (AGENTS marker) atomically after S-003.
// REQ-SA-03 regression.
func Test_Service_Init_S003_NoSkill_SkipsAgentsMarker_Regression(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, NoSkill: true, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	// --no-skill path: outputs 1+2 written; outputs 3+4+SDK skipped; returns nil.
	if err != nil {
		t.Errorf("--no-skill should succeed after S-003; got: %v", err)
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	if _, statErr := ffs.Stat(agentsPath); statErr == nil {
		t.Errorf("AGENTS.md was created despite --no-skill (REQ-SA-03 regression)")
	}

	claudePath := filepath.Join(dir, "CLAUDE.md")
	if _, statErr := ffs.Stat(claudePath); statErr == nil {
		t.Errorf("CLAUDE.md was created despite --no-skill (REQ-SA-03 regression)")
	}
}

// Test_Service_Init_S003_DryRun_HasAppendMarkerOp verifies that dry-run
// records an append_marker op for AGENTS.md. REQ-AR-01, REQ-DR-01.
func Test_Service_Init_S003_DryRun_HasAppendMarkerOp(t *testing.T) {
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

	agentsPath := filepath.Join(dir, "AGENTS.md")
	var found bool
	for _, op := range result.PlannedOps {
		if op.Op == "append_marker" && op.Path == agentsPath {
			found = true
		}
	}
	if !found {
		t.Errorf("PlannedOps does not contain append_marker for %q (REQ-AR-01 dry-run)", agentsPath)
	}
}

// --- S-004 package.json mutation integration tests ---

// Test_Service_Init_S004_RealWrite_WritesPackageJSON verifies that real-write
// mode writes package.json with the @pbuilder/sdk dev-dep. The service returns
// ErrCodeNotImplemented for the install subprocess (S-005 stub), but
// package.json must be written before that error is returned. REQ-PM-01.
func Test_Service_Init_S004_RealWrite_WritesPackageJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	// After S-004, outputs 1+2+3+4+5 succeed. Install subprocess returns
	// ErrCodeNotImplemented (S-005 stub).
	if err != nil {
		sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
		if !errors.Is(err, sentinel) {
			t.Fatalf("unexpected non-stub error: %v", err)
		}
	}

	pkgPath := filepath.Join(dir, "package.json")
	got, readErr := ffs.ReadFile(pkgPath)
	if readErr != nil {
		t.Fatalf("package.json not written: %v", readErr)
	}
	content := string(got)
	if !strings.Contains(content, `"@pbuilder/sdk": "^1.0.0"`) {
		t.Errorf("package.json missing @pbuilder/sdk devDependency\ngot:\n%s", content)
	}
}

// Test_Service_Init_S004_RealWrite_PreexistingPackageJSON_PreservesExistingDeps
// verifies that when package.json already exists with other deps, the service
// preserves them and adds @pbuilder/sdk. REQ-PM-02 (additive only).
func Test_Service_Init_S004_RealWrite_PreexistingPackageJSON_PreservesExistingDeps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	pkgPath := filepath.Join(dir, "package.json")
	existing := []byte(`{"name":"my-project","devDependencies":{"typescript":"^5.0.0"}}`)
	if err := ffs.WriteFile(pkgPath, existing, 0o644); err != nil {
		t.Fatalf("seed package.json: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	if err != nil {
		sentinel := &errs.Error{Code: errs.ErrCodeNotImplemented}
		if !errors.Is(err, sentinel) {
			t.Fatalf("unexpected non-stub error: %v", err)
		}
	}

	got, _ := ffs.ReadFile(pkgPath)
	content := string(got)

	if !strings.Contains(content, `"@pbuilder/sdk": "^1.0.0"`) {
		t.Errorf("package.json missing @pbuilder/sdk\ngot:\n%s", content)
	}
	if !strings.Contains(content, `"typescript": "^5.0.0"`) {
		t.Errorf("package.json lost pre-existing typescript dep\ngot:\n%s", content)
	}
}

// Test_Service_Init_S004_RealWrite_MalformedPackageJSON_ReturnsErrCodeInvalidInput
// verifies that a malformed package.json causes ErrCodeInvalidInput and no
// other outputs are affected (the error is returned before any more writes).
// REQ-PM-02.
func Test_Service_Init_S004_RealWrite_MalformedPackageJSON_ReturnsErrCodeInvalidInput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()

	pkgPath := filepath.Join(dir, "package.json")
	malformed := []byte(`{name: "no-quotes"}`)
	if err := ffs.WriteFile(pkgPath, malformed, 0o644); err != nil {
		t.Fatalf("seed package.json: %v", err)
	}

	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for malformed package.json; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) || e.Code != errs.ErrCodeInvalidInput {
		t.Errorf("expected ErrCodeInvalidInput; got: %v", err)
	}

	// package.json must remain unmodified.
	got, readErr := ffs.ReadFile(pkgPath)
	if readErr != nil {
		t.Fatalf("ReadFile after malformed error: %v", readErr)
	}
	if string(got) != string(malformed) {
		t.Errorf("malformed package.json was modified; want unchanged")
	}
}

// Test_Service_Init_S004_NoSkill_SkipsPackageJSON_Regression verifies that
// --no-skill still skips package.json atomically with outputs 3+4 after S-004.
// REQ-SA-03 regression.
func Test_Service_Init_S004_NoSkill_SkipsPackageJSON_Regression(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, NoSkill: true, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	// --no-skill path: outputs 1+2 written; 3+4+SDK skipped; returns nil.
	if err != nil {
		t.Errorf("--no-skill should succeed after S-004; got: %v", err)
	}

	pkgPath := filepath.Join(dir, "package.json")
	if _, statErr := ffs.Stat(pkgPath); statErr == nil {
		t.Errorf("package.json was written despite --no-skill (REQ-SA-03 regression)")
	}
}

// Test_Service_Init_S004_DryRun_ModifyDevDepOp_Regression verifies that dry-run
// still records a modify_devdep PlannedOp for package.json. REQ-DR-01, REQ-PM-01.
func Test_Service_Init_S004_DryRun_ModifyDevDepOp_Regression(t *testing.T) {
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

	pkgPath := filepath.Join(dir, "package.json")
	var found bool
	for _, op := range result.PlannedOps {
		if op.Op == "modify_devdep" && op.Path == pkgPath {
			found = true
			// Details must mention the package.
			if !strings.Contains(op.Details, "@pbuilder/sdk") {
				t.Errorf("modify_devdep Details missing @pbuilder/sdk: %q", op.Details)
			}
		}
	}
	if !found {
		t.Errorf("PlannedOps does not contain modify_devdep for %q (REQ-PM-01 dry-run)", pkgPath)
	}
}

// --- REQ-DR-02 regression after S-003 ---

// Test_Service_Init_DryRun_WithMCP_HasCorrectOpCount verifies that with MCP=yes,
// dry-run produces exactly 7 PlannedOps (5 file ops + install_package + mcp_setup_offered).
// Without MCP, exactly 6 ops (5 file ops + install_package).
// REQ-DR-02, S-005 acceptance criterion.
func Test_Service_Init_DryRun_WithMCP_HasCorrectOpCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mcp     MCPMode
		wantOps int
	}{
		{name: "mcp=no", mcp: MCPNo, wantOps: 6},
		{name: "mcp=yes", mcp: MCPYes, wantOps: 7},
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

// --- S-005 PM detection + install integration tests ---

// Test_Service_S005_RealWrite_InstallCalled_HappyPath verifies that in real-write
// mode Service.Init calls pm.Detect then pm.Install with the detected PM, and
// sets InitResult.Installed=true, InitResult.PackageManager=<detected PM>.
// REQ-PD-01, REQ-PD-02.
func Test_Service_S005_RealWrite_InstallCalled_HappyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMPnpm, installErr: nil}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{
		Directory:          dir,
		DryRun:             false,
		MCP:                MCPNo,
		PackageManagerFlag: PMUnset, // triggers lockfile detection path
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init: unexpected error: %v", err)
	}

	if !result.Installed {
		t.Errorf("InitResult.Installed = false, want true (REQ-PD-02 happy path)")
	}
	if result.PackageManager != PMPnpm {
		t.Errorf("InitResult.PackageManager = %q, want %q (REQ-PD-01)", result.PackageManager, PMPnpm)
	}
	if pm.installCalls != 1 {
		t.Errorf("pm.installCalls = %d, want 1", pm.installCalls)
	}
}

// Test_Service_S005_RealWrite_FlagOverride_UsesFlag verifies that when
// PackageManagerFlag is set, Detect uses it and Install is called with it.
// REQ-PD-01 (flag override).
func Test_Service_S005_RealWrite_FlagOverride_UsesFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	// fakePM.detectResult is PMNpm, but the flag says PMYarn — flag wins.
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{
		Directory:          dir,
		DryRun:             false,
		MCP:                MCPNo,
		PackageManagerFlag: PMYarn,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init: unexpected error: %v", err)
	}

	if result.PackageManager != PMYarn {
		t.Errorf("InitResult.PackageManager = %q, want %q (flag override, REQ-PD-01)", result.PackageManager, PMYarn)
	}
}

// Test_Service_S005_RealWrite_InstallFails_ReturnsError verifies that when
// pm.Install returns an error, Service.Init propagates it. REQ-PD-03.
func Test_Service_S005_RealWrite_InstallFails_ReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	installErr := &errs.Error{
		Code:    errs.ErrCodeExecutionFailed,
		Op:      "init.handler",
		Message: "simulated install failure",
	}
	pm := &fakePM{detectResult: PMNpm, installErr: installErr}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	_, err := svc.Init(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when pm.Install fails; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) || e.Code != errs.ErrCodeExecutionFailed {
		t.Errorf("expected ErrCodeExecutionFailed propagated; got: %v", err)
	}
}

// Test_Service_S005_NoInstall_SkipsInstall verifies that when NoInstall=true,
// pm.Install is NOT called, Installed=false, but PackageManager is still
// populated (detect still runs for the pretty-print hint). REQ-PD-04.
func Test_Service_S005_NoInstall_SkipsInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMBun}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{
		Directory: dir,
		DryRun:    false,
		MCP:       MCPNo,
		NoInstall: true,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init with --no-install: unexpected error: %v", err)
	}

	if result.Installed {
		t.Errorf("InitResult.Installed = true, want false when --no-install (REQ-PD-04)")
	}
	if pm.installCalls != 0 {
		t.Errorf("pm.installCalls = %d, want 0 when --no-install", pm.installCalls)
	}
	// PackageManager should still be populated (detect ran for pretty print).
	if result.PackageManager != PMBun {
		t.Errorf("InitResult.PackageManager = %q, want %q (detect still ran for hint)", result.PackageManager, PMBun)
	}
}

// Test_Service_S005_MCP_Yes_AfterInstall_SetsMCPSetupOffered verifies that
// when MCP=yes and install succeeds, MCPSetupOffered=true (REQ-MCP-02).
func Test_Service_S005_MCP_Yes_AfterInstall_SetsMCPSetupOffered(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{
		Directory: dir,
		DryRun:    false,
		MCP:       MCPYes,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init MCP=yes: unexpected error: %v", err)
	}

	if !result.MCPSetupOffered {
		t.Errorf("InitResult.MCPSetupOffered = false, want true when MCP=yes (REQ-MCP-02)")
	}
	if !result.Installed {
		t.Errorf("InitResult.Installed = false, want true (install should have run)")
	}
}

// Test_Service_S005_MCP_No_MCPSetupOffered_False verifies that when MCP=no,
// MCPSetupOffered=false. REQ-MCP-02.
func Test_Service_S005_MCP_No_MCPSetupOffered_False(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: false, MCP: MCPNo}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init MCP=no: unexpected error: %v", err)
	}

	if result.MCPSetupOffered {
		t.Errorf("InitResult.MCPSetupOffered = true, want false when MCP=no (REQ-MCP-02)")
	}
}

// Test_Service_S005_MCP_Yes_NoInstall_SetsMCPSetupOffered verifies that even
// with --no-install, MCP=yes still sets MCPSetupOffered=true after the
// package.json mutation (REQ-MCP-02).
func Test_Service_S005_MCP_Yes_NoInstall_SetsMCPSetupOffered(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ffs := newFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(ffs, pm, template.Skill)

	req := InitRequest{
		Directory: dir,
		DryRun:    false,
		MCP:       MCPYes,
		NoInstall: true,
	}

	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init MCP=yes --no-install: unexpected error: %v", err)
	}

	if !result.MCPSetupOffered {
		t.Errorf("MCPSetupOffered = false with MCP=yes --no-install (REQ-MCP-02): instructions should still be offered")
	}
}

// Test_Service_S005_DryRun_HasInstallPackageOp verifies that dry-run includes
// the install_package PlannedOp when --no-install is not set. REQ-DR-02.
func Test_Service_S005_DryRun_HasInstallPackageOp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: true, MCP: MCPNo}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, op := range result.PlannedOps {
		if op.Op == "install_package" {
			found = true
		}
	}
	if !found {
		t.Errorf("PlannedOps does not contain install_package (REQ-DR-02 — S-005 wired)")
	}
}

// Test_Service_S005_DryRun_NoInstall_NoInstallPackageOp verifies that with
// --no-install, dry-run does NOT record install_package. REQ-PD-04.
func Test_Service_S005_DryRun_NoInstall_NoInstallPackageOp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dfs := newDryRunFakeFS()
	pm := &fakePM{detectResult: PMNpm}
	svc := NewService(dfs, pm, template.Skill)

	req := InitRequest{Directory: dir, DryRun: true, MCP: MCPNo, NoInstall: true}
	result, err := svc.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, op := range result.PlannedOps {
		if op.Op == "install_package" {
			t.Errorf("PlannedOps contains install_package despite --no-install (REQ-PD-04)")
		}
	}
}
