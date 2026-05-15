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

	// Collections is the mutable collections map for PATH-mode schematics.
	// Each collection maps schematic names to their path entries.
	// Outer key: collection name (e.g. "default").
	// Inner key: schematic name. Value: json.RawMessage ({"path": "..."}).
	Collections map[string]map[string]json.RawMessage

	// Inlines is the mutable map for INLINE-mode schematics (REQ-PJ-06).
	// Inline entries nest under "schematics" within the collection object:
	//   "collections": { "default": { "schematics": { "<name>": {"inputs": {}} } } }
	// Outer key: collection name (e.g. "default").
	// Inner key: schematic name. Value: json.RawMessage ({"inputs": {}}).
	Inlines map[string]map[string]json.RawMessage

	// CollectionPaths is the mutable map for top-level collection registrations
	// (REQ-NC-01). Each entry is a collection name → relative path to collection.json.
	// Serialised as: "collections": { "<name>": {"path": "<relPath>"} }
	// These are siblings of "default" at the top level of the collections object.
	CollectionPaths map[string]string

	// Extra holds all OTHER top-level fields not explicitly decoded above.
	// They are written back verbatim on WriteConfig (REQ-PJ-04).
	Extra map[string]json.RawMessage

	// Warnings accumulates non-fatal issues discovered during ReadConfig.
	// Callers propagate these into NewResult.Warnings so Renderer prints them
	// (ADR-019). Example: UTF-8 BOM detected and stripped (ADV-06).
	Warnings []string
}

// schematicPathEntry is the JSON shape for a path-mode schematic registration.
// REQ-PJ-05: {"path": "./schematics/<name>"}
type schematicPathEntry struct {
	Path string `json:"path"`
}

// collectionEntry is the JSON shape for a top-level collection registration.
// REQ-NC-01: {"path": "./schematics/<name>/collection.json"}
type collectionEntry struct {
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

	// Strip UTF-8 BOM if present (ADV-06: some editors prepend \xEF\xBB\xBF).
	// Note: WARN surfacing for BOM detection is implemented in Group B (next commit).
	data, _ = StripBOM(data)

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
		Collections:     make(map[string]map[string]json.RawMessage),
		Inlines:         make(map[string]map[string]json.RawMessage),
		CollectionPaths: make(map[string]string),
		Extra:           make(map[string]json.RawMessage),
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
				var colEntries map[string]json.RawMessage
				if err := json.Unmarshal(colRaw, &colEntries); err != nil {
					// Tolerate malformed collection entry — store as empty map.
					colEntries = make(map[string]json.RawMessage)
				}

				// Detect collection-level path entry (REQ-NC-01).
				// A collection entry has {"path": "<string>"} where the path value is
				// a JSON string (no other keys, or "path" is the only meaningful key).
				// This distinguishes it from a schematic container ("default") which
				// has schematic name keys or a "schematics" sub-key.
				if pathRaw, haspath := colEntries["path"]; haspath && isJSONString(pathRaw) {
					var colPath string
					if err := json.Unmarshal(pathRaw, &colPath); err == nil {
						cfg.CollectionPaths[colName] = colPath
						// Do NOT also add to Collections — it is a collection, not a schematic container.
						continue
					}
				}

				// Separate inline entries (under "schematics" key) from path entries.
				pathEntries := make(map[string]json.RawMessage)
				for entryKey, entryRaw := range colEntries {
					if entryKey == "schematics" {
						// Inline schematics are nested under "schematics" sub-key.
						var inlineMap map[string]json.RawMessage
						if err := json.Unmarshal(entryRaw, &inlineMap); err == nil {
							cfg.Inlines[colName] = inlineMap
						}
					} else {
						pathEntries[entryKey] = entryRaw
					}
				}
				cfg.Collections[colName] = pathEntries
			}
		default:
			cfg.Extra[key] = raw
		}
	}

	return cfg, nil
}

// isJSONString returns true iff the raw JSON token is a quoted string.
// Used to distinguish collection-level path entries from schematic container maps.
func isJSONString(raw json.RawMessage) bool {
	return len(raw) >= 2 && raw[0] == '"'
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

	// Marshal collections (merging path entries + inline entries + top-level collection paths).
	colsEncoded, err := marshalCollections(cfg.Collections, cfg.Inlines, cfg.CollectionPaths)
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

// marshalCollections merges path-mode entries (cfg.Collections), inline-mode
// entries (cfg.Inlines), and top-level collection paths (cfg.CollectionPaths)
// into a single JSON object for project-builder.json.
//
// For each schematic container (e.g. "default"), the output object combines:
//   - Direct path-mode entries: { "<name>": { "path": "..." }, ... }
//   - Inline entries nested under "schematics": { "schematics": { "<name>": { "inputs": {} } } }
//
// For each top-level collection (e.g. "bar"), the output is a simple path object:
//   - { "path": "./schematics/bar/collection.json" }
func marshalCollections(
	collections map[string]map[string]json.RawMessage,
	inlines map[string]map[string]json.RawMessage,
	collectionPaths map[string]string,
) (json.RawMessage, error) {
	// Collect all collection names (union of all maps).
	allCols := make(map[string]struct{})
	for colName := range collections {
		allCols[colName] = struct{}{}
	}
	for colName := range inlines {
		allCols[colName] = struct{}{}
	}
	for colName := range collectionPaths {
		allCols[colName] = struct{}{}
	}

	if len(allCols) == 0 {
		return json.RawMessage("{}"), nil
	}

	// Use a raw map to allow heterogeneous value types (schematic containers vs collection paths).
	resultRaw := make(map[string]json.RawMessage, len(allCols))

	for colName := range allCols {
		// Top-level collection path entries (REQ-NC-01) — serialised as {"path": "..."}.
		if relPath, isColPath := collectionPaths[colName]; isColPath {
			entry := collectionEntry{Path: relPath}
			b, err := json.Marshal(entry)
			if err != nil {
				return nil, err
			}
			resultRaw[colName] = b
			continue
		}

		// Schematic container (e.g. "default") — build merged map.
		colObj := make(map[string]json.RawMessage)

		// Add path-mode entries directly.
		for schName, schRaw := range collections[colName] {
			colObj[schName] = schRaw
		}

		// Add inline entries under "schematics" sub-key.
		if inlineEntries, ok := inlines[colName]; ok && len(inlineEntries) > 0 {
			inlineBytes, err := json.Marshal(inlineEntries)
			if err != nil {
				return nil, err
			}
			colObj["schematics"] = json.RawMessage(inlineBytes)
		}

		colObjBytes, err := json.Marshal(colObj)
		if err != nil {
			return nil, err
		}
		resultRaw[colName] = colObjBytes
	}

	b, err := json.Marshal(resultRaw)
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

// schematicInlineEntry is the JSON shape for an inline schematic registration.
// REQ-PJ-06: {"inputs": {}}
type schematicInlineEntry struct {
	Inputs map[string]json.RawMessage `json:"inputs"`
}

// RegisterSchematicInline mutates cfg to add (or overwrite) an inline schematic
// entry in the given collection (REQ-PJ-06).
//
// Inline-mode entry shape: {"inputs": {}}
// Key: cfg.Inlines[collection][name]
//
// Idempotent: calling with the same (collection, name) is a no-op in terms of
// output content (REQ-PJ-02).
func RegisterSchematicInline(cfg *Config, collection, name string) error {
	entry := schematicInlineEntry{Inputs: make(map[string]json.RawMessage)}
	b, err := json.Marshal(entry)
	if err != nil {
		return &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "projectconfig.registerSchematicInline",
			Message: "failed to serialise inline schematic entry",
			Cause:   err,
		}
	}

	if _, ok := cfg.Inlines[collection]; !ok {
		cfg.Inlines[collection] = make(map[string]json.RawMessage)
	}
	cfg.Inlines[collection][name] = b
	return nil
}

// SchematicExists returns true iff a schematic entry for name exists in the given
// collection, regardless of whether it is a path-mode or inline-mode entry.
func SchematicExists(cfg *Config, collection, name string) bool {
	// Check path-mode entries.
	if col, ok := cfg.Collections[collection]; ok {
		if _, exists := col[name]; exists {
			return true
		}
	}
	// Check inline-mode entries.
	if inl, ok := cfg.Inlines[collection]; ok {
		if _, exists := inl[name]; exists {
			return true
		}
	}
	return false
}

// SchematicExistsInPathMode returns true iff a schematic with the given name exists
// as a path-mode entry (direct key under the collection, with a "path" field).
// Returns false for inline-mode entries. Used by mode-conflict detection (ADR-026).
func SchematicExistsInPathMode(cfg *Config, collection, name string) bool {
	col, ok := cfg.Collections[collection]
	if !ok {
		return false
	}
	_, exists := col[name]
	return exists
}

// CountInlineSchematics returns the total number of inline schematics in the given
// collection. Used by soft-warning threshold checks (REQ-NSI-04).
func CountInlineSchematics(cfg *Config, collection string) int {
	if inl, ok := cfg.Inlines[collection]; ok {
		return len(inl)
	}
	return 0
}

// RegisterCollection mutates cfg to add (or overwrite) a top-level collection
// entry in the collections map (REQ-NC-01).
//
// Collection entry shape: {"path": relPath}
// The collection is stored in cfg.CollectionPaths[name] = relPath, which is
// serialised by WriteConfig/marshalCollections as a sibling of "default" at the
// top level of the "collections" JSON object.
//
// Idempotent: calling with the same (name, relPath) is a no-op (REQ-PJ-02).
func RegisterCollection(cfg *Config, name, relPath string) error {
	if cfg.CollectionPaths == nil {
		cfg.CollectionPaths = make(map[string]string)
	}
	cfg.CollectionPaths[name] = relPath
	return nil
}

// CollectionExists returns true iff a top-level collection entry exists for name.
func CollectionExists(cfg *Config, name string) bool {
	if cfg.CollectionPaths == nil {
		return false
	}
	_, ok := cfg.CollectionPaths[name]
	return ok
}
