docs/architecture.md

# chameleonDB Architecture

## System Overview
![System Overview diagram](diagrams/system_overview.png)

## 2. Compilation Flow
![Compilation Flow](diagrams/Compilation_Flow.png)


## 3. Data Flow
![Schema DSL Pipeline](diagrams/Data%20Flow.png)

## Component Responsibilities

### Rust Core (`chameleon-core`)
- **Parser**: Schema DSL → AST (LALRPOP grammar)
- **Type Checker**: Validates schema integrity, relation consistency
- **Code Generator**: Produces type-safe bindings for Rust/Go
- **Optimizer**: Rule-based query optimization (v1.0)

### Go Runtime (`chameleon`)
- **Query Engine**: Translates validated queries to SQL
- **Executor**: Manages query execution and result mapping
- **Connection Pool**: pgx-based PostgreSQL connection management
- **CLI Tool**: Developer tooling and migrations

### FFI Boundary
- **C ABI**: Stable interface between Rust and Go
- **Zero-copy where possible**: Strings via pointers
- **Error handling**: Result codes + error messages
- **Overhead**: ~100ns per call (negligible)

## Design Decisions

### Why Rust for Core?
- True lambdas and closures (essential for query API)
- Extreme type safety (catch errors at compile time)
- Operator overloading (natural query syntax)
- Performance (parser and type-checker are hot paths)

### Why Go for Runtime?
- Simple concurrency model (goroutines for connection pooling)
- Excellent PostgreSQL driver (pgx)
- Easy deployment (single binary)
- Great tooling and debugging experience

### Why FFI instead of pure Rust or pure Go?
- **Best of both worlds**: Rust's type system + Go's runtime simplicity
- **Future-proof**: Can add bindings for other languages (Node, Java, Python)
- **Performance**: FFI overhead is minimal (~100ns)
- **Development velocity**: Go runtime is faster to iterate on

## Future Architecture (v2.0+)
![Future Architecture](diagrams/Future%20Architecture.png)

## Performance Targets (v1.0)

| Operation | Target | Notes |
|-----------|--------|-------|
| Schema parse | <10ms | Cold start, one-time |
| Type check query | <1ms | Per query compilation |
| FFI call overhead | <100ns | Per boundary crossing |
| Query execution | Database-bound | Optimized SQL generation |

## Next Steps

1. ✅ Schema parser (LALRPOP)
2. 🚧 FFI layer design
3. ⏳ Type checker implementation
4. ⏳ Code generator (Rust/Go bindings)
5. ⏳ Query executor (Go runtime)