# Changelog
All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog,
and this project adheres to Semantic Versioning (pre-1.0).

---

## [Unreleased]

---

## [1.1.0-alpha] - 2026-11

### Security
- Hardened query and mutation generation. See GHSA-2r6g-424r-6xmr.

### Fixed
- **Migrations are no longer destructive.** `chameleon migrate --apply` used to emit `DROP TABLE IF EXISTS ... CASCADE` for every entity before recreating it, which deleted all data in those tables. The generated script is now create-only: one `CREATE TABLE IF NOT EXISTS` per entity, in dependency order, with no `DROP`, `TRUNCATE` or `ALTER`.

### Changed
- **Behavior change:** because the script is create-only, tables that already exist are left untouched. Schema changes to existing tables (new columns, type changes) are not applied by `migrate` yet and must be applied manually. Applying differences is planned work.
- Generated DDL is deterministic: columns are ordered primary key first, then alphabetically, and constraints are sorted, so the same schema always produces the same SQL.
- `migrate` now says that `backup_on_migrate` is not implemented instead of printing "Creating backup...".
- Hardened the mutation path: table and column names in insert, update and delete are validated against the schema and quoted. Delete filters now reject unknown fields with `UnknownFieldError` before reaching the database.
- Float filter values are bound as `numeric`, so comparisons with integer columns keep their exact value.
- Added `engine.QuoteIdent` for validating and quoting identifiers.

## [1.1.0-alpha] - 2026-11

### Added
- Enforced vault-based schema loading model for the engine.
- New `status` and `verify` commands for runtime and schema health checks.
- Improved security model documentation aligned with vault-only execution.

### Changed
- Refactored documentation structure and clarified engine security guarantees.
- Hardened schema loading paths to prevent non-vault execution.
- Hardened query generation: values are bound as parameters (`$1..$n`) instead of being written into SQL text, and identifiers are validated against the schema and quoted.
- `GeneratedSQL` gained `main_params` and `eager_params` (optional in the JSON). `ToSQL`, `Debug` and `playground-sql` show placeholder SQL and parameters separately.
- Eager loading binds parent ids as one array parameter (`= ANY($1)`).
- `like` filters escape `%`, `_` and `\` in the value and declare `ESCAPE '\'`.
- Query generation now rejects the `in` operator, `like` on non-string values and relation paths deeper than `relation.field`; these did not produce working SQL before.
- Generated identifiers are quoted (and folded to lower case, as unquoted names are in PostgreSQL), so `FROM users` is now `FROM "users"`.

### Fixed
- Corrected `pkg-config` installation logic in install scripts.
- Normalized internal library naming conventions.

---

## [0.2.6-alpha] – 2026-02

### Added
- Updated installation and uninstallation scripts supporting:
  - New shared library layout
  - `pkg-config` integration
- Improved uninstall flow with user confirmation and verbose output.

### Fixed
- Corrected `pkg-config` file installation paths on Unix systems.

---

## [0.2.5-alpha] – 2026-01

### Added
- Enhanced entity name generation with singularization logic.
- Validation tests for entity naming consistency.
- Improved introspection logging and diagnostics.
- Additional vault-mode checks during introspection.

### Fixed
- Paranoid mode validation during `introspect`.
- Connection string resolution issues in introspection workflows.

---

## [0.2.4-alpha]

### Changed
- Minor internal version upgrade and stabilization.

---

## [0.2.3-alpha]

### Added
- `status` and `verify` commands for:
  - Schema integrity
  - Vault consistency
  - Engine readiness

---

## [0.2.2-alpha]

### Added
- Mutation execution layer with:
  - Full PostgreSQL error mapping
  - Structured error propagation across FFI
- IdentityMap-based row deduplication during query execution.
- Improved eager query path handling and relation mapping.

### Changed
- Refactored engine mutation handling and error contracts.
- Migration generation now includes safe FK `DROP` statements.

### Fixed
- Cyclical import issues in mutation factory initialization.
- Improved error diagnostics for constraint violations.

---

## [0.2.1-alpha]

### Changed
- Updated README roadmap and milestone descriptions.

---

## [0.2.0-alpha]

### Added
- Administrative structure (`.chameleon/`)
- Journaling system for schema and migration state.
- Multi-file schema support.
- Updated `init` and `migrate` commands aligned with admin layout.

### Changed
- Configuration loading via `.chameleon.yml`.

---

## [0.1.5-alpha]

### Added
- Administrative structure and journaling foundation.
- Schema lifecycle management.
- Migration command enhancements.

---

## [0.1.4-alpha]

### Added
- Windows support in release pipeline.
- Cross-platform installer updates.

---

## [0.1.3-alpha]

### Changed
- Release workflow simplification and artifact stabilization.

---

## [0.1.2-alpha]

### Changed
- Improved release artifacts with SHA verification.
- Simplified CI matrix.

---

## [0.1.1-alpha]

### Added
- Database introspection infrastructure (PostgreSQL).
- `DATABASE_URL` support for cloud-native deployments.
- Structured debug and query tracing infrastructure.

---

## [0.1.0-beta]

### Added
- `check --json` CLI command for editor and tooling integration.
- Structured JSON error model across:
  - Parser
  - Core
  - FFI
  - CLI
- Rich error output with position, context, and suggestions.

---

## [0.1.0]

### Added
- SQL generator and Go Query Builder.
- PostgreSQL connector and executor.
- End-to-end integration tests with Docker.
- Schema DSL with type checking and validation.
- Migration generation from validated schemas.
- Rust ↔ Go FFI bridge.
- CLI implemented with Cobra.
- Full parser implementation using LALRPOP.
- Documentation:
  - Architecture overview
  - Query reference
  - Project philosophy

---

## [0.0.1]

### Added
- Initial schema parser, AST, and runtime foundations.