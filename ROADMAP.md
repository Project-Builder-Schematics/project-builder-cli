# Project Builder CLI — Roadmap

> **Status**: Phase 1 (repo bootstrap & tooling) merged via PR #9. Phase 2 (architectural-skeleton, #7) merged via PR #12 — full inert Go skeleton landed (35/35 REQs, 9/9 fitness functions, 43 tests, all 5 CI jobs green). Phase 2 follow-on (renderer-adapters, #10) and Phase 3 (angular-subprocess-adapter, #11) have `/plan` complete + slices ready and are UNBLOCKED — can `/build` in parallel.
> **Last updated**: 2026-05-11 (post `/build` + merge of architectural-skeleton via PR #12)
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
| **3** | **Pretty + JSON Renderer adapters** | **M** | architecture | 3 capabilities (`pretty-renderer`, `json-renderer`, `renderer-factory`) + composition-root delta — TUI deferred to Phase 1B; 14 REQs across 4 domains; 4 ADRs; 3 SPIDR slices | 🚧 **READY TO BUILD** — spec [#10](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/10) (`/plan` complete, V1 signed, slices ready). UNBLOCKED post-#2 merge — can start now. C10 (Error.MarshalJSON verbose-mode opt-in) lands alongside JSON Renderer here. |
| **4** | **AngularSubprocessAdapter** | **L** (sensitivity override: subprocess + supply-chain + privilege) | angular-adapter | First concrete `Engine` impl. NDJSON over stdio + embedded `runner.js` via `//go:embed` + SIGTERM/SIGKILL build-tag isolation + injectable Discoverer. 8 domains, 38 REQs, 4 ADRs, 9 existing fitness functions reused; 6 SPIDR slices (incl. integration slice with real Node) | 🚧 **READY TO BUILD** — spec [#11](https://github.com/Project-Builder-Schematics/project-builder-cli/issues/11) (`/plan` complete, V1 signed, slices ready). UNBLOCKED post-#2 merge — can run in parallel with #3. C8 + C9 (AST-based FF-07/FF-08 hardening) recommended alongside this slice. |
| 5 | `builder init` end-to-end | M | angular-adapter | First real command: generates `project-builder.json` + skill stub + folder structure + package.json script alias | 📋 PENDING — after #3 |
| 6 | `builder execute` end-to-end | L | angular-adapter | Central command. Full 6-step pipeline. Requires #4 functional | 📋 PENDING — after #4 + #5 |
| 7 | `builder add` | M | angular-adapter | Scaffold a new local schematic (factory + schema.json stub) | 📋 PENDING — post-execute |
| 8 | `builder info` | S-M | angular-adapter | Inspection of collection / specific schematic | 📋 PENDING — post-execute |
| 9 | `builder sync` | M | angular-adapter | Fetch declared remote collections into global cache; registry auth model decided here | 📋 PENDING — post-execute |
| 10 | `builder validate` | M | angular-adapter | Lint mode for schematics (structural + semantic + dep graph) | 📋 PENDING — post-execute |
| 11 | `builder remove` | S | angular-adapter | Cleanup local schematic from workspace | 📋 PENDING — post-execute |
| 12 | `builder skill update` | M | angular-adapter | Regenerate AI skill artifact when CLI version changes; depends on #13 | 📋 PENDING — after #13 |
| **13** | **AI Skill artifact — content design** | M-L | angular-adapter | Detailed skill markdown design (heuristics, frontmatter, exact path). Spec #3 deferred this explicitly | 📋 PENDING — when ready |
| 14 | npm multi-platform distribution | L | foundations (release sub-slice) | JS wrapper + platform packages `@my-cli/{darwin-arm64,...}` + real CI release (replaces `release.yml` stub) | 📋 PENDING — final v1.0 |

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

1. **When `/plan` starts a change**: mark its row 🚧 IN FLIGHT.
2. **When `/sdd-archive` closes a change**: mark ✅ DONE with the issue link.
3. **When new pending work surfaces during explore/spec/design/apply/verify**: append to the relevant section (A / B / C) here AND in engram.
4. **The orchestrator MAY update this file** during `/plan` and `/sdd-archive` automatically. Manual edits are also fine — engram and this file should stay in sync; if they diverge, engram wins for SDD-level decisions.
