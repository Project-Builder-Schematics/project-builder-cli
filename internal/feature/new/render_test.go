// Package newfeature — render_test.go covers the render helpers.
//
// ADR-04: RenderPretty tests use outputtest.Spy to assert (method, args) —
// NOT bytes. RenderJSON tests retain io.Writer since JSON is structured data.
//
// REQ coverage: ADR-019 (all output via Output port — no direct fmt.Println).
package newfeature_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output/outputtest"
)

// Test_RenderPretty_DryRun_EmitsHeadingAndNewline verifies that RenderPretty
// in dry-run mode calls Heading("DRY RUN — no files written") and Newline.
func Test_RenderPretty_DryRun_EmitsHeadingAndNewline(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := newfeature.NewResult{
		DryRun: true,
	}
	newfeature.RenderPretty(spy, result)

	spy.AssertCalledWith(t, "Heading", "DRY RUN — no files written")
	spy.AssertCalled(t, "Newline")
}

// Test_RenderPretty_DryRun_CreateOp_EmitsBody verifies that a create_file
// planned op is emitted as Body with the path string.
func Test_RenderPretty_DryRun_CreateOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := newfeature.NewResult{
		DryRun: true,
		PlannedOps: []fswriter.PlannedOp{
			{Op: "create_file", Path: "/tmp/foo/factory.ts"},
		},
	}
	newfeature.RenderPretty(spy, result)

	spy.AssertCalledWith(t, "Body", "  Would create: /tmp/foo/factory.ts")
}

// Test_RenderPretty_DryRun_DefaultOp_EmitsBody verifies that non-create_file
// planned ops are emitted as Body with the op + path.
func Test_RenderPretty_DryRun_DefaultOp_EmitsBody(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := newfeature.NewResult{
		DryRun: true,
		PlannedOps: []fswriter.PlannedOp{
			{Op: "modify_config", Path: "/tmp/foo/project-builder.json"},
		},
	}
	newfeature.RenderPretty(spy, result)

	spy.AssertCalledWith(t, "Body", "  Would modify_config: /tmp/foo/project-builder.json")
}

// Test_RenderPretty_RealWrite_EmitsPaths verifies that real-write mode emits
// Path calls for each file in FilesCreated.
func Test_RenderPretty_RealWrite_EmitsPaths(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := newfeature.NewResult{
		DryRun:       false,
		FilesCreated: []string{"/tmp/foo/factory.ts", "/tmp/foo/schema.json"},
	}
	newfeature.RenderPretty(spy, result)

	spy.AssertCalledWith(t, "Path", "/tmp/foo/factory.ts")
	spy.AssertCalledWith(t, "Path", "/tmp/foo/schema.json")
}

// Test_RenderPretty_Warnings_EmitsWarning verifies that warnings are emitted
// as Warning calls regardless of dry-run mode (ADR-019).
func Test_RenderPretty_Warnings_EmitsWarning(t *testing.T) {
	t.Parallel()

	spy := outputtest.New()
	result := newfeature.NewResult{
		DryRun:   false,
		Warnings: []string{"collection 'default' now has 10 inline schematics"},
	}
	newfeature.RenderPretty(spy, result)

	spy.AssertCalledWith(t, "Warning", "collection 'default' now has 10 inline schematics")
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

// Test_WarnApproachingSchematicLimit verifies the soft warning message when the
// inline schematic count reaches the threshold (REQ-NSI-04).
func Test_WarnApproachingSchematicLimit(t *testing.T) {
	t.Parallel()

	// At threshold (10), the message must mention the collection and count.
	msg := newfeature.WarnApproachingSchematicLimit("default", 10)
	if msg == "" {
		t.Fatal("WarnApproachingSchematicLimit: returned empty string at threshold 10")
	}
	if !strings.Contains(msg, "10") {
		t.Errorf("WarnApproachingSchematicLimit: message does not mention count 10; got: %q", msg)
	}
	if !strings.Contains(msg, "default") {
		t.Errorf("WarnApproachingSchematicLimit: message does not mention collection 'default'; got: %q", msg)
	}
}

// Test_WarnApproachingSchematicLimit_Triangulate verifies a different count appears in the message.
func Test_WarnApproachingSchematicLimit_Triangulate(t *testing.T) {
	t.Parallel()

	msg := newfeature.WarnApproachingSchematicLimit("my-col", 15)
	if !strings.Contains(msg, "15") {
		t.Errorf("WarnApproachingSchematicLimit: message does not mention count 15; got: %q", msg)
	}
	if !strings.Contains(msg, "my-col") {
		t.Errorf("WarnApproachingSchematicLimit: message does not mention collection 'my-col'; got: %q", msg)
	}
}

// Test_WarnApproachingFileSize verifies the soft warning message when project-builder.json
// approaches the 20KB size limit (REQ-NSI-05).
func Test_WarnApproachingFileSize(t *testing.T) {
	t.Parallel()

	// 20480 bytes = 20 KB.
	msg := newfeature.WarnApproachingFileSize(20480)
	if msg == "" {
		t.Fatal("WarnApproachingFileSize: returned empty string at 20480 bytes")
	}
	if !strings.Contains(msg, "20") {
		t.Errorf("WarnApproachingFileSize: message does not mention KB size (20); got: %q", msg)
	}
}

// Test_WarnApproachingFileSize_Triangulate verifies a different byte size appears.
func Test_WarnApproachingFileSize_Triangulate(t *testing.T) {
	t.Parallel()

	// 25600 bytes = 25 KB.
	msg := newfeature.WarnApproachingFileSize(25600)
	if !strings.Contains(msg, "25") {
		t.Errorf("WarnApproachingFileSize: message does not mention KB size (25); got: %q", msg)
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
