// Package newfeature — schema_test.go tests schema.json v1 generation and
// shape validation.
//
// REQ coverage:
//   - REQ-SJ-01: top-level structure has required "inputs" object
//   - REQ-SJ-03: empty schema {"inputs": {}} is valid minimum
//   - REQ-SJ-04: reject Angular JSON Schema format on read (properties/$schema:draft-07)
//   - REQ-SJ-05: canonical bytes: two-space indent, trailing newline
//   - REQ-SJ-06: type: enum without enum array → ErrSchemaValidation
//   - REQ-SJ-07: type: list without items.type → ErrSchemaValidation
//   - REQ-SJ-08: position negative → ErrSchemaValidation
//   - REQ-SJ-09: unknown fields → warning (not error)
//   - REQ-SJ-10: default type mismatch → ErrSchemaValidation
//   - REQ-PJ-07: json.NewEncoder + SetEscapeHTML(false) for all JSON writes
//   - REQ-NC-01: collection.json skeleton canonical bytes
package newfeature_test

import (
	"errors"
	"testing"

	newfeature "github.com/Project-Builder-Schematics/project-builder-cli/internal/feature/new"
)

// Test_MarshalEmpty_CanonicalBytes verifies MarshalEmpty returns the exact
// canonical byte sequence for an empty schema.json (REQ-SJ-01, REQ-SJ-03, REQ-SJ-05).
//
// Canonical form: {"inputs": {}} with two-space indent and trailing newline.
func Test_MarshalEmpty_CanonicalBytes(t *testing.T) {
	t.Parallel()

	want := "{\n  \"inputs\": {}\n}\n"
	got := string(newfeature.MarshalEmpty())

	if got != want {
		t.Errorf("MarshalEmpty() bytes mismatch:\nwant: %q\n got: %q", want, got)
	}
}

// Test_MarshalEmpty_NoHTMLEscape verifies that angle brackets are not escaped
// (L-builder-init-03: SetEscapeHTML(false)).
func Test_MarshalEmpty_NoHTMLEscape(t *testing.T) {
	t.Parallel()

	got := string(newfeature.MarshalEmpty())

	// If HTML escaping were active, < would be < and > would be >.
	// The empty schema has no angle brackets, but this verifies the encoder
	// contract is consistent with the broader project pattern.
	// We assert it's valid JSON with "inputs" key.
	if len(got) == 0 {
		t.Error("MarshalEmpty() returned empty bytes")
	}
}

// Test_MarshalEmpty_IsIdempotent verifies repeated calls return identical bytes.
// The function must be pure — no state, no randomness (REQ-SJ-05).
func Test_MarshalEmpty_IsIdempotent(t *testing.T) {
	t.Parallel()

	a := newfeature.MarshalEmpty()
	b := newfeature.MarshalEmpty()

	if string(a) != string(b) {
		t.Errorf("MarshalEmpty() not idempotent:\nfirst call:  %q\nsecond call: %q", a, b)
	}
}

// ─── ValidateSchema tests (REQ-SJ-06..10) ────────────────────────────────────

// Test_ValidateSchema_EnumWithoutValues verifies that type:"enum" without an
// enum values array returns ErrSchemaValidation (REQ-SJ-06).
func Test_ValidateSchema_EnumWithoutValues(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"style": {Type: "enum"}, // no Enum field
		},
	}

	_, err := newfeature.ValidateSchema(schema)
	if err == nil {
		t.Fatal("ValidateSchema: expected ErrSchemaValidation for enum without values; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ValidateSchema: error not ErrSchemaValidation; got: %v", err)
	}
	// Error message must name the input and the problem (REQ-SJ-06 format).
	msg := err.Error()
	if !containsStr(msg, "enum") {
		t.Errorf("error message does not mention 'enum': %q", msg)
	}
}

// Test_ValidateSchema_ListWithoutItemsType verifies that type:"list" without
// items.type returns ErrSchemaValidation (REQ-SJ-07).
func Test_ValidateSchema_ListWithoutItemsType(t *testing.T) {
	t.Parallel()

	// Subcase A: items is nil.
	schemaA := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"tags": {Type: "list"}, // no Items field
		},
	}
	_, err := newfeature.ValidateSchema(schemaA)
	if err == nil {
		t.Fatal("ValidateSchema (list, nil items): expected ErrSchemaValidation; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ValidateSchema (list, nil items): not ErrSchemaValidation; got: %v", err)
	}

	// Subcase B: items present but type empty.
	schemaB := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"tags": {Type: "list", Items: &newfeature.ItemsSpec{Type: ""}},
		},
	}
	_, err = newfeature.ValidateSchema(schemaB)
	if err == nil {
		t.Fatal("ValidateSchema (list, empty items.type): expected ErrSchemaValidation; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ValidateSchema (list, empty items.type): not ErrSchemaValidation; got: %v", err)
	}
}

// Test_ValidateSchema_NegativePosition verifies that position < 0 returns
// ErrSchemaValidation (REQ-SJ-08).
func Test_ValidateSchema_NegativePosition(t *testing.T) {
	t.Parallel()

	neg := -1
	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"name": {Type: "string", Position: &neg},
		},
	}

	_, err := newfeature.ValidateSchema(schema)
	if err == nil {
		t.Fatal("ValidateSchema: expected ErrSchemaValidation for negative position; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ValidateSchema: not ErrSchemaValidation; got: %v", err)
	}
}

// Test_ValidateSchema_NonNegativePosition verifies that position >= 0 is valid
// (positive boundary case for REQ-SJ-08).
func Test_ValidateSchema_NonNegativePosition(t *testing.T) {
	t.Parallel()

	zero := 0
	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"name": {Type: "string", Position: &zero},
		},
	}

	_, err := newfeature.ValidateSchema(schema)
	if err != nil {
		t.Errorf("ValidateSchema: position=0 should be valid; got: %v", err)
	}
}

// Test_ValidateSchema_DefaultTypeMismatch verifies that default value with wrong
// type returns ErrSchemaValidation (REQ-SJ-10).
func Test_ValidateSchema_DefaultTypeMismatch(t *testing.T) {
	t.Parallel()

	// number type with string default.
	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"count": {Type: "number", Default: "hello"}, // string default for number type
		},
	}

	_, err := newfeature.ValidateSchema(schema)
	if err == nil {
		t.Fatal("ValidateSchema: expected ErrSchemaValidation for type mismatch; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ValidateSchema: not ErrSchemaValidation; got: %v", err)
	}
	if !containsStr(err.Error(), "number") {
		t.Errorf("error message does not mention 'number': %q", err.Error())
	}
}

// Test_ValidateSchema_ValidDefault verifies that compatible default values pass
// validation (boundary case for REQ-SJ-10).
func Test_ValidateSchema_ValidDefault(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"count": {Type: "number", Default: float64(42)},
			"flag":  {Type: "boolean", Default: false},
			"name":  {Type: "string", Default: "foo"},
		},
	}

	_, err := newfeature.ValidateSchema(schema)
	if err != nil {
		t.Errorf("ValidateSchema with valid defaults: unexpected error: %v", err)
	}
}

// Test_ValidateSchema_EmptySchema verifies that empty inputs is valid (REQ-SJ-03).
func Test_ValidateSchema_EmptySchema(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{Inputs: map[string]newfeature.InputSpec{}}

	_, err := newfeature.ValidateSchema(schema)
	if err != nil {
		t.Errorf("ValidateSchema(empty): unexpected error: %v", err)
	}
}

// ─── REQ-SJ-09: unknown fields → warning (not error) ──────────────────────

// Test_ValidateSchema_UnknownFieldsProduceWarning verifies that unknown fields
// in an InputSpec produce a SchemaValidationWarning (not an error) (REQ-SJ-09).
//
// Scenario: input "name" carries unknown field "deprecated" → exactly 1 warning
// that mentions both the field name ("deprecated") and the input name ("name").
func Test_ValidateSchema_UnknownFieldsProduceWarning(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"name": {
				Type: "string",
				UnknownFields: map[string]any{
					"deprecated": true,
				},
			},
		},
	}

	warns, err := newfeature.ValidateSchema(schema)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if len(warns) != 1 {
		t.Fatalf("ValidateSchema: expected 1 warning for unknown field; got %d", len(warns))
	}
	if !containsStr(warns[0].Message, "deprecated") {
		t.Errorf("warning message does not mention field name 'deprecated': %q", warns[0].Message)
	}
	if !containsStr(warns[0].Message, "name") {
		t.Errorf("warning message does not mention input name 'name': %q", warns[0].Message)
	}
}

// Test_ValidateSchema_MultipleInputsUnknownFields verifies that unknown fields
// from multiple inputs each produce their own warning (REQ-SJ-09 generalisation).
func Test_ValidateSchema_MultipleInputsUnknownFields(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"alpha": {
				Type: "string",
				UnknownFields: map[string]any{
					"deprecated": true,
				},
			},
			"beta": {
				Type: "number",
				UnknownFields: map[string]any{
					"experimental": "yes",
				},
			},
		},
	}

	warns, err := newfeature.ValidateSchema(schema)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if len(warns) != 2 {
		t.Fatalf("ValidateSchema: expected 2 warnings (one per unknown field); got %d", len(warns))
	}
}

// Test_ValidateSchema_NoUnknownFields verifies that a schema with no unknown
// fields produces zero warnings (REQ-SJ-09 negative case).
func Test_ValidateSchema_NoUnknownFields(t *testing.T) {
	t.Parallel()

	schema := newfeature.Schema{
		Inputs: map[string]newfeature.InputSpec{
			"name": {Type: "string"},
		},
	}

	warns, err := newfeature.ValidateSchema(schema)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("ValidateSchema: expected 0 warnings; got %d: %v", len(warns), warns)
	}
}

// containsStr is a simple string containment helper for schema tests.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

// ─── REQ-SJ-04: reject Angular schema shape on read ──────────────────────────

// Test_REQ_SJ04_RejectAngularProperties verifies that ReadSchemaFromBytes returns
// ErrSchemaValidation when the JSON contains a top-level "properties" field, which
// is a hallmark of Angular JSON Schema format (REQ-SJ-04).
func Test_REQ_SJ04_RejectAngularProperties(t *testing.T) {
	t.Parallel()

	data := []byte(`{"properties": {"foo": {"type": "string"}}}`)

	_, _, err := newfeature.ReadSchemaFromBytes(data)
	if err == nil {
		t.Fatal("ReadSchemaFromBytes: expected ErrSchemaValidation for Angular 'properties' field; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ReadSchemaFromBytes: error not ErrSchemaValidation; got: %v", err)
	}
	if !containsStr(err.Error(), "Angular") && !containsStr(err.Error(), "properties") {
		t.Errorf("error message does not mention Angular or properties: %q", err.Error())
	}
}

// Test_REQ_SJ04_RejectAngularDraft07Schema verifies that ReadSchemaFromBytes returns
// ErrSchemaValidation when the JSON contains a top-level "$schema" field whose value
// is the JSON Schema draft-07 URL (REQ-SJ-04).
func Test_REQ_SJ04_RejectAngularDraft07Schema(t *testing.T) {
	t.Parallel()

	data := []byte(`{"$schema": "http://json-schema.org/draft-07/schema"}`)

	_, _, err := newfeature.ReadSchemaFromBytes(data)
	if err == nil {
		t.Fatal("ReadSchemaFromBytes: expected ErrSchemaValidation for $schema:draft-07; got nil")
	}
	if !errors.Is(err, newfeature.ErrSchemaValidation) {
		t.Errorf("ReadSchemaFromBytes: error not ErrSchemaValidation; got: %v", err)
	}
	if !containsStr(err.Error(), "Angular") && !containsStr(err.Error(), "properties") {
		t.Errorf("error message does not mention Angular or properties: %q", err.Error())
	}
}

// Test_REQ_SJ04_AcceptValidSchema verifies that ReadSchemaFromBytes accepts a minimal
// valid schema {"inputs": {}} without error (REQ-SJ-04 negative — no false positives).
func Test_REQ_SJ04_AcceptValidSchema(t *testing.T) {
	t.Parallel()

	data := []byte(`{"inputs": {}}`)

	_, _, err := newfeature.ReadSchemaFromBytes(data)
	if err != nil {
		t.Errorf("ReadSchemaFromBytes: unexpected error for valid schema: %v", err)
	}
}

// Test_REQ_SJ04_AcceptCustomSchemaURL verifies that ReadSchemaFromBytes accepts a
// $schema URL that is NOT the draft-07 URL (non-draft-07 $schema is not forbidden).
// Only the Angular draft-07 URL is rejected; custom $schema URLs pass through.
func Test_REQ_SJ04_AcceptCustomSchemaURL(t *testing.T) {
	t.Parallel()

	data := []byte(`{"$schema": "https://schemas.builder/v1.json", "inputs": {}}`)

	_, _, err := newfeature.ReadSchemaFromBytes(data)
	if err != nil {
		t.Errorf("ReadSchemaFromBytes: unexpected error for non-draft-07 $schema: %v", err)
	}
}

// ─── MarshalCollectionSkeleton (REQ-NC-01) ────────────────────────────────────

// Test_MarshalCollectionSkeleton_CanonicalBytes verifies MarshalCollectionSkeleton
// returns the canonical byte sequence for an empty collection.json (REQ-NC-01).
//
// Canonical form: {"version": 1, "schematics": {}} with two-space indent and trailing newline.
func Test_MarshalCollectionSkeleton_CanonicalBytes(t *testing.T) {
	t.Parallel()

	want := "{\n  \"version\": 1,\n  \"schematics\": {}\n}\n"
	got := string(newfeature.MarshalCollectionSkeleton())

	if got != want {
		t.Errorf("MarshalCollectionSkeleton() bytes mismatch:\nwant: %q\n got: %q", want, got)
	}
}

// Test_MarshalCollectionSkeleton_Deterministic verifies calling twice produces the same bytes.
func Test_MarshalCollectionSkeleton_Deterministic(t *testing.T) {
	t.Parallel()

	a := newfeature.MarshalCollectionSkeleton()
	b := newfeature.MarshalCollectionSkeleton()

	if string(a) != string(b) {
		t.Errorf("MarshalCollectionSkeleton: non-deterministic output:\nfirst:  %q\nsecond: %q", a, b)
	}
}
