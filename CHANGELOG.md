# Changelog
All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog,
and this project adheres to Semantic Versioning (pre-1.0).

---

## [Unreleased]

### Added
- A `Makefile` at the repository root that delegates to `chameleon/Makefile` and `cargo` (`make help` lists the targets).

### Changed
- `make tag` moved to the root (`scripts/tag-release.sh`). It now refuses to tag unless the branch is `main` and equal to `origin/main`, and the CHANGELOG and RELEASE.md already describe the version; the version can be passed as `make tag VERSION=X.Y.Z`; output messages are in English; the release commit no longer carries a `Co-authored-by` trailer.
- `chameleon init` no longer writes `features.rollback_enabled`, `features.audit_logging`, `features.backup_on_migrate` or `safety.backup_before_apply`. None of them had any behavior behind it. Files that still set one of them to `true` keep loading, and every command prints one warning per key saying it is not implemented and has no effect.
- `migrate --apply` always prints that ChameleonDB creates no backup (before, it printed it only when `backup_on_migrate` was set).
### Fixed
- `Include` on a belongs-to relation (`author: User`) failed with a missing-foreign-key error. It now loads the referenced rows, once each, and skips NULL keys. A filter through a belongs-to relation (`author.name`) now joins on the local key.

### Changed
- The key of a belongs-to relation is resolved from the `via` of the has-many on the other entity that points back, or from a field named `<relation>_id`. `chameleon validate` now reports a belongs-to relation whose key cannot be resolved (or is ambiguous, or has a type different from the target's `id`) instead of leaving the error to the first query.
- **Go runtime:** `engine.NewEngineAt(projectDir)` opens the project in a given directory, so a program no longer has to run from the project root. `engine.NewEngine()` does the same through `CHAMELEON_PROJECT_DIR` and keeps using the working directory when the variable is not set.
- The errors of `NewEngine`/`NewEngineAt` name the directory that was searched. A missing vault says to run `chameleon migrate --apply` or to point to the project, and a failed integrity check says which check failed (it said only "integrity check failed").

## [v1.1.3] - 2026-11-08

### Fixed
- The install script requested `chameleon-darwin-<arch>.tar.gz`, but releases published `chameleon-macos-universal.tar.gz`, so the install failed with a 404 on macOS. It now requests the archive that the release actually publishes.
- On an unsupported platform or a missing archive, the script says which archive it looked for.

### Changed
- **Release assets:** the macOS archive is now `chameleon-macos-arm64.tar.gz`. It was named `universal`, but it was built on an Apple Silicon runner and contains arm64 binaries only. Intel Macs have no prebuilt binary; the install script says so and points to the build from source.
- Linux archives are built on Ubuntu 22.04 to lower the glibc version they require.
- The macOS binary is linked with an rpath to `/usr/local/lib`, where the install script puts `libchameleon.dylib`.
- The Windows checksum file uses the same format as the others (`<hash>  <file>`).

### Added
- Linux arm64 archive (`chameleon-linux-arm64.tar.gz`).
- The `.sha256` file of each archive is now attached to the release. The install script verifies it and aborts on a mismatch; if the release has no checksum, or no `sha256sum`/`shasum` is available, it warns and continues.
- `CHAMELEON_RELEASE_URL` overrides the download location of the install script (mirrors, testing a release before announcing it).

## [v1.1.2] - 2026-11-06

### Fixed
- Deleting a record that still has children now returns `ForeignKeyConstraintError` (with the dependent table) instead of a misleading `ForeignKeyError` that said the referenced record did not exist.
- `DELETE` filters are emitted in a deterministic order, as in `UPDATE`.
- `SafetyError` suggestions no longer mention `ForceUpdateAll` and `ForceDeleteAll`, which do not exist. They now say to use `Filter()`.
- `chameleon status` looked only for `schema.cham` in the current directory, so projects created with `init` (schemas in `schemas/`) never showed whether the schema was up to date. It now reads `schema.paths` from `.chameleon.yml` and merges the files as `migrate` does.
- `chameleon validate` and `chameleon check` with no file validate `schema.cham` if it exists and otherwise the project schemas, instead of failing on projects that use `schema.paths`.
- `CHAMELEON_DEBUG` was defined but never read. The engine now honors it (`1`, `true` or `sql`; `trace`; `explain`), and so do mutations.
- `.Debug()` on a query with `Include()` now prints the eager queries too, not only the main one. At trace level it adds the duration and row count of each.
- The SQL of a query was generated twice per execution (two passes through the core library); it is now generated once.

### Changed
- **Behavior change:** in mutations, `like` and `ilike` filters now mean "contains", as in the query builder: `Filter("name", "like", "ana")` matches `%ana%`. `%`, `_` and `\` in the value are escaped and match literally. Code that passed its own `%` pattern must drop the wildcards.
- **Behavior change:** a `nil` filter value in a mutation becomes `IS NULL` (`eq`) or `IS NOT NULL` (`neq`, `ne`). Before, it produced `= NULL`, which never matches. Any other operator with `nil` returns an error.
- Debug output redacts the values of columns whose name contains `password`, `passwd`, `secret`, `token`, `api_key`, `apikey`, `private_key`, `credential`, `hash` or `salt`, for queries and mutations (`$1 = [REDACTED]`). Other values are still printed. A parameter whose column cannot be determined is redacted too.
- Mutation debug output goes through the engine's debug context (its writer and level) instead of printing directly to stdout.
- The default `.chameleon/.gitignore` also ignores `vault/mode_auth.json`, the verifier of the mode password. The rest of `vault/` stays versioned. `config auth set-password` adds the rule to `.gitignore` files created by earlier versions. A file that was already committed stays tracked until you run `git rm --cached .chameleon/vault/mode_auth.json`.
- `chameleon version` prints one line; `--verbose` no longer shows separate CLI and core versions, because both came from the same place.
- Debug traces of the FFI layer use `CHAMELEON_DEBUG` instead of `DEBUG=1`.

### Removed
- Dead duplicate error types in `pkg/engine/mutation/errors.go` (the engine's own definitions are the ones in use).

### Added
- Integration tests for `IS NULL` filters, `like`/`ilike` as "contains" with a literal `%`, and deleting a parent that still has children.
- More unit tests for the CLI commands and the debug output. `make test` now prints a summary of the failing packages and tests.

## [v1.1.1] - 2026-11-06

### Fixed
- The error for a `uuid()` default on a non-uuid field now names `uuid()` instead of `uuid_v4()`.
- Self-referential has-many relations (for example `children: [Category] via parent_id`) are no longer reported as a circular dependency. Cycles between two or more entities are still rejected.
- `introspect` recognizes `timestamp without time zone`, the type PostgreSQL reports for the `TIMESTAMP` columns ChameleonDB itself creates. Before, those columns were introspected as `string`.
- Journal: details and `duration_ms` are read back, so `journal migrations` shows the Version and Duration columns instead of empty ones.
- Journal: error messages are no longer cut at the first space in `journal errors` and `journal last`.
- Journal: entries are written deterministically (detail keys sorted) and every value that needs it is quoted, so an entry is always one line. A value with a newline can no longer forge a second entry or change the status.
- Journal: `Log` and `logEntry` were the same code twice; `Log` now delegates.
- `journal` output truncates by characters, not bytes, so a multi-byte character is never split.

### Changed
- `validate` now rejects, before `migrate` reaches PostgreSQL:
  - a field that is both `primary` and `nullable`;
  - `vector(0)` (also inside arrays);
  - a has-many foreign key whose type differs from the parent's primary key type;
  - a has-many whose parent primary key is not named `id`, since the generated foreign key and the query layer both reference `id`.
  Schemas that failed in PostgreSQL for these reasons now fail earlier with a clearer message.
- `introspect` adds a `// WARNING` comment to every column whose PostgreSQL type has no ChameleonDB equivalent (`jsonb`, arrays, `bytea`, `time`, ...) instead of silently using `string`.
- `journal last` reads back across days when today has fewer entries than requested. `journal migrations` and `journal errors` cover all days.
- `journal errors` lists every failed operation: status `error`, `failed`, `denied` or `aborted_readonly`, or any entry with an error message.
- `journal ... --format=json` includes the entry details.

### Added
- `chameleon journal search <text>`, advertised in the help but not implemented until now: case-insensitive match over the whole log line across all days, with `--limit` (default 50, 0 for all).
- Integration test that applies `default uuid()`, literal and `now()` defaults against PostgreSQL and checks the inserted record.

### Documentation
- README: `default uuid()` needs PostgreSQL 13 or newer (`gen_random_uuid()`), or `pgcrypto` on older versions.

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
