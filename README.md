<div align="center">

![ChameleonDB](docs/logo-200x150.png)

*Type-safe, graph-oriented database access without the magic*

[![License: Apache](https://img.shields.io/badge/license-Apache%20License%202.0-blue)](https://www.apache.org/licenses/LICENSE-2.0)
[![Rust Version](https://img.shields.io/badge/rust-1.75%2B-orange.svg)](https://www.rust-lang.org)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://golang.org)
[![Status](https://img.shields.io/badge/status-beta-yellow)](https://github.com/chameleon-db/chameleondb)

[Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples) • [Discord](https://chameleondb.dev/discord)

</div>

---

## ⚠️ Beta Notice

ChameleonDB is in **beta** (v1.0-beta). Core features are stable, API may have minor changes. 
Production-ready for evaluation and non-critical workloads.

**Early adopters welcome** — your feedback shapes the product.

---

## Overview

ChameleonDB is a **graph-oriented, strongly-typed database access layer** that brings compile-time safety to database queries.

Instead of writing SQL or dealing with ORM magic, you define a schema once and navigate relationships naturally.

### The Problem

Traditional database access has friction:

- **SQL** requires manual JOINs and is error-prone
- **ORMs** hide behavior and have runtime errors
- **Type-safety** is missing at the query level
- **N+1 queries** are easy to introduce accidentally
- **Debugging** complex queries is painful

### The ChameleonDB Solution

**1. Define your schema** (or import from existing database)
```chameleon
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

**2. Write natural queries with field projection**
```go
// Query only the fields you need
users := db.Query("User").
    Select("id", "name", "email").  // Partial selection
    Filter("age", "gt", 25).
    Include("posts").                // Eager load relations
    Execute(ctx)
```

**3. See exactly what runs** (Debug Mode)
```go
// Enable debug to see generated SQL
users := db.Query("User").
    Select("id", "name").
    Filter("email", "like", "ana").
    Debug().  // Shows SQL before execution
    Execute(ctx)
```

Output:
```sql
[SQL] Query User
SELECT id, name FROM users WHERE email LIKE '%ana%'

[TRACE] Query on User: 2.3ms, 3 rows
```

**4. Mutations with safety guards**
```go
// Insert with validation
result, err := db.Insert("User").
    Set("email", "ana@mail.com").
    Set("name", "Ana Garcia").
    Debug().
    Execute(ctx)

// Update with mandatory WHERE clause
db.Update("User").
    Filter("id", "eq", userID).
    Set("name", "Ana María").
    Execute(ctx)

// Delete with safety guard (prevents accidental full table delete)
db.Delete("Post").
    Filter("published", "eq", false).
    Execute(ctx)
```

**What you get:**

✅ **Compile-time schema validation** — Catch errors before runtime  
✅ **Field projection** — Query only what you need (performance++)  
✅ **Graph navigation** — No manual JOINs required  
✅ **Full SQL transparency** — See generated queries with `.Debug()`  
✅ **Mutation safety** — Prevent UPDATE/DELETE without WHERE  
✅ **Zero magic** — Predictable, explicit behavior  
✅ **Native performance** — Rust core, minimal overhead  

---

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 14+

### Installation
```bash
# Install ChameleonDB CLI
curl -sSL https://chameleondb.dev/install | sh

# Or build from source
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make build
```

### Your First Project
```bash
# Create new project
cd my-project
chameleon init

# Validate schema
chameleon validate

# Generate migration
chameleon migrate --dry-run

# Apply to database
chameleon migrate --apply
```

### Your First Query
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

func main() {
    // Connect to database
    eng := engine.NewEngine()
    eng.LoadSchemaFromFile("schema.cham")
    
    ctx := context.Background()
    config := engine.ConnectorConfig{
        Host:     "localhost",
        Port:     5432,
        Database: "my_blog",
        User:     "postgres",
        Password: "postgres",
    }
    
    eng.Connect(ctx, config)
    defer eng.Close()
    
    // Query with field projection and eager loading
    result, err := eng.Query("User").
        Select("id", "name", "email").  // Only fetch needed fields
        Filter("email", "eq", "ana@mail.com").
        Include("posts").                 // Eager load (no N+1)
        Execute(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Access results
    for _, user := range result.Rows {
        fmt.Printf("User: %s (%s)\n", 
            user["name"], user["email"])
        
        if posts, ok := result.Relations["posts"]; ok {
            fmt.Printf("  Posts: %d\n", len(posts))
        }
    }
}
```

---

## Core Features

### 🎯 Field Projection (v1.0)

Query only the fields you need for optimal performance:

```go
// Fetch only specific fields
users := db.Query("User").
    Select("id", "name").  // No email, age, etc.
    Execute(ctx)

// Combine with filters
activeUsers := db.Query("User").
    Select("id", "email").
    Filter("active", "eq", true).
    Execute(ctx)

// Default behavior (backward compatible)
allUsers := db.Query("User").Execute(ctx)  // SELECT * FROM users
```

**Benefits:**
- 📉 Reduced network traffic
- ⚡ Faster queries
- 💾 Lower memory usage
- 🔒 Better security (don't expose unnecessary data)

### 🛡️ Mutation Safety (v1.0)

Built-in safety guards prevent common mistakes:

```go
// ✅ SAFE: Update with filter
db.Update("User").
    Filter("id", "eq", userID).
    Set("name", "New Name").
    Execute(ctx)

// ❌ BLOCKED: Update without filter (would affect entire table)
db.Update("User").
    Set("name", "Same Name").
    Execute(ctx)
// Error: UPDATE requires a WHERE clause

// ✅ SAFE: Explicit confirmation for dangerous operations
db.Update("User").
    Set("verified", true).
    ForceUpdateAll().  // Explicit opt-in
    Execute(ctx)
```

**Safety Features:**
- 🚫 No UPDATE/DELETE without WHERE clause
- 🔐 Primary key update prevention
- ✅ Required field validation
- 📝 Type checking (UUID, email, etc.)
- ⚠️ Clear error messages with suggestions

### 🔍 Debug Mode (v1.0)

See exactly what SQL runs, with timing information:

```go
// Enable debug for a single query
users := db.Query("User").
    Select("id", "name").
    Filter("age", "gt", 25).
    Debug().  // Shows SQL + execution time
    Execute(ctx)
```

Output:
```
[SQL] Query User
SELECT id, name FROM users WHERE age > 25

[TRACE] Query on User: 1.2ms, 42 rows
```

**Debug Levels:**
- `DebugOff` - No output (production)
- `DebugSQL` - Show generated SQL
- `DebugTrace` - SQL + timing + row count

**Use cases:**
- 🐛 Debugging slow queries
- 📊 Performance optimization
- 🎓 Learning SQL generation
- 🔧 Development workflow

### 🔗 Graph Navigation

Navigate relationships without manual JOINs:

```go
// Eager load nested relations
posts := db.Query("Post").
    Include("author").           // User entity
    Include("comments").         // Comment entities
    Include("comments.author").  // Nested: comment authors
    Execute(ctx)

// Filter through relations (automatic JOIN)
posts := db.Query("Post").
    Filter("author.verified", "eq", true).  // Joins users table
    Filter("published", "eq", true).
    Execute(ctx)
```

Generated SQL (with automatic JOIN):
```sql
SELECT DISTINCT posts.*
FROM posts
INNER JOIN users ON users.id = posts.author_id
WHERE users.verified = true AND posts.published = true
```

---

## Architecture

ChameleonDB uses a **hybrid Rust + Go architecture** for type-safety and performance:

```
┌───────────────────────────────────────────────────┐
│  Go Application Layer                             │
│  - Query Builder (fluent API)                     │
│  - Mutation Factory (Insert/Update/Delete)        │
│  - Connection Management (pgx)                    │
└──────────────────┬────────────────────────────────┘
                   │ Go Runtime
                   ↕ FFI (C ABI, ~100ns overhead)
┌──────────────────▼────────────────────────────────┐
│  Rust Core (libchameleon.so)                      │
│  - Schema Parser (LALRPOP)                        │
│  - Type Checker & Validator                       │
│  - SQL Generator (PostgreSQL)                     │
│  - Migration Generator                            │
└───────────────────────────────────────────────────┘
```

### Design Principles

**Contract-Driven Architecture:**
- Interfaces define boundaries between layers
- Factory pattern for extensibility
- Dependency injection for testing
- No circular dependencies

**Example: Mutation System**
```go
// Contract (interface)
type InsertMutation interface {
    Set(field string, value interface{}) InsertMutation
    Debug() InsertMutation
    Execute(ctx context.Context) (*InsertResult, error)
}

// Factory creates implementations
factory := mutation.NewFactory(schema)
insert := factory.NewInsert("User")

// Usage (works with any implementation)
result := insert.
    Set("email", "ana@mail.com").
    Set("name", "Ana").
    Debug().
    Execute(ctx)
```

**Why this architecture?**

- **Rust core**: Type-safety, zero-cost abstractions, fast parsing
- **Go runtime**: Simple deployment, great concurrency, familiar tooling
- **FFI overhead**: ~100ns per call (negligible for DB operations)
- **Future-proof**: Easy to add Node, Python, Java bindings

---

## Advanced Features

### Raw SQL Escape Hatch

For complex queries beyond the query builder:

```go
// Access the underlying pgx connection pool
pool := engine.Connector().Pool()

// Execute raw SQL for complex operations
rows, err := pool.Query(ctx, `
    SELECT u.name, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON p.author_id = u.id
    GROUP BY u.id, u.name
    HAVING COUNT(p.id) > 5
    ORDER BY post_count DESC
`)
```

**When to use:**
- ✅ Complex aggregations (GROUP BY, HAVING)
- ✅ Subqueries and CTEs
- ✅ Database-specific features (full-text search, JSON operators)
- ✅ Performance-critical queries with custom indexes

See [Raw SQL Documentation](docs/raw_sql.md) for details.

### Validation Pipeline

Three-stage validation for data integrity:

**Stage 1: Go-level validation**
- Type checking (string, int, UUID)
- Format validation (email, UUID format)
- Null constraints

**Stage 2: Schema validation (Rust)**
- Field existence
- Relation validation
- Primary key constraints

**Stage 3: Database constraints (PostgreSQL)**
- Unique constraints
- Foreign keys
- Check constraints

```go
// Example: All three stages in action
result, err := db.Insert("User").
    Set("email", "not-an-email").  // ❌ Stage 1: Invalid email format
    Set("age", "twenty").          // ❌ Stage 1: Type mismatch (string vs int)
    Set("unknown_field", "val").   // ❌ Stage 2: Field doesn't exist in schema
    Execute(ctx)
    // ❌ Stage 3: Database unique constraint (if email exists)
```

---

## Features Status

### ✅ Available Now (v1.0-beta)

**Query System:**
- [x] Schema parser and validator
- [x] Query builder with filters
- [x] **Field projection** (`.Select()`)
- [x] Eager loading (`.Include()`)
- [x] Nested includes
- [x] Relation filtering (automatic JOINs)
- [x] **Debug mode** (`.Debug()`)

**Mutation System:**
- [x] **Insert builder** with validation
- [x] **Update builder** with safety guards
- [x] **Delete builder** with safety guards
- [x] **Mutation factory** pattern
- [x] Three-stage validation pipeline

**Tooling:**
- [x] CLI tools (init, validate, migrate, check)
- [x] Rich error messages with suggestions
- [x] PostgreSQL migration generator
- [x] VSCode extension (syntax highlighting, diagnostics)
- [x] Connection string support (`DATABASE_URL`)
- [x] 80+ integration tests

### 🚧 Working on - This Quarter (v1.1)

- [ ] **IdentityMap** (object deduplication)
- [ ] Database introspection (`chameleon introspect`)
- [ ] Transaction support
- [ ] Batch operations

### 🔮 Planned (v1.2+)

- [ ] Code generation (type-safe DTOs)
- [ ] Query explain and optimization hints
- [ ] Redis backend (@cache annotation)
- [ ] Migration from Prisma/TypeORM
- [ ] Multi-language support (TypeScript, Python)

---

## Project Structure
```
chameleondb/
├── chameleon-core/          # Rust core library
│   ├── src/
│   │   ├── ast/             # Schema AST structures
│   │   ├── parser/          # LALRPOP grammar
│   │   ├── typechecker/     # Validation logic
│   │   ├── query/           # Query AST + filters
│   │   ├── sql/             # SQL generation + SELECT projection
│   │   ├── migration/       # DDL generation
│   │   └── ffi/             # C ABI bridge
│   └── Cargo.toml
│
├── chameleon/               # Go runtime
│   ├── cmd/chameleon/       # CLI tool
│   ├── pkg/
│   │   └── engine/          # Query executor + mutation factory
│   │       ├── mutation/    # Insert/Update/Delete builders
│   │       ├── query.go     # QueryBuilder with Select
│   │       ├── executor.go  # SQL execution
│   │       └── validation.go # Validation pipeline
│   └── go.mod
│
├── docs/                    # Documentation
│   ├── architecture.md
│   ├── what_is_chameleondb.md
│   ├── query-reference.md
│   └── raw_sql.md           # Raw SQL escape hatch guide
│
└── examples/                # Separate repo
    ├── 01-hello-world/
    ├── 02-blog/
    └── tutorial/
```

---

## Roadmap

### Q1 2026 ✅ v1.0-beta (Current)

✅ Schema parser & type checker  
✅ PostgreSQL backend  
✅ Query builder with SELECT projection  
✅ Mutation factory (Insert/Update/Delete)  
✅ Debug mode  
✅ CLI tools  
✅ VSCode extension  

**Goal:** Production-ready for simple to medium complexity apps

### Q2 2026 - v1.1

- IdentityMap (object deduplication)
- Database introspection
- Transaction support
- Batch operations
- Performance benchmarks

**Goal:** Feature parity with major ORMs

### Q3-Q4 2026 - v1.2 Stable

- Code generation
- Additional backends (MySQL, Redis)
- Migration tools
- Multi-language bindings

**Goal:** Enterprise-ready

### 2027+ - v2.0

- DuckDB backend (OLAP)
- ML-based query optimization
- Visual schema editor
- Advanced features

---

## Performance

Early benchmarks (v1.0-beta):

**Schema Operations:**
- Schema parsing: < 1ms for typical schemas
- Type checking: < 5ms for complex queries
- FFI overhead: ~100ns per call

**Query Performance:**
- Simple queries: On par with hand-written SQL
- Field projection: 20-40% faster than SELECT * (depends on table width)
- Eager loading: Eliminates N+1 queries

**Validation:**
- Go-level validation: < 1µs per field
- Schema validation: < 5µs per query
- Total overhead: Negligible vs network + DB time

Full benchmarks vs Prisma/GORM coming in v1.1.

---

## Contributing

We welcome contributions! ChameleonDB is actively developed and there's plenty to do.

### How to Contribute

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing`)
5. **Open** a Pull Request

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

- 🦀 **Rust**: Query optimizer, additional SQL dialects
- 🐹 **Go**: Runtime improvements, connection pooling
- 📚 **Documentation**: Tutorials, API docs, migration guides
- 🧪 **Testing**: Unit tests, integration tests, benchmarks
- 🎨 **Tooling**: VSCode extension, browser devtools

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

---

## Community

- **GitHub Discussions**: [Ask questions, share ideas](https://github.com/chameleon-db/chameleondb/discussions)
- **Discord**: [Join our server](https://chameleondb.dev/discord)
- **Twitter**: [@chameleondb](https://twitter.com/chameleondb)

---

## Why ChameleonDB?

### vs Raw SQL

| Raw SQL | ChameleonDB |
|---------|-------------|
| ❌ Manual JOINs | ✅ Graph navigation |
| ❌ No type safety | ✅ Compile-time validation |
| ❌ Easy to make mistakes | ✅ Safety guards |
| ❌ No field projection helpers | ✅ `.Select()` API |
| ✅ Full control | ✅ Full control + convenience |

### vs Traditional ORMs (Prisma, GORM, TypeORM)

| Traditional ORMs | ChameleonDB |
|------------------|-------------|
| ❌ Runtime errors | ✅ Compile-time errors |
| ❌ Magic behavior | ✅ Explicit, predictable |
| ❌ Hidden SQL | ✅ Full transparency (`.Debug()`) |
| ❌ TypeScript-only (Prisma) | ✅ Multi-language (Go, TS planned) |
| ✅ Rich ecosystem | 🚧 Growing ecosystem |

### Key Differentiators

1. **SQL Visibility**: See exactly what runs with `.Debug()`
2. **Field Projection**: Built-in SELECT optimization
3. **Safety by Default**: Prevent dangerous operations
4. **Multi-Language**: Not tied to one ecosystem
5. **No Magic**: Predictable behavior, no surprises

---

## License

ChameleonDB is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

Inspired by:

- **Prisma** — Schema-first approach and developer experience
- **GraphQL** — Graph-based data navigation
- **Rust** — Type safety and zero-cost abstractions
- **EdgeDB** — Rethinking database access layers

Special thanks to all [contributors](https://github.com/chameleon-db/chameleondb/graphs/contributors)!

---

<div align="center">

**Built with ❤️ by developers, for developers**

[Website](https://chameleondb.dev) • [Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples)

</div>