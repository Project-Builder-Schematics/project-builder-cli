# Contributing

Thanks for considering a contribution. The repo is still being bootstrapped (pre-`v1.0`); the toolchain below is the floor every contributor must match before opening a PR.

## Required tools

| Tool | Version | Purpose |
|---|---|---|
| Go | `1.23` (sourced from `go.mod` only — no `.go-version` / `.tool-versions`) | Compiler, test runner, module resolution |
| [`just`](https://github.com/casey/just) | `1.50.0` or newer | Task runner — `just`, `just build`, `just test`, `just lint`, `just fmt`, `just run` |
| [`gofumpt`](https://github.com/mvdan/gofumpt) | `v0.10.0` | Stricter Go formatter (superset of `gofmt`) |
| [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) | `v0.45.0` (or matching your Go version) | Import management |
| [`golangci-lint`](https://golangci-lint.run/) | `v2.12.2` | Aggregate linter — config in `.golangci.yml` (13 linters + 2 formatters) |
| [`lefthook`](https://github.com/evilmartians/lefthook) | `2.1.6` or newer | Git hook runner — `pre-commit`, `commit-msg` |
| [`@commitlint/cli`](https://commitlint.js.org/) | latest | Conventional-commit validator |

Install on macOS / Linux (Homebrew):

```sh
brew install just gofumpt golangci-lint lefthook
go install golang.org/x/tools/cmd/goimports@v0.45.0
npm install -g @commitlint/cli @commitlint/config-conventional
```

Editor support: this repo ships an `.editorconfig`. Use a plugin that respects it (VS Code / GoLand / Vim / Emacs all have one). Tabs in `*.go`, two-space indent in YAML / JSON / Markdown.

## First-time setup

```sh
git clone https://github.com/Project-Builder-Schematics/project-builder-cli.git
cd project-builder-cli
lefthook install     # wire up pre-commit + commit-msg hooks
just                 # list available targets
just build           # smoke check — should exit 0 on a fresh clone
```

## Day-to-day

```sh
just fmt    # gofumpt + goimports across the tree
just lint   # golangci-lint run
just test   # go test -race -cover ./...
just build  # go build ./...
just run -- <args>  # forwards to `go run ./cmd/builder`
```

## Conventional commits

All commit messages MUST follow [Conventional Commits](https://www.conventionalcommits.org/). The `commit-msg` hook enforces this via `@commitlint/config-conventional`.

Format:

```
type(scope): subject
```

Common types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`, `build`. Scope is optional but encouraged.

Examples:

- `feat(layout): add 11 directory placeholders + CI layout smoke (S-001)`
- `fix(ci): pin golangci-lint-action major version`
- `docs: update CONTRIBUTING with lefthook step`

## Branch naming

| Pattern | Use for |
|---|---|
| `feature/<short-name>` | New capability or non-trivial change |
| `fix/<short-name>` | Bug fix |
| `chore/<short-name>` | Tooling, deps, housekeeping |

PRs target `main`. CI must be green before merge.

## Hook smoke test (manual — REQ-06 scenarios)

Verify your local hooks are wired correctly before opening the first PR:

1. **Pre-commit fmt/lint gate**:
   - Stage a `*.go` file with deliberate formatting violations (e.g. extra blank lines, missing imports).
   - Run `git commit -m "test: hook smoke"`.
   - Expect: `gofumpt` / `goimports` re-stage corrected content, `golangci-lint --new-from-rev=HEAD` runs.
2. **Commit-msg conventional gate**:
   - Run `git commit -m "stuff"`.
   - Expect: commit is REJECTED with a `commitlint` diagnostic (`subject may not be empty` / `type may not be empty`).

If either hook does not fire, re-run `lefthook install` and confirm `.git/hooks/{pre-commit,commit-msg}` exist.

## Versioning and Releases

This project uses automated SemVer bumping on every merge to `main`.

### 0.x convention

The project is in the `0.x` phase (pre-GA). During this phase:

- **`bump:minor`** — any change that introduces a new top-level command (e.g. `builder execute`, `builder add`). Also use for BREAKING CHANGES while `major == 0` (e.g. removing or renaming a flag that users depend on). Under 0.x, breaking changes do NOT trigger a major bump — that decision is reserved for the explicit v1.0.0 cut.
- **`bump:patch`** — everything else: improvements, bug fixes, refactors, docs updates, internal tooling, dependency updates. When in doubt, use `bump:patch`.

The jump from `v0.x.y` to `v1.0.0` is a deliberate, human, manual decision tied to the v1.0 GA feature bundle — it is NOT automated.

### PR label policy

Every PR **must** carry exactly one of `bump:minor` or `bump:patch` before merging. The CI job `bump-label-validation` enforces this — a PR with zero labels or both labels will fail CI and cannot be merged.

**Examples:**

| Change | Correct label |
|---|---|
| New top-level command `builder execute` | `bump:minor` |
| Removing `--config` flag (breaking while `major == 0`) | `bump:minor` |
| Bug fix in `builder init` | `bump:patch` |
| Refactor internal packages | `bump:patch` |
| Update Go dependency | `bump:patch` |
| Docs / CONTRIBUTING update | `bump:patch` |

Add the label via GitHub UI (PR sidebar → Labels) or:
```sh
gh pr edit <number> --add-label bump:patch
```

### How the automation works

On merge to `main`, the `Release` workflow (`release.yml`) runs as `github-actions[bot]`:

1. Reads the merged PR's bump label.
2. Reads the current `const Version` from `cmd/builder/version.go`.
3. Calls `scripts/release/bump-version.sh` to compute the new version.
4. Updates `cmd/builder/version.go`, commits `chore(release): vX.Y.Z`, and pushes to `main`.
5. Creates an annotated tag `vX.Y.Z` and a GitHub Release.

The bot's own commit fires a second `push: main` event, which is suppressed by the job-level guard `if: github.actor != 'github-actions[bot]'` — no infinite loop.

Direct pushes to `main` (admin hotfixes without a PR) are detected and skipped cleanly — no tag is created.

### Wrong-tag recovery procedure (REQ-CVA-033)

If a wrong tag is pushed by the automation:

1. **Delete the GitHub Release** (safe — does not delete the tag):
   ```sh
   gh release delete vX.Y.Z --yes
   ```

2. **Leave the git tag in place** if at all possible. Git tags are part of the shared history of every collaborator's clone. Silently force-deleting a tag from the remote (`git push origin :refs/tags/vX.Y.Z`) will leave stale references in clones — coordinate with the team before doing this.

3. **If you must delete the tag** (e.g. the tagged commit itself was wrong), notify all contributors first, then:
   ```sh
   git push origin :refs/tags/vX.Y.Z     # delete remote tag
   git tag -d vX.Y.Z                     # delete local tag
   ```
   Ask collaborators to also run `git tag -d vX.Y.Z` locally.

4. **Fix the root cause** (wrong label on the merged PR, bug in the release script) in a hotfix PR with a `bump:patch` label.

5. The next qualifying merge will produce the correct next version based on the current `const Version` in `cmd/builder/version.go`. If the deleted tag's version was consumed, the next bump will naturally skip past it.

### Manually testing bump arithmetic

```sh
just test-bump      # runs scripts/release/bump-version.test.sh (10 cases)
```

### Disabling the automation (emergency)

GitHub UI → Actions → `Release` → Disable workflow. The workflow can be re-enabled after a fix is merged.

## SDD pipeline

This project uses Specification-Driven Development (SDD): every non-trivial change goes through `triage → explore → propose → spec → design → slice → apply → verify → archive`. The pipeline conventions live in the repo's [GitHub Discussions](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions) (#2–#5 cover triage classifications, persona lenses, and slice format).

When you open a PR, link to the spec issue (e.g. `Refs #6`) and reference the slice IDs you completed (`S-001`, `S-002`, ...). The spec is the source of truth — implementation deviations get flagged in the SDD verify step.

## License

[MIT](./LICENSE).
