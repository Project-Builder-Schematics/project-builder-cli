// Package fswriter provides the FSWriter port and its two production
// implementations: osFS (real writes via atomic rename) and dryRunFS
// (records PlannedOps without touching disk).
//
// ADR-020: all filesystem I/O in feature packages goes through FSWriter.
// FF-init-02 (post-feature): a fitness function script enforces that no
// production code calls os.* directly outside this file.
package fswriter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FSWriter is the filesystem port used by all feature packages (ADR-020).
// All filesystem I/O in the service layer MUST go through this interface.
// Three implementations exist: osFS (real writes), dryRunFS (records PlannedOps),
// fakeFS (in-memory, for tests).
type FSWriter interface {
	// Stat returns file metadata for path, or an error if the path does not
	// exist or is not accessible.
	Stat(path string) (os.FileInfo, error)

	// Lstat returns file metadata for path without following symlinks.
	// Use this to detect whether a path is itself a symlink (REQ-AR-05).
	Lstat(path string) (os.FileInfo, error)

	// EvalSymlinks returns the path with all symlinks resolved.
	// Returns an error if any component of the path does not exist.
	// Use this to verify the resolved target stays within the project
	// directory (REQ-AR-05 symlink-safety check).
	EvalSymlinks(path string) (string, error)

	// ReadFile returns the contents of path, or an error.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to path with the given permissions.
	// Implementations MUST use an atomic write (temp file + rename) to avoid
	// partial writes (FF-init-02).
	WriteFile(path string, data []byte, perm os.FileMode) error

	// MkdirAll creates path and all parents with the given permissions.
	MkdirAll(path string, perm os.FileMode) error

	// AppendFile appends data to path, creating the file if it does not exist.
	AppendFile(path string, data []byte) error

	// PlannedOps returns the list of operations recorded in dry-run mode.
	// Returns nil for non-dry-run implementations.
	PlannedOps() []PlannedOp
}

// OpRecorder is satisfied by FSWriter implementations that support
// recording arbitrary PlannedOps (e.g. dryRunFS, fakeFS). Services use
// a type assertion to this interface for custom op types that don't map
// to a standard FSWriter method (e.g. mcp_setup_offered, install_package).
type OpRecorder interface {
	RecordOp(PlannedOp)
}

// PlannedOp records a single intended file operation in dry-run mode.
// The Op field uses a stable 5-value enum (REQ-DR-02).
type PlannedOp struct {
	// Op is the stable operation discriminator.
	// Values: create_file | append_marker | modify_devdep | install_package | mcp_setup_offered
	Op string `json:"op"`

	// Path is the target file or directory path. Omitted for mcp_setup_offered.
	Path string `json:"path,omitempty"`

	// Details carries supplementary information (e.g. package name for install_package).
	Details string `json:"details,omitempty"`
}

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
//
// G304 is suppressed: the path is supplied by the handler after
// Canonicalise (REQ-DV-01: filepath.Abs + filepath.Clean + .. traversal
// rejection). This osFS is only used in production via composeApp wiring;
// tests use fakeFS.
func (o *osFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) // #nosec G304 — path validated by Canonicalise
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
//
// G302 is suppressed: 0o644 is the standard POSIX permission for
// user-readable text files like AGENTS.md and CLAUDE.md (the only files
// init's AppendFile is used against). Reducing to 0o600 would break the
// project's existing convention for these files which are intended to be
// committed and shared.
//
// G304 is suppressed: path is supplied by the handler after
// Canonicalise (REQ-DV-01); same justification as ReadFile above.
func (o *osFS) AppendFile(path string, data []byte) (retErr error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G302,G304 — see godoc
	if err != nil {
		return fmt.Errorf("osFS.AppendFile: open %q: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("osFS.AppendFile: close %q: %w", path, cerr)
		}
	}()
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

// NewDryRunWriter is the exported constructor for dry-run mode.
func NewDryRunWriter() FSWriter { return newDryRunFS() }

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

// RecordOp appends an arbitrary PlannedOp (used by service for ops that
// don't go through the standard filesystem methods, e.g. mcp_setup_offered
// and install_package).
func (d *dryRunFS) RecordOp(op PlannedOp) {
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
