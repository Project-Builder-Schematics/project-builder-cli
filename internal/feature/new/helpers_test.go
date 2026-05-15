// Package newfeature_test — helpers_test.go provides OS-level test helpers
// used by adversarial tests that require real filesystem interaction.
//
// These helpers are separated from the main test files to keep ADV-09 and
// ADV-08 OS-level tests self-contained.
package newfeature_test

import (
	"os"
	"runtime"
)

// isWindows returns true iff the current OS is Windows.
// Used to skip filesystem-permission tests (chmod semantics differ on Windows).
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// writeFileOS writes data to path using os.WriteFile directly.
// For use in test setup that must bypass FSWriter (e.g. writing project-builder.json
// before making a directory read-only in ADV-09).
func writeFileOS(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644) //nolint:gosec // test-only; path from t.TempDir()
}

// mkdirOS creates directory path with the given permissions using os.Mkdir.
func mkdirOS(path string, perm os.FileMode) error {
	return os.Mkdir(path, perm)
}

// chmodOS changes the permissions of path using os.Chmod.
func chmodOS(path string, perm os.FileMode) error {
	return os.Chmod(path, perm)
}

// dirExistsOS reports whether path exists and is a directory (real OS check).
func dirExistsOS(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
