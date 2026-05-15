// Package newfeature — schema.go generates schema.json v1 content.
//
// REQ coverage:
//   - REQ-SJ-01: top-level structure with required "inputs" object
//   - REQ-SJ-03: empty schema {"inputs": {}} is a valid minimum
//   - REQ-SJ-05: canonical byte sequence (two-space indent, trailing newline)
//   - REQ-PJ-07: json.NewEncoder + SetEscapeHTML(false) for all JSON writes
//     (L-builder-init-03)
package newfeature

import (
	"bytes"
	"encoding/json"
)

// emptySchema is the typed representation of an empty schema.json v1.
// The Inputs field uses map[string]any to produce the exact `{}` encoding.
type emptySchema struct {
	Inputs map[string]any `json:"inputs"`
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
