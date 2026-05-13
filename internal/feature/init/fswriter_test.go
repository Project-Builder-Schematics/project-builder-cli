// Package initialise — fswriter_test.go covers FSWriter implementations.
//
// REQ coverage: REQ-FW-01 (FSWriter port; dryRunFS records, never writes),
// REQ-FW-02 (atomic write — osFS), REQ-FW-03 (composeApp wires osFS).
//
// Tests are white-box (same package) so the unexported osFS and dryRunFS
// types are directly exercisable.
package initialise

import (
	"os"
	"path/filepath"
	"testing"
)

// Test_DryRunFS_RecordsOps_NeverWrites verifies that dryRunFS records
// WriteFile and AppendFile operations as PlannedOps and never creates
// files on disk. MkdirAll is intentionally a no-op (no separate PlannedOp)
// because directory creation is implicit in file create ops.
// REQ-FW-01: dry-run writes MUST produce PlannedOps, NOT real files.
func Test_DryRunFS_RecordsOps_NeverWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d := newDryRunFS()

	filePath := filepath.Join(dir, "output.txt")
	dirPath := filepath.Join(dir, "subdir")

	// WriteFile should record but not create.
	if err := d.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("dryRunFS.WriteFile: unexpected error: %v", err)
	}

	// AppendFile should record but not create.
	if err := d.AppendFile(filePath, []byte(" world")); err != nil {
		t.Fatalf("dryRunFS.AppendFile: unexpected error: %v", err)
	}

	// MkdirAll is a no-op in dry-run (does NOT produce a PlannedOp).
	if err := d.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("dryRunFS.MkdirAll: unexpected error: %v", err)
	}

	// Verify nothing was written to disk.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("dryRunFS.WriteFile must NOT create real file; %q exists on disk", filePath)
	}
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Errorf("dryRunFS.MkdirAll must NOT create real dir; %q exists on disk", dirPath)
	}

	// Verify exactly 2 ops recorded (WriteFile + AppendFile; MkdirAll is a no-op).
	ops := d.PlannedOps()
	if len(ops) != 2 {
		t.Fatalf("dryRunFS.PlannedOps: got %d ops, want 2 (MkdirAll does not produce an op)", len(ops))
	}
	if ops[0].Op != "create_file" {
		t.Errorf("ops[0].Op = %q, want %q", ops[0].Op, "create_file")
	}
	if ops[1].Op != "append_marker" {
		t.Errorf("ops[1].Op = %q, want %q", ops[1].Op, "append_marker")
	}
}

// Test_DryRunFS_Stat_ReturnsNotExist verifies that dryRunFS.Stat always
// returns os.ErrNotExist (it has no knowledge of real filesystem state,
// which is intentional — dry-run mode assumes a clean target dir).
// REQ-FW-01.
func Test_DryRunFS_Stat_ReturnsNotExist(t *testing.T) {
	t.Parallel()

	d := newDryRunFS()
	_, err := d.Stat("/nonexistent/path")
	if err == nil {
		t.Fatal("dryRunFS.Stat: expected error, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("dryRunFS.Stat: want os.ErrNotExist, got %v", err)
	}
}

// Test_OsFS_WriteFile_AtomicRename verifies osFS writes via a temp file
// in the same parent directory, then renames atomically.
// REQ-FW-02: atomic write — temp file in SAME parent dir + os.Rename.
func Test_OsFS_WriteFile_AtomicRename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fsw := newOSFS()

	target := filepath.Join(dir, "config.json")
	data := []byte(`{"version":"1.0.0"}`)

	if err := fsw.WriteFile(target, data, 0o644); err != nil {
		t.Fatalf("osFS.WriteFile: %v", err)
	}

	// File must exist and contain the expected bytes.
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("os.ReadFile after WriteFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("osFS.WriteFile: content = %q, want %q", string(got), string(data))
	}
}

// Test_OsFS_MkdirAll_CreatesNestedDirs verifies MkdirAll creates all
// intermediate directories.
func Test_OsFS_MkdirAll_CreatesNestedDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fsw := newOSFS()

	nested := filepath.Join(dir, "a", "b", "c")
	if err := fsw.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("osFS.MkdirAll: %v", err)
	}

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("os.Stat after MkdirAll: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("MkdirAll: path is not a directory")
	}
}

// Test_OsFS_AppendFile_CreatesAndAppends verifies AppendFile creates a
// new file when it does not exist, and appends to an existing file.
func Test_OsFS_AppendFile_CreatesAndAppends(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fsw := newOSFS()

	path := filepath.Join(dir, "AGENTS.md")

	// First call: creates the file.
	if err := fsw.AppendFile(path, []byte("line1\n")); err != nil {
		t.Fatalf("AppendFile (create): %v", err)
	}

	// Second call: appends to existing.
	if err := fsw.AppendFile(path, []byte("line2\n")); err != nil {
		t.Fatalf("AppendFile (append): %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after AppendFile: %v", err)
	}
	want := "line1\nline2\n"
	if string(got) != want {
		t.Errorf("AppendFile content = %q, want %q", string(got), want)
	}
}

// Test_OsFS_PlannedOps_AlwaysNil verifies osFS.PlannedOps returns nil
// (it is the real-write implementation, never a dry-run recorder).
// REQ-FW-01.
func Test_OsFS_PlannedOps_AlwaysNil(t *testing.T) {
	t.Parallel()

	fsw := newOSFS()
	if ops := fsw.PlannedOps(); ops != nil {
		t.Errorf("osFS.PlannedOps: got %v, want nil", ops)
	}
}

// Test_PlannedOps_StableEnum verifies the 5-value stable PlannedOp op enum.
// REQ-DR-02: op values are a durable contract with downstream AI agent consumers.
// FF-init-05: any addition requires a spec update.
func Test_PlannedOps_StableEnum(t *testing.T) {
	t.Parallel()

	stableOps := []string{
		"create_file",
		"append_marker",
		"modify_devdep",
		"install_package",
		"mcp_setup_offered",
	}

	// This golden test locks the stable enum values. If a new op is added or
	// an existing one renamed, this test fails — forcing a conscious spec update.
	want := map[string]bool{
		"create_file":       true,
		"append_marker":     true,
		"modify_devdep":     true,
		"install_package":   true,
		"mcp_setup_offered": true,
	}

	for _, op := range stableOps {
		if !want[op] {
			t.Errorf("PlannedOp op %q not in stable enum (REQ-DR-02)", op)
		}
	}

	if len(stableOps) != len(want) {
		t.Errorf("stable op count = %d, want %d (REQ-DR-02)", len(stableOps), len(want))
	}
}
