# ChameleonDB releases

**Current release: [v1.2.0-alpha](https://github.com/chameleon-db/chameleondb/releases/tag/v1.2.0-alpha)** (2026-11-09)

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

## In this release: v1.2.0-alpha

The first run is easier and the mode password is stored properly. One command was removed, so read "Behavior changes" before upgrading.

### First run

- `chameleon init` asks how to connect to PostgreSQL when a terminal is attached: keep `${DATABASE_URL}`, detect a connection (environment, `.env`, `.env.example`, compose files), write a `docker-compose.yml` with PostgreSQL 16 and start it, or type a connection string. A connection is tested before it is used. Every question has a flag (`--db-url`, `--detect-db`, `--compose`, `--no-example`), and nothing is asked without a terminal, with `CI` set or with `--no-input`.
- `init` never overwrites a file it did not create. The guide is `CHAMELEON.md` (your `README.md` is never touched), `.env.example` only gets an appended `# chameleon env` block, and the example schema is created only if `schemas/` has no `.cham` file.
- After the `migrate --apply` that creates the vault, a terminal session is offered a password and `standard` mode. Scripts can use `--set-mode standard` with `CHAMELEON_MODE_PASSWORD`. If the project stays in `readonly`, the apply output says so and lists the two commands that unlock it.
- The example schema declares `id: uuid primary default uuid()`.

### Security

- The mode password is stored with Argon2id. Existing passwords keep working and are upgraded when first verified. Changing a password requires the current one.
- `verify`, `migrate` and the engine fail when `mode.json` was edited outside the CLI. This detects the edit; it does not prevent it. See [ADR-0013](docs/adr/0013-mode-and-password-threat-model.md) for what modes and the password do and do not protect.
- The `chameleon dashboard` server was removed: it accepted record writes with no session token and only an origin filter. It will return only with its own design.

### Runtime and core

- `Include` on a belongs-to relation (`Post.author`) works, and a filter through one (`author.name`) joins on the local key. `validate` reports a belongs-to relation whose key cannot be resolved.
- `engine.NewEngineAt(projectDir)` and `CHAMELEON_PROJECT_DIR` open a project from any working directory. Errors name the directory that was searched.
- `init` writes the four config flags that have no behavior (`rollback_enabled`, `audit_logging`, `backup_on_migrate`, `backup_before_apply`) as `false`, with a comment, and warns when a file sets one to `true` ([ADR-0012](docs/adr/0012-config-flags-roadmap.md)). `migrate --apply` always says that no backup is created.
- A `Makefile` at the repository root; `make tag` runs `scripts/tag-release.sh`.

### Behavior changes

Read these before upgrading from v1.1.3:

| Change | What to do |
|---|---|
| `chameleon dashboard` and `dashboard serve` no longer exist. | Use the CLI. If you scripted the dashboard API, it has no replacement yet. |
| `init` writes `CHAMELEON.md` instead of `README.md`. | Nothing for existing projects. |
| Setting a mode password when one exists requires the current one (`CHAMELEON_MODE_PASSWORD`, and `CHAMELEON_MODE_PASSWORD_NEW` for the new one when not interactive). | Update scripts that rotate the password. |
| A `mode.json` that disagrees with the last mode change in the integrity log makes `verify`, `migrate` and the engine fail. | Change the mode with `chameleon config set mode=...`, not by editing the file. |
| Configs that set `rollback_enabled`, `audit_logging`, `backup_on_migrate` or `backup_before_apply` to `true` print a warning on every command. | Set them to `false`; they had no effect. |
| New dependency `golang.org/x/crypto` (Argon2). | Only when building from source: run `go mod tidy`. |

No migration is needed. Existing tables are never altered by ChameleonDB.

## Previous release: v1.1.3-alpha

Changed the install path only. The install script works on macOS (Apple Silicon) and Linux arm64; every archive has a SHA-256 file that the script verifies; the macOS archive is `chameleon-macos-arm64` (it was called `universal`). Changes of v1.1.2 and earlier are in the [CHANGELOG](CHANGELOG.md).

## What works today

| Area | State | Notes |
|---|---|---|
| Schema language (`.cham`) | Working | Entities, fields, relations, defaults (`uuid()`, `now()`, literals), `unique`, `nullable`, `vector(N)`, arrays |
| Type checker (`validate`, `check`) | Working | Rejects invalid primary keys, `vector(0)`, mismatched has-many keys and cycles between entities |
| Migrations (`migrate`) | Partial | Create-only: `CREATE TABLE IF NOT EXISTS` in a deterministic order. Existing tables are left as they are. |
| Schema Vault | Working | Versions with a SHA-256 hash of the schema source and the parent version, stored locally. Detects edits made outside the tool. |
| Integrity modes | Working | `readonly`, `standard`, `privileged`, `emergency`. They guard against accidental changes through the CLI; they are not access control and do not set database permissions. The password is stored with Argon2id. |
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
# ChameleonDB v1.2.0-alpha
```

## Try it

```bash
mkdir test-chameleondb && cd test-chameleondb
chameleon init                        # asks how to connect; CHAMELEON.md explains the layout
chameleon validate
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
