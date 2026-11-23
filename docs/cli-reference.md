# CLI reference and disposition (proposal)

> **Status:** draft · **Date:** 2026-10-24
> The *Today* column describes the commands registered in `chameleon/cmd/chameleon` (descriptions come from the CLI's own help text). The *Proposed disposition* column is a proposal for discussion, not a commitment. Flags and subcommands are not listed yet.

Dispositions: **keep** (core to the vision), **rework** (keep the name, change behaviour), **freeze** (no new features until the related decision is made), **deprecate** (remove after a notice), **add** (new).

## Existing commands

| Command | Today | Proposed disposition | Notes |
|---|---|---|---|
| `init [name]` | Initialise a new project | keep | Should create the minimal structure for observe mode first |
| `check [file]` | Check a schema for errors (used by editor extensions) | keep | Stable machine-readable output is a contract; test it |
| `validate [file]` | Validate a schema | rework | Overlaps with `check`; merge or document the difference |
| `migrate` | Manage database migrations | rework | Becomes plan/apply with risk classification; destructive changes gated |
| `journal schema [version]` | Show schema version history (vault) | keep | Revisit with the canonical digest |
| `journal migrations` | Show migration history | keep | |
| `journal last [n]`, `journal errors` | Last journal entries; error entries | keep | Journal becomes part of the audit trail |
| `introspect <database-url>` | Generate a schema from an existing database | keep | Core of observe/drift; define the supported subset first |
| `verify` | Verify vault integrity | rework | Extend to verify control level and drift; state the threat model |
| `status` | Show status | keep | |
| `config get`, `config set key=value` | Local configuration, including the integrity mode | keep | |
| `config auth set-password` | Set or rotate the admin password for mode upgrades | rework | Depends on [ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md); password hashing to be reviewed |
| `query [entity]` | Interactive query execution (for testing) | freeze | Data-access scope: [ADR-0007](adr/0007-data-access-layer-scope.md) |
| `playground-sql`, `playground-sql serve` | Compile SQL from a JSON payload; minimal HTTP server for it | freeze | Developer tool; must bind to loopback and require a token if kept |
| `uninstall` | Uninstall ChameleonDB | keep | |
| `version` | Show version | keep | |

## Candidate new commands

| Command | Purpose | Depends on |
|---|---|---|
| `digest` | Print the canonical digest of a schema | [ADR-0005](adr/0005-canonical-schema-digest.md) |
| `snapshot` | Store a baseline snapshot of a database (no data rows, no credentials) | introspection, canonical model |
| `drift` | Compare a reference (schema, snapshot or another database) with a live database and classify differences | introspection, canonical model, `snapshot` |
| `plan` | Produce a risk-classified change plan without applying it | diff and planning layer |
| `compat <revA> <revB>` | Compatibility verdict between two revisions | [ADR-0009](adr/0009-offline-first-and-distributed-data.md) |
| `doctor` | Check environment, versions and control level | [ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md) |

## Conventions to adopt

- Machine-readable output (`--json`) for every command that reports state. Available today for `validate`, `status`, `verify`, `migrate` and `introspect`; see [JSON output and exit codes](#json-output-and-exit-codes).
- Stable exit codes, documented.
- No command prints a guarantee it did not verify.
- Commands that change a database support `--dry-run` and show the SQL.

## JSON output and exit codes

`validate`, `status`, `verify`, `migrate` and `introspect` accept `--json`. With it, **stdout holds exactly one JSON document** and nothing else; the human output (progress, warnings) goes to stderr. Any other command with `--json` fails with `usage_error`. `--json` also implies `--no-input`: nothing is ever asked, and a step that would need a confirmation fails with `usage_error` and a hint (for example `introspect` over an existing schema needs `--force`).

### Envelope

```json
{
  "schema_version": 1,
  "command": "chameleon validate",
  "ok": false,
  "data": { "source": "bad.cham", "valid": false },
  "errors": [
    {
      "code": "schema_invalid",
      "message": "Unexpected token: ...",
      "hint": "Entity definitions must have an opening brace.",
      "file": "bad.cham",
      "line": 1,
      "column": 8,
      "snippet": "  1 │ entity {\n    │        ^"
    }
  ]
}
```

- `schema_version` changes only when a field is removed or changes meaning. New fields can be added without a bump, so parse leniently.
- `ok` is `true` exactly when `errors` is empty. `data` is always an object and `errors` always an array, never `null`.
- `data` may be filled on failure too (for example `verify` lists the versions it checked).
- `errors[].code` is one of the codes below. `hint`, `file`, `line`, `column` and `snippet` appear only when known.
- Passwords in connection strings (`scheme://user:password@host`, `password=...`) are redacted in every output, with or without `--json`.

### Exit codes

The exit code is the same with and without `--json`.

| Exit | `errors[].code` | Meaning |
|---|---|---|
| 0 | | Success |
| 1 | `internal_error` | Unexpected failure with no more specific class |
| 2 | `usage_error` | Wrong flag, argument or command; also a prompt that `--json` cannot answer |
| 3 | `schema_invalid` | The schema does not parse or type-check |
| 4 | `mode_blocked` | The operation is refused by the paranoid mode |
| 5 | `integrity_failed` | Vault integrity check failed, or `mode.json` was edited outside the CLI |
| 6 | `connection_failed` | The database could not be reached |
| 7 | `not_initialized` | Not a ChameleonDB project (no configuration or no vault) |
| 8 | `apply_failed` | The database rejected the migration |

`status` is the diagnostic command: it exits 0 and reports a failed integrity check in `data.vault.integrity`. Use `verify` when you need a non-zero exit.

### `data` per command

| Command | `data` |
|---|---|
| `validate` | `valid` (bool), `source` (file or `project schemas`) |
| `status` | `schema` (`state`, `current_version`, `hash`, `last_modified`, `error`), `vault` (`initialized`, `versions`, `integrity`, `issues`, `mode`, `error`) |
| `verify` | `valid`, `versions[]` (`version`, `ok`, `issue`), `issues[]`, `schema_file` (`exists`, `matches_current`) |
| `migrate` | `mode` (`check`, `dry_run`, `apply`), `up_to_date`, `changed`, `pending_unapplied`, `summary`, `sql` (the plan), `applied`, `version`, `hash`, `duration_ms` |
| `introspect` | `output`, `tables`, `duration_ms` |

Without a database you can already use `validate`, `migrate --dry-run`, `status` and `verify`.

## Output order

The same schema gives the same output, byte for byte, on every run. Where the order is not the order of the entities in your files, it is:

- Columns (`CREATE TABLE`, `SELECT` lists, includes): the primary key first, then alphabetical by name.
- Relations and constraints: alphabetical by name.
- Entities: the order of the schema files.
- Errors and listings: by entity order, then alphabetical.

The tests in `chameleon-core/tests/determinism.rs` parse a schema 100 times and compare the output.
