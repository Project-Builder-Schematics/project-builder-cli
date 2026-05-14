# Project Builder CLI — Roadmap

> **Status**: Phase 1 (repo bootstrap & tooling) merged via PR #9. Phase 2 (architectural-skeleton, #7) merged via PR #12. Phase 2 follow-on (renderer-adapters, #10) merged via PR #13. Phase 3 (angular-subprocess-adapter, #11) merged via PR #14 — `AngularSubprocessAdapter` is now the first concrete `Engine` implementation: NDJSON over stdio, `//go:embed` runner.js, SIGTERM→SIGKILL build-tag isolation, default-deny env allowlist, full validation against shell metachars + path traversal. 38/38 REQs, 9/9 fitness functions, race-clean, council pass-with-followups. Phase 4 (`builder init` end-to-end) and Phase 5 (`builder execute` end-to-end) are now the natural next picks.
> **Last updated**: 2026-05-14 (post archive of builder-init-end-to-end; branch pushed; PR pending)
> **Canonical source**: This file mirrors the SDD pending-changes registry stored in engram under topic key `project/pending-changes`. When the two diverge, **engram is authoritative** for the SDD orchestrator; this file is the human-readable mirror.

The CLI is being rewritten from TypeScript (legacy `@pbuilder/cli` v1.9.4) to Go, distributed via npm with platform-specific binaries (esbuild / turbo / Biome pattern). This document tracks the v1.0 milestone breakdown.

For architectural background, read the canonical specs in [Discussions](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions): #2 (Mental Model), #3 (Capabilities), #4 (Execute Flow), #5 (Phasing). The original RFC is #1.

## Legend

- ✅ **DONE** — change is archived
- 🚧 **IN FLIGHT** — `/plan` running or `/build` in progress
- 📋 **PENDING** — not yet started; in `Backlog` of the [Project board](https://github.com/orgs/Project-Builder-Schematics/projects/1)

---

## Section A — Remaining `/plan` invocations

Each row is a future SDD `/plan` candidate. When picked up, the user invokes `/plan <change-name>` and the orchestrator runs triage → explore → propose → spec → design → slice. Order is recommended, not strict — dependencies noted.

| # | Change | Triage | Phase | Coverage | Status |
|---|---|---|---|---|---|
| 1 | v1.0 Repo Bootstrap & Tooling | L | foundations | go.mod, layout placeholders, Justfile, golangci-lint v2, lefthook, CI (lint+test+build+smoke), README, CONTRIBUTING, .gitignore, deferred release.yml stub | ✅ DONE — spec [#6](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/6) closed via PR [#9](https://github.com/Project-Builder-Schematics/project-builder-cli/pull/9); followup [#8](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/8) tracks multi-platform release |
| 2 | Architectural Skeleton | L | architecture | 7 ADRs (ADR-006..ADR-012) + Engine/Renderer ports + 12-event sealed catalogue (with `Sensitive` flags) + structured `Error` (with `SafeMessage`/`MarshalJSON`/`Error()` lock) + `composeApp()` ≤120 LOC + 8 Cobra stubs (handlers ≤100 LOC) + 9 fitness functions | ✅ **DONE** — spec [#7](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/7) closed via PR [#12](https://github.com/Project-Builder-Schematics/project-builder-cli/pull/12); verify-final PASS (35/35 REQs, 9/9 FFs); followups C8–C14 registered in engram `project/pending-changes` |
| 3 | Pretty + JSON Renderer adapters | M | architecture | 3 capabilities (`pretty-renderer`, `json-renderer`, `renderer-factory`) + composition-root delta — TUI deferred to Phase 1B; 14 REQs across 4 domains; 4 ADRs (1 amended at apply); 3 SPIDR slices | ✅ **DONE** — spec [#10](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/10) closed via PR [#13](https://github.com/Project-Builder-Schematics/project-builder-cli/pull/13); verify-final pass-with-followups (14/14 REQs, 30/30 scenarios, 9/9 FFs, 140 tests race-clean); ADRs 013–015 promoted; followups C15–C17 registered. ADR-03 amended at apply-time (inline per-adapter `mask()` to dodge import cycle). C10 deferred — JSON Renderer scope did not require Error.MarshalJSON verbose-mode; remains pending. |
| **4** | **AngularSubprocessAdapter** | **L** (sensitivity override: subprocess + supply-chain + privilege) | angular-adapter | First concrete `Engine` impl. NDJSON over stdio + embedded `runner.js` via `//go:embed` + SIGTERM/SIGKILL build-tag isolation + injectable Discoverer. 8 domains, 38 REQs, 4 ADRs, 9 existing fitness functions reused; 6 SPIDR slices (incl. integration slice with real Node) | 🚧 **READY TO BUILD** — spec [#11](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/11) (`/plan` complete, V1 signed, slices ready). UNBLOCKED post-#2 + #3 merges. C8 + C9 (AST-based FF-07/FF-08 hardening) recommended alongside this slice. |
| 5 | `builder init` end-to-end | L (sensitivity override) | angular-adapter | First real command: generates `project-builder.json` + SKILL.md + `schematics/.gitkeep` + AGENTS.md/CLAUDE.md marker + `@pbuilder/sdk` dev-dep + PM install. Augment-mode only. MCP-prompt (bounded: print instructions only). 43 REQs, 6 FFs, 4 ADRs (020–023). | ✅ **DONE** — spec [#15](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/15) closed via PR pending push; verify-final pass-with-followups (43/43 REQs, 6/6 FFs, 7/7 e2e, 3/3 mutation; coverage 65.3% non-blocking); ADRs 020–023 promoted; 5 followups registered (`builder-init-coverage-glue`, `builder-init-stdin-script-detection`, `builder-init-publishable`, `builder-init-mcp-install`, `sdk-publish-project-builder-schema` [CRITICAL — blocks v1.0.0]). |
| 6 | `builder execute` end-to-end | L | angular-adapter | Central command. Full 6-step pipeline. Requires #4 functional | 📋 PENDING — after #4 + #5 |
| 7 | `builder new` *(was `builder add`)* | M | angular-adapter | Scaffold a new local schematic (factory + schema.json stub). Verb changed from `add` per inventory revision 2026-05-12; scope unchanged from original entry. | 📋 PENDING — post-execute |
| 8 | `builder add` *(new purpose: external registration)* | M-L | angular-adapter | Register an externally published schematic as a dependency. Adds the package to `project-builder.json` `collections` + as a dev dep in `package.json`. Validates via cache lookup or registry fetch. Introduced per inventory revision 2026-05-12. | 📋 PENDING — post-execute |
| 9 | **Templates system foundation** | L | foundations / angular-adapter | Template runtime (initial set: `consumer`, `publishable`). JSON merge primitives, in-place application semantics, conflict resolution policy, idempotency invariants. Prerequisite for #10 and #11. Introduced per inventory revision 2026-05-12; design discussion deferred to its own session. | 📋 PENDING — after #6 |
| 10 | `builder create` | M-L | angular-adapter | Scaffold a new project from scratch via `--template=<name>`. Templates supplied by #9. TTY: prompt for template; non-interactive: explicit flag required. Introduced per inventory revision 2026-05-12. | 📋 PENDING — after #9 |
| 11 | `builder migrate` | L | angular-adapter | Transform an existing project between modes / versions / adapters via `--template=<name>` applied in-place (default; explicit `--inline` as alias) or extracted to a new directory (`--to=<path>`). Templates from #9. Introduced per inventory revision 2026-05-12; future home for v1 → v2 engine migration and adapter swaps. | 📋 PENDING — after #9 |
| 12 | `builder info` | S-M | angular-adapter | Inspection of collection / specific schematic | 📋 PENDING — post-execute |
| 13 | `builder sync` | M | angular-adapter | Fetch declared remote collections into global cache; registry auth model decided here | 📋 PENDING — post-execute |
| 14 | `builder validate` | M | angular-adapter | Lint mode for schematics (structural + semantic + dep graph) | 📋 PENDING — post-execute |
| 15 | `builder remove` *(polymorphic)* | S-M | angular-adapter | Remove a local schematic (by name) OR unregister an external collection (by package identifier). Argument disambiguation by name resolution. Widened from spec #3 entry per inventory revision 2026-05-12. | 📋 PENDING — post-execute |
| 16 | `builder skill update` | M | angular-adapter | Regenerate AI skill artifact when CLI version changes; depends on #17 | 📋 PENDING — after #17 |
| **17** | **AI Skill artifact — content design** | M-L | angular-adapter | Detailed skill markdown design (heuristics, frontmatter, exact path). Spec #3 deferred this explicitly. Design discussion 2026-05-12: skill teaches **CLI + SDK** (the union, not just CLI surface); init flow asks user whether to install the project-builder MCP server (concrete MCP design pending — see C2). | 📋 PENDING — when ready |
| 18 | npm multi-platform distribution | L | foundations (release sub-slice) | JS wrapper + platform packages `@my-cli/{darwin-arm64,...}` + real CI release (replaces `release.yml` stub) | 📋 PENDING — final v1.0 |
---

## Section B — Architectural details deferred until specific `/plan` invocations

These were intentionally NOT pre-decided during initial planning. Each surfaces as an ADR within the relevant `/plan`.

| # | Detail | Decided in |
|---|---|---|
| B1 | Exact shape of `Engine` interface (Go) — RFC #1 sketched, finalised at design | `/plan #2` Architectural Skeleton |
| B2 | Concrete catalogue of `Event` types (`FileCreated`, `ScriptStarted`, `InputRequested`, `Done`, `Failed`, `Cancelled`, etc.) and their fields | `/plan #2` Architectural Skeleton |
| B3 | Cobra flags → `schema.json` inputs mapping pattern | `/plan #5` builder init OR `/plan #6` builder execute |
| B4 | `Tree` API shape (staging area abstraction the engine adapter exposes) | `/plan #4` AngularSubprocessAdapter |
| B5 | Registry auth + cache model for `builder sync` (Go HTTP vs subprocess `npm pack`) | `/plan #9` builder sync |
| B6 | `project-builder.json` JSON Schema validation rules (forward-compat for unknown keys per spec #3) | `/plan #5` builder init |
| B7 | Bidirectional input protocol concrete wire format for AI agents (MCP / stdin JSON-RPC / etc.) | `/plan #5` or dedicated `/plan` for MCP exposure |

---

## Section C — Strategic / operational concerns

Touched in planning conversations but not yet in any formal `/plan`. Some need their own `/plan`, some are smaller config + docs.

| # | Item | Why it matters | Recommended action |
|---|---|---|---|
| C1 | TUI v1.1B layer (Bubble Tea + Bubbles + Huh + Lip Gloss + Glamour) | RFC #1 lists as Milestone 1B. Architecture is event-driven so swap-in is feasible | After v1.0 ships — own `/plan` post-#14 |
| C2 | MCP server exposure | Spec #2 leaves "design for v1.x or v2?" open. Affects how `builder execute` exposes its contract to AI agents | Decide BEFORE `/plan #5` (early enough to shape command interfaces) |
| C3 | Branch protection + CODEOWNERS on `main` | Repo has NO protection rules. Sensitive area UNFLAGGED in current registry. Supply-chain weakpoint if contributors come | Small `/plan` (S-M) before first external PR |
| C4 | Release & versioning strategy (semver, conventional commits → changelog automation, tag conventions) | Implicit in `/plan #14` but deserves an ADR sooner | ADR within `/plan #14`, OR a quick standalone `/plan` |
| ~~C5~~ | ~~Followup issue for the deferred `release.yml`~~ | ~~Required for S-005 acceptance~~ | ✅ **DONE** — issue [#8](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/8) filed 2026-05-10 |
| C6 | Migration path: v1 Angular schematics → v2 native engine | Spec #5 leaves "Whether AngularSubprocessAdapter stays in v2 indefinitely" open | When v2 engine repo (separate) arrives |
| ~~C7~~ | ~~`go-testing` skill audit~~ | ~~User added the skill mid-session~~ | ✅ **DONE** — audited 2026-05-10, confirmed covers table-driven + small fakes + teatest |
| **C8** | **AST-based FF-07 ctx-guard replacement** (InputRequested.Reply send sites) | Current `scripts/fitness/input-reply-ctx-guard.sh` uses grep-window heuristic. May produce false negatives on refactored code (Reply send split across helpers). | pre-`/plan #4` or alongside it |
| **C9** | **AST-based FF-08 EnvAllowlist exemption** (no-untyped-args rule) | Current rule uses inline `// fitness:allow-untyped-args env-allowlist` marker. AST walker would be more maintainable (recognise field by name + type vs. marker string). | pre-`/plan #4` or alongside it |
| **C10** | **`Error.MarshalJSON` verbose-mode opt-in** | Verify-final R-03 deferred until /plan #3 (JSON Renderer needs debug visibility). Context-driven verbose mode would re-enable Cause/Details/Path. | implement alongside `/plan #3` JSON Renderer |
| **C11** | **Graphify post-commit hook `.git/index.lock` orphan** | Build gotcha (2026-05-11): graphify rebuild can leave stale `.git/index.lock` after commits. Recurred ~3 times during architectural-skeleton build. Workaround: `fuser .git/index.lock` (must be empty) + `rm -f`. Root cause TBD — likely graphify's final git op crashes pre-release. | investigate before next major build session — patch hook (flock, cleanup on exit) or shift to post-merge/post-checkout |
| **C12** | **Fish config `$HOME/go/bin` PATH for WSL contributors** | Build gotcha (2026-05-11): `goimports` requires `$HOME/go/bin` in PATH; user's `~/.config/fish/config.fish` Linux branch was missing it. Patched during S-000 cycle. macOS branch (line 29) also missing — replicate if dev on macOS. | Add to CONTRIBUTING.md WSL/macOS notes |
| **C13** | **Slice doc obs #175 outdated path reference** | S-000.2 task says `internal/features/.gitkeep` (plural). Post-rename commit `6696112`, the actual path is `internal/feature/` (singular). Build sub-agent resolved at apply time; doc still drifts. | Fix via `mem_update` on obs #175 at next /plan |
| **C14** | **Stale CI assertions surface mid-PR after phase transitions** | Post-archive gotcha (2026-05-11): PR #12 CI initially failed on (a) `golangci-lint-action@v6` not supporting `golangci-lint v2.x` (masked on main pre-Phase-2), (b) `smoke (layout invariants)` step asserting 0 .go files (bootstrap-state safety net, obsolete in Phase 2). Fixed in commit `65b1eb7`. **General lesson**: at the start of each phase /plan, audit existing CI assertions for ones the phase will invalidate by design. | each future phase transition; surface in `/plan` design or `/verify final` scope |
| **C15** | **Document `script_started.args` type-variance in NDJSON schema GoDoc** | Verify-final INFO (renderer-adapters): `args` field renders as the JSON string `"[REDACTED]"` when `Sensitive=true`, otherwise as `[]string`. Intentional (redacted value should not look like a valid array), but downstream NDJSON consumers must handle both type variants. | Documentation polish; pair with C17 |
| **C16** | **FF-04 carve-out drift policy comment** | Verify-final INFO (renderer-adapters): `scripts/fitness/shared-isolation.sh` carve-out for `render/pretty → charmbracelet/* + lucasb-eyer/*` is enumerated (not prefix-based). Future lipgloss direct-dep growth WILL fire FF-04 — that IS the desired alarm. A policy comment documenting this expectation would prevent reactive carve-out creep. | Hygiene; do alongside next `scripts/fitness/` touch |
| **C17** | **External NDJSON consumer reference doc (12 stable discriminators)** | Verify-final INFO (renderer-adapters): the NDJSON `"type"` discriminator is a public, locked contract (ADR-04). No external-facing reference doc enumerates the 12 values. Required before the first downstream consumer (AI agent / CI pipeline) ships against the stream. | medium priority; ship before first external NDJSON integration |

---

## Active flags / sequencing constraints

These constraints apply across the whole roadmap and **must not be relaxed silently**:

- **Strict TDD enabled**: every `/plan` from #2 onwards must produce tests for every REQ-ID; fitness functions enforce in CI.
- **Spec V1 of v1.0-repo-bootstrap-and-tooling is signed**. Adjustments require `unfreeze=true` to `sdd-spec`.
- **Phase 1 boundary still active until #14 ships**: no Go product code in `/cmd/builder/` or `/internal/` until `/plan #2` runs (which fills these dirs with stub Go code).
- **`gh.exe` required from WSL**: never plain `gh`. Token has `project, workflow, repo` scopes.
- **License is Apache 2.0**: do not regress to MIT / ISC.
- **`handler.go ≤ 100 LOC` fitness function**: encoded in CI from `/plan #2` onwards.

---

## How this file is updated


## Roadmap — Updated Rows

Replace row 7 with two rows (7a and 7b) and add row 7c:

---

### Row 7a — `builder new schematic`

| # | Change | Triage | Phase | Coverage | Status |
|---|---|---|---|---|---|
| 7a | `builder new schematic` / `builder new s` | M | angular-adapter | Scaffold a new local schematic: creates folder, empty factory (default export), simplified `schema.json` (inputs-only format), registers in target collection's `collection.json`. Flags: `--collection` (default: `default`), `--description` (prompt if omitted), `--extends` (validates against `externals`). Runs `builder generate-types` automatically post-scaffold. Validates: name collision, collection existence, extends→externals consistency. See [Spec] CLI — `builder new`. | 📋 PENDING — post-execute |

### Row 7b — `builder new collection`

| # | Change | Triage | Phase | Coverage | Status |
|---|---|---|---|---|---|
| 7b | `builder new collection` / `builder new c` | S-M | angular-adapter | Scaffold a new local collection: creates `collection.json` with two lifecycle schematics (`add`, `remove`) pre-wired, registers in `project-builder.json` under `collections`. Flags: `--description` (prompt if omitted). Validates: name collision. See [Spec] CLI — `builder new`. | 📋 PENDING — post-execute |

### Row 7c — `builder generate-types`

| # | Change | Triage | Phase | Coverage | Status |
|---|---|---|---|---|---|
| 7c | `builder generate-types` | S | angular-adapter | Generate `.d.ts` files from simplified `schema.json` inputs. Produces typed interfaces for factory consumption. Auto-triggered by `builder new s` and `builder add`; also available as standalone command. Introduced per `builder new` design session 2026-05-14. | 📋 PENDING — post-execute |

---

## Notes on changes

### Roadmap row 7 (original)
**Before:**
> `builder new` *(was `builder add`)* — Scaffold a new local schematic (factory + schema.json stub). Verb changed from `add` per inventory revision 2026-05-12; scope unchanged from original entry.

**After:** Split into 7a (`new schematic`), 7b (`new collection`), and 7c (`generate-types`). The `builder new` command is now an extensible subcommand tree — future subcommands (`builder new orchestrator`, `builder new template`, etc.) can be added without breaking the command surface.

### Command Inventory changes
1. `builder new` → split into `builder new schematic <name>` (alias `s`) and `builder new collection <name>` (alias `c`)
2. Added `builder generate-types` as a new command

### Cross-references to new specs
- **[Spec] CLI — `builder new`**: full design spec for both subcommands, simplified schema format, extension model, validation rules
- **Addendum — collection.json / project-builder.json**: decisions from this design session that affect neighbouring file formats

1. **When `/plan` starts a change**: mark its row 🚧 IN FLIGHT.
2. **When `/sdd-archive` closes a change**: mark ✅ DONE with the issue link.
3. **When new pending work surfaces during explore/spec/design/apply/verify**: append to the relevant section (A / B / C) here AND in engram.
4. **The orchestrator MAY update this file** during `/plan` and `/sdd-archive` automatically. Manual edits are also fine — engram and this file should stay in sync; if they diverge, engram wins for SDD-level decisions.
