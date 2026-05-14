// Package initialise — schematics_skel.go creates the schematics/ directory
// skeleton via the FSWriter port.
//
// The schematics/ directory is the workspace for local schematics. It is
// bootstrapped with a single .gitkeep file whose content is locked by the spec
// (REQ-SF-01). The constant schematicsFolderName (defined in types.go) holds
// the stable folder name — changing it after v1.0.0 is a BREAKING CHANGE.
//
// No direct os.* calls (FF-init-02 invariant — all I/O via FSWriter).
package initialise

import (
	"path/filepath"
)

// gitkeepContent is the locked byte content of schematics/.gitkeep (REQ-SF-01).
// This is a durable contract — changing post-v1.0.0 breaks existing git histories
// for users who committed this file.
const gitkeepContent = "# This folder holds local schematics for this project.\n" +
	"# Use `builder add <name>` to scaffold a new schematic here.\n"

// writeSchematicsSkel creates the schematics/ directory and writes the locked
// .gitkeep content via the FSWriter port.
//
// Returns the absolute path of the written .gitkeep file on success.
//
// No direct os.* calls (FF-init-02 invariant).
func writeSchematicsSkel(fs FSWriter, req InitRequest) (string, error) {
	schematicsDir := filepath.Join(req.Directory, schematicsFolderName)
	gitkeepPath := filepath.Join(schematicsDir, ".gitkeep")

	if err := fs.MkdirAll(schematicsDir, 0o755); err != nil {
		return "", err
	}

	if err := fs.WriteFile(gitkeepPath, []byte(gitkeepContent), 0o644); err != nil {
		return "", err
	}

	return gitkeepPath, nil
}
