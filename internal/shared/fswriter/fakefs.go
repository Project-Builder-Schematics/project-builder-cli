// Package fswriter — fakefs.go provides the in-memory FSWriter implementation
// for use in tests across all feature packages.
//
// fakeFS satisfies the FSWriter interface and the OpRecorder extension.
// It is designed for test isolation: no real filesystem state is read or written.
// Thread-safe via sync.Mutex.
//
// Exported via NewFakeFS() so external test packages can use it without
// violating package boundaries.
package fswriter

import (
	"io/fs"
	"os"
	"sync"
	"time"
)

// FakeFS is an in-memory FSWriter implementation for use in tests.
// Exported so external test packages can reference the type directly
// when they need to add symlinks or inspect internal state.
type FakeFS struct {
	mu    sync.Mutex
	files map[string][]byte
	ops   []PlannedOp

	// Symlinks maps a path to its resolved absolute target (simulates symlinks).
	// When EvalSymlinks(path) is called and path is in this map, the mapped
	// value is returned. For Lstat, paths in Symlinks report as symlinks.
	Symlinks map[string]string
}

// NewFakeFS returns an empty FakeFS ready for use in tests.
// Use this as a drop-in replacement for osFS when you need real in-memory reads.
func NewFakeFS() *FakeFS {
	return &FakeFS{
		files:    make(map[string][]byte),
		Symlinks: make(map[string]string),
	}
}

// AddSymlink registers path as a symlink that resolves to target.
// Use this in tests that exercise symlink safety checks (ADV-08).
func (f *FakeFS) AddSymlink(path, target string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Symlinks[path] = target
}

// Stat returns FileInfo for path. Returns os.ErrNotExist if path is not
// present in the in-memory map. For symlinked paths, Stat follows the link.
func (f *FakeFS) Stat(path string) (os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if target, ok := f.Symlinks[path]; ok {
		size := int64(len(f.files[target]))
		return &fakeFSFileInfo{name: path, size: size}, nil
	}
	data, ok := f.files[path]
	if !ok {
		return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}
	return &fakeFSFileInfo{name: path, size: int64(len(data))}, nil
}

// Lstat returns FileInfo for path WITHOUT following symlinks.
// If path is a registered symlink, it returns info with ModeSymlink set.
func (f *FakeFS) Lstat(path string) (os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Symlinks[path]; ok {
		return &fakeFSFileInfo{name: path, size: 0, isSymlink: true}, nil
	}
	data, ok := f.files[path]
	if !ok {
		return nil, &os.PathError{Op: "lstat", Path: path, Err: os.ErrNotExist}
	}
	return &fakeFSFileInfo{name: path, size: int64(len(data))}, nil
}

// EvalSymlinks resolves any registered symlink for path.
// Returns the mapped target if path is a known symlink, else returns path unchanged.
func (f *FakeFS) EvalSymlinks(path string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if target, ok := f.Symlinks[path]; ok {
		return target, nil
	}
	return path, nil
}

// ReadFile returns the bytes stored at path.
// Returns os.ErrNotExist if path has not been written.
func (f *FakeFS) ReadFile(path string) ([]byte, error) {
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

// WriteFile stores data at path (permissions are ignored in memory).
// Records a create_file PlannedOp in addition to storing the data.
func (f *FakeFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := make([]byte, len(data))
	copy(stored, data)
	f.files[path] = stored
	f.ops = append(f.ops, PlannedOp{Op: "create_file", Path: path})
	return nil
}

// MkdirAll is a no-op: directory creation is implicit in file create ops.
// No PlannedOp is recorded (matches dryRunFS behaviour).
func (f *FakeFS) MkdirAll(_ string, _ os.FileMode) error { return nil }

// AppendFile appends data to the existing content at path.
// Records an append_marker PlannedOp.
func (f *FakeFS) AppendFile(path string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = append(f.files[path], data...)
	f.ops = append(f.ops, PlannedOp{Op: "append_marker", Path: path})
	return nil
}

// PlannedOps returns a copy of the accumulated PlannedOps slice.
func (f *FakeFS) PlannedOps() []PlannedOp {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ops) == 0 {
		return nil
	}
	out := make([]PlannedOp, len(f.ops))
	copy(out, f.ops)
	return out
}

// RecordOp appends an arbitrary PlannedOp (implements OpRecorder).
// Use this for service-layer ops that don't map to standard FSWriter methods.
func (f *FakeFS) RecordOp(op PlannedOp) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops = append(f.ops, op)
}

// FileCount returns the number of files written to the in-memory store.
// Useful in tests to assert no writes occurred (e.g., dry-run / error paths).
func (f *FakeFS) FileCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.files)
}

// HasFile returns true iff path exists in the in-memory store.
func (f *FakeFS) HasFile(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.files[path]
	return ok
}

// --- fakeFSFileInfo: minimal os.FileInfo for FakeFS.Stat ---

type fakeFSFileInfo struct {
	name      string
	size      int64
	isSymlink bool
}

func (fi *fakeFSFileInfo) Name() string { return fi.name }
func (fi *fakeFSFileInfo) Size() int64  { return fi.size }
func (fi *fakeFSFileInfo) Mode() fs.FileMode {
	if fi.isSymlink {
		return fs.ModeSymlink | 0o644
	}
	return 0o644
}
func (fi *fakeFSFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *fakeFSFileInfo) IsDir() bool        { return false }
func (fi *fakeFSFileInfo) Sys() any           { return nil }
