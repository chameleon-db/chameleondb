# ChameleonDB releases

**Current release: [v1.3.0-alpha](https://github.com/chameleon-db/chameleondb/releases/tag/v1.3.0-alpha)** (2026-11-23)

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

## In this release: v1.3.0-alpha

Scripts and coding agents can now drive ChameleonDB: `--json` output, exit codes that say what failed, and rules for agents written into the project. Read "Behavior changes" before upgrading.

### Machine-readable output

- `validate`, `status`, `verify`, `migrate` and `introspect` accept `--json`. Stdout then holds exactly one JSON document (`schema_version`, `command`, `ok`, `data`, `errors[]`); the human output goes to stderr. `--json` never asks questions.
- Errors have a stable `code` and a `hint`; schema errors also carry `file`, `line`, `column` and `snippet`. `migrate --json` returns the plan (`data.sql`) and, with `--apply`, the result.
- Passwords in connection strings are redacted in every output.
- `validate`, `migrate --dry-run`, `status` and `verify` need no database.

```bash
chameleon validate --json | jq '.errors[] | {code, file, line, message}'
chameleon migrate --dry-run --json | jq -r '.data.sql'
```

### Exit codes

The same with and without `--json`:

| Exit | Meaning |
|---|---|
| 0 | Success |
| 1 | Unexpected failure |
| 2 | Wrong flag, argument or command |
| 3 | The schema is invalid |
| 4 | Blocked by the integrity mode |
| 5 | Integrity check failed (vault or `mode.json`) |
| 6 | Database unreachable |
| 7 | Not a ChameleonDB project |
| 8 | The database rejected the migration |

The full contract is in [JSON output and exit codes](docs/cli-reference.md#json-output-and-exit-codes).

### Coding agents

- `CHAMELEON.md` has a "For coding agents" section: run `validate` and `migrate --dry-run`, ask a human before `migrate --apply`, do not edit `.chameleon/`, readonly mode is expected, no real `DATABASE_URL` in files.
- `chameleon init --agents` adds the same rules to `AGENTS.md`, creating it or appending one block. On an existing project it only touches `AGENTS.md`.

### Deterministic output

The same schema gives the same bytes on every run. Before, the column list of `SELECT *` and of include queries, the schema JSON between the core and the runtime, and the order of type-check errors depended on hash-map order. The order is documented in [Output order](docs/cli-reference.md#output-order).

### Install script

With a `CHAMELEON_LIB_DIR` other than the default, the script no longer reports a failed check; it explains how to export the library path.

### Behavior changes

| Change | What to do |
|---|---|
| Failures exit with 2 to 8 instead of 1. | Scripts that test `$? -eq 1` should test `$? -ne 0`, or the specific code. |
| Errors are printed once, to stderr. | Nothing, unless you parse stdout for errors: use `--json`. |
| `SELECT *` columns come in a fixed order (primary key, then alphabetical). | Nothing. Rows are decoded by column name. |

No migration is needed.

## Previous release: v1.2.0-alpha

`init` asks how to connect and never overwrites your files, the mode password is stored with Argon2id, edits to `mode.json` are detected, `Include` works on belongs-to relations, `engine.NewEngineAt` opens a project from any directory, and the `chameleon dashboard` command was removed. Details are in the [CHANGELOG](CHANGELOG.md).

## Earlier: v1.1.3-alpha

Changed the install path only: the install script works on macOS (Apple Silicon) and Linux arm64, and every archive has a SHA-256 file that the script verifies.

## What works today

| Area | State | Notes |
|---|---|---|
| Schema language (`.cham`) | Working | Entities, fields, relations, defaults (`uuid()`, `now()`, literals), `unique`, `nullable`, `vector(N)`, arrays |
| Type checker (`validate`, `check`) | Working | Rejects invalid primary keys, `vector(0)`, mismatched has-many keys and cycles between entities |
| Migrations (`migrate`) | Partial | Create-only: `CREATE TABLE IF NOT EXISTS` in a deterministic order. Existing tables are left as they are. |
| Schema Vault | Working | Versions with a SHA-256 hash of the schema source and the parent version, stored locally. Detects edits made outside the tool. |
| Integrity modes | Working | `readonly`, `standard`, `privileged`, `emergency`. They guard against accidental changes through the CLI; they are not access control and do not set database permissions. The password is stored with Argon2id. |
| JSON output and exit codes | Working | `--json` on `validate`, `status`, `verify`, `migrate`, `introspect`; stable exit codes. Other commands do not support `--json` yet. |
| Journal | Working | Operation log with `journal last`, `migrations`, `errors`, `schema` and `search` |
| Introspection | Working | Generates a schema from PostgreSQL and warns about columns it cannot map |
| Go data-access layer | Working, frozen | Query and mutation builders with eager loading, write guards (no update or delete without a filter) and typed constraint errors. Scope under review ([ADR-0007](docs/adr/0007-data-access-layer-scope.md)). |
| Annotations (`@cache`, `@olap`, `@vector`, `@ml`) | Parsed only | They have no effect today |
| Other databases | Not available | Only PostgreSQL |

### What the guarantees mean

- The vault detects changes made to its files outside the tool, including an edit of `mode.json`. It does not stop someone who can edit the files and the integrity log together.
- Integrity modes do not replace database permissions. A user with direct access to the database is not restricted by them.
- The schema hash is computed over the source text, so formatting changes alter it. A canonical form is planned ([ADR-0005](docs/adr/0005-canonical-schema-digest.md)).

## Known limitations

- **No schema diff.** `migrate` creates missing tables only. Changes to existing tables (new columns, type changes) must be applied by hand.
- **No rollback and no backups.** The old `rollback_enabled`, `audit_logging`, `backup_on_migrate` and `backup_before_apply` settings are not implemented: `init` no longer writes them and a file that sets them gets a warning. `migrate --apply` says that no backup is created.
- **No transactions** in the data-access layer; each mutation is its own statement.
- **Types:** `int` maps to a 32-bit `INTEGER` and `timestamp` to `TIMESTAMP` without time zone. Both are planned to change; see the plan below.
- **Table names** are derived differently by the migration generator and the mutation layer for some multi-word or irregular names. A fix that makes them identical is planned and will come with migration notes.
- **Filters:** the `in` operator, `OR` and `time.Time` values are not supported. Pass dates as strings. A filter can follow one relation (`posts.published`); two or more fail with `UnsupportedPath`.
- **`updated_at`:** ChameleonDB does not maintain it. Set it in your code when you update a row.
- **Intel Macs and Windows on ARM** have no prebuilt binary. Build from source.
- **Install script:** it does not install on Windows (download the archive by hand). It verifies the SHA-256 checksum of the archive, but the checksum comes from the same release page as the archive, so it detects corrupted downloads, not a compromised release. Release signing is not available yet.
- **Readonly retry:** if the first `migrate --apply` cannot reach the database, the version is registered but a retry is refused while the project is in `readonly` mode. Set a password and `standard` mode to retry.
- **`migrate --dry-run` creates the vault**, so a later `migrate --apply --set-mode standard` does not switch the mode (and does not say so). Run `chameleon config auth set-password` and `chameleon config set mode=standard` instead.
- **`introspect` without a vault** skips the mode check (it prints a warning). Inside a project in `readonly` mode it is blocked.
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

Supported: Linux x86-64 and arm64, macOS on Apple Silicon. The script installs `chameleon`, the shared library, the C header and a `pkg-config` file, and verifies the SHA-256 checksum of the archive. It asks before reinstalling. Set `CHAMELEON_INSTALL_DIR`, `CHAMELEON_LIB_DIR`, `CHAMELEON_INCLUDE_DIR` or `CHAMELEON_PKGCONFIG_DIR` to change the destinations. With a `CHAMELEON_LIB_DIR` other than `/usr/local/lib`, the script reports a failed check even though the files are installed; export `LD_LIBRARY_PATH` (Linux) or `DYLD_LIBRARY_PATH` (macOS) with that directory.

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
# ChameleonDB v1.3.0-alpha
```

## Try it

```bash
mkdir test-chameleondb && cd test-chameleondb
chameleon init                        # asks how to connect; CHAMELEON.md explains the layout
chameleon validate                    # add --json for scripts and agents
chameleon migrate --dry-run           # preview the SQL
chameleon migrate --apply             # then offers a password and standard mode
chameleon status
chameleon journal schema
```

If you decline the password, the project stays in `readonly` and the next `migrate --apply` is refused until you run `chameleon config auth set-password` and `chameleon config set mode=standard`.

Use a test database. See the [documentation](https://docs.chameleondb.dev) for the schema language, queries, mutations and the security model.

## Release history

| Version | Date | Highlights |
|---|---|---|
| [v1.3.0-alpha](CHANGELOG.md) | 2026-11-23 | `--json` output and stable exit codes, coding-agent rules in `CHAMELEON.md` and `init --agents`, deterministic generated output, install script fix for custom library directories |
| [v1.2.0-alpha](CHANGELOG.md) | 2026-11-09 | `init` asks how to connect and never overwrites files, password and `standard` mode offered after the first apply, Argon2id mode password, `mode.json` tampering detected, belongs-to `Include`, `NewEngineAt`, `dashboard` removed |
| [v1.1.3-alpha](CHANGELOG.md) | 2026-11-08 | Install script fixes for macOS and Linux arm64, SHA-256 files attached to releases |
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
