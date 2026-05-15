# project-builder-cli

[![CI](https://github.com/Project-Builder-Schematics/project-builder-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Project-Builder-Schematics/project-builder-cli/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](go.mod)

> **AI-first schematics runner**. Bootstrap, author, and orchestrate project scaffolds from composable AI skills — driven from your terminal or directly by an AI agent.

This is the Go rewrite of the legacy TypeScript [`@pbuilder/cli`](https://www.npmjs.com/package/@pbuilder/cli) (v1.9.4). The Go implementation will be distributed as platform-specific binaries via npm wrappers (esbuild / turbo / Biome pattern), with Angular schematics as the first execution backend.

---

## Why this exists

Scaffolding tooling today is a fragmented landscape: one CLI per framework, no unified mental model, no first-class AI integration, and configuration scattered across config files, package scripts, and ad-hoc README sections. `project-builder-cli` is a single binary that:

1. **Treats schematics as first-class artefacts** — every transformation is a reproducible, versioned schematic that can be authored locally, shared as an npm package, or extended by AI agents.
2. **Is AI-first by design** — every command has a `--json` mode for machine consumption, a stable error contract with actionable suggestions, and ships an `SKILL.md` artefact so AI agents know how to use the tool without prompt engineering.
3. **Decouples the runner from the engine** — the CLI orchestrates; concrete engines (Angular schematics today, native Go engine planned) plug in via a stable `Engine` port. Swap backends without rewriting the CLI.
4. **Preserves your control** — `--dry-run` previews every operation as structured JSON before any write. Filesystem mutations are atomic (write-to-temp + rename). Agent-readable files (`AGENTS.md`, `CLAUDE.md`) are touched only via durable, idempotent, line-exact markers.

If you want the architectural background, read the canonical specs in [Discussions](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions):

- [#1 RFC — original motivation](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/1)
- [#2 Mental Model & 4 Atomic Responsibilities](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/2)
- [#3 Capabilities & Commands Inventory](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/3) — **canonical command reference**
- [#4 Execute Flow](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/4)
- [#5 Phasing v1/v2 (Angular Subprocess → Native Engine)](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/5)

---

## What's in this repo

This repository is the **CLI binary** — the user-facing entry point. Three sibling components round out the Project Builder ecosystem:

| Component | Repo | Role |
|---|---|---|
| **CLI** (this repo) | `project-builder-cli` | The `builder` binary. Parses commands, validates input, orchestrates engines. |
| **SDK** | `@pbuilder/sdk` *(npm, pending)* | Developer library for authoring schematics. Ships the canonical `project-builder.schema.json`. |
| **Engine** | `@pbuilder/engine` *(pending)* | Native Go engine that runs schematics directly. v1 uses Angular schematics via subprocess; v2 replaces it. |
| **MCP server** | *(pending — see roadmap row 17)* | Optional MCP server exposing CLI primitives to AI clients (Claude Desktop, Claude Code, Cursor, etc.). |

For the SDD (spec-driven development) pipeline used to plan and build every change in this repo, see [SDD orchestrator docs](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/2#discussioncomment-) and the [ROADMAP](./ROADMAP.md).

---

## Status

Pre-`v1.0` — repository is being bootstrapped slice by slice via SDD. No installable releases yet.

| Phase | Status |
|---|---|
| Phase 1 — Repo bootstrap & tooling | ✅ Done (PR #9) |
| Phase 2 — Architectural skeleton | ✅ Done (PR #12) |
| Phase 3 — Renderer adapters (pretty + JSON) | ✅ Done (PR #13) |
| Phase 3 — AngularSubprocessAdapter | ✅ Done (PR #14) |
| Phase 4 — `builder init` end-to-end | ✅ Done (PR #17) |
| Phase 4.5 — CLI versioning automation (auto-bump on merge) | ✅ Done (PR #21) |
| **Phase 5 — `builder new` (schematic + collection scaffolding)** | **✅ Done (PR #22, v0.1.0)** |
| Phase 6 — `builder execute` end-to-end | 📋 Next |
| Phases 7..17 — remaining commands + npm distribution | 📋 Backlog |

See [ROADMAP.md](./ROADMAP.md) for the full breakdown.

---

## Install

Pending `v1.0`. Once cut, the CLI will install via:

```sh
npm i -g @pbuilder/cli
```

To build from source today (requires Go 1.23+):

```sh
git clone https://github.com/Project-Builder-Schematics/project-builder-cli.git
cd project-builder-cli
just build           # or: go build -o builder ./cmd/builder/
./builder --help
```

For contributors: [`CONTRIBUTING.md`](./CONTRIBUTING.md).

---

## Command inventory

The canonical inventory lives in [Discussion #3](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/3). Status reflects what currently works in the binary.

| Command | Status | Purpose |
|---|---|---|
| **`builder init`** | ✅ **Available** | Initialise a Project Builder workspace in the current repo |
| `builder execute` (`e`, `generate`, `g`) | 📋 Pending | Run a schematic against the current project |
| **`builder new schematic`** (alias `s`) | ✅ **Available** | Scaffold a new local schematic (path mode + `--inline`); auto-generates `.d.ts` types |
| **`builder new collection`** (alias `c`) | ✅ **Available** | Scaffold a new local collection (with optional `--publishable` lifecycle stubs) |
| `builder generate-types` | ⚠️ Auto-only | TS type emission runs inline inside `builder new schematic`; standalone command pending |
| `builder add` | 📋 Pending | Register an externally published collection |
| `builder create` | 📋 Pending | Scaffold a new project from scratch (templates) |
| `builder migrate` | 📋 Pending | Transform a project between modes/versions/adapters |
| `builder info` | 📋 Pending | Inspect a collection or specific schematic |
| `builder sync` | 📋 Pending | Fetch declared remote collections into the global cache |
| `builder validate` | 📋 Pending | Lint mode for schematics |
| `builder remove` | 📋 Pending | Remove a local schematic or unregister a collection |
| `builder skill update` | 📋 Pending | Regenerate the AI skill artefact when the CLI version changes |

### Transversal flags (planned, not all wired yet)

These flags will work consistently across commands as they ship:

| Flag | Effect |
|---|---|
| `--dry-run` | Show what would happen without applying. Free byproduct of the staging architecture. |
| `--non-interactive` | Fail clean on missing required inputs; implies `--auto-install` (suitable for CI and AI agents). |
| `--auto-install` | Skip the bunx-style "install this collection?" prompt for the current invocation. |
| `--strict` | Undeclared deps fail before execution (vs default dev-mode warning + auto-install). |
| `--force` | Override the `dry-run-incompatible` flag on schematics (preview may be inaccurate). |
| `--conflict-policy=child-wins\|strict` | Per-invocation override of the project's conflict policy. |
| `--source=local\|cached` | Force the Discoverer to use a specific source (default: LOCAL → CACHED). |
| `--json` | Machine-readable JSON output for AI and CI consumption. |

Boolean conventions: `--flag` = `true`, `--no-flag` = `false`, `--flag=value` = the explicit value.

---

## `builder init` — full reference

`builder init` bootstraps a Project Builder workspace in an existing repository. CLI-only (does not call any engine) — the first command in the inventory with that property.

### Synopsis

```
builder init [directory] [flags]
```

- `directory` is optional. When omitted, init operates on the current working directory. The chosen directory is taken literally — init does **not** climb the filesystem looking for `.git` or `package.json`.

### What it creates

A successful `init` produces **five outputs**:

1. **`project-builder.json`** at project root — the workspace anchor file (schema v1, `$schema` pointed at the locally-installed SDK package).
2. **`schematics/.gitkeep`** — skeleton folder for local schematic authoring (later filled in by `builder new schematic`).
3. **`.claude/skills/pbuilder/SKILL.md`** — bundled AI skill artefact (Anthropic Agent Skills format).
4. **Fenced reference block in `AGENTS.md`** (preferred) or `CLAUDE.md` — idempotent, line-exact, durable post-v1.0.0.
5. **`@pbuilder/sdk` added to `devDependencies`** in `package.json` — additive merge; existing deps preserved.

After the writes, init optionally invokes the user's detected package manager to install the SDK (with a 120-second timeout). Then, in TTY mode, init prompts for MCP server setup; an affirmative reply prints setup instructions (the actual install is a [future change](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/3)).

### Flags

| Flag | Effect |
|---|---|
| `--force` | Overwrite existing `project-builder.json` (and existing SKILL.md / ambiguous AGENTS-and-CLAUDE marker scenarios). |
| `--dry-run` | Preview every planned operation as structured output. Writes nothing. |
| `--json` | Emit machine-readable JSON output. Combines with `--dry-run` for a full structured plan. |
| `--non-interactive` | Disable all prompts. With `--mcp` unset, defaults to `--mcp=no`. Suitable for CI and AI agents. |
| `--package-manager=<npm\|pnpm\|yarn\|bun>` | Override package-manager detection. Default: lockfile sniff (pnpm > yarn > bun > npm) → `npm` fallback. |
| `--no-install` | Skip the package-manager install step. The SDK is still declared in `package.json` — run install manually later. |
| `--no-skill` | Atomically skip the SKILL.md output, the AGENTS/CLAUDE reference block, and the SDK dev-dep. Use when you want only `project-builder.json` + `schematics/`. |
| `--mcp=<yes\|no\|prompt>` | Control MCP setup prompt. Default: `prompt` in TTY, `no` in `--non-interactive`. `--mcp=prompt` is incompatible with `--non-interactive`. |
| `--publishable` | Reserved — currently returns `ErrCodeInitNotImplemented`. Planned for `builder-init-publishable`. |

### Examples

```sh
# Standard init — five outputs + npm install + prompt for MCP setup
builder init

# Init a sibling directory
builder init ./my-new-workspace

# Preview the full plan as JSON (no writes, no subprocess)
builder init --dry-run --json /tmp/preview

# CI / AI agent flow — non-interactive, JSON output, explicit PM, no MCP
builder init --non-interactive --json --package-manager=pnpm --mcp=no .

# Skip install (you'll run it manually later)
builder init --no-install

# Minimal init — only project-builder.json + schematics/ (no SKILL, no SDK)
builder init --no-skill

# Force re-init over an existing workspace
builder init --force
```

### `project-builder.json` shape (schema v1)

```json
{
  "$schema": "./node_modules/@pbuilder/sdk/schemas/project-builder.schema.json",
  "version": "1",
  "collections": {},
  "dependencies": {},
  "settings": {
    "autoInstall": true,
    "conflictPolicy": "child-wins",
    "depValidation": "dev"
  },
  "skill": {
    "enabled": true,
    "path": ".claude/skills/pbuilder/SKILL.md"
  }
}
```

| Section | Purpose |
|---|---|
| `$schema` | Relative path to the SDK-shipped JSON Schema. IDE autocomplete + validation once `npm install` completes. |
| `version` | Schema version. v1 is locked. Future v2 readers will check this field. |
| `collections` | Local collections declared with relative paths. Read by the Discoverer (`LOCAL` source). |
| `dependencies` | Remote collections this project uses. Read by `builder sync` to populate the global cache. |
| `settings.autoInstall` | If `true`, skip the bunx-style prompt globally for this project. |
| `settings.conflictPolicy` | `child-wins` or `strict`. Engine respects this for inter-schematic merges. |
| `settings.depValidation` | `dev` or `strict`. Applied to undeclared deps at execute time. |
| `skill` | Where the AI skill lives. `enabled: false` opts out for users without AI tooling. |

Forward-compat: readers ignore unknown top-level keys (Discussion #3 invariant).

### Errors you might hit

Every error includes a structured `code`, a human-readable `message`, and a non-empty `suggestions` array — making them actionable for AI agents and humans alike.

| Code | When | Remediation |
|---|---|---|
| `init_dir_not_empty` | Target directory has files and `--force` not passed | re-run with `--force`, use `--dry-run` first, or choose an empty directory |
| `init_config_exists` | `project-builder.json` already exists and `--force` not passed | re-run with `--force`, delete the file, or use `--dry-run` to preview |
| `init_agent_file_ambiguous` | Both `AGENTS.md` and `CLAUDE.md` already contain the pbuilder marker | re-run with `--force` (writes to AGENTS.md), or remove one marker manually |
| `init_package_manager_not_found` | Resolved PM binary not on PATH | use `--no-install`, install the PM (`npm`/`pnpm`/etc.), or set `--package-manager=<other>` |
| `init_skill_exists` | SKILL.md already exists; emitted as a *warning* (exit 0, not fatal) | re-run with `--force` to overwrite, or use `builder skill update` (planned) |
| `init_not_implemented` | `--publishable` flag used before its implementation lands | re-run without `--publishable` |
| `invalid_input` | Path traversal (`..` escapes cwd), malformed `package.json`, symlink-out-of-project, or invalid `--mcp` value | message names the offending input + suggests valid alternatives |

---

## `builder new` — full reference

`builder new` is a parent command with two leaves: `builder new schematic <name>` and `builder new collection <name>`. Both operate on the workspace anchored by `project-builder.json` (created by `builder init`). CLI-only — does not call any engine.

### Synopsis

```
builder new schematic <name> [flags]
builder new s <name> [flags]                    # alias

builder new collection <name> [flags]
builder new c <name> [flags]                    # alias
```

`<name>` is mandatory and validated against shell metacharacters, path separators, null bytes, and reserved characters via the shared `validate.RejectMetachars` helper.

### `builder new schematic <name>` — what it creates

Two modes, controlled by the `--inline` flag.

**Path mode (default)** — produces 4 outputs:

1. `schematics/<name>/factory.{ts,js}` — factory stub (TS or JS depending on language detection)
2. `schematics/<name>/schema.json` — canonical v1 shape `{"inputs": {}}` (two-space indent, trailing newline, no BOM, no HTML escaping)
3. `schematics/<name>/schema.d.ts` — auto-generated TypeScript interface from `schema.json` inputs (always emitted regardless of `--language`)
4. `project-builder.json` — adds `collections.default.<name>: { "path": "./schematics/<name>" }`

**Inline mode (`--inline`)** — embeds the schematic directly inside `project-builder.json`:

```json
"collections": {
  "default": {
    "schematics": {
      "<name>": { "inputs": {} }
    }
  }
}
```

No `schematics/<name>/` files are created. Soft warnings fire when a collection accumulates ≥10 inline schematics or when `project-builder.json` exceeds 20KB after the write.

### `builder new collection <name>` — what it creates

**Default mode** — produces 2 outputs:

1. `schematics/<name>/collection.json` — skeleton `{"version": 1, "schematics": {}}`
2. `project-builder.json` — adds `collections.<name>: { "path": "./schematics/<name>/collection.json" }`

**Publishable mode (`--publishable`)** — produces 7 outputs (collection skeleton + lifecycle stubs):

1. `schematics/<name>/collection.json`
2. `schematics/<name>/add/factory.ts` + `schema.json` + `schema.d.ts`
3. `schematics/<name>/remove/factory.ts` + `schema.json` + `schema.d.ts`
4. `project-builder.json` entry includes `add` + `remove` paths

`--publishable` cannot be combined with `--inline` — they are mutually exclusive (mode conflict).

### Flags

| Flag | Applies to | Effect |
|---|---|---|
| `--force` | both | Overwrite existing schematic / collection of the same name |
| `--dry-run` | both | Preview planned operations as `PlannedOp[]`. Writes nothing. Output as JSON via `--output=json`. |
| `--inline` | `schematic` | Embed schema in `project-builder.json` instead of creating files (mutually exclusive with `--publishable` for collection) |
| `--language=<ts\|js>` | `schematic` | Force TypeScript or JavaScript factory. Auto-detect default: TS if `devDependencies.typescript` OR `tsconfig.json` exists; falls back to TS with a warning otherwise. |
| `--extends=<@scope/pkg:base>` | `schematic` | Declare a base schematic this one extends. Grammar enforced: `^@[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+$`. Path traversal rejected. |
| `--publishable` | `collection` | Generate add/remove lifecycle stubs (turns the collection into a publishable npm package skeleton) |
| `--output=<pretty\|json>` | both | Output format (inherited persistent flag from root) |

### Examples

```sh
# Standard schematic — 3 files + project-builder.json entry, TS auto-detected
builder new schematic my-component

# Schematic with explicit JavaScript factory
builder new s my-helper --language=js

# Inline schematic — no files, embedded in project-builder.json
builder new schematic config-only --inline

# Schematic that extends an external base (no network call at create-time)
builder new schematic feature-flags --extends=@my-org/core:base

# Preview as JSON without writing anything
builder new schematic preview-test --dry-run --output=json

# Force overwrite an existing schematic
builder new schematic my-component --force

# Plain collection — collection.json + project-builder.json entry only
builder new collection ui-kit

# Publishable collection — adds add/remove lifecycle stubs
builder new collection my-pkg --publishable

# Collection alias
builder new c shared-utils --publishable
```

### `schema.json` v1 format

The generated `schema.json` is a closed shape — only the fields below are accepted on read:

```json
{
  "version": 1,
  "inputs": {
    "<name>": {
      "type": "string|number|boolean|enum|list",
      "description": "...",
      "position": 0,
      "default": <type-compatible value>,
      "enum": ["a", "b"],
      "items": { "type": "string" }
    }
  }
}
```

**FORBIDDEN top-level keys** (rejected on read with explicit error):

- `properties` — Angular JSON Schema draft-07 sentinel
- `$schema` referencing `http://json-schema.org/draft-07/schema` — same legacy contract

This is the breaking departure from Angular schematics. The `$schema` written by `builder init` points to the SDK-bundled meta-schema, not draft-07.

`{"inputs": {}}` is the canonical empty form (two-space indent, trailing newline, no BOM). Fitness function `FF-15 schema-json-canonical.sh` enforces this for generated output.

### TypeScript codegen (`.d.ts`)

For each path-mode schematic, `schema.d.ts` is generated alongside `schema.json` using a hand-rolled `strings.Builder` (NOT `text/template` — silent-injection class avoided per ADR-032). Property names are escaped via the `tsident` package which:

- Replaces hyphens, dots, spaces with `_` (collapsed: `my--name` → `my_name`)
- Prefixes leading-digit names with `_` (`123x` → `_123x`)
- Suffixes TypeScript reserved words with `_` (`class` → `class_`, `interface` → `interface_`)
- Replaces non-ASCII bytes with `_`

The 69 ECMAScript + TypeScript reserved words are enumerated in `internal/shared/tsident/reserved.go` and every entry is asserted by fitness function `FF-14 tsident-reserved-coverage.sh`.

Header comment format (deterministic, no timestamp — clean `git diff`):

```ts
// Auto-generated by builder new — do not edit manually
// @builder-cli v0.1.0

export interface MySchematicSchematicInputs {
  class_: string; // original: class
  count?: number;
}
```

### Errors you might hit

Every error includes a structured `code`, message, and `suggestions` array. Exit code: `2` for `*errs.Error` (structured user-correctable), `1` for unexpected.

| Code | When | Remediation |
|---|---|---|
| `new_schematic_exists` | Schematic with this name exists in the default collection (path or inline) | re-run with `--force`, or pick a different name |
| `new_collection_exists` | Collection with this name already exists | re-run with `--force`, or pick a different name |
| `new_invalid_name` | Name contains shell metacharacters, path separators, null bytes, or is empty | rename — kebab-case only, no separators, no metacharacters |
| `new_invalid_extends` | `--extends` value fails grammar `@scope/pkg:collection` (path traversal, missing `@`/`:`, whitespace) | fix the format per the grammar |
| `new_mode_conflict` | `--publishable --inline` together; or `--inline --force` when path-mode entry exists | pick one mode; for migration use `builder remove` first |
| `new_invalid_language` | `--language=<value>` not in `{ts, js}` | use `--language=ts` or `--language=js` (or omit for auto-detect) |

### Project-builder.json mutation invariants

- **Atomic writes**: write-temp + rename. Partial state impossible (REQ-PJ-01). Concurrent runs rely on OS-level rename atomicity — at least one wins.
- **Idempotent**: `--force` re-runs produce byte-identical output (REQ-PJ-02).
- **Forward-compat**: unknown top-level fields preserved verbatim on read-mutate-write (REQ-PJ-04).
- **`version` field preserved verbatim** — string `"1"` stays string; integer `1` stays integer (REQ-PJ-03).
- **No HTML escaping** — uses `json.NewEncoder` + `SetEscapeHTML(false)` per L-builder-init-03 (REQ-PJ-07).
- **UTF-8 BOM detected and stripped on read** with a Renderer warning (ADV-06).
- **Symlink rejection** — schematic dir resolving outside workspace via `EvalSymlinks` is rejected before any write (ADV-08).

### Adversarial coverage

13 adversarial scenarios are tested end-to-end:

- TS reserved word as schematic name (`class` → `class_` in `.d.ts`, original preserved in `schema.json`)
- Path traversal in name and `--extends` (rejected via `validate.RejectMetachars` and grammar regex)
- Shell metacharacter / null byte in name
- UTF-8 BOM in existing `schema.json` (stripped + warned)
- Concurrent runs (race detector clean; OS rename atomicity guarantees single winner)
- Symlink pointing outside workspace
- Read-only filesystem (no partial state; atomic write fails before rename)
- `--inline --force` when path-mode entry exists (mode conflict error)
- All 69 TypeScript reserved words tested by data-driven matrix
- 255-char max-length name (no panic, no OOM)

---

## Architectural highlights

- **Hexagonal**: handler → service → port. The init feature is the first to *not* compose the `Engine` port — it's CLI-only.
- **FSWriter port** (ADR-020): all filesystem mutations route through a 6-method interface with `osFS` (production, atomic temp+rename), `dryRunFS` (records `PlannedOp[]`), and `fakeFS` (in-memory tests). No direct `os.WriteFile` in feature code (enforced by `scripts/fitness/no-direct-os-io.sh`).
- **Locked durable contracts** post-v1.0.0:
  - `project-builder.json` v1 byte shape (golden test)
  - SKILL.md placeholder bytes (`//go:embed` + SHA-256 fixture)
  - AGENTS/CLAUDE marker begin/end literals (`<!-- pbuilder:skill:begin -->` / `<!-- pbuilder:skill:end -->`)
  - `--json --dry-run` envelope schema (5 stable `op` values: `create_file`, `append_marker`, `modify_devdep`, `install_package`, `mcp_setup_offered`)
- **Strict TDD**: every REQ has a passing test. 18+ CI-enforced fitness functions (handler LOC ceiling, marker uniqueness, error-code additivity, embed byte-stability, no cross-feature imports, tsident reserved-word coverage, schema.json canonical bytes, projectconfig F-01 promotion marker, release workflow guards, etc.).
- **Compliance with the [Project Builder Mental Model](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions/2)**: four atomic responsibilities (config, AI skill, schematic authoring scaffold, dependency declaration) — each materialised as exactly one of the five outputs.

For the full ADR list (now ADR-001..ADR-034), browse the [`design`](./design) directory or the [Architectural Decisions discussion](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions). The most recent additions:

- **ADR-031** — `schema.json` v1 closed shape (rejects Angular draft-07 `properties` field on read)
- **ADR-032** — TS codegen via `strings.Builder` + `tsident` package (silent-injection class avoided vs `text/template`)
- **ADR-033** — inline-vs-path service method split with explicit mode-conflict rejection
- **ADR-034** — `projectconfig` in-line + F-01 followup commitment to promote to `internal/shared/projectconfig/` before `builder add` lands

---

## Development

```sh
just build       # compile the binary to ./builder
just test        # run all tests with -race
just fitness     # run all 13 CI fitness functions
just fmt         # gofumpt + goimports (idempotent)
just lint        # golangci-lint (15-tool curated bundle)
```

Hooks (`lefthook`) run formatters on commit and tests on push. See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for the full workflow including the SDD pipeline used for every substantive change.

---

## Roadmap

The full breakdown lives at [ROADMAP.md](./ROADMAP.md). Highlights of what's next:

1. **`builder execute`** (Phase 6, L) — the central command. Full 6-step pipeline against the AngularSubprocessAdapter.
2. **`f-01-projectconfig-promotion`** (M, **HIGH**) — promote `internal/feature/new/projectconfig.go` to `internal/shared/projectconfig/` before `builder add` lands. Tracked by FF-17.
3. **`builder add`** (M-L) — register external collections (depends on F-01).
4. **Templates system foundation** (L) — prerequisite for `builder create` + `builder migrate`.
5. **Followups from `builder new`**: `s002-tdd-cleanup` (retroactive RED→GREEN split), `schema-forbidden-field-wire-up` (wire `ReadSchemaFromBytes` into `--force` overwrite path), `flaky-race-investigation`.
6. **Followups from `builder init`**: coverage glue (raise to ≥70%), TTY-suppression flag for scripted contexts, `--publishable` mode, actual MCP server install.
7. **npm multi-platform distribution** (L) — JS wrapper + platform packages — gates v1.0 release.

---

## Documentation

- [Contributing guide](./CONTRIBUTING.md)
- [Roadmap](./ROADMAP.md)
- [Architectural decisions, mental model, and command inventory](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions)
- [License (Apache 2.0)](./LICENSE)

---

## License

Apache License 2.0 — see [LICENSE](./LICENSE).
