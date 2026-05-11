package angular

import (
	"os"
	"strings"
)

// buildEnv constructs the environment slice for cmd.Env using a default-deny
// allowlist strategy (REQ-07).
//
// Security contract:
//   - PATH is always injected from the host OS environment, exactly once (REQ-07.3).
//     PATH is required for Node.js to resolve modules; it is NOT user-controlled.
//   - Only variables named in allowlist are propagated from os.Environ() (REQ-07.2).
//   - Variables NOT in allowlist are silently excluded (REQ-07.1).
//   - Allowlist entries that do not exist in os.Environ() are silently skipped (REQ-07.4).
//   - If allowlist is empty (or nil), only PATH is present in the result.
//
// FF-08 note: the []string parameter is intentional — the allowlist is a list // fitness:allow-untyped-args env-allowlist
// of environment variable names, not arbitrary command arguments.
func buildEnv(allowlist []string) []string { // fitness:allow-untyped-args env-allowlist
	// Pre-allocate: PATH + however many allowlist entries resolve.
	env := make([]string, 0, len(allowlist)+1) // fitness:allow-untyped-args env-allowlist

	// PATH is always included exactly once (REQ-07.3).
	// We read it from os directly — never from user-controlled input.
	env = append(env, "PATH="+os.Getenv("PATH"))

	// Propagate allowlisted env vars from the host environment.
	// Only keys explicitly listed in allowlist are forwarded (default-deny).
	hostEnv := os.Environ()
	for _, key := range allowlist {
		// Skip PATH — already injected above; prevents duplicates.
		if strings.EqualFold(key, "PATH") {
			continue
		}
		// Look up the value in the host environment.
		// os.LookupEnv is used instead of os.Getenv to distinguish "not set"
		// from "set to empty string" (REQ-07.4: not-set → skip silently).
		val, found := lookupInEnv(hostEnv, key)
		if !found {
			// REQ-07.4: non-existent allowlist key — silently skipped.
			continue
		}
		env = append(env, key+"="+val)
	}

	return env
}

// lookupInEnv searches environ (a slice of "KEY=VALUE" strings) for key and
// returns the value and true if found, or "" and false if not.
//
// We parse from the raw slice rather than calling os.LookupEnv to avoid the
// os.Environ() call in a loop (which would re-snapshot the environment each
// time). The caller passes the snapshot once.
func lookupInEnv(environ []string, key string) (string, bool) { // fitness:allow-untyped-args env-allowlist
	prefix := key + "="
	for _, entry := range environ {
		if strings.HasPrefix(entry, prefix) {
			return entry[len(prefix):], true
		}
	}
	return "", false
}
