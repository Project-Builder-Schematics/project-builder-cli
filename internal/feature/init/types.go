// Package initialise provides the data types and interfaces for the
// `builder init` command, which bootstraps a new project workspace.
package initialise

import (
	"context"
	"os"
	"time"
)

// MCPMode is the resolved value of the --mcp flag (REQ-MCP-01).
// It is determined in the handler after TTY detection and --non-interactive
// defaults are applied. The resolved value is stored in InitRequest.MCP.
type MCPMode string

const (
	// MCPPrompt causes the handler to prompt the user (TTY only) with the
	// locked prompt string. Incompatible with --non-interactive.
	MCPPrompt MCPMode = "prompt"

	// MCPYes causes the service to print the locked MCP instructions block
	// to stdout after all writes and install complete.
	MCPYes MCPMode = "yes"

	// MCPNo suppresses the MCP instructions block entirely.
	MCPNo MCPMode = "no"
)

// PackageManager is the typed identifier for a Node.js package manager.
type PackageManager string

const (
	// PMUnset means no package manager was specified via --pm flag.
	// Detection logic will infer the manager from lockfiles (S-005).
	PMUnset PackageManager = ""

	// PMNpm selects npm as the package manager.
	PMNpm PackageManager = "npm"

	// PMPnpm selects pnpm as the package manager.
	PMPnpm PackageManager = "pnpm"

	// PMYarn selects yarn as the package manager.
	PMYarn PackageManager = "yarn"

	// PMBun selects bun as the package manager.
	PMBun PackageManager = "bun"
)

// schematicsFolderName is the stable folder name for the schematics directory.
// REQ-SF-01 lock: must not be changed post-v1.0.0.
const schematicsFolderName = "schematics"

// mcpInstructions is the locked MCP instructions block (REQ-MCP-02 durable contract).
const mcpInstructions = `To complete MCP server setup:
  1. The MCP server design is in progress — see https://github.com/Project-Builder-Schematics/project-builder-cli (roadmap row 17 + C2).
  2. When available, run: builder mcp install (planned for builder-init-mcp-install).`

// mcpPromptQuestion is the locked prompt string shown to the user (REQ-MCP-01).
const mcpPromptQuestion = "Would you like to set up the Project Builder MCP server for your AI client? [y/N]: "

// InitRequest is the canonical input to Service.Init.
// The handler validates and canonicalises all fields before building this struct.
type InitRequest struct {
	// Directory is the absolute, cleaned target directory (REQ-DV-01).
	Directory string

	// Force allows overwriting an existing project-builder.json (REQ-EC-03).
	Force bool

	// DryRun records all intended file operations as PlannedOps without
	// writing anything to disk (REQ-DR-01).
	DryRun bool

	// JSON selects NDJSON output mode for the result (REQ-JO-01).
	JSON bool

	// NonInteractive disables all interactive prompts (REQ-CS-04).
	NonInteractive bool

	// PackageManagerFlag is the value from --pm flag (PMUnset if not supplied).
	PackageManagerFlag PackageManager

	// NoInstall skips the install subprocess (REQ-PD-03).
	NoInstall bool

	// NoSkill skips writing outputs 3, 4, and the SDK dev-dep atomically
	// (REQ-SA-03).
	NoSkill bool

	// Publishable selects the publishable template (REQ-CS-05).
	// Returns ErrCodeInitNotImplemented — planned for builder-init-publishable.
	Publishable bool

	// MCP is the resolved MCP mode after TTY detection and flag parsing
	// (REQ-MCP-01). Set by the handler before calling Service.Init.
	MCP MCPMode
}

// InitResult is the output produced by a successful Service.Init call.
// It is serialised to JSON via --json or rendered in pretty mode.
type InitResult struct {
	// Directory is the absolute path of the initialised project workspace.
	Directory string `json:"directory"`

	// DryRun is true iff the run recorded PlannedOps without writing files.
	DryRun bool `json:"dry_run"`

	// PlannedOps records the intended file operations in dry-run mode.
	// Omitted (empty array) in real-write mode.
	PlannedOps []PlannedOp `json:"planned_ops,omitempty"`

	// OutputsCreated lists the paths written in real-write mode.
	// Omitted in dry-run mode.
	OutputsCreated []string `json:"outputs_created,omitempty"`

	// PackageManager is the resolved package manager (e.g. "npm", "pnpm").
	PackageManager PackageManager `json:"package_manager,omitempty"`

	// Installed is true iff the package manager install subprocess ran and
	// exited successfully.
	Installed bool `json:"installed"`

	// MCPSetupOffered is true iff MCP=yes was resolved AND the instructions
	// block was emitted (or in dry-run: the mcp_setup_offered PlannedOp was
	// recorded). (REQ-MCP-03)
	MCPSetupOffered bool `json:"mcp_setup_offered"`

	// Warnings lists any non-fatal issues encountered during initialisation.
	Warnings []string `json:"warnings,omitempty"`
}

// PlannedOp records a single intended file operation in dry-run mode.
// The Op field uses a stable 5-value enum (REQ-DR-02).
type PlannedOp struct {
	// Op is the stable operation discriminator.
	// Values: create_file | append_marker | modify_devdep | install_package | mcp_setup_offered
	Op string `json:"op"`

	// Path is the target file or directory path. Omitted for mcp_setup_offered.
	Path string `json:"path,omitempty"`

	// Details carries supplementary information (e.g. package name for install_package).
	Details string `json:"details,omitempty"`
}

// FSWriter is the filesystem port for the init feature (ADR-020).
// All filesystem I/O in the service layer MUST go through this interface.
// Three implementations exist: osFS (real writes), dryRunFS (records PlannedOps),
// fakeFS (in-memory, for tests).
type FSWriter interface {
	// Stat returns file metadata for path, or an error if the path does not
	// exist or is not accessible.
	Stat(path string) (os.FileInfo, error)

	// Lstat returns file metadata for path without following symlinks.
	// Use this to detect whether a path is itself a symlink (REQ-AR-05).
	Lstat(path string) (os.FileInfo, error)

	// EvalSymlinks returns the path with all symlinks resolved.
	// Returns an error if any component of the path does not exist.
	// Use this to verify the resolved target stays within the project
	// directory (REQ-AR-05 symlink-safety check).
	EvalSymlinks(path string) (string, error)

	// ReadFile returns the contents of path, or an error.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to path with the given permissions.
	// Implementations MUST use an atomic write (temp file + rename) to avoid
	// partial writes (FF-init-02).
	WriteFile(path string, data []byte, perm os.FileMode) error

	// MkdirAll creates path and all parents with the given permissions.
	MkdirAll(path string, perm os.FileMode) error

	// AppendFile appends data to path, creating the file if it does not exist.
	AppendFile(path string, data []byte) error

	// PlannedOps returns the list of operations recorded in dry-run mode.
	// Returns nil for non-dry-run implementations.
	PlannedOps() []PlannedOp
}

// PackageManagerRunner is the port for package manager detection and install
// (ADR-023). Implemented by realPM (S-005) and fakePM (tests).
type PackageManagerRunner interface {
	// Detect returns the resolved package manager for the given directory.
	// flag overrides auto-detection when not PMUnset.
	Detect(dir string, flag PackageManager) (PackageManager, error)

	// Install runs the package manager install subprocess in dir.
	// The context carries the 120-second timeout (ADR-023).
	Install(ctx context.Context, dir string, pm PackageManager) error
}

// Service orchestrates the five init outputs via FSWriter and PackageManagerRunner.
type Service struct {
	fs    FSWriter
	pm    PackageManagerRunner
	skill []byte
	now   func() time.Time
}

// NewService constructs a Service with the given dependencies.
// skill is the bundled SKILL.md bytes (//go:embed, S-002).
// Pass []byte{} for S-000 walking skeleton — the real bytes land in S-002.
func NewService(fs FSWriter, pm PackageManagerRunner, skill []byte) *Service {
	return &Service{
		fs:    fs,
		pm:    pm,
		skill: skill,
		now:   time.Now,
	}
}
