docs/en/what_is_chameleondb.md

# ChameleonDB

![Chameleon logo](../logo-200x150.png)

## What is ChameleonDB?

ChameleonDB is a **domain-first data language and runtime** designed to define, validate, and query data models as **semantic domains**, not as raw SQL tables.
It allows developers to work with relational, analytical, cached, and other data backends using a **single typed language**, while delegating execution details to a runtime that adapts to the underlying storage engines.

ChameleonDB focuses on **correctness, clarity, and long-term maintainability** of data-heavy applications.

---

## What problem does it solve?

Modern applications often deal with multiple types of data:

- transactional data (users, orders)
- analytical data (metrics, aggregates)
- cached or ephemeral data
- vector or ML-related data

Today, this usually results in:
- duplicated SQL logic
- fragile joins
- business rules spread across queries
- inconsistent models between layers
- silent runtime errors

ChameleonDB addresses this by:
- defining the data model once, at the domain level
- validating relationships and queries before execution
- providing a single semantic entry point for different data backends
- reducing coupling between application logic and storage details

---

## What is ChameleonDB NOT?

ChameleonDB is **not**:

- a database engine
- a replacement for PostgreSQL or other databases
- a traditional ORM
- a BI or reporting tool
- a web framework

ChameleonDB does not store data.
It **orchestrates access to existing data backends**.

---

## Is ChameleonDB a programming language?

ChameleonDB includes a **domain-specific language (DSL)** with its own syntax, parser, type system, and validation rules.

It is **not a general-purpose language**. It is a language designed specifically for defining and querying data models, compiled into a validated semantic plan consumed by the runtime.

---

## Is it compiled or interpreted?

ChameleonDB schemas and queries are **compiled into a validated semantic plan**, not into machine code.

This compilation step:
- validates types and relationships
- detects invalid queries early
- produces an execution plan consumed by the runtime

There is no virtual machine or bytecode runtime.

---

## How does ChameleonDB work?

### 1. Define your domain

You describe entities, fields, relationships, and annotations using the ChameleonDB language:

```rust
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    created_at: timestamp default now(),
    orders: [Order] via user_id,
    session: string @cache,
}

entity Order {
    id: uuid primary,
    total: decimal,
    status: string,
    user_id: uuid,
    user: User,
    items: [OrderItem] via order_id,
}
```

### 2. Validate and plan

The ChameleonDB core validates schemas and queries at compile time:
- entity references exist and are consistent
- relation targets are valid and foreign keys match
- each entity has exactly one primary key
- backend annotations are used correctly
- no circular ownership dependencies exist

If validation fails, you get clear, contextual error messages before any code runs.

### 3. Execute via runtime

The runtime executes queries against configured backends:

```go
users := db.Users().
    Filter(expr.Field("email").Eq("ana@mail.com")).
    Include("orders").
    Execute()
```

The application expresses **intent**. The runtime handles execution details.

---

## Where is data stored?

Data is stored in **external backends**, such as:
- PostgreSQL (OLTP)
- analytical stores (future)
- caches (future)
- vector stores (future)

ChameleonDB does not own storage.
It defines **how data is accessed and validated**, not where it lives.

---

## What are annotations?

Annotations provide **semantic hints** about how data should be treated:

```rust
session_token: string @cache
monthly_spent: decimal @olap
embedding: vector(384) @vector
```

Annotations:
- do not change the logical domain model
- do not force a specific backend
- are validated at compile time (e.g., `@vector` requires `vector(N)` type)
- allow future backend specialization without rewriting schemas

In early versions, annotations are declarative metadata.
Execution routing evolves over time as backends are added.

---

## How is ChameleonDB used from applications?

ChameleonDB provides a runtime API (currently in Go) that allows applications to:
- load and validate schemas
- build queries
- execute them safely

Example (Go):
```go
users := db.Users().
    Filter(expr.Field("email").Eq("ana@mail.com")).
    Include("orders").
    Execute()
```

The runtime:
- validates queries against the schema
- translates them to backend-specific operations
- returns structured results

---

## Who is ChameleonDB for?

ChameleonDB is designed for:
- backend developers working with complex data models
- teams maintaining data-heavy systems
- applications that mix transactional and analytical data
- developers who want stronger guarantees than raw SQL or traditional ORMs

It is also designed to be **approachable to non-SQL experts**, such as data analysts, through higher-level abstractions built on top of the core.

---

## Project scope and philosophy

ChameleonDB prioritizes:
- correctness over convenience
- explicit models over implicit magic
- validation before execution
- long-term evolution over short-term hacks

Not all features are implemented at once.
The architecture is designed to **grow without breaking existing models**.

---

## Project structure

```
chameleondb/
├── chameleon-core/          Rust — parser, type checker, AST, FFI
│   ├── src/
│   │   ├── ast/             Schema data structures
│   │   ├── parser/          LALRPOP grammar and parser
│   │   ├── typechecker/     Compile-time validation
│   │   └── ffi/             C ABI bridge to Go
│   └── tests/               Integration tests
│
├── chameleon/               Go — runtime, CLI, backend connectors
│   ├── cmd/chameleon/       CLI tool
│   ├── pkg/engine/          Public engine API
│   └── internal/ffi/        CGO bindings
│
├── examples/                Example .cham schemas
└── docs/
    ├── en/                  English documentation
    └── sp/                  Spanish documentation
```

- **Rust core** — parsing, validation, and optimization
- **Go runtime** — query execution and backend connectivity
- **FFI bridge** — C ABI interface between Rust and Go (~100ns overhead)

---

## Current status

ChameleonDB is in **early development (v0.1)**.

**What works today:**
- Schema DSL with entities, fields, relations, and backend annotations
- Parser with full syntax support (Rust core, LALRPOP)
- Type checker with compile-time validation (relations, constraints, cycles)
- Runtime with CLI tool (Go)
- Field types: `uuid`, `string`, `int`, `decimal`, `bool`, `timestamp`, `float`, `vector(N)`, arrays
- Backend annotations: `@cache`, `@olap`, `@vector`, `@ml` (declarative, not yet routed)
- 27 tests passing across the full stack

**What is coming next:**
- Query builder (filter, include, select)
- PostgreSQL backend execution
- Backend routing based on annotations
- Migration generation

ChameleonDB still is not ready for production use.