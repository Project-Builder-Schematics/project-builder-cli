# Output Port — Semantic API for Command Authors

`output.Output` is the unified port through which all builder commands emit user-facing bytes.
Instead of calling `fmt.Println` or writing to `os.Stdout` directly, every feature handler
receives an `Output` dependency and calls semantic methods on it. This keeps handlers
decoupled from rendering concerns, makes them trivially testable via `outputtest.Spy`,
and ensures the `--theme` flag drives every byte written to the terminal.

---

## API Surface

| Method | Signature | When to use |
|---|---|---|
| `Heading` | `Heading(text string)` | Section titles and command result headers |
| `Body` | `Body(text string)` | Descriptive lines and operation summaries |
| `Hint` | `Hint(text string)` | Soft suggestions, secondary notes |
| `Success` | `Success(text string)` | Positive confirmation lines |
| `Warning` | `Warning(text string)` | Non-fatal advisory messages |
| `Error` | `Error(text string)` | User-correctable error announcements |
| `Path` | `Path(p string)` | Filesystem paths, URLs, identifiers |
| `Prompt` | `Prompt(text string) (string, error)` | Synchronous single-line user input |
| `Newline` | `Newline()` | Visual spacing between sections |
| `Stream` | `Stream(ctx context.Context, ch <-chan events.Event) error` | Engine event rendering |

---

## Semantic Role Table

| Method | Token | Visual role | Example call |
|---|---|---|---|
| `Heading` | `Primary` | Bold accent colour — signals a new logical section | `out.Heading("Initialising workspace")` |
| `Body` | `Foreground` | Default terminal foreground — plain prose | `out.Body("Created schematics/.gitkeep")` |
| `Hint` | `Muted` | Dimmed — supplementary, low-priority | `out.Hint("Re-run with --force to overwrite")` |
| `Success` | `Success` | Green — operation completed successfully | `out.Success("Project Builder is ready.")` |
| `Warning` | `Warning` | Yellow/amber — something to note, not fatal | `out.Warning("10 inline schematics — consider splitting")` |
| `Error` | `Error` | Red — user-correctable problem | `out.Error("project-builder.json already exists")` |
| `Path` | `Accent` | Distinct accent — stands out from prose | `out.Path("schematics/my-component/factory.ts")` |
| `Prompt` | `Primary` (prefix `? `) | Interactive question — blocks for input | `name, err := out.Prompt("MCP server name?")` |
| `Newline` | — | Blank line — breathing room between sections | `out.Newline()` |
| `Stream` | — | Engine events — delegates to `pretty.Renderer` | `out.Stream(ctx, eventsCh)` |

Token mappings are resolved by `theme.Theme` at construction time; exact hex values live
in `internal/shared/render/theme/` and are enforced by FF-24 (`no-hex-leak`).

---

## Prompt vs Stream Rule (ADR-05)

These two methods handle user interaction but they are **independent and serve different flows**:

**`Prompt(text string) (string, error)`** — synchronous, blocking.
Used for simple CLI questions where you need one line of input before proceeding.
The adapter writes a styled `? <text>` prefix and reads a single line from the configured
reader (default: `os.Stdin`). Control returns only after the user presses Enter.

```go
// init/mcp.go — one-liner for a CLI question
serverName, err := out.Prompt("MCP server name?")
```

**`Stream(ctx context.Context, ch <-chan events.Event) error`** — asynchronous, channel-driven.
Used for the engine execution path where the `Engine` emits `events.Event` values
(including `events.InputRequested` for engine-driven prompts). Delegates byte-for-byte
to `pretty.Renderer.Render`. Returns nil on clean channel close.

```go
// execute handler — engine events flow through Stream
if err := out.Stream(ctx, resultCh); err != nil {
    return err
}
```

The two are independent. A command can call `Prompt` during setup and then `Stream`
during execution without conflict.

---

## Adding a New Command

Inject `output.Output` into the command constructor. The binary wires the production
`themed.Adapter`; tests inject `outputtest.Spy`.

```go
// internal/feature/mycommand/command.go
func NewCommand(svc *Service, out output.Output) *cobra.Command {
    run := func(cmd *cobra.Command, args []string) error {
        result, err := svc.DoWork(cmd.Context(), args)
        if err != nil {
            out.Error(err.Error())
            return err
        }
        out.Heading("Done")
        out.Path(result.OutputPath)
        out.Success("All files written.")
        return nil
    }
    // ... cobra setup
}
```

In `cmd/builder/main.go::composeApp`, add one line alongside the existing wires:

```go
myCmd := mycommand.NewCommand(mySvc, out)
```

The `out` variable is already constructed and profile-wired by `composeApp` — no
additional setup needed.

---

## Constructor Injection: Struct Field OR Closure Capture (ADR-03)

ADR-03 specified that feature handlers should store `Output` as an unexported struct field
(`h.out output.Output`), with constructors receiving it once. In practice, the `init` and
`new` handlers instead use **closure capture** — the `Output` value is captured by the
`RunE` closure at construction time rather than stored on a struct:

```go
// internal/feature/init/handler.go (closure pattern — actually used)
func newRunE(svc *Service, out output.Output) func(*cobra.Command, []string) error {
    return func(cmd *cobra.Command, args []string) error {
        // out captured from constructor argument — same DI semantics
        out.Heading("Initialising workspace")
        ...
    }
}
```

Both patterns satisfy the **"Output injected once at construction"** intent of ADR-03.
The distinction is stylistic, not behavioural:

- **Prefer the struct field** (`h.out`) when the handler has multiple exported methods
  that all need `Output` — the field avoids repeating the argument across methods.
- **Closure capture is idiomatic** for Cobra `RunE`-style handlers (a single anonymous
  function per command), which is the pattern `init` and `new` use today.

FF-25 enforces the architectural boundary (`no-direct-stdout-in-features`) regardless of
which construction style is used — both patterns route output through the `Output` port.

---

## Golden Update Workflow

The `themed.Adapter` is covered by a golden matrix in
`internal/shared/render/output/themed/themed_test.go`. Goldens pin the exact ANSI bytes
produced for each semantic method under specific profiles, catching token regressions
(wrong colour, missing reset sequence, etc.) at the unit level.

When a golden legitimately drifts — for example after updating a palette hex value in
`internal/shared/render/theme/` — regenerate with the `-update` flag:

```sh
go test ./internal/shared/render/output/themed/... -run Test_Themed -update
```

This overwrites every `.golden` file under
`internal/shared/render/output/themed/testdata/golden/output/`. Review the diff
(expected: only SGR colour codes change, visible text stays the same), then commit.

Do **not** run `-update` to silence a test failure you do not understand — always
investigate the root cause first.
