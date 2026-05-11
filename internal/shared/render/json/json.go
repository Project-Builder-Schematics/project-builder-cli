// Package json provides Renderer, a machine-facing output adapter that
// emits one JSON object per line (NDJSON format) for each event received.
//
// Import note: this package intentionally does NOT import
// internal/shared/render to avoid an import cycle (factory.go in render/
// imports render/json). Interface satisfaction is asserted in factory_test.go.
package json

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// mask returns "[REDACTED]" when sensitive is true, otherwise returns v unchanged.
// Inlined per ADR-03 amendment (2026-05-11): import-cycle prevents sharing the
// render-level helper; each adapter owns an identical copy.
func mask(v string, sensitive bool) string {
	if sensitive {
		return "[REDACTED]"
	}
	return v
}

// maskSlice returns []string{"[REDACTED]"} when sensitive is true,
// otherwise returns args unchanged (nil-safe).
func maskSlice(args []string, sensitive bool) any {
	if sensitive {
		return "[REDACTED]"
	}
	return args
}

// Renderer is the machine-facing output adapter for project-builder-cli.
// It emits NDJSON (one JSON object per line) for each event, with sensitive
// fields replaced by the "[REDACTED]" placeholder.
//
// Renderer structurally satisfies render.Renderer via its Render method.
// The compile-time assertion lives in factory_test.go (cycle-free).
type Renderer struct {
	enc *stdjson.Encoder
}

// New constructs a Renderer writing to w.
func New(w io.Writer) *Renderer {
	enc := stdjson.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Renderer{enc: enc}
}

// Render satisfies the render.Renderer interface. It emits one JSON object
// per event, newline-terminated (NDJSON). Sensitive fields are replaced with
// "[REDACTED]". Returns nil on channel close; respects ctx cancellation.
func (r *Renderer) Render(ctx context.Context, ch <-chan events.Event) error {
	var seq uint64
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			seq++
			env, err := toEnvelope(ev, seq)
			if err != nil {
				return fmt.Errorf("json.Renderer.Render: %w", err)
			}
			if err := r.enc.Encode(env); err != nil {
				return fmt.Errorf("json.Renderer.Render: encode: %w", err)
			}
		}
	}
}
