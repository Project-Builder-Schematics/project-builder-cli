// Package newfeature — projectconfig.go reads, mutates, and writes project-builder.json.
//
// This implementation is INLINE in feature/new/ for v1 per ADR-027.
// // FOLLOWUP F-01: promote to internal/shared/projectconfig/ before builder-add lands.
// FF-17 asserts the marker above exists in this file.
//
// # Design (ADR-027)
//
// Unlike init's project_config.go (write-only with locked bytes), this file
// implements a full read/mutate/write cycle. The read preserves ALL top-level
// fields via json.RawMessage (REQ-PJ-04), and the write uses json.NewEncoder
// with SetEscapeHTML(false) + SetIndent("  ") (L-builder-init-03).
//
// # Version field verbatim preservation (R-RES-1 / REQ-PJ-03)
//
// builder init writes `"version": "1"` (string). REQ-PJ-03 mandates this value
// is preserved VERBATIM after any mutation. The Config.Version field is typed as
// json.RawMessage so the exact JSON token bytes are round-tripped unchanged —
// whether the token is `"1"` (string) or `1` (integer) depends on what was in the
// file; we never coerce.
//
// # Concurrent writes (ADV-07 / REQ-PJ-01)
//
// All writes use FSWriter.WriteFile which uses write-temp + rename (atomic on
// POSIX). No advisory lock is used in v1 — OS-level rename atomicity is the only
// guarantee. See GoDoc on WriteConfig below.
//
// # ADR-012 compliance
//
// This file does NOT import from internal/feature/init/. It is an independent
// implementation. The init package's project_config.go is referenced only as a
// pattern (write pattern, not import).
package newfeature

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/fswriter"
)

// Config is the parsed, mutable representation of project-builder.json.
//
// Known top-level fields are decoded into typed fields; all OTHER top-level
// fields are preserved as json.RawMessage in Extra (REQ-PJ-04 unknown-field
// preservation).
//
// Version is typed as json.RawMessage to preserve the token verbatim on write
// (R-RES-1): init writes `"1"` (string), and we must not silently coerce it to
// the integer 1.
type Config struct {
	// Version is the raw JSON token from "version" — preserved verbatim on write.
	Version json.RawMessage

	// Collections is the mutable collections map. Each collection maps
	// schematic names to their entries (path or inline shape).
	// Outer key: collection name (e.g. "default").
	// Inner key: schematic name. Value: json.RawMessage for flexibility.
	Collections map[string]map[string]json.RawMessage

	// Extra holds all OTHER top-level fields not explicitly decoded above.
	// They are written back verbatim on WriteConfig (REQ-PJ-04).
	Extra map[string]json.RawMessage
}

// schematicPathEntry is the JSON shape for a path-mode schematic registration.
// REQ-PJ-05: {"path": "./schematics/<name>"}
type schematicPathEntry struct {
	Path string `json:"path"`
}

// ReadConfig reads and parses project-builder.json from dir via the given FSWriter.
//
// Returns ErrCodeInvalidInput (REQ-PJ-08) on parse failure.
// Returns os.ErrNotExist-wrapped error if the file is absent.
func ReadConfig(dir string, fs fswriter.FSWriter) (*Config, error) {
	path := filepath.Join(dir, "project-builder.json")
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// First pass: decode into a flat map to capture ALL top-level keys.
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "projectconfig.read",
			Message: "project-builder.json: failed to parse JSON. Run 'builder validate' to diagnose.",
			Cause:   err,
		}
	}

	cfg := &Config{
		Collections: make(map[string]map[string]json.RawMessage),
		Extra:       make(map[string]json.RawMessage),
	}

	// Extract known fields by key; put everything else in Extra.
	for key, raw := range rawMap {
		switch key {
		case "version":
			cfg.Version = raw
		case "collections":
			var colsRaw map[string]json.RawMessage
			if err := json.Unmarshal(raw, &colsRaw); err != nil {
				return nil, &errs.Error{
					Code:    errs.ErrCodeInvalidInput,
					Op:      "projectconfig.read",
					Message: "project-builder.json: 'collections' field is not a JSON object.",
					Cause:   err,
				}
			}
			for colName, colRaw := range colsRaw {
				var entries map[string]json.RawMessage
				if err := json.Unmarshal(colRaw, &entries); err != nil {
					// Tolerate malformed collection entry — store as empty map.
					entries = make(map[string]json.RawMessage)
				}
				cfg.Collections[colName] = entries
			}
		default:
			cfg.Extra[key] = raw
		}
	}

	return cfg, nil
}

// WriteConfig serialises cfg back to project-builder.json in dir via the FSWriter.
//
// Write strategy:
//   - Rebuilds the full JSON object by merging Extra (unknown fields) + version +
//     collections into a single ordered map for encoding.
//   - Uses json.NewEncoder + SetEscapeHTML(false) + SetIndent("", "  ") per
//     L-builder-init-03.
//   - Appends a trailing newline (consistent with init's locked-bytes contract).
//   - Writes atomically via FSWriter.WriteFile (write-temp + rename per REQ-PJ-01).
//
// Concurrent writes rely on OS-level rename atomicity; no advisory lock is used in v1.
func WriteConfig(dir string, cfg *Config, fs fswriter.FSWriter) error {
	// Merge everything into one flat map for ordered encoding.
	out := make(map[string]json.RawMessage, len(cfg.Extra)+2)

	// Copy Extra fields first (unknown top-level keys; REQ-PJ-04).
	for k, v := range cfg.Extra {
		out[k] = v
	}

	// Overwrite / set known fields.
	if cfg.Version != nil {
		out["version"] = cfg.Version
	}

	// Marshal collections.
	colsEncoded, err := marshalCollections(cfg.Collections)
	if err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "projectconfig.write",
			Message: "failed to serialise collections map",
			Cause:   err,
		}
	}
	out["collections"] = colsEncoded

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	if err := enc.Encode(out); err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "projectconfig.write",
			Message: "failed to encode project-builder.json",
			Cause:   err,
		}
	}

	// json.Encoder.Encode already appends a newline — buf ends with \n.
	path := filepath.Join(dir, "project-builder.json")
	return fs.WriteFile(path, buf.Bytes(), 0o644)
}

// marshalCollections converts the in-memory collections map to a json.RawMessage.
func marshalCollections(collections map[string]map[string]json.RawMessage) (json.RawMessage, error) {
	if len(collections) == 0 {
		return json.RawMessage("{}"), nil
	}
	b, err := json.Marshal(collections)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RegisterSchematicPath mutates cfg to add (or overwrite) a path-mode schematic
// entry in the given collection (REQ-PJ-05).
//
// Path-mode entry shape: {"path": relPath}
// Key: cfg.Collections[collection][name]
//
// Idempotent: calling with the same (collection, name, relPath) is a no-op
// in terms of output content (REQ-PJ-02).
func RegisterSchematicPath(cfg *Config, collection, name, relPath string) error {
	entry := schematicPathEntry{Path: relPath}
	b, err := json.Marshal(entry)
	if err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "projectconfig.registerSchematicPath",
			Message: "failed to serialise schematic path entry",
			Cause:   err,
		}
	}

	if _, ok := cfg.Collections[collection]; !ok {
		cfg.Collections[collection] = make(map[string]json.RawMessage)
	}
	cfg.Collections[collection][name] = b
	return nil
}

// SchematicExists returns true iff a schematic entry for name exists in the given
// collection, regardless of whether it is a path-mode or inline-mode entry.
func SchematicExists(cfg *Config, collection, name string) bool {
	col, ok := cfg.Collections[collection]
	if !ok {
		return false
	}
	_, exists := col[name]
	return exists
}
