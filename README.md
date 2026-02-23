<div align="center">

![ChameleonDB](docs/logo-200x150.png)

*Schema-governed database platform with explicit integrity guarantees*

[![License: Apache](https://img.shields.io/badge/license-Apache%20License%202.0-blue)](https://www.apache.org/licenses/LICENSE-2.0)
[![Rust Version](https://img.shields.io/badge/rust-1.75%2B-orange.svg)](https://www.rust-lang.org)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://golang.org)
[![Status](https://img.shields.io/badge/status-v1.0--alpha-green)](https://github.com/chameleon-db/chameleondb)
[![codecov](https://codecov.io/gh/chameleon-db/chameleondb/graph/badge.svg)](https://codecov.io/gh/chameleon-db/chameleondb)

[ 🇺🇸 Documentation](https://chameleondb.dev/docs) • [🇪🇸 Spanish documentation](docs/sp/README.md) • [Examples](https://github.com/chameleon-db/chameleon-examples) • [Discord](https://chameleondb.dev/discord)

</div>

---

## ⚡ v1.0-alpha Released!

ChameleonDB **v1.0-alpha** is now available with **Schema Vault** and **Integrity Modes** — explicit schema governance built into the database layer.

**What's new:**
- 🔐 **Schema Vault**: Versioned, hash-verified schema storage
- 🛡️ **Integrity Modes**: Ring-based governance (readonly/standard/privileged/emergency)
- ✅ **IdentityMap**: Automatic object deduplication in memory
- 📊 **Complete audit trail**: Append-only integrity log
- 🚀 **Zero-config**: Auto-initialization on first migrate

**Early adopters welcome** — your feedback shapes the product.

---

## Overview

ChameleonDB is a **schema-governed database platform** that treats schemas as first-class, immutable artifacts with explicit integrity guarantees.

Unlike traditional databases that treat schema evolution as an auxiliary concern, ChameleonDB **governs schemas at runtime** through versioning, cryptographic integrity, and explicit operational modes.

### The Problem

Modern database systems enforce strong guarantees over data but treat schema evolution informally:

- **Schema drift** happens silently over time
- **Migration failures** leave databases in unknown states
- **Authority** for schema changes is implicit, not enforced
- **Audit trails** are external, incomplete, or missing
- **Rollback** is manual and error-prone

### The ChameleonDB Solution

**1. Define your schema** (versioned and hash-verified)
```go
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

**2. Initialize with zero config** (auto-creates Schema Vault)
```bash
chameleon init                  # Creates .chameleon/vault/
chameleon migrate --apply       # Registers v001, applies migration
```

**3. Schema Vault tracks everything**
```
.chameleon/vault/
├── manifest.json       # Current version + history
├── integrity.log       # Append-only audit trail
├── versions/
│   ├── v001.json      # Immutable snapshot
│   └── v002.json
└── hashes/
    ├── v001.hash      # SHA256 verification
    └── v002.hash
```

**4. Integrity enforcement** (automatic verification)
```bash
# Every migrate checks integrity
$ chameleon migrate

🔍 Verifying schema integrity...
   ✓ Current: v001 (3f2a8b9c...)
   ✓ No tampering detected

# If someone modifies vault files
❌ INTEGRITY VIOLATION DETECTED
   • v001.json: hash mismatch
   🚨 Schema vault has been modified!
   ❌ Migration aborted for safety
```

**5. Integrity Modes** (Unix-style protection rings)
```bash
# Default: readonly (schema locked)
$ chameleon migrate --apply
❌ readonly mode: schema modifications blocked

# Upgrade to standard (controlled changes)
$ chameleon config set mode=standard
🔐 Enter mode password: ****
✅ Mode upgraded to standard

# Now migrations are allowed
$ chameleon migrate --apply
✅ Migration applied successfully
✅ Schema v002 locked in vault
```

**What you get:**

✅ **Immutable schema versions** — Tamper-proof with SHA256 hashing  
✅ **Integrity verification** — Automatic checks before every operation  
✅ **Explicit governance** — Ring-based modes (readonly/standard/privileged/emergency)  
✅ **Complete audit trail** — Append-only log, never deleted  
✅ **Zero-config vault** — Auto-initializes on first migrate  
✅ **Password-protected upgrades** — Mode escalation requires auth  
✅ **Migration recovery** — Retry failed migrations automatically  

---

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 14+

### Installation (Linux and macOS)
```bash
# Install ChameleonDB CLI
curl -sSL https://chameleondb.dev/install | sh

# Or build from source
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make build
```

### Windows
Download the compressed `.gz` file from [https://www.chameleondb.dev/windows](https://www.chameleondb.dev/windows)  
Extract the `.gz` file (WinRAR, WinZip, 7zip, or similar). You will obtain two files:

* `chameleon.exe`
* `chameleon.dll`

> ⚠️ Both files must be kept together.

Add to Path to use in any terminal.  
(see [RELEASE](RELEASE.md) or [INSTALLATION GUIDE](https://www.chameleondb.dev/docs/pages/installation.html) for more details)

### Verify installation

Open a new terminal (CMD or PowerShell) and run:
```bash
chameleon --version
# Output: chameleon v1.0-alpha
```

### Using Chameleon as a Go SDK (from another repository)

If you import `github.com/chameleon-db/chameleondb/chameleon/pkg/engine` in another Go project,
the package now links `libchameleon` automatically via cgo (`-lchameleon`).

Requirements:

- `libchameleon.so` installed in `/usr/local/lib` (Linux)
- `chameleon.h` available in system include paths (recommended: `/usr/local/include`)

Build/install from source:

```bash
# from this monorepo root
cd chameleon-core
cargo build --release

sudo cp target/release/libchameleon.so /usr/local/lib/
sudo cp include/chameleon.h /usr/local/include/
sudo ldconfig
```

Then in your external Go repo:

```bash
go get github.com/chameleon-db/chameleondb/chameleon@latest
go build ./...
```

If you install the library in a non-standard location, set custom flags:

```bash
CGO_LDFLAGS="-L/path/to/lib -Wl,-rpath,/path/to/lib -lchameleon" \
CGO_CFLAGS="-I/path/to/include" \
go build ./...
```

Recommended packaging strategy (industry standard):

- Ship native library + public header together (`libchameleon.*` + `chameleon.h`)
- Keep C ABI stable across patch/minor releases
- Publish `pkg-config` metadata (`chameleon.pc`) for language/toolchain interoperability
- Use ABI-versioned shared libraries (e.g. `libchameleon.so.1 -> libchameleon.so`) for safe upgrades
- Use semantic versioning and avoid breaking C symbols without a major version bump

This makes Go, C/C++, Python, Node, and other future bindings easier to maintain.


### Your First Project
```bash
# Initialize project (creates vault)
cd my-project
chameleon init

# Create schema.cham
cat > schema.cham <<EOF
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
}
EOF

# Apply migration (auto-registers v001)
chameleon migrate --apply

# View history
chameleon journal schema
```

### Integrity Mode Management

```bash
# Check current mode
$ chameleon status

Schema:
  Current version:  v001
  Status:          ✓ Up to date

Vault:
  Versions:        1 registered
  Integrity:       ✓ OK
  Mode:            🔒 readonly (locked)

# Set password for mode upgrades (recommended)
$ chameleon config auth set-password
Enter new password: ********
✅ Mode password configured

# Upgrade to allow schema changes
$ chameleon config set mode=standard
🔐 Enter mode password: ********
✅ Mode upgraded to standard

# Downgrade (no password required)
$ chameleon config set mode=readonly
✅ Mode downgraded to readonly
```

---

## Core Features

### 🔐 Schema Vault (v1.0)

**Versioned, immutable schema storage with cryptographic integrity:**

```bash
# Every migration creates a new version
$ chameleon migrate --apply

📦 Registering new schema version...
   ✓ Registered as v002 (hash: 7d4e1c2a...)
   ✓ Parent: v001

✅ Migration applied successfully
✅ Schema v002 locked in vault

# View version history
$ chameleon journal schema

📖 Schema Version History

v002 (current) ✓
├─ Hash: 7d4e1c2a...
├─ Date: 2026-02-20 15:45:00
├─ Author: dperalta
├─ Changes: Added age field to User
└─ Parent: v001

v001
├─ Hash: 3f2a8b9c...
├─ Date: 2026-02-20 10:30:00
├─ Author: dperalta
├─ Changes: Initial schema
└─ Parent: none
```

**Features:**
- ✅ Immutable snapshots (once registered, never modified)
- ✅ SHA256 hash verification (tamper detection)
- ✅ Lineage tracking (parent versions)
- ✅ Automatic registration on migrate
- ✅ Complete audit trail (integrity.log)

---

### 🛡️ Integrity Modes (v1.0)

**Unix-style protection rings for schema governance:**

| Mode | Ring | Use Case | Schema Changes |
|------|------|----------|----------------|
| **readonly** | R3 | Production (default) | ❌ Blocked |
| **standard** | R2 | Development teams | ✅ Controlled |
| **privileged** | R1 | DBAs | ✅ Direct (logged) |
| **emergency** | R0 | Incident recovery | ✅ No checks (audited) |

**How it works:**

```bash
# Default mode: readonly (schema locked)
$ chameleon init
✅ Vault initialized in readonly mode
💡 Tip: Set mode password with 'chameleon config auth set-password'

# Try to migrate in readonly mode
$ chameleon migrate --apply
❌ readonly mode: schema modifications blocked

# Upgrade mode (requires password)
$ chameleon config set mode=standard
🔐 Enter mode password: ****
✅ Mode upgraded to standard

# Now migrations work
$ chameleon migrate --apply
✅ Schema v002 registered and applied
```

**Password protection:**
```bash
# Set password for mode upgrades
$ chameleon config auth set-password
Enter new password: ********
✅ Mode password configured

# Upgrades require password
$ chameleon config set mode=privileged
🔐 Enter mode password: ****

# Downgrades don't require password
$ chameleon config set mode=readonly
✅ Mode downgraded (no password needed)
```

---

### 🎯 Query System (v1.0)

**Graph-oriented, type-safe queries with field projection:**

```go
// Query only the fields you need
users := db.Query("User").
    Select("id", "name", "email").  // Partial selection
    Filter("age", "gt", 25).
    Include("posts").                // Eager load (no N+1)
    Execute(ctx)

// Debug mode (see generated SQL)
users := db.Query("User").
    Select("id", "name").
    Filter("email", "like", "ana").
    Debug().
    Execute(ctx)

// Output:
// [SQL] Query User
// SELECT id, name FROM users WHERE email LIKE '%ana%'
// [TRACE] Query on User: 2.3ms, 3 rows
```

---

### 🛡️ Mutation Safety (v1.0)

**Built-in safety guards with clear error messages:**

```go
// Insert with validation
result, err := db.Insert("User").
    Set("email", "ana@mail.com").
    Set("name", "Ana Garcia").
    Execute(ctx)

// If email already exists:
// ❌ UniqueConstraintError: Field 'email' must be unique
//    Value: ana@mail.com already exists
//    Suggestion: Use a different value or update the existing record

// Update requires WHERE clause
db.Update("User").
    Filter("id", "eq", userID).
    Set("name", "New Name").
    Execute(ctx)

// Trying to update without filter:
// ❌ SafetyError: UPDATE requires a WHERE clause
//    Suggestion: Use Filter() or ForceUpdateAll()
```

---

### 🔍 IdentityMap (v1.0)

**Automatic object deduplication in memory:**

```go
// Without IdentityMap (wasteful)
// If User has 100 posts, User object is duplicated 100 times in memory

// With IdentityMap (efficient)
result := db.Query("User").
    Include("posts").
    Execute(ctx)

// User object appears only once
// All 100 posts reference the same User instance
// Memory savings: ~99% for large result sets
```

---

## CLI Commands

### Migration Management

```bash
# View status
chameleon status

# Check for changes
chameleon migrate

# Preview SQL
chameleon migrate --dry-run

# Apply migration
chameleon migrate --apply

# Verify integrity
chameleon verify
```

### Schema Vault

```bash
# View version history
chameleon journal schema

# View specific version
chameleon journal schema v002

# View integrity log
cat .chameleon/vault/integrity.log
```

### Mode Management

```bash
# View current mode
chameleon config get mode

# Set mode password
chameleon config auth set-password

# Upgrade mode (requires password)
chameleon config set mode=standard

# Downgrade mode (no password)
chameleon config set mode=readonly
```

---

## Features Status

### ✅ Available Now (v1.0-alpha)

**Schema Governance:**
- [x] Schema Vault (versioned, hash-verified)
- [x] Integrity Modes (readonly/standard/privileged/emergency)
- [x] Password-protected mode upgrades
- [x] Automatic integrity verification
- [x] Append-only audit trail
- [x] Migration recovery (retry failed migrations)

**Query System:**
- [x] Schema parser and type checker
- [x] Query builder with filters
- [x] Field projection (`.Select()`)
- [x] Eager loading (`.Include()`)
- [x] Nested includes
- [x] IdentityMap (object deduplication)
- [x] Debug mode (`.Debug()`)

**Mutation System:**
- [x] Insert/Update/Delete builders
- [x] Safety guards (WHERE clause required)
- [x] Three-stage validation pipeline
- [x] Comprehensive error mapping

**Tooling:**
- [x] CLI tools (init, migrate, verify, status)
- [x] Rich error messages with suggestions
- [x] PostgreSQL migration generator
- [x] Database introspection (Only Postgres for now)
- [x] 300+ tests (unit + integration)

### 🚧 Coming Soon (v1.1 - March 2026)

- [ ] Schema Vault rollback
- [ ] Complete Paraniod modes (proposal/approval workflow)
- [ ] Transaction support
- [ ] Batch operations

### 🔮 Planned (v1.2+ - Q2 2026)

- [ ] Additional backends (MySQL, DuckDB)
- [ ] Code generation (type-safe DTOs)
- [ ] Multi-language support (TypeScript, Python)
- [ ] Query optimizer layer

### 🌟 Future (v1.5+ - 2027)

- [ ] ML-based query optimization
- [ ] Visual schema editor
- [ ] Distributed vault (multi-node)

---

## Architecture

### Schema Vault Structure

```
.chameleon/
├── vault/
│   ├── manifest.json           # Current version + history
│   ├── integrity.log           # Append-only audit trail
│   ├── versions/               # Immutable snapshots
│   │   ├── v001.json
│   │   ├── v002.json
│   │   └── v003.json
│   └── hashes/                 # SHA256 verification
│       ├── v001.hash
│       ├── v002.hash
│       └── v003.hash
├── state/                      # Migration tracking
│   └── migrations.json
└── journal/                    # Operation log
    └── 2026-02-20.log
```

### Execution Flow

```
1. User: chameleon migrate --apply
         ↓
2. Vault: Check integrity (verify all hashes)
         ↓
3. Mode:  Check if changes allowed (readonly blocks)
         ↓
4. Detect: Compare schema hash with current version
         ↓
5. Register: Create v002 snapshot + hash
         ↓
6. Execute: Apply SQL migration
         ↓
7. Log: Record in integrity.log + journal
```

---

## Roadmap

### Q1 2026 ✅ v1.0-alpha (Current)

✅ Schema Vault (versioned, hash-verified)  
✅ Integrity Modes (ring-based governance)  
✅ IdentityMap (object deduplication)  
✅ Complete audit trail  
✅ Zero-config initialization  
✅ Password-protected modes  

**Goal:** Production-ready core with explicit governance

### Q2 2026 - v1.1

- Schema Vault rollback
- Eagle mode (proposal workflow)
- Database introspection
- Transaction support
- Performance benchmarks

**Goal:** Feature parity with major ORMs + governance

### Q3-Q4 2026 - v1.2 Stable

- Additional backends (MySQL, DuckDB)
- Code generation
- Multi-language bindings
- Migration tools

**Goal:** Enterprise-ready

### 2027+ - v2.0

- ML-based query optimization
- Visual schema editor
- Distributed vault
- Advanced features

---

## Why ChameleonDB?

### vs Traditional Databases

| Traditional DB | ChameleonDB |
|----------------|-------------|
| ❌ Schema drift over time | ✅ Immutable, versioned schemas |
| ❌ Informal governance | ✅ Explicit Integrity Modes |
| ❌ No tamper detection | ✅ SHA256 hash verification |
| ❌ External audit logs | ✅ Built-in integrity log |
| ❌ Manual rollback | ✅ Version-based recovery |

### vs Migration Tools (Flyway, Liquibase)

| Migration Tools | ChameleonDB |
|-----------------|-------------|
| ❌ External to database | ✅ Built into platform |
| ❌ No schema identity | ✅ Cryptographic hashing |
| ❌ Limited governance | ✅ Ring-based modes |
| ❌ Rollback is manual | ✅ Version snapshots |
| ✅ Battle-tested | 🚧 New approach |

### Key Differentiators

1. **Schema as First-Class Artifact**: Versioned, immutable, hash-verified
2. **Runtime Governance**: Integrity Modes enforced by the system
3. **Zero-Config Vault**: Auto-initializes, works out of the box
4. **Complete Audit Trail**: Append-only, never deleted
5. **Explicit Authority**: Mode upgrades require password

---

## Contributing

We welcome contributions! ChameleonDB is actively developed.

### Development Setup
```bash
# Build Rust core
cd chameleon-core
cargo build --release
cargo test

# Build Go runtime
cd ../chameleon
make build
make test

# Run integration tests
make test-integration
```

### Areas We Need Help

- 🦀 **Rust**: Query optimizer, additional backends
- 🐹 **Go**: Runtime improvements, Schema Vault features
- 📚 **Documentation**: Tutorials, migration guides
- 🧪 **Testing**: Integration tests, benchmarks
- 🎨 **Tooling**: VSCode extension improvements

---

## License

ChameleonDB is licensed under the **Apache License 2.0**.

---

## Acknowledgments

Inspired by:

- **Unix/Linux** — Protection rings and explicit governance
- **Git** — Immutable, hash-verified history
- **Prisma** — Schema-first approach
- **Datomic** — Immutability as a design principle

---

<div align="center">

**Built with ❤️ for developers who care about schema integrity**

[Website](https://chameleondb.dev) • [Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples)  
[Discord](https://chameleondb.dev/discord) • [X/Twitter](https://x.com/ChameleonDB)
</div>