# ChameleonDB releases

**Current release: [v1.1.3-alpha](https://github.com/chameleon-db/chameleondb/releases/tag/v1.1.3-alpha)** (2026-11-08)

This page is the overview of the current release: what it contains, what works, what does not, how to install it and what we are working on. The complete, version-by-version list of changes is the [CHANGELOG](CHANGELOG.md).

## At a glance

| | |
|---|---|
| **Stage** | Alpha. Syntax, APIs and behavior can still change. Do not use it with untrusted input or production data yet. |
| **Database** | PostgreSQL only (integration tests run on 16; `default uuid()` needs 13 or newer, or `pgcrypto`) |
| **Platforms** | Linux x86-64 and arm64, macOS Apple Silicon (arm64), Windows x86-64. No prebuilt binary for Intel Macs. |
| **Components** | Rust core (parser, type checker, SQL and migration generation), Go runtime and CLI |
| **License** | Apache 2.0 |
| **Security** | Report vulnerabilities privately, see [SECURITY.md](SECURITY.md) |
| **Documentation** | [docs.chameleondb.dev](https://docs.chameleondb.dev) |

## In this release: v1.1.3-alpha

v1.1.3-alpha changes the install path only; the core and the runtime are the same as in v1.1.2-alpha.

- The install script works on macOS (Apple Silicon) and on Linux arm64. Before, it failed with a 404 on macOS because the archive name did not match.
- Every archive has its SHA-256 file attached to the release, and the install script verifies it.
- The macOS archive is now `chameleon-macos-arm64` (it was called `universal` but contained arm64 only).

## Previous release: v1.1.2-alpha

A maintenance release with fixes to mutations, debugging and the CLI. It does not change the schema language or the migrations.

### Mutations

- Deleting a record that still has children returns `ForeignKeyConstraintError`, with the dependent table. Before, it returned a `ForeignKeyError` that said the referenced record did not exist.
- `DELETE` filters are generated in a fixed order, as `UPDATE` already was.
- A `nil` filter value now means `IS NULL` (`eq`) or `IS NOT NULL` (`neq`, `ne`). Before, it produced `= NULL`, which never matches.
- `SafetyError` no longer suggests `ForceUpdateAll` and `ForceDeleteAll`, which do not exist.

### Debugging

- `CHAMELEON_DEBUG` works: `1`, `true` or `sql` print the SQL and its parameters; `trace` adds durations and row counts; `explain` is the most verbose. It applies to queries and mutations.
- `.Debug()` on a query with `Include()` prints the eager-loading queries as well.
- Parameter values of columns that look like secrets (`password`, `token`, `secret`, `hash`, `api_key`, ...) are shown as `[REDACTED]`.
- A query's SQL is generated once per execution instead of twice.

### CLI and vault

- `chameleon status` and `chameleon validate` read the schema files configured in `.chameleon.yml` (`schema.paths`). Before, projects created with `init` (schemas in `schemas/`) were not checked.
- `.chameleon/.gitignore` ignores `vault/mode_auth.json`, the verifier of the mode password.
- `chameleon version` prints a single line.
- The install script works when piped from `curl` and its closing message no longer repeats the product name.

### Behavior changes

Read these before upgrading from v1.1.1:

| Change | What to do |
|---|---|
| Mutation `like` and `ilike` now mean "contains", as in queries: `"ana"` matches `%ana%`. `%`, `_` and `\` in the value match literally. | If you passed your own `%` pattern, remove the wildcards. |
| A `nil` filter value in a mutation is `IS NULL` or `IS NOT NULL`. Any other operator with `nil` returns an error. | Check filters that passed `nil` and relied on matching nothing. |
| `vault/mode_auth.json` is ignored by git for new and upgraded projects. A file that is already committed stays tracked. | Run `git rm --cached .chameleon/vault/mode_auth.json` and commit. |
| Debug output redacts values of columns that look like secrets. | None. Other values are still printed, so do not enable debug output where logs hold personal data. |

No migration is needed. Existing tables are never altered by ChameleonDB.

## What works today

| Area | State | Notes |
|---|---|---|
| Schema language (`.cham`) | Working | Entities, fields, relations, defaults (`uuid()`, `now()`, literals), `unique`, `nullable`, `vector(N)`, arrays |
| Type checker (`validate`, `check`) | Working | Rejects invalid primary keys, `vector(0)`, mismatched has-many keys and cycles between entities |
| Migrations (`migrate`) | Partial | Create-only: `CREATE TABLE IF NOT EXISTS` in a deterministic order. Existing tables are left as they are. |
| Schema Vault | Working | Versions with a SHA-256 hash of the schema source and the parent version, stored locally. Detects edits made outside the tool. |
| Integrity modes | Working | `readonly`, `standard`, `privileged`, `emergency`. They govern what the CLI allows, not database permissions. |
| Journal | Working | Operation log with `journal last`, `migrations`, `errors`, `schema` and `search` |
| Introspection | Working | Generates a schema from PostgreSQL and warns about columns it cannot map |
| Go data-access layer | Working, frozen | Query and mutation builders with eager loading, write guards (no update or delete without a filter) and typed constraint errors. Scope under review ([ADR-0007](docs/adr/0007-data-access-layer-scope.md)). |
| Annotations (`@cache`, `@olap`, `@vector`, `@ml`) | Parsed only | They have no effect today |
| Other databases | Not available | Only PostgreSQL |

### What the guarantees mean

- The vault detects changes made to its files outside the tool. It does not stop someone who can edit both the files and their hashes.
- Integrity modes do not replace database permissions. A user with direct access to the database is not restricted by them.
- The schema hash is computed over the source text, so formatting changes alter it. A canonical form is planned ([ADR-0005](docs/adr/0005-canonical-schema-digest.md)).

## Known limitations

- **No schema diff.** `migrate` creates missing tables only. Changes to existing tables (new columns, type changes) must be applied by hand.
- **No rollback and no backups.** The old `rollback_enabled`, `audit_logging`, `backup_on_migrate` and `backup_before_apply` settings are not implemented: `init` no longer writes them and a file that sets them gets a warning. `migrate --apply` says that no backup is created.
- **No transactions** in the data-access layer; each mutation is its own statement.
- **Types:** `int` maps to a 32-bit `INTEGER` and `timestamp` to `TIMESTAMP` without time zone. Both are planned to change; see the plan below.
- **Table names** are derived differently by the migration generator and the mutation layer for some multi-word or irregular names. A fix that makes them identical is planned and will come with migration notes.
- **Filters:** the `in` operator, `OR` and `time.Time` values are not supported. Pass dates as strings.
- **Intel Macs and Windows on ARM** have no prebuilt binary. Build from source.
- **Install script:** it does not install on Windows (download the archive by hand). It verifies the SHA-256 checksum of the archive, but the checksum comes from the same release page as the archive, so it detects corrupted downloads, not a compromised release. Release signing is not available yet.
- **Developer server** (`playground-sql serve`) is for local use only. Do not expose it to a network.

## What we are working on

Work is organised by horizon, not by date. The board and the issues are the source of truth: [project board](https://github.com/orgs/chameleon-db/projects/1) and [issues](https://github.com/chameleon-db/chameleondb/issues). What follows is direction, not a commitment.

| Horizon | Goal | Examples |
|---|---|---|
| **H1a, observe and drift** | Point the tool at a PostgreSQL database with read-only access and see how it differs from a reference | Canonical schema digest, faithful introspection of a supported subset, baseline snapshots, `drift` command with text and JSON reports |
| **H1b, controlled change** | Review each change with its risk before it is applied | Risk-classified change plans, gates for destructive changes, verification of the control level |
| **Later** | Beyond PostgreSQL and a single team | Additional backends, offline-first compatibility between schema revisions |

See the [product definition](docs/product-definition.md) for scope and the [architecture overview](docs/architecture/overview.md) for how today's code relates to the target.

## Install

### Linux and macOS

```bash
curl -sSL https://chameleondb.dev/install | sh
```

Supported: Linux x86-64 and arm64, macOS on Apple Silicon. The script installs `chameleon`, the shared library, the C header and a `pkg-config` file, and verifies the SHA-256 checksum of the archive. It asks before reinstalling. Set `CHAMELEON_INSTALL_DIR`, `CHAMELEON_LIB_DIR`, `CHAMELEON_INCLUDE_DIR` or `CHAMELEON_PKGCONFIG_DIR` to change the destinations.

On Intel Macs, build from source (below).

### Windows

Download the archive from [chameleondb.dev/windows](https://www.chameleondb.dev/windows) or from the release page (`chameleon-windows-amd64.tar.gz`). It contains two files that must stay together:

- `chameleon.exe`
- `chameleon.dll`

Put them in a folder (for example `c:\chameleon\`), add that folder to `Path` (System Settings, Environment Variables), then open a new terminal.

### From source

```bash
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb
make build            # builds the Rust core first, then chameleon/bin/chameleon
```

You need Go 1.25.6 or newer, a stable Rust toolchain, a C compiler and `make`.

### Verify

```bash
chameleon version
# ChameleonDB v1.1.3-alpha
```

## Try it

```bash
mkdir test-chameleondb && cd test-chameleondb
chameleon init
# edit schemas/example.cham, or write your own entities
chameleon validate
chameleon migrate --dry-run           # preview the SQL
chameleon config auth set-password    # recommended before changing the mode
chameleon config set mode=standard    # new vaults start in readonly
chameleon migrate --apply
chameleon status
chameleon journal schema
```

Use a test database. See the [documentation](https://docs.chameleondb.dev) for the schema language, queries, mutations and the security model.

## Release history

| Version | Date | Highlights |
|---|---|---|
| [v1.1.2-alpha](CHANGELOG.md) | 2026-11-06 | Mutation filter fixes (`IS NULL`, `like` as contains), foreign-key delete error, `CHAMELEON_DEBUG`, debug of eager queries, CLI reads `schema.paths`, `mode_auth.json` ignored by git |
| [v1.1.1-alpha](CHANGELOG.md) | 2026-11-06 | Stricter `validate`, self-referential relations, `introspect` fixes and warnings, journal rewrite and `journal search` |
| [v1.1.0-alpha](CHANGELOG.md) | 2026-11 | Security hardening of query and mutation generation (GHSA-2r6g-424r-6xmr), `migrate` is create-only and no longer drops tables, deterministic DDL |
| v1.0-alpha | 2026-02-25 | Schema Vault, integrity modes, identity map for eager loading, typed database errors, `status`, `verify` and `journal schema` |

Earlier pre-1.0 releases are in the [CHANGELOG](CHANGELOG.md). Releases before v1.1.0-alpha have a security issue in query generation; upgrade if you run one.

## Feedback

- Bugs and requests: [GitHub Issues](https://github.com/chameleon-db/chameleondb/issues)
- Vulnerabilities: privately, through the [security advisory form](https://github.com/chameleon-db/chameleondb/security/advisories/new)
- Questions and chat: [Discord](https://chameleondb.dev/discord)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
