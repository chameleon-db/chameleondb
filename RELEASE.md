# ChameleonDB v1.0-alpha Release

**Release Date:** February 25, 2026  
**Status:** Alpha (early adopter preview)

---

## 🎉 Overview

ChameleonDB v1.0-alpha introduces **Schema Vault** and **Integrity Modes** — bringing explicit schema governance to the database layer.

This release transforms ChameleonDB from a type-safe query builder into a **schema-governed database platform** with built-in integrity guarantees.

---

## 🚀 What's New

### Schema Vault

**Versioned, immutable schema storage with cryptographic integrity:**

- ✅ Automatic version registration on every migrate
- ✅ SHA256 hash verification (tamper detection)
- ✅ Immutable snapshots (once registered, never modified)
- ✅ Lineage tracking (parent versions)
- ✅ Append-only integrity log
- ✅ Zero-config (auto-initializes on first migrate)

**Usage:**
```bash
# Initialize project (creates vault)
chameleon init

# Apply migration (auto-registers v001)
chameleon migrate --apply

# View version history
chameleon journal schema

# Verify integrity
chameleon verify
```

---

### Integrity Modes

**Unix-style protection rings for schema governance:**

| Mode | Description | Schema Changes |
|------|-------------|----------------|
| **readonly** (R3) | Production default | ❌ Blocked |
| **standard** (R2) | Development teams | ✅ Controlled |
| **privileged** (R1) | DBAs | ✅ Direct (logged) |
| **emergency** (R0) | Incident recovery | ✅ No checks (audited) |

**Usage:**
```bash
# Set password for mode upgrades
chameleon config auth set-password

# Upgrade mode (requires password)
chameleon config set mode=standard

# Check current mode
chameleon status
```

**Features:**
- ✅ Password-protected mode upgrades
- ✅ Automatic enforcement on migrate
- ✅ Downgrades don't require password
- ✅ Complete audit trail of mode changes

---

### IdentityMap

**Automatic object deduplication in memory:**

- ✅ Deduplicates objects in Include queries
- ✅ Memory savings: ~99% for large result sets
- ✅ Session-based (per-query isolation)
- ✅ Zero configuration

**Example:**
```go
// User with 100 posts
result := db.Query("User").
    Include("posts").
    Execute(ctx)

// Without IdentityMap: User object duplicated 100 times
// With IdentityMap: User object appears only once
```

---

### Error Mapping

**PostgreSQL errors mapped to ChameleonDB types with suggestions:**

```go
// Unique constraint violation
❌ UniqueConstraintError: Field 'email' must be unique
   Value: test@mail.com already exists
   Suggestion: Use a different value or update the existing record

// Foreign key violation
❌ ForeignKeyError: Invalid reference
   Field: author_id
   Referenced: users(id=uuid-999)
   Suggestion: Ensure the referenced user exists

// NOT NULL violation
❌ NotNullError: Field 'email' cannot be null
   Suggestion: Provide a value for email (this field is required)
```

**Supported errors:**
- `UniqueConstraintError` (23505)
- `ForeignKeyError` (23503)
- `NotNullError` (23502)
- `ConstraintError` (23514)
- `UnknownFieldError` (42703)

---

### CLI Improvements

**New commands:**

```bash
# Schema Vault
chameleon journal schema           # View version history
chameleon journal schema v002      # View specific version
chameleon verify                   # Verify integrity
chameleon status                   # Overview

# Mode Management
chameleon config get mode          # View current mode
chameleon config set mode=MODE     # Change mode
chameleon config auth set-password # Set mode password
```

**Improved migrate:**
- ✅ Auto-initializes vault on first run
- ✅ Integrity verification before every migrate
- ✅ Detects schema changes via hash
- ✅ Registers new version before applying
- ✅ Retry failed migrations automatically
- ✅ Mode enforcement (readonly blocks changes)

---

## 📊 Stats

**Code:**
- Rust: 94 tests ✅
- Go: 80 tests ✅
- Integration: 30 tests ✅
- **Total: 204 tests passing**

**Features:**
- Schema Vault
- Integrity Modes (4 modes)
- IdentityMap
- Error mapping (6 error types)
- 13 CLI commands

**Lines of code:**
- Rust core: ~8,500 lines
- Go runtime: ~6,200 lines
- Total: ~14,700 lines

---

## 🔧 Breaking Changes

### ⚠️ Mode names changed

**Old (pre-v1.0):**
- `dragon` → `readonly`
- `eagle` → `standard`
- `lion` → `privileged`
- `phoenix` → `emergency`

**Migration:**
```bash
# Update .chameleon/vault/manifest.json
{
  "integrity_mode": "readonly"  // was "dragon"
}
```

### ⚠️ First migrate creates vault

**Before:** Vault had to be manually initialized  
**Now:** Auto-creates on first `chameleon migrate`

**No action required** - existing projects will auto-upgrade.

---

## 🐛 Bug Fixes

- Fixed: Dragon mode not enforcing on migrate
- Fixed: rows.Next() false positive in mutations
- Fixed: Error silencing in Execute methods
- Fixed: UUID primary key handling
- Fixed: Table naming (PascalCase → snake_case)

---

## 📚 Documentation

**Updated:**
- README with Schema Vault section
- Quick Start guide
- CLI reference
- Integrity Modes guide

**New:**
- RELEASE notes (this file)
- Migration guide (v0.x → v1.0)
- Schema Vault architecture docs

---

## 🚧 Known Limitations

**v1.0-alpha limitations:**

- **No rollback** - Schema Vault tracks versions but rollback not implemented yet (coming in v1.1)
- **Eagle mode not fully implemented** - Proposal/approval workflow coming in v1.1
- **Single backend** - Only PostgreSQL supported (MySQL/DuckDB coming in v1.2)
- **No distributed vault** - Multi-node support coming in v1.5

**These are intentional scope decisions for alpha release.**

---

## 🎯 What's Next

### v1.1 (March 2026)

- Schema Vault rollback
- Eagle mode (proposal/approval workflow)
- Database introspection
- Transaction support
- Performance benchmarks

### v1.2 (Q2 2026)

- Additional backends (MySQL, DuckDB)
- Code generation
- Multi-language bindings

---

## 📦 Installation

### From Release

```bash
# Auto-install (macOS | Linux)
curl -sSL https://chameleondb.dev/install | sh

# Or manual installation (Linux example)
curl -LO https://github.com/chameleon-db/chameleondb/releases/download/v1.0-alpha/chameleon-linux-amd64
chmod +x chameleon-linux-amd64
sudo mv chameleon-linux-amd64 /usr/local/bin/chameleon
```

### From Source

```bash
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make build
sudo make install
```

### Verify Installation

```bash
chameleon --version
# Output: chameleon v1.0-alpha
```

## Windows Install
ChameleonDB is distributed as a portable binary. It does not require an installer.

### 1. Download

Download the compressed `.gz` file from [here](https://www.chameleondb.dev/windows)

### 2. Extract

Extract the `.gz` file (WinRAR, WinZip, 7zip, or similar). You will obtain two files:

* `chameleon.exe`
* `chameleon.dll`

> ⚠️ Both files must be kept together.

### 3. Move to a safe folder

Create a folder for ChameleonDB, for example:

```bash
c:\chameleon\
```

Move **both files** (`.exe` and `.dll`) into that folder.

### 4. Add to PATH

To be able to run `chameleon` from any console:

1. Open **System Settings** -> **Environment Variables**
2. Under **System variables**, select `Path` -> **Edit**
3. Add the path:

```bash
c:\chameleon\
# Or the path you chose
```

4. Accept all changes

> 📌 Close and reopen the terminal for the change to take effect.

### 5. Verify installation

Open a new terminal (CMD or PowerShell) and run:
```bash
chameleon --version
# Output: chameleon v1.0-alpha
```

---

## 🧪 Testing

We encourage early adopters to test v1.0-alpha in **non-production** environments.

**To try it:**

```bash
# Create test project
mkdir test-chameleondb
cd test-chameleondb

# Initialize
chameleon init

# Create simple schema
cat > schema.cham <<EOF
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
}
EOF

# Apply migration
chameleon migrate --apply

# View vault
chameleon journal schema
chameleon status
chameleon verify
```

**Please report:**
- Bugs: [GitHub Issues](https://github.com/chameleon-db/chameleondb/issues)
- Feature requests: [Discussions](https://github.com/chameleon-db/chameleondb/discussions)
- Feedback: [Discord](https://chameleondb.dev/discord)

---

## 💬 Community

**Get involved:**

- **GitHub Discussions**: [Share ideas](https://github.com/chameleon-db/chameleondb/discussions)
- **Discord**: [Join our server](https://chameleondb.dev/discord)
- **Twitter**: [@chameleondb](https://twitter.com/chameleondb)

**Early adopter program:**

If you're testing v1.0-alpha in a real project, let us know! We're offering:
- Direct support in Discord
- Priority for feature requests
- Recognition in v1.0 stable release notes

---

## 🙏 Acknowledgments

**Contributors to v1.0-alpha:**

- Daniel Peralta ([@dperalta](https://github.com/dperalta)) - Core architecture & implementation
- Community feedback from early testers

**Special thanks:**

- Early adopters who tested pre-alpha builds
- Contributors who reported bugs and suggested improvements

---

## 📄 License

ChameleonDB is licensed under the **Apache License 2.0**.

---

<div align="center">

**Ready to try v1.0-alpha?**

[Download](https://github.com/chameleon-db/chameleondb/releases/tag/v1.0-alpha) • [Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples)

</div>