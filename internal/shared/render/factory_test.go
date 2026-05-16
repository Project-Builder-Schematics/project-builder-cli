package render_test

import (
	"testing"

	renderjson "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/json"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/pretty"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/theme"
)

// compile-time interface satisfaction assertions (REQ-01.1, REQ-05.1).
// These live here (not in adapter packages) to avoid a render → render/json →
// render import cycle. The factory is the natural declaration site since it
// is the only package that imports both adapters AND the render interface.
var (
	_ render.Renderer = (*pretty.Renderer)(nil)
	_ render.Renderer = (*renderjson.Renderer)(nil)
)

// isTTYTrue and isTTYFalse are test stubs for the isTTY injection parameter (ADR-02).
var (
	isTTYTrue  = func() bool { return true }
	isTTYFalse = func() bool { return false }
)

// noColorTheme is a deterministic NoColor theme for use in factory tests.
var noColorTheme = theme.New(theme.Palette{}, theme.NoColor, theme.Light)

// Test_NewRenderer_ExplicitMode covers REQ-09.1, REQ-09.2, REQ-09.3.
// When --output is explicitly set, TTY state is irrelevant.
func Test_NewRenderer_ExplicitMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mode        render.OutputMode
		isTTY       func() bool
		wantPretty  bool
		wantJSON    bool
		wantErrCode errors.ErrCode
	}{
		{
			name:       "pretty mode returns PrettyRenderer",
			mode:       render.OutputModePretty,
			isTTY:      isTTYFalse, // irrelevant when mode explicit
			wantPretty: true,
		},
		{
			name:     "json mode returns JSONRenderer",
			mode:     render.OutputModeJSON,
			isTTY:    isTTYTrue, // irrelevant when mode explicit
			wantJSON: true,
		},
		{
			name:        "unknown mode returns ErrCodeInvalidInput",
			mode:        render.OutputMode("xml"),
			isTTY:       isTTYFalse,
			wantErrCode: errors.ErrCodeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := render.NewRenderer(tt.mode, noColorTheme, tt.isTTY)

			if tt.wantErrCode != "" {
				if err == nil {
					t.Fatalf("expected error with code %q, got nil", tt.wantErrCode)
				}
				var appErr *errors.Error
				if !asError(err, &appErr) {
					t.Fatalf("expected *errors.Error, got %T: %v", err, err)
				}
				if appErr.Code != tt.wantErrCode {
					t.Errorf("error code = %q; want %q", appErr.Code, tt.wantErrCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r == nil {
				t.Fatal("NewRenderer returned nil renderer")
			}

			if tt.wantPretty {
				if _, ok := r.(*pretty.Renderer); !ok {
					t.Errorf("got %T; want *pretty.Renderer", r)
				}
			}
			if tt.wantJSON {
				if _, ok := r.(*renderjson.Renderer); !ok {
					t.Errorf("got %T; want *renderjson.Renderer", r)
				}
			}
		})
	}
}

// Test_NewRenderer_AutoMode covers REQ-10.1, REQ-11.1, REQ-12.1.
// When mode is OutputModeAuto (""), the isTTY function determines the adapter.
func Test_NewRenderer_AutoMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		isTTY      func() bool
		wantPretty bool
		wantJSON   bool
	}{
		{
			name:     "non-TTY stdout defaults to json.Renderer (REQ-10.1)",
			isTTY:    isTTYFalse,
			wantJSON: true,
		},
		{
			name:       "TTY stdout defaults to pretty.Renderer (REQ-11.1)",
			isTTY:      isTTYTrue,
			wantPretty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := render.NewRenderer(render.OutputModeAuto, noColorTheme, tt.isTTY)
			if err != nil {
				t.Fatalf("unexpected error for auto mode: %v", err)
			}
			if r == nil {
				t.Fatal("NewRenderer returned nil renderer")
			}

			if tt.wantPretty {
				if _, ok := r.(*pretty.Renderer); !ok {
					t.Errorf("got %T; want *pretty.Renderer", r)
				}
			}
			if tt.wantJSON {
				if _, ok := r.(*renderjson.Renderer); !ok {
					t.Errorf("got %T; want *renderjson.Renderer", r)
				}
			}
		})
	}
}

// Test_NewRenderer_TTYInjectionHonoured covers REQ-12.1.
// Verifies the factory strictly honours the injected isTTY function and does
// not fall back to os.Stdout or any other system call.
func Test_NewRenderer_TTYInjectionHonoured(t *testing.T) {
	t.Parallel()

	// isTTY=true → pretty.Renderer regardless of the real terminal state.
	r, err := render.NewRenderer(render.OutputModeAuto, noColorTheme, isTTYTrue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := r.(*pretty.Renderer); !ok {
		t.Errorf("isTTY=true: got %T; want *pretty.Renderer", r)
	}

	// isTTY=false → json.Renderer regardless of the real terminal state.
	r2, err2 := render.NewRenderer(render.OutputModeAuto, noColorTheme, isTTYFalse)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if _, ok := r2.(*renderjson.Renderer); !ok {
		t.Errorf("isTTY=false: got %T; want *renderjson.Renderer", r2)
	}
}

// asError is a minimal helper that checks if err is (or wraps) *errors.Error.
// We use errors.As pattern inline to avoid importing "errors" package aliases.
func asError(err error, target **errors.Error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*errors.Error); ok {
		*target = e
		return true
	}
	return false
}
