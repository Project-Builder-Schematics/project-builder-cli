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

## SDD pipeline

This project uses Specification-Driven Development (SDD): every non-trivial change goes through `triage → explore → propose → spec → design → slice → apply → verify → archive`. The pipeline conventions live in the repo's [GitHub Discussions](https://github.com/Project-Builder-Schematics/project-builder-cli/discussions) (#2–#5 cover triage classifications, persona lenses, and slice format).

When you open a PR, link to the spec issue (e.g. `Refs #6`) and reference the slice IDs you completed (`S-001`, `S-002`, ...). The spec is the source of truth — implementation deviations get flagged in the SDD verify step.

## License

[MIT](./LICENSE).
