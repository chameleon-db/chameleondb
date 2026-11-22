# ADR-0012: Configuration flags for planned features

- **Status:** Proposed (the horizons below need the maintainer's confirmation)
- **Date:** 2026-11-09
- **Related:** [ADR-0002](0002-honest-claims-policy.md), [ADR-0006](0006-levels-of-control-and-role-enforcement.md), [ADR-0013](0013-mode-and-password-threat-model.md)

## Context

`chameleon init` has been writing four keys that were added with a feature in mind:

| Key | Intent |
|---|---|
| `safety.backup_before_apply` | Take a backup before `migrate --apply` |
| `features.backup_on_migrate` | The same intent, under another section |
| `features.audit_logging` | Control the journal |
| `features.rollback_enabled` | Allow undoing a migration |

None of them has behavior. The code never reads them (the only trace was a warning printed when `backup_on_migrate` was true), and `init` wrote them as `true`. A user who opens their own `.chameleon.yml` and reads `backup_before_apply: true` reasonably believes a backup exists. That is the kind of statement [ADR-0002](0002-honest-claims-policy.md) rules out: a capability described in the present tense that is only planned.

The keys also record real intent, and deleting them would lose it. The decision is therefore how to keep the intent without making a claim, and what each feature needs, so that the plan lives in the repository.

## Decision

1. **Planned keys stay in the file, as `false`, with a comment.** `init` and every later save write them with `# Planned, not implemented: true has no effect. See ADR-0012.` The file shows what is coming and never says it is active.
2. **Setting one to `true` is allowed but warned.** Existing files keep loading. Every command prints one warning per key and per process: the key is planned and has no effect.
3. **The two backup keys become one when the feature is built.** The name is decided at implementation time; until then both are accepted.
4. **Each feature has a horizon** (below). A key stops being "planned" only in the release that ships its behavior, together with the test that proves it and the removal of the comment.
5. **`migrate --apply` states that ChameleonDB creates no backup**, always, because that is true whatever the file says.

### Plan per feature

| Feature | Now | H1 | H2 |
|---|---|---|---|
| **Backup before apply** (`backup_before_apply`, `backup_on_migrate`) | The key is planned; `migrate --apply` says no backup is created | `pg_dump` of the schema (no data) before the apply, saved under `.chameleon/backups/` with the migration id (`backups/` is already in `.gitignore`). If the dump fails or `pg_dump` is missing, nothing is applied | Dump with data as an option, retention, restore command |
| **Audit logging** (`audit_logging`) | The journal exists (`internal/journal`) and is written by the CLI regardless of the key. The key is planned | Decide whether the key is a switch or a verbosity level. Turning the journal off conflicts with the paranoid default and with [ADR-0013](0013-mode-and-password-threat-model.md), so the likely outcome is a level (`standard`, `verbose`), not an off switch | Retention, export, tamper evidence for the journal |
| **Rollback** (`rollback_enabled`) | The key is planned; `init` already says rollback is planned | Nothing. Migrations are create-only, so there is nothing to undo except the `DROP` of what a migration created | Needs down-migrations or a snapshot restore, and an alter/diff engine. The state tracker stores only the DDL hash today. Depends on the canonical schema digest ([ADR-0005](0005-canonical-schema-digest.md)) |

### What this ADR does not decide

- The exact name and shape of the merged backup key.
- Whether the audit setting becomes a level.
- The rollback mechanism.

Each gets its own issue when its horizon starts.

## Related finding: other keys with no behavior

While checking the four keys, a search of the Go code outside the `config` package found no reader for these keys either, and `init` writes them with values that look active:

- `features.auto_migration` (written `true`), `features.dry_run_default`, `safety.require_confirmation`, `safety.validate_schema` (written `true`), `schema.validation_strict`
- `database.max_connections` (10), `database.connection_timeout` (30), `database.migration_timeout` (300)

`database.driver` is also not read (the tool is PostgreSQL only). This ADR does not change them. They should get the same treatment as the planned keys, or be removed from the generated file, in a follow-up issue. That choice is the maintainer's: some of them may be meant to be wired up soon.

## Consequences

- Positive: the generated file no longer claims anything it does not do; the intent is kept and visible; each feature has a place in the plan; existing files still load.
- Negative: four extra lines with comments in every generated file; users who copied `true` get a warning on each command until they change it.
- Follow-up: one issue per feature at its horizon; one issue for the keys listed in the related finding; update `docs/cli/config` in the documentation repository.
