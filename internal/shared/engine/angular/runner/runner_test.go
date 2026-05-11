// Package runner_test covers the embedded runner script.
package runner_test

import (
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/engine/angular/runner"
)

// Test_Script_NonEmpty covers REQ-19.1: embedded runner bytes are non-empty.
func Test_Script_NonEmpty(t *testing.T) {
	t.Parallel()

	if len(runner.Script) == 0 {
		t.Fatal("runner.Script is empty — REQ-19.1 violated: //go:embed runner.js must produce non-empty bytes")
	}
}
