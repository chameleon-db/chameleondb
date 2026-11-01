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
| `dashboard`, `dashboard serve` | Local dashboard and its API server | freeze | Scope to be defined after the core epics |
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

- Machine-readable output (`--output json`) for every command that reports state.
- Stable exit codes, documented.
- No command prints a guarantee it did not verify.
- Commands that change a database support `--dry-run` and show the SQL.
