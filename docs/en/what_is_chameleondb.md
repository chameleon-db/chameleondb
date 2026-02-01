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
```rust
entity User {
    id: uuid primary,
    email: string unique,
    orders: [Order] via user_id,
    session: string @cache,
}

entity Order {
    id: uuid primary,
    total: decimal,
    user: User,
}
```

### 2. Validate and plan

The Chameleon core validates schemas and queries at compile time:
- entity references exist
- relation targets are valid
- backend annotations are consistent
- types are correct

### 3. Execute via runtime
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

Annotations provide **semantic hints** about how data should be treated.

Example:

```rust
monthly_spent: decimal @olap
session_token: string @cache
embedding: vector(384) @vector
```
Annotations:  
- do not change the logical domain model  
- do not force a specific backend  
- allow future backend specialization without rewriting schemas  

In early versions, annotations are metadata.  
Execution strategies evolve over time.  

## How is ChameleonDB used from applications?

ChameleonDB provides a runtime API (currently in Go) that allows applications to:  
- load schemas  
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
- validates queries
- translates them to backend-specific operations
- returns structured results  

## Who is ChameleonDB for?
ChameleonDB is designed for:  
- backend developers working with complex data models  
- teams maintaining data-heavy systems  
- applications that mix transactional and analytical data  
- developers who want stronger guarantees than raw SQL or traditional ORMs  

It is also designed to be **approachable to non-SQL experts**, such as data analysts, through higher-level abstractions built on top of the core.

## Project scope and philosophy
ChameleonDB prioritizes:  
- correctness over convenience  
- explicit models over implicit magic  
- validation before execution  
- long-term evolution over short-term hacks  

Not all features are implemented at once.  
The architecture is designed to **grow without breaking existing models.**

## Project structure
```
chameleondb/
├── chameleon-core/          Rust — parser, AST, type checker, FFI
├── chameleon/               Go  — runtime, CLI, backend connectors
├── examples/                Example .cham schemas
└── docs/
    ├── en/                  English documentation
    └── sp/                  Spanish documentation
```

- **Rust core**: responsible for parsing, validation, and optimization
- **Go runtime**: responsible for query execution and backend connectivity
- **FFI bridge**: C ABI interface between Rust and Go (~100ns overhead)

## Current status

ChameleonDB is in **early development (v0.1)**.

What works today:
- Schema DSL with entities, fields, relations, and backend annotations
- Parser with compile-time validation (Rust core)
- Runtime with CLI tool (Go)
- Field types: uuid, string, int, decimal, bool, timestamp, float, vector(N), arrays
- Backend annotations: @cache, @olap, @vector, @ml (declarative, not yet routed)

What is coming next:
- Type checker for deep schema validation
- Query builder and execution against PostgreSQL
- Backend routing based on annotations
- Migration generation

ChameleonDB still is not ready for production use.

