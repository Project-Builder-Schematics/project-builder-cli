// Package discoverer locates the Node.js binary and the Angular schematics-cli
// executable required by AngularSubprocessAdapter.
//
// # Priority chain — FindNode
//
// S-000 (skeleton): NODE_BINARY env var only.
// S-004 (full chain): NODE_BINARY → PATH exec.LookPath → well-known paths.
//
// # Concrete struct — no interface
//
// Per project ADR (locked in sdd-init/project-builder-cli): Discoverer is a
// concrete struct, not an interface. Testability is achieved via the
// NODE_BINARY environment variable override (highest-priority discovery path).
package discoverer

import (
	"os"

	appErrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// Discoverer locates runtime binaries needed by AngularSubprocessAdapter.
type Discoverer struct{}

// New returns a new Discoverer instance.
func New() *Discoverer {
	return &Discoverer{}
}

// FindNode returns the absolute path to a Node.js binary.
//
// S-000 skeleton: checks NODE_BINARY env var only.
// Full priority chain (PATH + well-known paths + version validation) is
// implemented in S-004.
//
// Returns *errors.Error{Code: ErrCodeEngineNotFound, Op: "angular.discover_node"}
// when Node.js cannot be located.
func (d *Discoverer) FindNode() (string, error) {
	if bin := os.Getenv("NODE_BINARY"); bin != "" {
		return bin, nil
	}

	// S-000 skeleton: full discovery chain (PATH + well-known paths) added in S-004.
	return "", &appErrors.Error{
		Code:    appErrors.ErrCodeEngineNotFound,
		Op:      "angular.discover_node",
		Message: "Node.js binary not found; set NODE_BINARY or ensure node is on PATH",
		Suggestions: []string{
			"Install Node.js >= 18 and ensure it is on your PATH",
			"Or set the NODE_BINARY environment variable to the absolute path of the node binary",
		},
	}
}

// FindSchematics returns the absolute path to the @angular-devkit/schematics-cli
// binary for the given workspace directory.
//
// S-000 skeleton: stub — always returns ErrCodeEngineNotFound.
// Full discovery (workspace node_modules + PATH + version validation) is
// implemented in S-004.
func (d *Discoverer) FindSchematics(_ string) (string, error) {
	return "", &appErrors.Error{
		Code:    appErrors.ErrCodeEngineNotFound,
		Op:      "angular.discover_schematics",
		Message: "schematics-cli not found; run: npm install --save-dev @angular-devkit/schematics-cli",
		Suggestions: []string{
			"Install @angular-devkit/schematics-cli >= 17 in the project workspace",
			"Or install globally: npm install -g @angular-devkit/schematics-cli",
		},
	}
}
