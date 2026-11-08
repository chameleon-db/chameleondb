<div align="center">

![ChameleonDB](docs/logo-720x180.png)

*Schema-governed database platform with explicit integrity guarantees*
 
[![License: Apache](https://img.shields.io/badge/license-Apache%20License%202.0-blue)](https://www.apache.org/licenses/LICENSE-2.0)
[![Rust](https://img.shields.io/badge/rust-stable-orange.svg)](https://www.rust-lang.org)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8.svg)](https://golang.org)
[![Status](https://img.shields.io/badge/status-alpha-orange)](https://github.com/chameleon-db/chameleondb)
[![codecov](https://codecov.io/github/chameleon-db/chameleondb/branch/main/graph/badge.svg)](https://codecov.io/github/chameleon-db/chameleondb)


[ 🇺🇸 Documentation](docs/en/README.md) • [🇪🇸 Spanish documentation](docs/sp/README.md) • [Examples](https://github.com/chameleon-db/chameleon-examples) • [Discord](https://chameleondb.dev/discord)

</div>

---

> **Status: alpha.** Hardening work on query generation and related components is in progress and is tracked privately until fixes are released. Please do not use this version with untrusted input or for production data. See [SECURITY.md](SECURITY.md). Early adopters and feedback are welcome.

## What is ChameleonDB

ChameleonDB helps you keep a reliable relationship between the schema you declare, the changes you make to it, and the real state of your database. It treats the schema as a versioned, hash-checked artifact, records every change in a local audit trail, and lets you restrict who may change the schema through integrity modes.

It is not a database engine. Today it works with PostgreSQL only.

### Where the project is going

The near-term goal is to make schema change **observable and reviewable**: know how a database differs from what you think it has, and review every change together with its risk, without having to rewrite your existing setup first. The longer-term ambition is an adaptive platform for governing the lifecycle of data models across engines and environments.

That direction is a plan, not a description of the current code. Read the [product definition](docs/product-definition.md) for scope, horizons and what is explicitly out of scope, and the [architecture overview](docs/architecture/overview.md) for how today's implementation relates to the target.

## What works today

- **Schema language and checker.** Declare entities, fields and relations in `.cham` files; parse and type-check them.
- **Migration generation for PostgreSQL.** Preview the SQL (`--dry-run`) and apply it.
- **Schema Vault.** Each registered schema version is stored locally with a SHA-256 hash of its source text and its parent version.
- **Integrity modes.** Four modes (`readonly`, `standard`, `privileged`, `emergency`) decide whether schema changes are allowed. Moving to a higher privilege asks for a password.
- **Integrity verification and audit log.** Checks the vault before operations and records events in an append-only log.
- **Introspection.** Generate a schema from an existing PostgreSQL database.
- **Data-access layer.** A query and mutation builder for Go with field selection, eager loading and write guards (updates and deletes without a filter are blocked). This layer is a supporting component and its scope is under review ([ADR-0007](docs/adr/0007-data-access-layer-scope.md)).

### What the guarantees mean

- The vault **detects** changes made to its files outside the tool. It does **not** protect against someone who can edit both the files and their hashes.
- Integrity modes control what the **CLI** allows. They do not replace database permissions; a user with direct database access is not restricted by them.
- The hash is computed over the schema source text, so formatting changes alter it. A canonical form is planned ([ADR-0005](docs/adr/0005-canonical-schema-digest.md)).

## Quick start

### Prerequisites

- PostgreSQL. Integration tests run against PostgreSQL 16; other versions are not yet verified. `default uuid()` uses `gen_random_uuid()`, which needs PostgreSQL 13 or newer (or the `pgcrypto` extension on older versions).
- To build from source: Go 1.25.6 or newer, a stable Rust toolchain, a C compiler and `make`.

### Install (Linux and macOS)

```bash
curl -sSL https://chameleondb.dev/install | sh
```

Prebuilt binaries: Linux (amd64, arm64) and macOS on Apple Silicon (arm64). Intel Macs have no prebuilt binary; build from source. The install script verifies the SHA-256 checksum published with each release and aborts on a mismatch.

Build from source:

```bash
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make build            # builds the Rust core first, then bin/chameleon
```

### Install (Windows)

Download the compressed file from [chameleondb.dev/windows](https://www.chameleondb.dev/windows) and extract it. You get two files that must stay together:

- `chameleon.exe`
- `chameleon.dll`

Add the folder to your `PATH`. See [RELEASE.md](RELEASE.md) or the [installation guide](https://www.chameleondb.dev/docs/pages/installation.html) for details.

Check the installation:

```bash
chameleon version
```

### Your first project

```bash
mkdir my-project && cd my-project
chameleon init                      # creates .chameleon/ with the vault

cat > schema.cham <<'EOF'
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
}
EOF

chameleon check schema.cham         # parse and type-check
chameleon migrate --dry-run         # preview the SQL
chameleon migrate --apply           # register the version and apply it
chameleon journal schema            # version history
```

New vaults start in `readonly` mode, which blocks schema changes. To apply a migration, raise the mode first:

```bash
chameleon config auth set-password  # recommended: password for mode upgrades
chameleon config set mode=standard
```

Downgrading to a lower mode does not ask for a password.

## Schema language

```
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    posts: [Post] via author_id,
}

entity Post {
    id: uuid primary,
    title: string,
    content: string,
    published: bool,
    author_id: uuid,
    author: User,
}
```

## Schema Vault

The vault lives under `.chameleon/vault/`:

```
.chameleon/vault/
├── manifest.json       # current version and history
├── mode.json           # current integrity mode
├── integrity.log       # append-only audit trail
├── versions/           # one snapshot per registered version (v001.json, ...)
└── hashes/             # SHA-256 of each version (v001.hash, ...)
```

Every `migrate` verifies the vault first. If a version file no longer matches its hash, the operation stops and reports which file differs. Treat this as tamper **detection** for accidental or unexpected edits; see "What the guarantees mean" above.

## Integrity modes

| Mode | Ring | Intended use | Schema changes |
|------|------|--------------|----------------|
| `readonly` | R3 | Production (default) | Blocked |
| `standard` | R2 | Development teams | Allowed through the normal flow |
| `privileged` | R1 | DBAs | Allowed directly, logged |
| `emergency` | R0 | Incident recovery | Allowed with checks relaxed, audited |

```bash
chameleon config get mode
chameleon config set mode=standard    # upgrade: asks for the mode password
chameleon config set mode=readonly    # downgrade: no password
```

The password check is a local safeguard. Its storage is being reviewed and it is not a substitute for database roles.

## Using the Go library

The Go package `github.com/chameleon-db/chameleondb/chameleon/pkg/engine` calls the Rust core through cgo, so the native library must be available at build time.

```go
users, err := db.Query("User").
    Select("id", "name", "email").
    Filter("age", "gt", 25).
    Include("posts").
    Execute(ctx)

result, err := db.Update("User").
    Filter("id", "eq", userID).
    Set("name", "New Name").
    Execute(ctx)
// An update or delete without a filter is rejected with a SafetyError.
```

To use the package from another module you must build and install the core library yourself:

```bash
cd chameleon-core
cargo build --release
sudo cp target/release/libchameleon.so /usr/local/lib/   # Linux
sudo cp include/chameleon.h /usr/local/include/
sudo ldconfig
```

Then `go get github.com/chameleon-db/chameleondb/chameleon@latest`. If the library lives elsewhere, set `CGO_LDFLAGS` and `CGO_CFLAGS`. This installation path is a known source of friction and is being re-evaluated ([ADR-0011](docs/adr/0011-language-runtime-and-distribution.md)).

## Command overview

| Area | Commands |
|---|---|
| Project | `init`, `status`, `version`, `uninstall` |
| Schema | `check`, `validate`, `introspect <database-url>` |
| Migrations | `migrate` (`--dry-run`, `--apply`), `verify` |
| History and audit | `journal schema`, `journal migrations`, `journal last`, `journal errors` |
| Configuration | `config get`, `config set`, `config auth set-password` |
| Development tools | `query`, `playground-sql`, `dashboard` |

See the [CLI reference](docs/cli-reference.md) for the current state and proposed changes of each command.

## Documentation

| Document | Contents |
|---|---|
| [Documentation index](docs/en/README.md) | User documentation |
| [Product definition](docs/product-definition.md) | Scope, horizons, principles |
| [Vision and foundations](docs/vision-and-foundations.md) | Long-term concepts |
| [Architecture overview](docs/architecture/overview.md) | Current architecture and target |
| [ADR index](docs/adr/README.md) | Architecture decisions |
| [Planning](docs/planning/README.md) | How work is organised |

The roadmap is kept as issues and a project board, ordered by horizon and priority, without calendar dates.

## Contributing

Contributions are welcome: code, tests, documentation, bug reports and design feedback.

- Start with [CONTRIBUTING.md](CONTRIBUTING.md) and [DEVELOPERS.md](DEVELOPERS.md).
- Look for issues labelled `good first issue` and `help wanted`.
- Coding agents: see [AGENTS.md](AGENTS.md).
- Report security problems privately, as described in [SECURITY.md](SECURITY.md).

Areas where help is especially useful: Rust (core, canonical model), Go (runtime, vault), tests (adversarial and integration), documentation, and feedback from people who manage PostgreSQL schema changes in real projects.

## License

ChameleonDB is licensed under the **Apache License 2.0**.

## Acknowledgments

Ideas that influenced the project: Unix protection rings, Git's immutable hash-verified history, Prisma's schema-first approach and Datomic's immutability.

---

<div align="center">

[Website](https://chameleondb.dev) • [Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples)  
[Discord](https://chameleondb.dev/discord) • [X/Twitter](https://x.com/ChameleonDB)

</div>