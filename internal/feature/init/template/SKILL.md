---
name: pbuilder
description: AI agent skill for Project Builder CLI (preview)
---

# Project Builder — AI Skill (v0 / preview)

This is a placeholder skill artefact bundled with `builder init` in v1.0.
Full content design (decision heuristics, examples, when-to-formalise rules)
is tracked at: https://github.com/Project-Builder-Schematics/project-builder-cli (roadmap row 13)

## CLI Operations (current command inventory)

- `builder init` — initialise a Project Builder workspace
- `builder execute` (alias: `e`, `generate`, `g`) — run a schematic
- `builder add` — scaffold a new local schematic
- `builder info` — inspect a collection or schematic
- `builder sync` — fetch declared remote collections
- `builder validate` — lint mode for schematics
- `builder remove` — remove a local schematic
- `builder skill update` — regenerate this skill when the CLI version changes

## Decision Heuristics

TODO — content design deferred to roadmap row 13.

## Update

When the CLI version changes, run `builder skill update` to refresh this file.
