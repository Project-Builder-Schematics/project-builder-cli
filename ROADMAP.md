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
