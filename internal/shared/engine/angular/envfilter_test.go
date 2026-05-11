package angular

// envfilter_test.go — white-box tests for buildEnv (unexported).
// Lives in package angular (not angular_test) to access unexported buildEnv.
//
// S-000 scope: PATH always present in result (REQ-07.3).
// Full allowlist filter tests are in S-002.

import (
	"strings"
	"testing"
)

// Test_BuildEnv_PathAlwaysPresent covers REQ-07.3: PATH is always included in
// cmd.Env regardless of the allowlist.
func Test_BuildEnv_PathAlwaysPresent(t *testing.T) {
	t.Parallel()

	// Empty allowlist — only PATH should appear.
	got := buildEnv(nil) // fitness:allow-untyped-args env-allowlist

	pathFound := false
	for _, entry := range got {
		if strings.HasPrefix(entry, "PATH=") {
			pathFound = true
		}
	}

	if !pathFound {
		t.Errorf("buildEnv(nil) = %v — PATH is missing; REQ-07.3 violated", got)
	}
}

// Test_BuildEnv_EmptyAllowlist_OnlyPath covers REQ-07.3 (skeleton variant):
// with an empty allowlist, ONLY PATH should appear (no other vars leaked).
func Test_BuildEnv_EmptyAllowlist_OnlyPath(t *testing.T) {
	t.Parallel()

	got := buildEnv(nil) // fitness:allow-untyped-args env-allowlist

	for _, entry := range got {
		if !strings.HasPrefix(entry, "PATH=") {
			t.Errorf("buildEnv(nil) contains unexpected entry %q — only PATH expected in S-000 skeleton", entry)
		}
	}
}
