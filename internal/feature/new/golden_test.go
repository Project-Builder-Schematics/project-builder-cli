// Package newfeature — golden_test.go validates generated output against committed
// golden fixtures under testdata/golden/.
//
// REQ coverage:
//   - REQ-SJ-05: schema.json canonical bytes match golden fixture
//
// When generated output changes intentionally, update the golden files:
//
//	go test ./internal/feature/new/... -run Test_Golden -update
package newfeature_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
)

var updateGolden = flag.Bool("update", false, "overwrite golden fixture files with current output")

// Test_Golden_SchemaEmpty verifies MarshalEmpty bytes match the committed golden
// schema.json fixture (REQ-SJ-05).
func Test_Golden_SchemaEmpty(t *testing.T) {
	t.Parallel()

	goldenPath := filepath.Join("testdata", "golden", "schematic_empty", "schema.json")

	got := newfeature.MarshalEmpty()

	if *updateGolden {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil { //nolint:gosec // test fixture update path
			t.Fatalf("update golden: WriteFile: %v", err)
		}
		t.Logf("updated golden: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath) // #nosec G304 — test fixture path
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("schema.json golden mismatch:\nwant: %q\n got: %q", string(want), string(got))
	}
}
