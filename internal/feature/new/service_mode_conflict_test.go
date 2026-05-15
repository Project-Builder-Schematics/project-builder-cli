// Package newfeature — service_mode_conflict_test.go covers the preflight
// mode-conflict detection (ADR-026, REQ-NS-07, REQ-EC-05).
//
// REQ coverage:
//   - REQ-NS-07: --inline --force when path-mode entry exists → ErrCodeModeConflict
//   - REQ-EC-05: error message names "builder remove"
package newfeature_test

import (
	"context"
	"errors"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// setupWorkspaceWithPathSchematic creates a workspace with a path-mode schematic already
// registered in project-builder.json. Used to test mode-conflict scenarios.
func setupWorkspaceWithPathSchematic(t *testing.T, schematicName string) (string, *fswriter.FakeFS) {
	t.Helper()
	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// Create a path-mode schematic first.
	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:     schematicName,
		Language: "ts",
		WorkDir:  dir,
	})
	if err != nil {
		t.Fatalf("setup: failed to create path-mode schematic %q: %v", schematicName, err)
	}
	return dir, fs
}

// Test_ModeConflict_InlineWhenPathExists verifies ErrCodeModeConflict is returned
// when --inline is requested but the schematic already exists in path mode (REQ-NS-07).
func Test_ModeConflict_InlineWhenPathExists(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspaceWithPathSchematic(t, "existing-sch")
	svc := newfeature.NewService(fs)

	// Attempt to register the same schematic in inline mode (no --force).
	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:    "existing-sch",
		WorkDir: dir,
		Inline:  true,
	})
	if err == nil {
		t.Fatal("expected ErrCodeModeConflict; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeModeConflict {
		t.Errorf("error code = %q; want %q", e.Code, errs.ErrCodeModeConflict)
	}
}

// Test_ModeConflict_InlineForceWhenPathExists verifies ErrCodeModeConflict is returned
// even with --force when the schematic exists in path mode (ADV-10, REQ-EC-05).
func Test_ModeConflict_InlineForceWhenPathExists(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspaceWithPathSchematic(t, "existing-sch")
	svc := newfeature.NewService(fs)

	// Attempt with --inline --force.
	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:    "existing-sch",
		WorkDir: dir,
		Inline:  true,
		Force:   true,
	})
	if err == nil {
		t.Fatal("expected ErrCodeModeConflict even with --force; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}
	if e.Code != errs.ErrCodeModeConflict {
		t.Errorf("error code = %q; want %q", e.Code, errs.ErrCodeModeConflict)
	}

	// REQ-EC-05: error message MUST mention "builder remove" (recovery path).
	if e.Message == "" {
		t.Error("error message is empty")
	}
}

// Test_ModeConflict_MessageMentionsBuilderRemove verifies the error message names
// "builder remove" as the recovery path (REQ-EC-05).
func Test_ModeConflict_MessageMentionsBuilderRemove(t *testing.T) {
	t.Parallel()

	dir, fs := setupWorkspaceWithPathSchematic(t, "existing-sch")
	svc := newfeature.NewService(fs)

	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:    "existing-sch",
		WorkDir: dir,
		Inline:  true,
		Force:   true,
	})
	if err == nil {
		t.Fatal("expected error; got nil")
	}

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("error not *errs.Error; got: %T %v", err, err)
	}

	// The message MUST contain "builder remove" so the user knows the recovery path.
	if !errorMentionsBuilderRemove(e) {
		t.Errorf("ErrCodeModeConflict message does not mention 'builder remove'; got message: %q; suggestions: %v",
			e.Message, e.Suggestions)
	}
}

// Test_ModeConflict_NoConflict_NewSchematic verifies NO mode conflict when creating
// a brand-new schematic in inline mode (no prior path entry).
func Test_ModeConflict_NoConflict_NewSchematic(t *testing.T) {
	t.Parallel()

	dir, fs := setupE2EWorkspace(t)
	svc := newfeature.NewService(fs)

	// New schematic in inline mode — no prior entry → no conflict.
	_, err := svc.RegisterSchematic(context.Background(), newfeature.NewSchematicRequest{
		Name:    "brand-new",
		WorkDir: dir,
		Inline:  true,
	})
	if err != nil {
		t.Errorf("expected no error for brand-new inline schematic; got: %v", err)
	}
}

// errorMentionsBuilderRemove returns true if either the Error.Message or any
// Suggestion contains "builder remove".
func errorMentionsBuilderRemove(e *errs.Error) bool {
	if containsString(e.Message, "builder remove") {
		return true
	}
	for _, s := range e.Suggestions {
		if containsString(s, "builder remove") {
			return true
		}
	}
	return false
}

// containsString is a case-sensitive substring check.
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(haystack) > 0 && stringContains(haystack, needle))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
