package discoverer

import "testing"

// SetWellKnownNodePathsFn replaces the package's well-known-node-paths lookup
// for the duration of a single test. The original function is restored via
// t.Cleanup. Tests use this to neutralise CI runners that pre-install node
// at /usr/bin/node (which the production discovery chain would otherwise find
// and break "not-found" REQ-10.3 assertions).
func SetWellKnownNodePathsFn(t testing.TB, fn func() []string) {
	t.Helper()
	orig := wellKnownNodePathsFn
	wellKnownNodePathsFn = fn
	t.Cleanup(func() { wellKnownNodePathsFn = orig })
}
