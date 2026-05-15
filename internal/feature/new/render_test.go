// Package newfeature — render_test.go covers the stub render helpers.
//
// REQ coverage: ADR-019 (all output via Renderer — no direct fmt.Println).
// These tests verify the render helpers produce non-empty output and valid JSON.
package newfeature_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// Test_RenderPretty_DryRun_NonEmpty verifies that pretty-rendering a dry-run
// NewResult produces a non-empty string containing the dry-run indicator.
func Test_RenderPretty_DryRun_NonEmpty(t *testing.T) {
	t.Parallel()

	result := newfeature.NewResult{
		DryRun: true,
		PlannedOps: []fswriter.PlannedOp{
			{Op: "create_file", Path: "/tmp/foo/factory.ts"},
		},
	}

	var buf bytes.Buffer
	newfeature.RenderPretty(&buf, result)

	got := buf.String()
	if got == "" {
		t.Error("RenderPretty produced empty output for dry-run result")
	}
	if !strings.Contains(got, "DRY RUN") {
		t.Errorf("RenderPretty dry-run output missing 'DRY RUN' marker; got: %q", got)
	}
}

// Test_RenderJSON_ProducesValidJSON verifies that JSON-rendering a NewResult
// produces a valid JSON object with the expected fields.
func Test_RenderJSON_ProducesValidJSON(t *testing.T) {
	t.Parallel()

	result := newfeature.NewResult{
		DryRun: true,
		PlannedOps: []fswriter.PlannedOp{
			{Op: "create_file", Path: "/tmp/foo/factory.ts"},
		},
	}

	var buf bytes.Buffer
	if err := newfeature.RenderJSON(&buf, result); err != nil {
		t.Fatalf("RenderJSON: unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("RenderJSON output is not valid JSON: %v — raw: %s", err, buf.Bytes())
	}

	if _, ok := parsed["dry_run"]; !ok {
		t.Errorf("RenderJSON output missing 'dry_run' field; got: %s", buf.Bytes())
	}
}

// Test_RenderJSON_DoesNotEscapeHTML verifies SetEscapeHTML(false) per L-builder-init-03.
// Go's default json.Encoder replaces < with < and > with >.
// With SetEscapeHTML(false) those Unicode escapes must NOT appear in the output.
func Test_RenderJSON_DoesNotEscapeHTML(t *testing.T) {
	t.Parallel()

	result := newfeature.NewResult{
		SchematicName: "my-test",
	}

	var buf bytes.Buffer
	if err := newfeature.RenderJSON(&buf, result); err != nil {
		t.Fatalf("RenderJSON: unexpected error: %v", err)
	}

	got := buf.String()
	// The Unicode escape sequences < and > must NOT be present —
	// they would indicate the encoder is using HTML escaping mode.
	if strings.Contains(got, `<`) || strings.Contains(got, `>`) {
		t.Errorf("RenderJSON used HTML escaping (L-builder-init-03 violation); got: %s", got)
	}

	// The schematic_name field must contain the literal string.
	if !strings.Contains(got, "my-test") {
		t.Errorf("RenderJSON output missing 'my-test'; got: %s", got)
	}
}
