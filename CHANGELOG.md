# Changelog

All notable changes to ChameleonDB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Milestone: End-to-End System Operational

The complete stack is now functional! Schema files (.cham) can be parsed, validated, and queried through the CLI.

**Stack verified:** `.cham files` -> `Rust Parser` -> `FFI (C ABI)` -> `Go Runtime` -> `CLI`

### Added

#### Core Parser (Rust)
- ✅ LALRPOP-based schema parser with complete DSL support
- ✅ AST representation for entities, fields, and relations
- ✅ Support for field types: `uuid`, `string`, `int`, `decimal`, `bool`, `timestamp`
- ✅ Field modifiers: `primary`, `unique`, `nullable`, `default`
- ✅ Relation types: `HasOne`, `HasMany`, `BelongsTo` (ManyToMany pending)
- ✅ Default value functions: `now()`, `uuid_v4()`
- ✅ Serde serialization for all AST types
- ✅ 7 parser tests passing

#### FFI Layer (Rust ↔ Go Bridge)
- ✅ C ABI interface with `#[no_mangle]` exports
- ✅ JSON serialization for data interchange
- ✅ Safe memory management (explicit free functions)
- ✅ Error handling with `error_out` parameter pattern
- ✅ Functions: `chameleon_parse_schema`, `chameleon_validate_schema`, `chameleon_free_string`, `chameleon_version`
- ✅ 3 FFI integration tests passing

#### Go Runtime
- ✅ CGO wrapper for Rust core (`internal/ffi`)
- ✅ Engine API for schema loading (`pkg/engine`)
- ✅ Schema type definitions mirroring Rust AST
- ✅ Load from string or `.cham` file
- ✅ RPATH linking for standalone binaries
- ✅ 4 Go engine tests passing

#### CLI Tool
- ✅ `chameleon version` - Show library version
- ✅ `chameleon parse <file>` - Parse and display schema as JSON
- ✅ `chameleon validate <file>` - Validate schema integrity
- ✅ Clean error messages
- ✅ Example schema working (`examples/basic_schema.cham`)

#### Build System
- ✅ Makefile orchestrating Rust + Go builds
- ✅ Proper library path management (LD_LIBRARY_PATH + RPATH)
- ✅ CGO configuration for FFI linking
- ✅ Test targets for both Rust and Go

#### Documentation
- ✅ README with quick start
- ✅ Architecture diagrams (Mermaid)
- ✅ Example schema file
- ✅ CHANGELOG tracking progress

### Test Coverage
- **Total: 14 tests passing**
  - Rust parser: 7 tests
  - Rust FFI: 3 tests
  - Go engine: 4 tests

### Verified Capabilities
Successfully parses and validates complex schemas with:
- Multiple entities (e.g., User, Order, OrderItem)
- Field types (uuid, string, int, decimal, bool, timestamp)
- Constraints (primary key, unique, nullable)
- Default values (now(), uuid_v4(), literals)
- Relations (HasMany, BelongsTo with foreign keys)

### In Progress
- Type checker for advanced schema validation
- Relation consistency checks
- Circular dependency detection

### Known Limitations
- No query builder yet (v0.2)
- No database execution (v0.2)
- No code generation (v0.2)
- ManyToMany relations not implemented
- Basic validation only (entity existence, non-empty checks)

## [0.1.0] - TBD

Initial MVP release.

### Roadmap to v0.1 Completion
- [ ] Type checker implementation
- [ ] Enhanced validation (relation targets, foreign key consistency)
- [ ] Error messages improvement
- [ ] Performance benchmarks
- [ ] Documentation polish

---

## Development Timeline

**Week 1-2:** Parser + FFI + Go Runtime
- End-to-end system operational
- CLI functional
- 14 tests passing

**Week 3-4:** Type Checker + Validation
- Schema integrity checks
- Relation validation
- Better error messages

**Week 5-6:** Query Builder
- Type-safe query API
- Filter expressions
- Include/eager loading

**Week 7-8:** Database Execution
- PostgreSQL backend
- Connection pooling
- Query execution

---

*Last updated: 2025-02-01*