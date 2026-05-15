// Package newfeature — schema.go generates schema.json v1 content and provides
// the Schema type hierarchy used by tsgen.go for .d.ts codegen.
//
// REQ coverage:
//   - REQ-SJ-01: top-level structure with required "inputs" object
//   - REQ-SJ-02: InputSpec fields and type enum
//   - REQ-SJ-03: empty schema {"inputs": {}} is a valid minimum
//   - REQ-SJ-05: canonical byte sequence (two-space indent, trailing newline)
//   - REQ-SJ-06: enum without values → ErrSchemaValidation
//   - REQ-SJ-07: list without items.type → ErrSchemaValidation
//   - REQ-SJ-08: negative position → ErrSchemaValidation
//   - REQ-SJ-09: unknown fields → warning (not error)
//   - REQ-SJ-10: default type mismatch → ErrSchemaValidation
//   - REQ-PJ-07: json.NewEncoder + SetEscapeHTML(false) for all JSON writes
//     (L-builder-init-03)
package newfeature

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrSchemaValidation is returned by ValidateSchema for structural violations.
// Use errors.Is to check for this sentinel.
var ErrSchemaValidation = errors.New("schema validation error")

// Schema represents the parsed schema.json v1 structure (REQ-SJ-01/02).
// All inputs are optional in v1 — tsgen marks every property as "T?".
type Schema struct {
	// Inputs maps input names to their specifications.
	// Keys are the raw names as written in schema.json (pre-EscapeIdent).
	Inputs map[string]InputSpec `json:"inputs"`
}

// InputSpec is a single input definition within schema.json (REQ-SJ-02).
type InputSpec struct {
	// Type is REQUIRED: one of "string", "number", "boolean", "enum", "list".
	Type string `json:"type"`

	// Description is an optional human-readable description.
	Description string `json:"description,omitempty"`

	// Position is an optional non-negative integer for CLI positional ordering.
	Position *int `json:"position,omitempty"`

	// Default is optional; must be type-compatible with Type.
	Default any `json:"default,omitempty"`

	// Enum lists string literals (REQUIRED when Type == "enum"; ≥1 item).
	Enum []string `json:"enum,omitempty"`

	// Items specifies the element type (REQUIRED when Type == "list").
	Items *ItemsSpec `json:"items,omitempty"`

	// UnknownFields captures any JSON fields not listed above (REQ-SJ-09).
	// Populated by custom UnmarshalJSON when decoding schema.json from disk.
	// In-memory construction (e.g. tests, code-gen) may set this directly.
	// The json:"-" tag ensures unknown fields are NOT re-serialised on output.
	UnknownFields map[string]any `json:"-"`
}

// ItemsSpec describes the element type for list inputs (REQ-SJ-02).
type ItemsSpec struct {
	// Type is the element type (same enum as InputSpec.Type, minus "list").
	Type string `json:"type"`
}

// SchemaValidationWarning carries a non-fatal schema validation issue (REQ-SJ-09).
type SchemaValidationWarning struct {
	Field   string
	Message string
}

// ValidateSchema checks the structural invariants of a Schema (REQ-SJ-06..10).
// Returns (warnings, error). Error wraps ErrSchemaValidation on hard violations.
// Warnings are non-fatal; processing continues.
func ValidateSchema(s Schema) ([]SchemaValidationWarning, error) {
	var warns []SchemaValidationWarning

	for name, spec := range s.Inputs {
		// REQ-SJ-06: enum type must have enum values.
		if spec.Type == "enum" && len(spec.Enum) == 0 {
			return warns, fmt.Errorf("%w: input %q has type 'enum' but no 'enum' values array", ErrSchemaValidation, name)
		}

		// REQ-SJ-07: list type must have items.type.
		if spec.Type == "list" && (spec.Items == nil || spec.Items.Type == "") {
			return warns, fmt.Errorf("%w: input %q has type 'list' but no 'items.type' defined", ErrSchemaValidation, name)
		}

		// REQ-SJ-08: position must be non-negative.
		if spec.Position != nil && *spec.Position < 0 {
			return warns, fmt.Errorf("%w: input %q: 'position' must be a non-negative integer", ErrSchemaValidation, name)
		}

		// REQ-SJ-10: default type mismatch.
		if spec.Default != nil {
			if err := checkDefaultType(name, spec); err != nil {
				return warns, err
			}
		}

		// REQ-SJ-09: unknown fields → warning (not error). Processing continues.
		for field := range spec.UnknownFields {
			warns = append(warns, SchemaValidationWarning{
				Field:   field,
				Message: fmt.Sprintf("input %q: unknown field %q — unrecognised fields are ignored but may indicate a typo", name, field),
			})
		}
	}

	return warns, nil
}

// checkDefaultType validates that the Default value is type-compatible (REQ-SJ-10).
func checkDefaultType(name string, spec InputSpec) error {
	switch spec.Type {
	case "number":
		switch spec.Default.(type) {
		case float64, int, int64, float32:
			return nil
		}
		return fmt.Errorf("%w: input %q: 'default' value type does not match declared type 'number'", ErrSchemaValidation, name)
	case "boolean":
		if _, ok := spec.Default.(bool); !ok {
			return fmt.Errorf("%w: input %q: 'default' value type does not match declared type 'boolean'", ErrSchemaValidation, name)
		}
	case "string":
		if _, ok := spec.Default.(string); !ok {
			return fmt.Errorf("%w: input %q: 'default' value type does not match declared type 'string'", ErrSchemaValidation, name)
		}
	}
	return nil
}

// emptySchema is the typed representation of an empty schema.json v1.
// The Inputs field uses map[string]any to produce the exact `{}` encoding.
type emptySchema struct {
	Inputs map[string]any `json:"inputs"`
}

// collectionSkeleton is the typed representation of a minimal collection.json.
// REQ-NC-01: {"version": 1, "schematics": {}}
type collectionSkeleton struct {
	Version    int            `json:"version"`
	Schematics map[string]any `json:"schematics"`
}

// MarshalCollectionSkeleton returns the canonical byte sequence for an empty collection.json.
//
// Canonical form (REQ-NC-01):
//
//	{
//	  "version": 1,
//	  "schematics": {}
//	}
//
// (Two-space indent, trailing newline, no BOM, no HTML escaping per L-builder-init-03.)
//
// Pure function; deterministic; no side effects; no I/O.
func MarshalCollectionSkeleton() []byte {
	v := collectionSkeleton{Version: 1, Schematics: map[string]any{}}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		panic("schema.MarshalCollectionSkeleton: unexpected encode error: " + err.Error())
	}
	return buf.Bytes()
}

// MarshalEmpty returns the canonical byte sequence for an empty schema.json.
//
// Canonical form (REQ-SJ-05):
//
//	{
//	  "inputs": {}
//	}
//
// (Two-space indent, trailing newline, no BOM, no HTML escaping per L-builder-init-03.)
//
// Pure function; deterministic; no side effects; no I/O.
func MarshalEmpty() []byte {
	v := emptySchema{Inputs: map[string]any{}}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	// Encode never fails for a fixed, valid struct — panic is unreachable in practice.
	if err := enc.Encode(v); err != nil {
		panic("schema.MarshalEmpty: unexpected encode error: " + err.Error())
	}

	return buf.Bytes()
}
