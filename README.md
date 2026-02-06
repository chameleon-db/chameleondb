<div align="center">

![ChameleonDB](docs/logo-200x150.png)

*Type-safe, graph-oriented database access without the magic*

[![License: Apache](https://img.shields.io/badge/license-Apache%20License%202.0-blue)](https://www.apache.org/licenses/LICENSE-2.0)
[![Rust Version](https://img.shields.io/badge/rust-1.75%2B-orange.svg)](https://www.rust-lang.org)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://golang.org)
[![Status](https://img.shields.io/badge/status-alpha-yellow)](https://github.com/chameleon-db/chameleondb)

[Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples) • [Discord](https://chameleondb.dev/discord)

</div>

---

## ⚠️ Early Development Notice

ChameleonDB is in **active development** (v0.1 alpha). The API is not stable yet. 
For production use, wait for v1.0 (Q3 2026).

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

### The ChameleonDB Solution

**1. Define your schema** (or import from existing database)
```chameleon
entity User {
    id: uuid primary,
    email: string unique,
    posts: [Post] via author_id,
}

entity Post {
    id: uuid primary,
    title: string,
    published: bool,
    author_id: uuid,
    author: User,
}
```

**2. Write natural queries**
```go
// Get user with all their posts
user := db.Query("User").
    Filter("email", "eq", "ana@mail.com").
    Include("posts").
    Execute(ctx)
```

**3. See exactly what runs**
```sql
-- Main query
SELECT id, email FROM users WHERE email = 'ana@mail.com';

-- Eager load (no N+1)
SELECT id, title FROM posts WHERE author_id IN ('...');
```

**What you get:**

✅ **Compile-time schema validation** — Catch errors before runtime  
✅ **Graph navigation** — No manual JOINs required  
✅ **Full SQL transparency** — See generated queries  
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
curl -sSL https://chameleondb.dev/install.sh | sh

# Or build from source
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make build
```

### Your First Project
```bash
# Create new project
chameleon init my-blog
cd my-blog

# Validate schema
chameleon validate

# Generate migration
chameleon migrate --dry-run

# Apply to database
chameleon migrate --apply

# Insert sample data
psql my_blog < seed.sql
```

### Your First Query
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/chameleon-db/chameleondb/pkg/engine"
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
    
    // Query with eager loading
    result, err := eng.Query("User").
        Filter("email", "eq", "ana@mail.com").
        Include("posts").
        Execute(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Access results
    for _, user := range result.Rows {
        fmt.Printf("User: %s\n", user.String("email"))
        
        if posts, ok := result.Relations["posts"]; ok {
            fmt.Printf("  Posts: %d\n", len(posts))
        }
    }
}
```

---

## Architecture

ChameleonDB uses a **hybrid Rust + Go architecture**:
```
┌─────────────────────────────────┐
│  Rust Core (libchameleon.so)    │
│  - Parser (LALRPOP)             │
│  - Type checker                 │
│  - SQL generator                │
└──────────────┬──────────────────┘
               ↕ FFI (C ABI)
┌──────────────▼──────────────────┐
│  Go Runtime                     │
│  - Query executor               │
│  - Connection pooling (pgx)     │
│  - CLI tool                     │
└─────────────────────────────────┘
```

**Why this architecture?**

- **Rust core**: Type-safety, zero-cost abstractions, fast parsing
- **Go runtime**: Simple deployment, great concurrency, familiar tooling
- **FFI overhead**: ~100ns per call (negligible)
- **Future-proof**: Easy to add Node, Python, Java bindings

---

## Features

### ✅ Available Now (v0.1)

- [x] Schema parser and validator
- [x] Rich error messages with suggestions
- [x] PostgreSQL migration generator
- [x] Query builder with filters
- [x] Eager loading (Include)
- [x] Nested includes
- [x] Relation filtering (automatic JOINs)
- [x] CLI tools (init, validate, migrate, check)
- [x] 60+ integration tests

### 🚧 Coming This Month

- [ ] Database introspection (`chameleon introspect`)
- [ ] VSCode extension (syntax highlighting, diagnostics)
- [ ] Connection string support (`DATABASE_URL`)
- [ ] Debug mode (show SQL before execution)
- [ ] Performance benchmarks vs Prisma/GORM

### 🔮 Planned (v0.2)

- [ ] Code generation (type-safe DTOs)
- [ ] Redis backend (@cache annotation)
- [ ] Migration from Prisma/TypeORM
- [ ] Multi-language support (TypeScript, Python)
- [ ] Query explain and optimization hints

---

## Project Structure
```
chameleondb/
├── chameleon-core/          # Rust core library
│   ├── src/
│   │   ├── ast/             # Schema structures
│   │   ├── parser/          # LALRPOP grammar
│   │   ├── typechecker/     # Validation
│   │   ├── query/           # Query AST
│   │   ├── sql/             # SQL generation
│   │   ├── migration/       # DDL generation
│   │   └── ffi/             # C ABI bridge
│   └── Cargo.toml
│
├── chameleon/               # Go runtime
│   ├── cmd/chameleon/       # CLI tool
│   ├── pkg/
│   │   └── engine/          # Query executor
│   └── go.mod
│
├── docs/                    # Documentation
│   ├── architecture.md
│   ├── what_is_chameleondb.md
│   └── query-reference.md
│
└── examples/                # Separate repo
    ├── 01-hello-world/
    ├── 02-blog/
    └── tutorial/
```

---

## Roadmap

### Q1 2026 (Current) - MVP v0.1

✅ Schema parser & type checker  
✅ PostgreSQL backend  
✅ Query builder & SQL generation  
✅ CLI tools  
🚧 VSCode extension  
🚧 Database introspection  

**Goal:** Production-ready for simple use cases

### Q2 2026 - v0.2

- Code generation (DTOs)
- Redis backend
- Migration tools
- Multi-language support

**Goal:** Feature parity with major ORMs

### Q3-Q4 2026 - v1.0 Stable

- Performance optimization
- Production hardening
- Complete documentation
- Case studies

**Goal:** Enterprise-ready

### 2027+ - v2.0

- Additional backends (MySQL, DuckDB)
- ML-based query optimization
- Advanced features

---

## Contributing

We welcome contributions! ChameleonDB is in early stages and there's plenty to do.

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

- 🦀 **Rust**: Parser improvements, query optimizer
- 🐹 **Go**: Runtime improvements, testing
- 📚 **Documentation**: Tutorials, API docs
- 🧪 **Testing**: Unit tests, integration tests
- 🎨 **Design**: VSCode extension, tooling

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

---

## Community

- **GitHub Discussions**: [Ask questions, share ideas](https://github.com/chameleon-db/chameleondb/discussions)
- **Discord**: [Join our server](https://chameleondb.dev/discord)
- **Twitter**: [@chameleondb](https://twitter.com/chameleondb)

---

## Why ChameleonDB?

### vs Raw SQL

❌ Manual JOINs  
❌ No type safety  
❌ Easy to make mistakes  

✅ Graph navigation  
✅ Compile-time validation  
✅ Clear, maintainable code  

### vs Traditional ORMs

❌ Runtime errors  
❌ Magic behavior  
❌ Poor visibility into SQL  

✅ Compile-time errors  
✅ Explicit, predictable  
✅ Full SQL transparency  

---

## Performance

Early benchmarks (v0.1):

- **Schema parsing**: < 1ms for typical schemas
- **Type checking**: < 5ms for complex queries
- **FFI overhead**: ~100ns per call
- **Query execution**: On par with hand-written SQL

Full benchmarks vs Prisma/GORM coming in v0.2.

---

## License

ChameleonDB is licensed under the **Apache 2.0** - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

Inspired by:

- **Prisma** — Schema-first approach
- **GraphQL** — Graph-based querying
- **Rust** — Type safety and performance
- **EdgeDB** — Rethinking database access

Special thanks to all [contributors](https://github.com/chameleon-db/chameleondb/graphs/contributors)!

---

<div align="center">

**Built with ❤️ by developers, for developers**

[Website](https://chameleondb.dev) • [Documentation](https://chameleondb.dev/docs) • [Examples](https://github.com/chameleon-db/chameleon-examples)

</div>