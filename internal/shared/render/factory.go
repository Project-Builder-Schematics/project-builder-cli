package render

import (
	"io"
	"os"

	apperrors "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	renderjson "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/json"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"
)

// OutputMode is the typed flag value for --output.
// Using a named type prevents untyped strings from being passed where an
// OutputMode is expected, making misuse a compile-time error.
type OutputMode string

const (
	// OutputModePretty selects PrettyRenderer — human-readable, lipgloss-styled.
	OutputModePretty OutputMode = "pretty"

	// OutputModeJSON selects JSONRenderer — NDJSON, one object per line.
	OutputModeJSON OutputMode = "json"

	// OutputModeAuto defers adapter selection to TTY detection at runtime.
	// Production callers pass OutputModeAuto when --output is unset.
	// The factory resolves this via the injected isTTY function (ADR-02).
	OutputModeAuto OutputMode = ""
)

// NewRenderer constructs the appropriate Renderer implementation based on mode
// and TTY detection.
//
//   - OutputModePretty → *pretty.PrettyRenderer (stdout as writer)
//   - OutputModeJSON   → *renderjson.JSONRenderer (stdout as writer)
//   - OutputModeAuto   → delegates to isTTY(); true → Pretty, false → JSON
//   - Any other value  → *errors.Error{Code: ErrCodeInvalidInput}
//
// The isTTY parameter is injected for testability (ADR-02). Production callers
// should pass:
//
//	func() bool { return isatty.IsTerminal(os.Stdout.Fd()) }
//
// or use the convenience constructor NewRendererForStdout.
func NewRenderer(mode OutputMode, isTTY func() bool) (Renderer, *apperrors.Error) {
	return newRenderer(mode, isTTY, os.Stdout)
}

// newRenderer is the internal constructor, accepting an explicit writer so
// tests can capture output without replacing os.Stdout.
func newRenderer(mode OutputMode, isTTY func() bool, w io.Writer) (Renderer, *apperrors.Error) {
	switch mode {
	case OutputModePretty:
		return pretty.New(w), nil
	case OutputModeJSON:
		return renderjson.New(w), nil
	case OutputModeAuto:
		if isTTY() {
			return pretty.New(w), nil
		}
		return renderjson.New(w), nil
	default:
		return nil, &apperrors.Error{
			Code:    apperrors.ErrCodeInvalidInput,
			Op:      "render.NewRenderer",
			Message: "invalid output mode: must be \"pretty\", \"json\", or \"\" (auto); got \"" + string(mode) + "\"",
			Suggestions: []string{
				"Use --output=pretty for human-readable terminal output.",
				"Use --output=json for machine-readable NDJSON output.",
				"Omit --output to let the CLI detect the terminal automatically.",
			},
		}
	}
}
