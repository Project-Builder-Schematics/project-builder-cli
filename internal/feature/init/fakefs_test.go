// Package initialise — fakefs_test.go provides an in-memory FSWriter
// implementation for use in unit and integration tests.
//
// fakeFS is NOT a production implementation. It satisfies the FSWriter
// interface so tests can exercise service and handler logic without
// touching the real filesystem.
package initialise

import (
	"context"
	"io/fs"
	"os"
	"sync"
	"time"
)

// fakeFS is an in-memory FSWriter implementation for tests.
// All operations are recorded; writes and stats reflect the in-memory state.
// Thread-safe (mu protects files and ops).
type fakeFS struct {
	mu         sync.Mutex
	files      map[string][]byte
	ops        []PlannedOp
	recordOnly bool // when true, record ops without updating in-memory files

	// symlinks maps a path to its resolved absolute target (simulates symlinks).
	// When EvalSymlinks(path) is called and path is in this map, the mapped
	// value is returned. For Lstat, paths in symlinks report as symlinks.
	symlinks map[string]string
}

// newFakeFS returns an empty fakeFS ready for use.
func newFakeFS() *fakeFS {
	return &fakeFS{
		files:    make(map[string][]byte),
		symlinks: make(map[string]string),
	}
}

// newDryRunFakeFS returns a fakeFS in record-only mode, mimicking dryRunFS.
func newDryRunFakeFS() *fakeFS {
	return &fakeFS{
		files:      make(map[string][]byte),
		symlinks:   make(map[string]string),
		recordOnly: true,
	}
}

// addSymlink registers path as a symlink that resolves to target.
// Use this in tests that exercise REQ-AR-05 symlink safety.
func (f *fakeFS) addSymlink(path, target string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.symlinks[path] = target
}

// Stat returns FileInfo for path.
// Returns os.ErrNotExist if the path is not present in the in-memory map.
// For symlinked paths, Stat follows the link (returns info for the target).
func (f *fakeFS) Stat(path string) (os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Follow symlink if present.
	if target, ok := f.symlinks[path]; ok {
		// Symlink exists; return info for the target (if the target is also in
		// the in-memory map, use its size; otherwise size=0).
		size := int64(len(f.files[target]))
		return &fakeFI{name: path, size: size}, nil
	}
	data, ok := f.files[path]
	if !ok {
		return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}
	return &fakeFI{name: path, size: int64(len(data))}, nil
}

// Lstat returns FileInfo for path WITHOUT following symlinks.
// If path is a registered symlink, it returns a symlinkFI (Mode includes ModeSymlink).
// Returns os.ErrNotExist if path is neither a symlink nor a file.
func (f *fakeFS) Lstat(path string) (os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.symlinks[path]; ok {
		return &fakeFI{name: path, size: 0, isSymlink: true}, nil
	}
	data, ok := f.files[path]
	if !ok {
		return nil, &os.PathError{Op: "lstat", Path: path, Err: os.ErrNotExist}
	}
	return &fakeFI{name: path, size: int64(len(data))}, nil
}

// EvalSymlinks resolves any registered symlink for path.
// If path is a known symlink, the registered target is returned.
// Otherwise path itself is returned (no-op for regular files or non-existent paths).
func (f *fakeFS) EvalSymlinks(path string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if target, ok := f.symlinks[path]; ok {
		return target, nil
	}
	return path, nil
}

// ReadFile returns the bytes stored at path.
func (f *fakeFS) ReadFile(path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.files[path]
	if !ok {
		return nil, &os.PathError{Op: "read", Path: path, Err: os.ErrNotExist}
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// WriteFile stores data at path with the given permissions (perm ignored in memory).
// In record-only mode, records a create_file PlannedOp without storing data.
func (f *fakeFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recordOnly {
		f.ops = append(f.ops, PlannedOp{Op: "create_file", Path: path})
		return nil
	}
	stored := make([]byte, len(data))
	copy(stored, data)
	f.files[path] = stored
	return nil
}

// MkdirAll is a no-op for in-memory usage. Directory creation is implicit
// in file create ops — no separate PlannedOp is recorded (matches dryRunFS).
func (f *fakeFS) MkdirAll(_ string, _ os.FileMode) error { return nil }

// AppendFile appends data to the existing content at path.
// In record-only mode, records an append_marker PlannedOp.
func (f *fakeFS) AppendFile(path string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recordOnly {
		f.ops = append(f.ops, PlannedOp{Op: "append_marker", Path: path})
		return nil
	}
	f.files[path] = append(f.files[path], data...)
	return nil
}

// PlannedOps returns the accumulated PlannedOps slice (copy).
func (f *fakeFS) PlannedOps() []PlannedOp {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ops) == 0 {
		return nil
	}
	out := make([]PlannedOp, len(f.ops))
	copy(out, f.ops)
	return out
}

// RecordOp appends an arbitrary PlannedOp (used by service stubs in dry-run mode
// to record ops that don't go through Stat/WriteFile/AppendFile).
// Implements fswriter.OpRecorder (exported method — required for cross-package
// type assertion in service.go).
func (f *fakeFS) RecordOp(op PlannedOp) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops = append(f.ops, op)
}

// --- fakeFI: minimal os.FileInfo for fakeFS.Stat ---

// fakeFI is a minimal os.FileInfo implementation for fakeFS.
type fakeFI struct {
	name      string
	size      int64
	isSymlink bool
}

func (fi *fakeFI) Name() string { return fi.name }
func (fi *fakeFI) Size() int64  { return fi.size }
func (fi *fakeFI) Mode() fs.FileMode {
	if fi.isSymlink {
		return fs.ModeSymlink | 0o644
	}
	return 0o644
}
func (fi *fakeFI) ModTime() time.Time { return time.Time{} }
func (fi *fakeFI) IsDir() bool        { return false }
func (fi *fakeFI) Sys() any           { return nil }

// --- fakePM: minimal PackageManagerRunner for tests ---

// fakePM is a test double for PackageManagerRunner.
// DetectResult and InstallErr are pre-configured fields; the test controls them.
type fakePM struct {
	detectResult PackageManager
	detectErr    error
	installErr   error
	installCalls int
}

func (p *fakePM) Detect(_ string, flag PackageManager) (PackageManager, error) {
	if flag != PMUnset {
		return flag, nil
	}
	return p.detectResult, p.detectErr
}

func (p *fakePM) Install(_ context.Context, _ string, _ PackageManager) error {
	p.installCalls++
	return p.installErr
}
