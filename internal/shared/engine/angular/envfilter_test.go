package angular

// envfilter_test.go — white-box tests for buildEnv (unexported).
// Lives in package angular (not angular_test) to access unexported buildEnv.
//
// S-000 scope: PATH always present in result (REQ-07.3).
// S-002 scope:
//   - REQ-07.1: var NOT in allowlist does not reach child
//   - REQ-07.2: var IN allowlist is propagated to child
//   - REQ-07.3: PATH always included (regardless of allowlist)
//   - REQ-07.4: non-existent allowlist keys silently skipped

import (
	"os"
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
			t.Errorf("buildEnv(nil) contains unexpected entry %q — only PATH expected in empty allowlist", entry)
		}
	}
}

// Test_BuildEnv_VarNotInAllowlist covers REQ-07.1:
// env var NOT in EnvAllowlist does NOT appear in result.
func Test_BuildEnv_VarNotInAllowlist(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	const secretKey = "PB_TEST_SECRET_TOKEN_12345" //nolint:gosec // test constant — not a real credential
	t.Setenv(secretKey, "supersecret")

	// Allow an unrelated var — NOT secretKey.
	got := buildEnv([]string{"NODE_PATH"}) // fitness:allow-untyped-args env-allowlist

	for _, entry := range got {
		if strings.HasPrefix(entry, secretKey+"=") {
			t.Errorf("buildEnv result contains %q — should not leak; REQ-07.1 violated", entry)
		}
	}
}

// Test_BuildEnv_VarInAllowlist covers REQ-07.2:
// env var IN allowlist is propagated to child.
func Test_BuildEnv_VarInAllowlist(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	const key = "PB_TEST_NODE_PATH_12345"
	const val = "/usr/local/lib"
	t.Setenv(key, val)

	got := buildEnv([]string{key}) // fitness:allow-untyped-args env-allowlist

	found := false
	for _, entry := range got {
		if entry == key+"="+val {
			found = true
		}
	}
	if !found {
		t.Errorf("buildEnv result does not contain %s=%s — REQ-07.2 violated; got: %v", key, val, got)
	}
}

// Test_BuildEnv_NonExistentAllowlistKey covers REQ-07.4:
// non-existent allowlist key is silently skipped (no error, no empty entry).
func Test_BuildEnv_NonExistentAllowlistKey(t *testing.T) {
	t.Parallel()

	const nonExistent = "PB_TEST_NONEXISTENT_VAR_99999"
	// Guarantee it doesn't exist.
	os.Unsetenv(nonExistent) //nolint:errcheck // test cleanup

	got := buildEnv([]string{nonExistent}) // fitness:allow-untyped-args env-allowlist

	for _, entry := range got {
		if strings.HasPrefix(entry, nonExistent+"=") {
			t.Errorf("buildEnv result contains %q — non-existent key must be skipped; REQ-07.4 violated", entry)
		}
	}
}

// Test_BuildEnv_PathExactlyOne covers REQ-07.3 (invariant):
// PATH appears exactly once even if explicitly in allowlist too.
func Test_BuildEnv_PathExactlyOne(t *testing.T) {
	t.Parallel()

	// Include PATH explicitly in the allowlist.
	got := buildEnv([]string{"PATH"}) // fitness:allow-untyped-args env-allowlist

	count := 0
	for _, entry := range got {
		if strings.HasPrefix(entry, "PATH=") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("PATH appears %d times, want exactly 1 — REQ-07.3 violated; env: %v", count, got)
	}
}
