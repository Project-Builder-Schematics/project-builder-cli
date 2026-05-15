// Package newfeature provides the `builder new` command, which scaffolds
// schematics and collections in an existing workspace.
//
// The package is structured to mirror internal/feature/init: a parent command
// with leaf subcommands (schematic, collection), a service orchestrator, and
// stub handlers that return ErrCodeNewNotImplemented until real logic lands
// in S-001 through S-005.
package newfeature

import (
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// PlannedOp records a single intended file operation in dry-run mode.
// Alias for fswriter.PlannedOp — no duplication; same underlying type.
type PlannedOp = fswriter.PlannedOp

// NewSchematicRequest is the canonical input to Service.RegisterSchematic.
// The handler validates and canonicalises all fields before building this struct.
type NewSchematicRequest struct {
	// Name is the schematic name (kebab-case; validated against metachar list).
	Name string

	// Collection is the target collection name. Defaults to "default" when empty.
	Collection string

	// WorkDir is the workspace root directory (absolute path).
	// The service uses this to locate project-builder.json and construct output paths.
	WorkDir string

	// Force allows overwriting an existing schematic (REQ-NS-03).
	Force bool

	// DryRun records all intended file operations as PlannedOps without writing
	// to disk (REQ-NS-05).
	DryRun bool

	// Inline selects inline-embedded mode instead of path mode (REQ-NSI-01).
	Inline bool

	// Language is the explicit language override ("ts" or "js").
	// Empty string means auto-detect (REQ-LG-01..03).
	Language string

	// Extends is the optional --extends value in @scope/pkg:collection grammar
	// (REQ-EX-01). Empty string means no extends.
	Extends string

	// OutputJSON selects NDJSON output mode for the result.
	OutputJSON bool
}

// NewCollectionRequest is the canonical input to Service.RegisterCollection.
type NewCollectionRequest struct {
	// Name is the collection name (kebab-case; validated against metachar list).
	Name string

	// WorkDir is the workspace root directory (absolute path).
	WorkDir string

	// Force allows overwriting an existing collection (REQ-NC-03).
	Force bool

	// DryRun records all intended file operations as PlannedOps without writing
	// to disk (REQ-NC-06).
	DryRun bool

	// Publishable selects the --publishable template which creates add/ and
	// remove/ lifecycle stubs (REQ-NCP-01).
	Publishable bool

	// OutputJSON selects NDJSON output mode for the result.
	OutputJSON bool
}

// NewResult is the output produced by a successful Service call.
// It is serialised to JSON via --output=json or rendered in pretty mode.
type NewResult struct {
	// DryRun is true iff the run recorded PlannedOps without writing files.
	DryRun bool `json:"dry_run"`

	// PlannedOps records the intended file operations in dry-run mode.
	// Omitted (nil) in real-write mode.
	PlannedOps []PlannedOp `json:"planned_ops,omitempty"`

	// FilesCreated lists the absolute paths written in real-write mode.
	// Omitted in dry-run mode.
	FilesCreated []string `json:"files_created,omitempty"`

	// SchematicName is the name of the created schematic (if applicable).
	SchematicName string `json:"schematic_name,omitempty"`

	// CollectionName is the name of the created collection (if applicable).
	CollectionName string `json:"collection_name,omitempty"`

	// Warnings lists any non-fatal issues encountered during the operation.
	Warnings []string `json:"warnings,omitempty"`

	// ExtendsUsed is the --extends value that was actually wired into the schematic
	// registration (either from the flag or from the interactive PromptExtends call).
	// Empty when no extends was specified or the prompt was skipped.
	ExtendsUsed string `json:"extends_used,omitempty"`
}
