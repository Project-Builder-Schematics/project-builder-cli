package angular

import "os"

// buildEnv constructs the environment slice for cmd.Env.
//
// Security contract (default-deny):
//   - PATH is always injected from the host OS environment (REQ-07.3).
//   - No other variables are propagated in the S-000 skeleton.
//   - S-002 extends this with the full allowlist filter from req.EnvAllowlist.
//
// The allowlist parameter is reserved for S-002. In the skeleton it is
// unused (full filter wired in S-002).
func buildEnv(_ []string) []string { // fitness:allow-untyped-args env-allowlist
	// S-000 skeleton: inject PATH only.
	// S-002 will iterate allowlist and propagate matching os.Environ() entries.
	env := make([]string, 0, 1) // fitness:allow-untyped-args env-allowlist
	env = append(env, "PATH="+os.Getenv("PATH"))
	return env
}
