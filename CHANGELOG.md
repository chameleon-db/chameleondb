# Changelog

All notable changes to ChameleonDB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### 🏗️ Current Focus: Type Checker + Schema Validation

The core stack is fully operational and now includes compile-time schema validation.
Schemas are parsed, type-checked, and validated before reaching the runtime.

**Stack:** `.cham` → `Rust Parser` → `Type Checker` → `FFI (C ABI)` → `Go Runtime` → `CLI` ✅

---

### Added

#### Core Parser (Rust)
- LALRPOP-based schema parser with complete DSL support
- AST representation for entities, fields, and relations
- Field types: `uuid`, `string`, `int`, `decimal`, `bool`, `timestamp`, `float`
- Extended types: `vector(N)` parameterized, `[type]` arrays
- Field modifiers: `primary`, `unique`, `nullable`, `default`
- Default value functions: `now()`, `uuid_v4()`
- Relation types: `HasOne`, `HasMany`, `BelongsTo` (ManyToMany pending)
- Backend annotations: `@cache`, `@olap`, `@vector`, `@ml`
- Serde serialization for all AST types

#### Type Checker (Rust)
- Relation validation: target entity existence, foreign key consistency
- HasMany enforcement: requires explicit `via` foreign key
- Primary key validation: exactly one per entity (detects missing or multiple)
- Annotation consistency: `@vector` only valid with `vector(N)` type
- Constraint guards: `@backend` annotations blocked on `primary` and `unique` fields
- Circular dependency detection via DFS (correctly skips `BelongsTo` inverse relations)
- Granular error types with full entity/field context
- Formatted error reports for CLI output

#### FFI Layer (Rust ↔ Go Bridge)
- C ABI interface with stable function signatures
- JSON serialization for data interchange
- Safe memory management with explicit free functions
- Error propagation with `error_out` parameter pattern
- `chameleon_validate_schema` now runs the full type checker
- Functions: `chameleon_parse_schema`, `chameleon_validate_schema`, `chameleon_free_string`, `chameleon_version`

#### Go Runtime
- CGO wrapper for Rust core (`internal/ffi`)
- Engine API for schema loading (`pkg/engine`)
- Schema type definitions mirroring Rust AST (with custom JSON marshal/unmarshal)
- Load schema from string or `.cham` file
- RPATH linking for standalone binaries

#### CLI Tool
- `chameleon version` — Show library version
- `chameleon parse <file>` — Parse schema and output JSON
- `chameleon validate <file>` — Validate schema with full type checking

#### Build System
- Makefile orchestrating Rust + Go builds
- Library path management (LD_LIBRARY_PATH + RPATH)
- CGO configuration for FFI linking

#### Documentation
- README with project overview
- Architecture diagrams (Mermaid)
- `docs/en/what_is_chameleondb.md` — Project identity and philosophy
- Example schema (`examples/basic_schema.cham`)
- Bilingual docs structure (`docs/en/`, `docs/sp/`)

---

### Test Coverage

**Total: 27 tests passing** ✅

| Layer | Tests | Status |
|-------|-------|--------|
| Parser (basic) | 7 | ✅ |
| Parser (extended types + annotations) | 4 | ✅ |
| Type Checker | 13 | ✅ |
| FFI unit tests | 5 | ✅ |
| FFI integration | 3 | ✅ |

#### Type Checker Test Breakdown
- Valid schemas: simple, with relations, with annotations
- Relation errors: unknown target, invalid FK, missing FK on HasMany
- Primary key errors: missing, multiple
- Annotation errors: wrong vector type, annotation on PK, annotation on unique
- Circular dependency: A → B → C → A detection
- Error report formatting

---

### Verified Capabilities

Successfully parses and validates complex schemas with:
- Multiple entities with bidirectional relations (User ↔ Order ↔ OrderItem)
- All field types including `vector(N)` and primitive arrays
- Mixed backend annotations (`@cache`, `@olap`, `@vector`)
- Constraints (`primary`, `unique`, `nullable`, `default`)
- Deep validation catching schema errors at compile time

---

### Known Limitations
- No query builder yet (coming next)
- No database execution yet
- No code generation yet
- `ManyToMany` relations not implemented
- Backend annotations are declarative only (routing comes later)
- No migration generation yet

---

## [0.1.0] - TBD

Initial MVP release.

### Roadmap to v0.1 Completion
- [ ] Query builder (filter, include, select)
- [ ] PostgreSQL backend execution
- [ ] Migration generation
- [ ] Performance benchmarks
- [ ] Documentation polish

---

## Development Timeline

| Phase | Status | What |
|-------|--------|------|
| Week 1-2 | ✅ Done | Parser + FFI + Go Runtime |
| Week 2-3 | ✅ Done | Multi-backend annotations + extended types |
| Week 3-4 | ✅ Done | Type Checker + deep validation |
| Week 4-5 | 🚧 Next | Query Builder |
| Week 5-6 | ⏳ Upcoming | PostgreSQL execution |
| Week 7-8 | ⏳ Upcoming | Migrations + production hardening |

---

*Last updated: 2026-02-01*