// Package initialise — fswriter.go provides the two production FSWriter
// implementations: osFS (real writes via atomic rename) and dryRunFS
// (records PlannedOps without touching disk).
//
// ADR-020: all filesystem I/O in the init feature goes through FSWriter.
// FF-init-02 (post-feature): a fitness function script enforces that no
// init production code calls os.* directly outside this file.
package initialise

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// --- osFS: real filesystem implementation ---

// osFS is the production FSWriter implementation. All writes use atomic
// rename (temp file in the same parent directory + os.Rename) to prevent
// partial writes from leaving the target in an inconsistent state.
// REQ-FW-02.
type osFS struct{}

// newOSFS returns the production FSWriter.
// Called by composeApp (REQ-FW-03).
func newOSFS() *osFS { return &osFS{} }

// NewOSWriter is the exported constructor used by composeApp.
func NewOSWriter() FSWriter { return newOSFS() }

// Stat delegates to os.Stat.
func (o *osFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// Lstat delegates to os.Lstat (does not follow symlinks).
func (o *osFS) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

// EvalSymlinks delegates to filepath.EvalSymlinks.
func (o *osFS) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// ReadFile delegates to os.ReadFile.
func (o *osFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data atomically: create a temp file in the same parent
// directory, write data, then rename to target path.
// The same-directory constraint ensures the rename is atomic on most
// POSIX filesystems (no cross-device move). REQ-FW-02.
func (o *osFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Create temp file in the same directory as the target (same filesystem
	// mount point — ensures os.Rename is a single atomic syscall).
	tmp, err := os.CreateTemp(dir, ".pbuilder-tmp-*")
	if err != nil {
		return fmt.Errorf("osFS.WriteFile: create temp: %w", err)
	}
	tmpName := tmp.Name()

	// Best-effort cleanup of the temp file on any failure path.
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("osFS.WriteFile: write temp: %w", err)
	}

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("osFS.WriteFile: chmod temp: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("osFS.WriteFile: close temp: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("osFS.WriteFile: rename to %q: %w", path, err)
	}

	success = true
	return nil
}

// MkdirAll delegates to os.MkdirAll.
func (o *osFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// AppendFile opens path with O_APPEND|O_CREATE|O_WRONLY and writes data.
func (o *osFS) AppendFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("osFS.AppendFile: open %q: %w", path, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("osFS.AppendFile: write %q: %w", path, err)
	}
	return nil
}

// PlannedOps always returns nil for osFS — only dryRunFS records ops.
func (o *osFS) PlannedOps() []PlannedOp { return nil }

// --- dryRunFS: dry-run recorder implementation ---

// dryRunFS is the dry-run FSWriter implementation. All operations are
// recorded as PlannedOps; no files are created or modified on disk.
// REQ-DR-01, REQ-FW-01.
type dryRunFS struct {
	mu  sync.Mutex
	ops []PlannedOp
}

// newDryRunFS returns a fresh dryRunFS.
func newDryRunFS() *dryRunFS { return &dryRunFS{} }

// Stat always returns os.ErrNotExist. In dry-run mode the service assumes
// the target directory is clean (pre-run checks are skipped for dry-run).
// REQ-DR-01.
func (d *dryRunFS) Stat(_ string) (os.FileInfo, error) {
	return nil, &os.PathError{Op: "stat", Path: "", Err: os.ErrNotExist}
}

// Lstat always returns os.ErrNotExist in dry-run mode (no real filesystem
// state available). Symlink checks are skipped in dry-run.
func (d *dryRunFS) Lstat(path string) (os.FileInfo, error) {
	return nil, &os.PathError{Op: "lstat", Path: path, Err: os.ErrNotExist}
}

// EvalSymlinks returns path unchanged in dry-run mode (no real filesystem).
// Symlink safety checks are skipped in dry-run.
func (d *dryRunFS) EvalSymlinks(path string) (string, error) {
	return path, nil
}

// ReadFile returns os.ErrNotExist (dry-run has no on-disk state to read).
func (d *dryRunFS) ReadFile(path string) ([]byte, error) {
	return nil, &os.PathError{Op: "read", Path: path, Err: os.ErrNotExist}
}

// WriteFile records a create_file PlannedOp. Nothing is written to disk.
func (d *dryRunFS) WriteFile(path string, _ []byte, _ os.FileMode) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ops = append(d.ops, PlannedOp{Op: "create_file", Path: path})
	return nil
}

// MkdirAll is a no-op in dry-run mode: directory creation is implicit in the
// create_file ops for the files within those directories. Recording a
// separate dir op would inflate the PlannedOps count beyond the stable
// 5-value (+ mcp_setup_offered) contract (REQ-DR-02).
func (d *dryRunFS) MkdirAll(_ string, _ os.FileMode) error { return nil }

// AppendFile records an append_marker PlannedOp. Nothing is written to disk.
func (d *dryRunFS) AppendFile(path string, _ []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ops = append(d.ops, PlannedOp{Op: "append_marker", Path: path})
	return nil
}

// recordOp appends an arbitrary PlannedOp (used by service for ops that
// don't go through the standard filesystem methods, e.g. mcp_setup_offered
// and install_package).
func (d *dryRunFS) recordOp(op PlannedOp) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ops = append(d.ops, op)
}

// PlannedOps returns a copy of the recorded ops slice.
func (d *dryRunFS) PlannedOps() []PlannedOp {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.ops) == 0 {
		return nil
	}
	out := make([]PlannedOp, len(d.ops))
	copy(out, d.ops)
	return out
}
