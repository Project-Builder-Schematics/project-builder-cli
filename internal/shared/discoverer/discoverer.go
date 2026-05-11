// Package discoverer locates the Node.js binary and the Angular schematics-cli
// executable required by AngularSubprocessAdapter.
//
// # Priority chain — FindNode
//
// 1. NODE_BINARY env var (if set and the binary passes version check)
// 2. exec.LookPath("node") on PATH
// 3. Well-known platform paths (e.g. /usr/local/bin/node, ~/.nvm/...)
//
// # Priority chain — FindSchematics
//
// 1. {workspace}/node_modules/.bin/schematics (project-local install)
// 2. exec.LookPath("schematics") on PATH
//
// # Concrete struct — no interface
//
// Per project ADR (locked in sdd-init/project-builder-cli): Discoverer is a
// concrete struct, not an interface. Testability is achieved via the
// NODE_BINARY environment variable override (highest-priority discovery path).
package discoverer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	appErrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// minNodeMajor is the minimum Node.js major version required (REQ-10.2).
const minNodeMajor = 18

// minSchematicsMajor is the minimum schematics-cli major version required (REQ-11.3).
const minSchematicsMajor = 17

// Discoverer locates runtime binaries needed by AngularSubprocessAdapter.
type Discoverer struct{}

// New returns a new Discoverer instance.
func New() *Discoverer {
	return &Discoverer{}
}

// FindNode returns the absolute path to a Node.js binary that satisfies the
// minimum version requirement (>= 18.0.0).
//
// Priority chain (REQ-10):
//  1. NODE_BINARY env var — if set and executable, version-validated
//  2. exec.LookPath("node") on PATH — version-validated
//  3. Well-known paths (platform-specific) — version-validated
//
// Returns *errors.Error{Code: ErrCodeEngineNotFound, Op: "angular.discover_node"}
// when Node.js cannot be located or the found version is too old.
func (d *Discoverer) FindNode() (string, error) {
	// Priority 1: NODE_BINARY env var.
	if bin := os.Getenv("NODE_BINARY"); bin != "" {
		return validateNodeVersion(bin)
	}

	// Priority 2: exec.LookPath("node").
	if bin, err := exec.LookPath("node"); err == nil {
		return validateNodeVersion(bin)
	}

	// Priority 3: well-known paths.
	for _, candidate := range wellKnownNodePathsFn() {
		if _, err := os.Stat(candidate); err == nil {
			return validateNodeVersion(candidate)
		}
	}

	return "", &appErrors.Error{
		Code:    appErrors.ErrCodeEngineNotFound,
		Op:      "angular.discover_node",
		Message: "Node.js binary not found; set NODE_BINARY or ensure node is on PATH",
		Suggestions: []string{
			fmt.Sprintf("Install Node.js >= %d and ensure it is on your PATH", minNodeMajor),
			"Or set the NODE_BINARY environment variable to the absolute path of the node binary",
		},
	}
}

// FindSchematics returns the absolute path to the @angular-devkit/schematics-cli
// binary for the given workspace directory.
//
// Priority chain (REQ-11):
//  1. {workspace}/node_modules/.bin/schematics — version-validated
//  2. exec.LookPath("schematics") on PATH — version-validated
//
// Returns *errors.Error{Code: ErrCodeEngineNotFound, Op: "angular.discover_schematics"}
// when schematics-cli cannot be located or the found version is too old.
func (d *Discoverer) FindSchematics(workspace string) (string, error) {
	// Priority 1: project-local install.
	localBin := filepath.Join(workspace, "node_modules", ".bin", schematicsName())
	if _, err := os.Stat(localBin); err == nil {
		return validateSchematicsVersion(localBin)
	}

	// Priority 2: PATH lookup.
	if bin, err := exec.LookPath(schematicsName()); err == nil {
		return validateSchematicsVersion(bin)
	}

	return "", &appErrors.Error{
		Code:    appErrors.ErrCodeEngineNotFound,
		Op:      "angular.discover_schematics",
		Message: fmt.Sprintf("schematics-cli not found; version >= %d.0.0 required", minSchematicsMajor),
		Suggestions: []string{
			fmt.Sprintf("Install @angular-devkit/schematics-cli >= %d in the project workspace:", minSchematicsMajor),
			"  npm install --save-dev @angular-devkit/schematics-cli",
			"Or install globally: npm install -g @angular-devkit/schematics-cli",
		},
	}
}

// --- version validation ---

// validateNodeVersion runs `bin --version`, parses the output, and validates
// that the major version meets the minimum requirement.
func validateNodeVersion(bin string) (string, error) {
	versionStr, err := runVersionFlag(bin)
	if err != nil {
		return "", &appErrors.Error{
			Code:    appErrors.ErrCodeEngineNotFound,
			Op:      "angular.discover_node",
			Message: fmt.Sprintf("failed to run %q --version: %v", bin, err),
			Cause:   err,
		}
	}

	major, err := parseMajorVersion(versionStr)
	if err != nil {
		return "", &appErrors.Error{
			Code:    appErrors.ErrCodeEngineNotFound,
			Op:      "angular.discover_node",
			Message: fmt.Sprintf("could not parse Node.js version %q from %q", versionStr, bin),
			Cause:   err,
		}
	}

	if major < minNodeMajor {
		return "", &appErrors.Error{
			Code:    appErrors.ErrCodeEngineNotFound,
			Op:      "angular.discover_node",
			Message: fmt.Sprintf("Node.js %s is too old; minimum required is v%d.0.0 (found: %s)", bin, minNodeMajor, versionStr),
			Suggestions: []string{
				fmt.Sprintf("Upgrade Node.js to >= %d.0.0", minNodeMajor),
				"Use nvm: nvm install --lts",
			},
		}
	}

	return bin, nil
}

// validateSchematicsVersion runs `bin --version`, parses the output, and validates
// that the major version meets the minimum schematics requirement.
func validateSchematicsVersion(bin string) (string, error) {
	versionStr, err := runVersionFlag(bin)
	if err != nil {
		return "", &appErrors.Error{
			Code:    appErrors.ErrCodeEngineNotFound,
			Op:      "angular.discover_schematics",
			Message: fmt.Sprintf("failed to run %q --version: %v", bin, err),
			Cause:   err,
		}
	}

	major, err := parseMajorVersion(versionStr)
	if err != nil {
		return "", &appErrors.Error{
			Code:    appErrors.ErrCodeEngineNotFound,
			Op:      "angular.discover_schematics",
			Message: fmt.Sprintf("could not parse schematics version %q from %q", versionStr, bin),
			Cause:   err,
		}
	}

	if major < minSchematicsMajor {
		return "", &appErrors.Error{
			Code:    appErrors.ErrCodeEngineNotFound,
			Op:      "angular.discover_schematics",
			Message: fmt.Sprintf("schematics-cli version %s is too old; minimum required is %d.0.0", versionStr, minSchematicsMajor),
			Suggestions: []string{
				fmt.Sprintf("Upgrade @angular-devkit/schematics-cli to >= %d.0.0", minSchematicsMajor),
				"npm install --save-dev @angular-devkit/schematics-cli@latest",
			},
		}
	}

	return bin, nil
}

// runVersionFlag executes `bin --version` and returns the trimmed stdout output.
func runVersionFlag(bin string) (string, error) {
	//nolint:gosec // G204: bin is from NODE_BINARY / exec.LookPath, not user input.
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// parseMajorVersion parses a semver string like "v20.11.0" or "17.3.0"
// and returns the major version number.
func parseMajorVersion(version string) (int, error) {
	// Strip leading "v" if present.
	v := strings.TrimPrefix(version, "v")

	// Split on "." and parse the first segment.
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("empty version string")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("non-numeric major version %q: %w", parts[0], err)
	}
	return major, nil
}

// schematicsName returns the platform-appropriate binary name for schematics-cli.
func schematicsName() string {
	if runtime.GOOS == "windows" {
		return "schematics.cmd"
	}
	return "schematics"
}

// wellKnownNodePathsFn is the indirection used by FindNode to look up
// well-known Node.js install paths. Production code uses wellKnownNodePaths;
// tests override it via export_test.go SetWellKnownNodePathsFn to isolate
// from the host environment (notably CI runners that pre-install node).
var wellKnownNodePathsFn = wellKnownNodePaths

// wellKnownNodePaths returns a list of common Node.js installation paths for
// the current platform. Used as a fallback when NODE_BINARY and PATH fail.
func wellKnownNodePaths() []string {
	switch runtime.GOOS {
	case "darwin", "linux":
		home, _ := os.UserHomeDir()
		candidates := []string{
			"/usr/local/bin/node",
			"/usr/bin/node",
			"/opt/homebrew/bin/node",
		}
		if home != "" {
			// Add nvm-style paths (glob not supported — add common major versions).
			for _, major := range []string{"20", "18", "22", "21", "19"} {
				candidates = append(
					candidates,
					filepath.Join(home, ".nvm", "versions", "node", "v"+major+".0.0", "bin", "node"),
				)
			}
			// volta
			candidates = append(candidates, filepath.Join(home, ".volta", "bin", "node"))
		}
		return candidates
	case "windows":
		return []string{
			`C:\Program Files\nodejs\node.exe`,
			`C:\Program Files (x86)\nodejs\node.exe`,
		}
	default:
		return nil
	}
}
