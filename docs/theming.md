# Theming

Reference guide for the `builder` CLI color system: the 8-token semantic palette,
terminal capability detection, and the architectural constraints that keep hex literals
out of all code except the theme package.

---

## 1. The 8 Semantic Tokens

| Token        | Light     | Dark      | Semantic role                                                              |
|--------------|-----------|-----------|----------------------------------------------------------------------------|
| `Primary`    | `#8B5CF6` | `#A78BFA` | Identity, headings, primary borders, prompts                               |
| `Accent`     | `#0D9488` | `#2DD4BF` | Actions, touched paths, highlighted values (teal — distinct from Success)  |
| `Foreground` | `#0F172A` | `#F8FAFC` | Main body text                                                             |
| `Muted`      | `#64748B` | `#94A3B8` | Metadata, hints, footers, neutral borders                                  |
| `Background` | `#FFFFFF` | `#0F172A` | Canvas / panel fill                                                        |
| `Success`    | `#16A34A` | `#22C55E` | Completed operations (pure green)                                          |
| `Warning`    | `#D97706` | `#F59E0B` | Dry-run mode, destructive operations, detected drift                       |
| `Error`      | `#E11D48` | `#F43F5E` | Failed operations                                                          |

Source of truth: `internal/shared/render/theme/palette.go` (`DefaultPalette()`).
These hex values are canonical and byte-for-byte fixed (theme-tokens/REQ-02.1).

Design notes:
- `Accent` (teal) and `Success` (green) are intentionally distinct. `Accent` means
  "this is what was touched / highlighted"; `Success` means "operation completed
  successfully". Do not conflate them.
- `Muted` serves double duty: metadata/hints AND neutral decorative borders. For
  structural borders, prefer `Primary`; for decorative, `Muted` is correct.
- `Warning` covers three semantic cases: dry-run mode, destructive operations, and
  drift detection. Keep that mapping consistent across all callsites.

---

## 2. How to Consume Tokens in Code

Never reference a hex literal outside `internal/shared/render/theme/`. Reference
tokens by name via `theme.Resolve()`.

```go
import (
    "os"
    "internal/shared/render/theme"
    "internal/shared/render/pretty"
)

// In composeApp or a constructor — build the theme once.
t, err := theme.Default(os.Stdout, flagTheme, os.Getenv("BUILDER_THEME"))
if err != nil {
    return err
}

// Pass the theme to the renderer.
r := pretty.NewRenderer(os.Stdout, t)

// Render output using semantic style names.
r.styles.Primary.Render("text")
r.styles.Success.Render("done")
r.styles.Error.Render("failed")
```

Architecture: tokens are the **vocabulary** (`theme` package), `Styles` is the
**view** (a bag of `lipgloss.Style` values derived once at construction), and
`*PrettyRenderer` is the **consumer**. The resolver pre-computes a
`map[TokenName]lipgloss.TerminalColor` at theme construction time — per-render
lookup is O(1).

---

## 3. Theme Detection and Overrides

Appearance (Light / Dark) resolves via a three-level precedence chain:

```
--theme flag  >  BUILDER_THEME env  >  auto-detect
```

| Precedence | Source             | Values            |
|------------|--------------------|-------------------|
| Highest    | `--theme` flag     | `light`, `dark`, `auto` |
| Middle     | `BUILDER_THEME` env | `light`, `dark`  |
| Lowest     | Auto-detect        | terminal heuristic |

Example invocations:

```sh
# Auto-detect from terminal (default)
builder info

# Force dark theme regardless of terminal
builder --theme=dark info

# Force light via environment variable
BUILDER_THEME=light builder info

# Flag wins over env (appearance = Light)
BUILDER_THEME=dark builder --theme=light info
```

The `--theme` flag only accepts `light`, `dark`, or `auto`. Any other value causes
an immediate non-zero exit:

```
$ builder --theme=neon info
Error: invalid argument "neon" for "--theme" flag
```

Auto-detect reads terminal capability signals (`COLORTERM`, `TERM`, TTY state)
via `colorprofile.Detect`. When stdout is piped (not a TTY), auto-detect returns
`NoColor` — the same path as `--theme=auto` on a dumb terminal.

---

## 4. Profile Degradation

The CLI resolves four color profile tiers at startup:

| Tier        | Condition                              | Rendering behaviour              |
|-------------|----------------------------------------|----------------------------------|
| `TrueColor` | TTY + `COLORTERM=truecolor`            | Full 24-bit RGB (`#RRGGBB`)      |
| `ANSI256`   | TTY + `TERM=xterm-256color`            | Nearest 256-color index          |
| `ANSI16`    | TTY, limited terminal                  | Nearest 16-color (bright mapped) |
| `NoColor`   | Not a TTY, or no color support         | Zero SGR — plain text            |

When stdout is piped (`builder ... | cat`, CI environments without a TTY), the
profile is always `NoColor` — no SGR escape sequences are emitted. This makes
`builder` safe to pipe without stripping tools.

Token-to-color resolution per tier:
- `TrueColor`: hex used directly via `lipgloss.Color("#RRGGBB")`.
- `ANSI256` / `ANSI16`: lipgloss quantizes the hex to the nearest palette entry at
  render time via the global `termenv` profile.
- `NoColor`: `lipgloss.NoColor{}` — every token renders as empty style (plain text).

---

## 5. The `no-hex-leak` Fitness Rule

Hex literals are confined to `internal/shared/render/theme/`. Everything else
references tokens by name. This is enforced by FF-24:

```sh
just fitness-hex-leak
```

What it checks: `rg -n '"#[0-9a-fA-F]{6,8}"' --type go -g '!internal/shared/render/theme/**'`
must return zero matches.

This rule is part of the `just fitness` aggregate — it runs automatically in CI
and fails the build if any hex literal leaks outside the theme package.

Violation example (what CI will catch):

```
FF-24 no-hex-leak: raw hex color literal(s) outside internal/shared/render/theme/
internal/feature/foo/handler.go:42:    color := "#FF0000"
```

Fix: replace the literal with the appropriate token reference via `theme.Resolve()`.

Spec references: theme-tokens/REQ-03.1, render-pretty/REQ-05.1, REQ-05.2.

---

## 6. Updating Golden Files

The golden matrix test (`Test_Render_Golden_Matrix` in
`internal/shared/render/pretty/golden_test.go`) pins the exact ANSI byte output
for 3 styles (`Primary`, `Success`, `Error`) across 4 profiles
(`TrueColor`, `ANSI256`, `ANSI16`, `NoColor`) = 12 golden files.

When you change a token's hex value in `palette.go`, the golden files become stale
and the test will fail. Regenerate them:

```sh
go test -update ./internal/shared/render/pretty/...
```

After regenerating, inspect each `.golden` file manually before committing:

```sh
# View ANSI escape sequences as hex for verification
cat -v internal/shared/render/pretty/testdata/golden/styles/truecolor/primary.golden
```

Goldens live under `internal/shared/render/pretty/testdata/golden/styles/`. Each
file is the byte-for-byte regression net for its (style, profile) cell. A stale
golden caught by CI means a visual regression was introduced — treat it as a test
failure, not a nuisance.

Do not commit golden files without manually confirming that the visual change is
intentional.

---

## 7. Out of Scope (for Now)

The following are explicitly NOT implemented in this change and are tracked as
future work:

- **Fang chrome theming** (help text, version output, error banners): blocked on
  upstream fang support for custom renderers. Help/version/error output is
  currently unstyled regardless of theme.
- **Custom user palettes**: the palette is fixed at the 8 canonical tokens.
  User-defined overrides via config file or environment are not supported.
- **WCAG accessibility audit**: no contrast-ratio verification beyond the
  canonical light/dark palette variants. Accessibility audit is future work.
- **Dark-mode auto-detection via terminal background query**: the current
  auto-detect path uses `COLORTERM`/`TERM` env heuristics. Terminal background
  color querying (e.g., OSC 10/11) is not implemented.

---

## 8. Quick Reference for Slice Authors

Working on a new feature and need to emit styled output? Here is the checklist:

1. **Do not add hex literals.** If you need a color, it already has a token name.
   Use `theme.Resolve(<TokXxx>)` or reference `r.styles.<Field>` via the renderer.
2. **The wiring point is `composeApp` in `cmd/builder/main.go`.** Do not
   construct themes anywhere else (ADR-011: composition root discipline).
3. **Adding a new Styles field?** It must correspond to an existing token.
   Reflect-based test `Test_Styles_HasEightSemanticFields_ByReflection` will fail
   if you add a ninth field without a spec change.
4. **Changing a hex value?** Regenerate goldens (`go test -update`), inspect,
   commit both together.

Relevant ADRs (in `design/` or the Architectural Decisions discussion):
- ADR-01: theme as a sibling package, not nested under `pretty/`
- ADR-02: vendor `theme.Profile` enum (insulates from `colorprofile` v0.x churn)
- ADR-03: golden test strategy (4-profile x 3-style matrix)
- ADR-04: hex-leak fitness rule via Justfile + rg
