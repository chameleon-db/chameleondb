# ChameleonDB Architecture

![Chameleon logo](../logo-200x150.png)

## System Overview

![System Overview diagram](../diagrams/system_overview.png)

ChameleonDB is a **schema-governed database platform** with explicit integrity guarantees. Unlike traditional databases that treat schema evolution informally, ChameleonDB governs schemas at runtime through versioning, cryptographic integrity, and explicit operational modes.

---

## Architecture Layers

```
┌─────────────────────────────────────────────┐
│  Application Layer                          │
│  - Query Builder API                        │
│  - Mutations (Insert/Update/Delete)         │
│  - Debug Mode                               │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Security Layer                             │
│  - Integrity Modes (readonly/standard/...)  │
│  - Password-protected upgrades              │
│  - Mode enforcement                         │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Schema Vault                               │
│  - Versioned schemas (v001, v002, ...)      │
│  - SHA256 integrity verification           │
│  - Immutable snapshots                      │
│  - Append-only audit log                    │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Runtime Layer                              │
│  - Query Executor                           │
│  - Connection Pool (pgx)                    │
│  - Error Mapping                            │
│  - IdentityMap (deduplication)              │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  Database Backend                           │
│  - PostgreSQL (v1.0)                        │
│  - MySQL (planned v1.2)                     │
│  - DuckDB (planned v1.5)                    │
└─────────────────────────────────────────────┘
```

---

## Core Components

### 1. Rust Core (`chameleon-core`)

**Parser**  
Transforms `.cham` source into an AST using LALRPOP grammar. Handles entities, fields, relations, type modifiers, default values, and backend annotations.

**Type Checker**  
Validates the AST before runtime. Organized in three modules:
- `relations.rs` - Entity references and foreign key consistency
- `constraints.rs` - Primary keys and annotation rules  
- `mod.rs` - Pipeline orchestration and error reporting

**Query Optimizer** (Planned v1.5)  
Rule-based optimization for query execution plans. Deterministic, no ML in v1.x.

**Status:** ✅ Complete

---

### 2. Schema Vault

**Purpose:** Immutable, versioned schema storage with integrity guarantees.

**Structure:**
```
.chameleon/vault/
├── manifest.json       # Current version + history
├── integrity.log       # Append-only audit trail
├── versions/
│   ├── v001.json      # Immutable schema snapshot
│   └── v002.json
└── hashes/
    ├── v001.hash      # SHA256 verification
    └── v002.hash
```

**Features:**
- ✅ Automatic version registration on every migration
- ✅ SHA256 hash verification before every operation
- ✅ Tamper detection (if hash mismatch → abort)
- ✅ Lineage tracking (parent versions)
- ✅ Complete audit trail (who, what, when)

**Workflow:**
```
1. User modifies schema.cham
2. chameleon migrate detects changes
3. Compute SHA256 hash
4. Register as v002 (parent: v001)
5. Save snapshot to vault/versions/v002.json
6. Save hash to vault/hashes/v002.hash
7. Update manifest.json
8. Log to integrity.log
9. Apply migration to database
```

**Status:** ✅ Complete (v1.0)

---

### 3. Integrity Modes

**Purpose:** Ring-based access control for schema governance.

| Mode | Ring | Use Case | Schema Changes |
|------|------|----------|----------------|
| **readonly** | R3 | Production (default) | ❌ Blocked |
| **standard** | R2 | Development teams | ✅ Controlled |
| **privileged** | R1 | DBAs | ✅ Direct (logged) |
| **emergency** | R0 | Incident recovery | ✅ No checks (audited) |

**Mode Enforcement:**
- Application code checks mode before operations
- Mode upgrades require password authentication
- Downgrades don't require password
- All mode changes logged

**Password Protection:**
```bash
# Set password
chameleon config auth set-password

# Upgrade (requires password)
chameleon config set mode=standard
🔐 Enter password: ****

# Downgrade (no password)
chameleon config set mode=readonly
```

**Status:** ✅ Complete (v1.0)

---

### 4. Go Runtime (`chameleon`)

**Engine**  
Entry point for Go applications. Loads schemas from vault, verifies integrity, enforces modes, and exposes public API.

**Key change in v1.0:**
```go
// ❌ OLD (insecure - bypasses vault)
eng := engine.NewEngine()
eng.LoadSchemaFromFile("any.cham")

// ✅ NEW (secure - vault-enforced)
eng, err := engine.NewEngine()
// ↑ Loads ONLY from .chameleon/state/schema.merged.cham
// ↑ Verifies integrity automatically
// ↑ Enforces mode restrictions
```

**Query Executor**  
Translates validated queries into backend-specific SQL. Handles field projection (`.Select()`), eager loading (`.Include()`), and filters.

**Connection Pool**  
pgx-based PostgreSQL connection management with automatic reconnection and health checks.

**IdentityMap** (NEW in v1.0)  
Automatic object deduplication in memory. When querying with `.Include()`, parent objects are deduplicated to save memory.

Example:
```go
// User with 100 posts
result := db.Query("User").
    Include("posts").
    Execute(ctx)

// Without IdentityMap: User duplicated 100 times (wasteful)
// With IdentityMap: User appears only once (efficient)
```

**Error Mapping**  
Comprehensive PostgreSQL error → ChameleonDB error mapping with clear messages and suggestions.

**Status:** ✅ Complete (v1.0)

---

### 5. CLI Tools

**Commands:**

| Command | Purpose | Status |
|---------|---------|--------|
| `init` | Initialize project + vault | ✅ v1.0 |
| `migrate` | Generate & apply migrations | ✅ v1.0 |
| `validate` | Validate schema syntax | ✅ v1.0 |
| `verify` | Verify vault integrity | ✅ v1.0 |
| `status` | Show vault + mode status | ✅ v1.0 |
| `journal schema` | View version history | ✅ v1.0 |
| `config` | Manage modes & settings | ✅ v1.0 |
| `introspect` | DB → Schema generation | ✅ v1.0 |

**Status:** ✅ Complete (v1.0)

---

### 6. FFI Boundary

Communication between Rust core and Go runtime via C ABI.

**How it works:**
- Schemas serialized to JSON in Rust
- Passed to Go via C strings
- Go deserializes and uses
- Memory managed explicitly (Rust allocates, Go frees)

**Overhead:** ~100ns per call (negligible for DB operations)

**Status:** ✅ Complete

---

## Compilation & Validation Flow

![Compilation Flow](../diagrams/Compilation_Flow.png)

```
.cham files
    ↓
Parser (LALRPOP)
    ↓
AST
    ↓
Type Checker
  - Relations validation
  - Constraints validation
  - Cycle detection
    ↓
Validated Schema
    ↓
JSON Serialization
    ↓
FFI Boundary (C ABI)
    ↓
Go Runtime
    ↓
Schema Vault Registration
  - Compute SHA256
  - Save snapshot
  - Update manifest
    ↓
Migration Generation
    ↓
SQL Execution (PostgreSQL)
```

---

## Security Model

See [SECURITY.md](SECURITY.md) for complete details.

**Layers:**
1. **OS Permissions** - File access control (0700)
2. **Hash Integrity** - SHA256 tamper detection
3. **Integrity Modes** - Runtime access control
4. **Vault Enforcement** - No schema bypass
5. **Audit Trail** - Complete forensics

---

## Design Decisions

### Why Rust for Core?

- True lambdas and closures (essential for query API)
- Extreme type safety (catch errors at compile time)
- Operator overloading (natural query syntax)
- Excellent performance on parser hot paths

### Why Go for Runtime?

- Simple concurrency (goroutines for connection pooling)
- Excellent PostgreSQL driver (pgx)
- Single-binary deployment
- Great debugging tools

### Why FFI?

- Each language does what it does best
- Minimal overhead (~100ns)
- Future-proof (easy to add Node, Python, Java bindings)

### Why Schema Vault?

- Treats schemas as first-class artifacts
- Immutability prevents silent drift
- Cryptographic integrity (SHA256)
- Complete audit trail for compliance

### Why Integrity Modes?

- Explicit governance (not just config)
- Runtime enforcement (not optional)
- Password-protected escalation
- Compliance-ready out of the box

---

## Performance Targets (v1.0)

| Operation | Target | Notes |
|-----------|--------|-------|
| Schema parse | < 10ms | One-time, cold start |
| Type check | < 5ms | Per schema validation |
| FFI call | < 100ns | Per boundary crossing |
| Hash verification | < 1ms | SHA256 computation |
| Query compilation | < 1ms | Per query |
| Query execution | DB-bound | Optimized SQL generation |

---

## Component Status

| Component | Status | Version | Notes |
|-----------|--------|---------|-------|
| Parser | ✅ Complete | v1.0 | LALRPOP, all types |
| Type Checker | ✅ Complete | v1.0 | Relations, constraints, cycles |
| **Schema Vault** | ✅ Complete | v1.0 | Versioning, hashing, audit |
| **Integrity Modes** | ✅ Complete | v1.0 | 4 modes, password-protected |
| FFI Layer | ✅ Complete | v1.0 | C ABI bridge |
| Query Builder | ✅ Complete | v1.0 | Filter, Include, Select |
| Mutations | ✅ Complete | v1.0 | Insert, Update, Delete |
| **IdentityMap** | ✅ Complete | v1.0 | Object deduplication |
| Connection Pool | ✅ Complete | v1.0 | pgx-based |
| Error Mapping | ✅ Complete | v1.0 | Comprehensive |
| Migration Gen | ✅ Complete | v1.0 | PostgreSQL DDL |
| **Introspection** | ✅ Complete | v1.0 | DB → Schema |
| Debug Mode | ✅ Complete | v1.0 | SQL visibility |
| CLI Tools | ✅ Complete | v1.0 | 8 commands |
| Backend Registry | ⏳ Planned | v2.0 | Multi-backend routing |
| Code Generator | ⏳ Planned | v1.1+ | Boilerplate generation |
| Query Optimizer | ⏳ Planned | v1.5+ | Rule-based optimization |

---

## Future Architecture (v2.0+)

![Future Architecture](../diagrams/Future%20Architecture.png)

**Planned features (not in v1.x):**
- Multi-backend routing (PostgreSQL + DuckDB + Redis)
- ML-based query optimization
- Visual schema editor
- Distributed vault (multi-node)
- Advanced observability

**Note:** v2.0 features are not part of the open-source v1.x releases.

---

## Project Structure

```
chameleondb/
├── chameleon-core/          # Rust core
│   ├── src/
│   │   ├── ast/             # Schema structures
│   │   ├── parser/          # LALRPOP grammar
│   │   ├── typechecker/     # Validation
│   │   └── ffi/             # C ABI bridge
│   └── tests/               # Integration tests
│
├── chameleon/               # Go runtime
│   ├── cmd/chameleon/       # CLI tool
│   ├── pkg/
│   │   ├── engine/          # Public API
│   │   └── vault/           # Schema Vault (NEW)
│   └── internal/
│       ├── admin/           # Journal, state tracking
│       └── schema/          # Schema merge
│
├── examples/                # Example apps
│   └── todo-app/            # Complete CRUD example
│
└── docs/                    # Documentation
    ├── architecture.md      # This file
    ├── SECURITY.md          # Security model
    ├── QUICK_START.md       # 5 min tutorial
    └── ...
```

---

## Testing

**Test coverage:**
- Rust: 94 tests ✅
- Go: 80 tests ✅
- Integration: 30 tests ✅
- **Total: 204 tests passing**

**Test categories:**
- Parser tests (syntax, error handling)
- Type checker tests (relations, cycles, constraints)
- Vault tests (versioning, integrity, modes)
- Query tests (filters, includes, selects)
- Mutation tests (CRUD operations)
- Error mapping tests (PostgreSQL → ChameleonDB)

---

## Summary

ChameleonDB v1.0 provides:
- ✅ **Schema Vault** - Versioned, hash-verified schemas
- ✅ **Integrity Modes** - Explicit runtime governance
- ✅ **Type-safe queries** - Validated before execution
- ✅ **Complete audit trail** - Who, what, when
- ✅ **Zero-config security** - Fail-safe defaults
- ✅ **Production-ready** - 204 tests passing

**Philosophy:** Explicit over implicit, safety over convenience, governance over magic.