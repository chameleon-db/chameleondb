# ChameleonDB — Backlog

**Status:** Candidate for initial GitHub export  
**Version:** 0.2  
**Date:** 2026-11-01

This backlog is the starting point for the initial GitHub export. `scripts/planning/export_backlog.py` creates the issues in GitHub; after export, **GitHub is the source of truth** and this file is no longer edited for routine work. See [README](README.md) for the format and process.

Security findings are deliberately **not** listed here in detail; they are tracked in a private advisory.

## Backlog conventions

- **Milestone** indicates the product horizon to which an outcome belongs. It does not, by itself, define execution order.
- **Priority** indicates relative importance within the relevant horizon.
- **Dependencies** identify work that must happen first.
- **Blockers** are dependencies that genuinely prevent progress.
- Accepted ADRs are binding. Proposed ADRs are hypotheses until accepted.
- A task is complete only when its acceptance criteria are met and supported by evidence.
- A capability mentioned in an issue is not necessarily implemented.

---

## [E0] Repository hygiene and project setup

- type: epic
- labels: type:epic, area:docs, priority:P1
- milestone: H0

Make the repository easy to understand, build and contribute to, for people and coding agents.

**Stories**

- As a new contributor, I want a clear guide to build, test and propose changes, so that I can contribute without asking the maintainer.
- As a user, I want the README and docs to describe what the tool really does today, so that I can decide whether to use it.
- As a coding agent, I want explicit rules and playbooks, so that changes are safe and reviewable.

**Done when** all child issues are closed.

### [E0-01] Remove stray and generated files from version control

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

**Context.** Coverage outputs (`coverage.out`, `cmd_cov.out`, `introspect_cov.out`), `chameleon-core/test_sql_output.rs` and duplicate diagram images are tracked.

**Scope.** Remove them from the index, extend `.gitignore`, and keep one image per diagram. Do not rewrite history.

**Acceptance criteria**
- Given a fresh clone, when I run the test suites, then no tracked file changes.
- Given the repository, when I list tracked files, then no coverage output or scratch file is present.

### [E0-02] Add community and contributor files

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

Add `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, issue and pull request templates.

**Acceptance criteria**
- Given the repository root, when I open the GitHub Community page, then contributing, security, conduct, issue and PR templates are detected.
- Given the issue chooser, when a user starts a vulnerability report, then they are directed to the private reporting channel.
- Given `SECURITY.md`, when I read it, then the supported private reporting process and responsible disclosure expectations are clear.

### [E0-03] Adopt product, architecture and development documents in the repository

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

Add and link the current versions of:
- `docs/adr/`
- `docs/product-definition.md`
- `docs/vision-and-foundations.md`
- `docs/architecture/overview.md`
- `docs/cli-reference.md`
- `IMPLEMENTATION_PLAN.md`

Add links from the README. Confirm final document names and locations before creating duplicate copies.

**Acceptance criteria**
- Given the README, when I follow the documentation links, then I reach the product definition, vision, ADR index, architecture overview, CLI reference and implementation plan.
- Given any document, when I read it, then current behavior and future proposals are clearly distinguished.
- Given conflicting guidance, when I compare documents, then the documented hierarchy identifies which source takes precedence.

### [E0-04] Correct overclaims, dates and inconsistencies in README and docs

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P0
- milestone: H0

Run a claims review ([ADR-0002](../adr/0002-honest-claims-policy.md)) over the README and `docs/en`.

**Scope.** Remove calendar dates from the roadmap; align Go and Rust versions with `go.mod` and the build; describe integrity guarantees with their real scope (detects changes made outside the tool; not resistant to a privileged local user); remove references to methods that do not exist; add a short notice that the alpha is not recommended for untrusted input or production use until hardening work is finished.

**Acceptance criteria**
- Given each behavioral statement in the README, when I look for the code and test that support it, then I find them or the statement has been softened or removed.
- Given the README roadmap, when I read it, then it contains no calendar dates.
- Given the README, when I read the status section, then it states that the project is alpha and links to `SECURITY.md`.
- Given a public security or integrity claim, when I review it, then its scope and limitations are explicit.

### [E0-05] Decide the fate of the Spanish documentation

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P2
- milestone: H1a

`docs/sp` duplicates `docs/en`. Decide to archive it, remove it, or keep it as a translation generated from the English source.

**Acceptance criteria**
- Given `docs/`, when I look at the Spanish content, then it is either removed, clearly marked as an outdated archive, or generated from the English source with instructions.

### [E0-06] Reconcile versions and the changelog

- type: task
- epic: E0
- labels: type:task, area:ci, priority:P1
- milestone: H0

The Rust crate says `1.0.0-alpha`, tags go up to `v1.0.1-alpha`, and the changelog stops at `0.2.6-alpha`.

**Acceptance criteria**
- Given a release, when I compare the crate version, Go version output, tag and changelog, then they agree.
- Given the release process, when a version is cut, then a changelog entry exists.
- Given version information in documentation or CLI output, when it is displayed, then it comes from an identifiable source of truth.

### [E0-07] Add development and coding-agent guidance

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

Add `DEVELOPERS.md`, `AGENTS.md`, `.agents/skills/` and any project-specific agent playbooks needed to explain safe contribution and development workflows.

**Acceptance criteria**
- Given a contributor with Rust, Go and Docker, when they follow `DEVELOPERS.md`, then they can build the project and run unit tests without additional help.
- Given a coding agent working in the repository, when it reads `AGENTS.md`, then it can identify project conventions, required checks and security-sensitive areas.
- Given an agent playbook, when it describes a command or workflow, then that command or workflow exists and is reproducible.

### [E0-08] Record the current test baseline and execution evidence

- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

Publish a public-safe `docs/status.md` with the test baseline only (commands, environment, toolchain versions, results), derived from the private STATUS.md. Security findings stay out of it.

**Scope.** Preserve the initial report date and commit reference. Add the later Go tests, Cargo tests/build and integration tests as a separate execution record. Record exact dates, toolchain versions, commands and test counts from terminal output when available. No application code changed between the initial report and those successful test runs.

**Acceptance criteria**
- Given `docs/status.md`, when I read the test history, then the original inspection and later successful execution are clearly separate.
- Given a reported test result, when I look for evidence, then the command, environment and result are recorded or explicitly marked as developer-reported.
- Given the original report, when I read its findings, then limitations of the original review environment are not presented as current failures.
- Given a finding, when I read its status, then it is identified as reproduced, code-inspection finding, reported, fixed, not reproducible or unverified as appropriate.

---

## [E1] Trustworthy core

- type: epic
- labels: type:epic, area:core, area:runtime, priority:P0
- milestone: H0

The existing code behaves predictably, its claims match reality, and releases are protected by automated gates. Nothing new is promised until the required foundation is trustworthy.

**Stories**

- As a user, I want the tool to refuse malformed or hostile input safely, so that using it does not expose my database.
- As a maintainer, I want every pull request to build, lint, test and audit automatically, so that regressions are caught before release.
- As an integrator, I want equivalent supported schemas to have stable representations and identities, so that I can compare and review changes.
- As an installer, I want to verify what I download, so that I can trust the binary I run.

**Done when** all H0 child issues are closed, the required security and determinism checks pass, and the claims review (E0-04) is complete.

### [E1-01] Harden query generation (see private advisory)

- type: bug
- epic: E1
- labels: type:bug, area:core, area:runtime, priority:P0
- milestone: H0

Query generation must handle caller-supplied input safely and validate identifiers.

Details are in a private security advisory and will be published with the fix.

**Dependencies:** developed together with E1-05; both are published in the same change.

**Acceptance criteria**
- Given any supported query built through the public engine API, when SQL is generated, then values are passed as parameters and do not alter SQL structure.
- Given a field, entity or ordering identifier that is not valid for the schema, when a query is built, then it is rejected before SQL execution.
- Given the public query-generation and execution paths, when adversarial tests run, then hostile values are treated as data and invalid identifiers fail closed.
- Given the existing test suites, when they run after the fix, then they pass.
- Given the fix, when the private advisory is reviewed, then its proof of concept no longer reproduces through the affected supported paths.

### [E1-02] Update vulnerable dependencies

- type: task
- epic: E1
- labels: type:task, area:runtime, priority:P0
- milestone: H0

`govulncheck` previously reported vulnerable versions of `pgx` and `golang.org/x/text`. Reassess the current dependency graph, upgrade affected dependencies where warranted, and rerun the audit.

**Acceptance criteria**
- Given the current Go module, when I run `govulncheck ./...`, then every reported finding is triaged for affected version and reachable code path.
- Given an applicable and reachable vulnerability, when a fixed version is available, then the dependency is upgraded and the relevant tests pass.
- Given a remaining finding, when it cannot be remediated, then the reason and residual risk are documented.
- Given CI, when the dependency audit runs, then the result is visible and failures are not silently ignored.

### [E1-03] Fix default-deny lint findings and harden the FFI boundary

- type: task
- epic: E1
- labels: type:task, area:core, priority:P1
- milestone: H0

`cargo clippy` reports default-deny errors on exported functions that dereference raw pointers, plus unwraps and unreachable branches on paths reachable from FFI.

**Acceptance criteria**
- Given the Rust crate, when I run `cargo clippy --all-targets`, then there are no errors.
- Given a null pointer, invalid length, invalid UTF-8 or malformed JSON passed to an exported function, when it is called, then it returns a structured error and does not panic or crash.
- Given the Go bridge, when the core returns an error, then it is surfaced as a Go error.
- Given a Rust panic in a path reachable through the ABI, when it occurs, then it cannot unwind unsafely across the C ABI.
- Given the FFI boundary, when I read its documentation, then ownership, input and output contracts are explicit.

### [E1-04] Make current generated output deterministic

- type: task
- epic: E1
- labels: type:task, area:core, priority:P1
- milestone: H0

Generated SQL, column lists and listings must not depend on hash-map iteration order.

This issue covers deterministic output of existing generation paths. Semantic canonicalization and digest identity are handled by E1-12 and E1-13.

**Acceptance criteria**
- Given the same supported schema and query, when SQL is generated 100 times, then the output is byte-identical.
- Given the same schema, when DDL is generated, then entities and fields appear in a stable, documented order.
- Given repeated listing operations over the same state, when their output is compared, then ordering is stable.
- Given a deterministic behavior requirement, when it is tested, then the test does not depend on incidental process state or hash-map iteration.

### [E1-05] Add adversarial regression tests for query and mutation generation

- type: task
- epic: E1
- labels: type:task, area:core, area:runtime, priority:P0
- milestone: H0

**Scope.** Reproduce the issue described in the private advisory and add regression tests that assert safe properties (for example, values never alter query structure and unknown identifiers are rejected). Cover Rust-level generation and the supported Go engine paths against PostgreSQL.

Develop the tests and the E1-01 fix together, for example in the advisory's temporary private fork, and publish them in the same change. Do not publish failing or revealing tests ahead of the fix.

**Acceptance criteria**
- Given hostile strings as values, when queries and mutations are built and executed against PostgreSQL, then values are treated as data and cannot change query structure.
- Given invalid field, entity or ordering identifiers, when a query is built, then it is rejected before execution.
- Given the E1-01 fix reverted locally, when the regression tests run, then the relevant tests fail.
- Given the supported public query paths, when the tests run, then they exercise the actual Go-to-Rust-to-PostgreSQL path where applicable.
- Given a test failure, when it occurs, then it identifies the affected query path and input class.
- Given the repository's public history, when I review the tests and commit messages, then they contain no proof of concept from the private advisory and no description of the exploitation steps.

**Acceptance criteria**
- Given hostile strings as values, when queries and mutations are built and executed against PostgreSQL, then values are treated as data and cannot change query structure.
- Given invalid field, entity or ordering identifiers, when a query is built, then it is rejected before execution.
- Given the E1-01 fix reverted locally, when the regression tests run, then the relevant tests fail.
- Given the supported public query paths, when the tests run, then they exercise the actual Go-to-Rust-to-PostgreSQL path where applicable.
- Given a test failure, when it occurs, then it identifies the affected query path and input class.
- Given the repository's public history, when I review the tests and commit messages, then they contain no proof of concept from the private advisory and no description of the exploitation steps.

### [E1-06] Restrict developer servers to loopback and protect endpoints

- type: task
- epic: E1
- labels: type:task, area:runtime, priority:P1
- milestone: H0

The playground and dashboard HTTP servers must not be reachable from the network by default.

**Acceptance criteria**
- Given the default configuration, when a server starts, then it binds to `127.0.0.1` only.
- Given a request without a valid token, when it reaches a protected endpoint, then it is rejected.
- Given token generation and validation, when I review the implementation, then tokens are generated securely, are not exposed in logs, and are compared safely.
- Given a non-loopback binding, when it is explicitly configured, then the behavior and security implications are documented.
- Given the repository, when I inspect available servers and endpoints, then the issue covers every applicable developer server.

### [E1-07] Verify downloads in the install script and publish checksums

- type: task
- epic: E1
- labels: type:task, area:ci, priority:P1
- milestone: H0

**Acceptance criteria**
- Given a release, when it is published, then a checksums file is published with the downloadable artifacts.
- Given the install script, when the checksum of a downloaded artifact does not match, then installation stops with a clear error.
- Given a release artifact, when its checksum is published, then the checksum corresponds to the exact artifact and is reproducible.
- Given the installation documentation, when I follow it, then the verification process is explained.

### [E1-08] Review the vault threat model and password hashing

- type: task
- epic: E1
- labels: type:task, area:vault, priority:P1
- milestone: H1b

Document what the vault protects against and what it does not; replace single-round salted hashing of the admin password with a standard password-hashing function.

This issue should not block H1a unless the vault or its administrative interface is part of the actual observation path or exposes a relevant security boundary.

**Acceptance criteria**
- Given a newly set admin password, when it is stored, then it uses a memory-hard or iterated password-hashing function with documented parameters.
- Given existing password hashes, when the new implementation is introduced, then they are migrated or rejected with a clear message and no silent downgrade.
- Given the vault's actual role in H1a, when scope is reviewed, then any required hardening is identified as a dependency rather than assumed.

### [E1-09] CI on every pull request: build, lint, test and audit

- type: task
- epic: E1
- labels: type:task, area:ci, priority:P0
- milestone: H0

**Acceptance criteria**
- Given a pull request, when CI runs, then it builds the Rust core and Go CLI, runs `cargo fmt --check`, `cargo clippy`, `cargo test`, `gofmt`, `go vet`, Go unit tests and Go integration tests against PostgreSQL 16, and runs `cargo audit` and `govulncheck`.
- Given a failing required step, when the pull request is evaluated, then the check fails.
- Given repository branch protection settings, when a pull request is merged, then required checks must pass before merging.
- Given an optional or non-blocking check, when it fails, then its status is clearly distinguished from required checks.
- Given the CI configuration, when I inspect it, then no required check is masked by `|| true` or equivalent behavior.

### [E1-10] Portable integration test setup

- type: task
- epic: E1
- labels: type:task, area:ci, priority:P1
- milestone: H0

Integration tests must not depend on one developer's machine: connection settings come from environment variables, and the docs cover Docker and Podman.

**Acceptance criteria**
- Given a machine with Docker, when I follow `DEVELOPERS.md`, then `make test-integration` passes.
- Given a machine with Podman, when I follow the documented steps, then integration tests pass, or the documentation states the limitation.
- Given an existing PostgreSQL instance, when I set the documented environment variable, then tests run against it without starting a container.
- Given integration test configuration, when I inspect it, then credentials and connection details are not hardcoded in the repository.

### [E1-11] Consolidate duplicated mutation logic

- type: task
- epic: E1
- labels: type:task, area:core, area:runtime, priority:P2
- milestone: H1b

Mutation SQL is generated in Rust (currently unused) and in Go (currently used). Keep one implementation after the language/runtime boundary and the role of the data-access layer are decided.

**Dependencies:** ADR-0007 and ADR-0011 decisions.

**Acceptance criteria**
- Given the repository, when I search for mutation SQL generation after consolidation, then exactly one implementation exists and it is the one the engine uses.
- Given the integration tests, when they run, then they pass.
- Given the architectural decision, when I review the implementation, then there is no second active implementation with divergent behavior.
- Given the migration, when it is complete, then the chosen implementation boundary is documented.

### [E1-12] Spike: canonical schema digest

- type: spike
- epic: E1
- labels: type:spike, area:core, priority:P1
- milestone: H0

Time box: 2–3 days. Implements the experiment of [ADR-0005](../adr/0005-canonical-schema-digest.md) and informs E1-13 and E2-02. The canonical model must be constructible from both the DSL and database introspection for the supported subset.
The spike uses a provisional subset (tables, columns, types, nullability, defaults, primary, foreign and unique keys) that E2-01 later formalizes.

**Acceptance criteria**
- Given two schemas that differ only in formatting, comments or declaration order of independent items, when their canonical identities are computed, then they are equal.
- Given two schemas that differ semantically within the supported subset, when their canonical identities are computed, then they differ.
- Given a database created from schema S, when it is introspected and its canonical identity computed, then it equals the identity of S from the DSL for the supported subset, or the differences are listed and explained.
- Given unsupported schema constructs, when the spike evaluates them, then they are identified rather than silently discarded.
- Given the results, when the spike ends, then ADR-0005 is updated to Accepted, revised or rejected with evidence and explicit trade-offs.

### [E1-13] Implement canonical model and schema digest

- type: story
- epic: E1
- labels: type:story, area:core, area:runtime, priority:P1
- milestone: H0

As a developer and integrator, I want a stable semantic representation and digest for a supported schema, so that schema states can be compared independently of formatting.

**Dependencies:** E1-12; ADR-0005 and relevant ADR-0008 decisions.

**Acceptance criteria**
- Given a valid `.cham` schema, when it is parsed and canonicalized, then a versioned canonical representation and digest are produced.
- Given the canonical model definition, when I inspect it, then it is independent of any front-end and does not leak PostgreSQL-specific types ([ADR-0010](../adr/0010-backend-strategy-and-capability-declarations.md)).
- Given equivalent supported schemas with different formatting or declaration order of independent items, when their digests are computed, then the digests are identical.
- Given semantically different supported schemas, when their digests are computed, then the digests differ.
- Given an unsupported or uninterpretable object, when canonicalization runs, then the limitation is reported.
- Given a digest, when it is displayed or stored, then it identifies its canonical format version and hash algorithm.
- Given the canonical format, when its semantics change incompatibly, then a new format version is used.

### [E1-14] Spike: implementation languages, boundary and distribution

- type: spike
- epic: E1
- labels: type:spike, area:ci, priority:P1
- milestone: H0

Time box: 3 days. Implements the spike of [ADR-0011](../adr/0011-language-runtime-and-distribution.md).

This spike informs the architecture and packaging strategy; it does not automatically block H1a unless the result identifies a concrete obstacle to building or distributing the observation workflow.

**Acceptance criteria**
- Given the spike report, when I read it, then it contains installation steps, binary size, build time and cross-compilation results for each option tried.
- Given each option, when I compare it with the current Rust/Go/cgo design, then maintenance cost, FFI complexity, user installation and migration effort are documented.
- Given the results, when the spike ends, then ADR-0011 is accepted, revised or rejected with evidence and a migration estimate for CLI commands.

### [E1-15] Expose a public `digest` CLI command

- type: story
- epic: E1
- labels: type:story, area:runtime, priority:P2
- milestone: H1a

As a developer, I want to print the stable identity of a schema or database, so that I can inspect and compare schema identities directly.

This is a public CLI interface decision, separate from implementing the canonical model. Its scope should be confirmed against the H1a workflow before implementation.

**Dependencies:** E1-13 and database introspection support from E2-02.

**Acceptance criteria**
- Given a `.cham` file, when I run `chameleon digest file.cham`, then it prints a versioned digest and exits with code 0.
- Given the same file formatted differently, when I run the command, then the digest is identical.
- Given an invalid schema, when I run the command, then it prints diagnostics and exits with a non-zero code.
- Given a database URL and a read-only role, when I run the command, then it prints the digest of the introspected supported subset and lists unsupported objects.
- Given the H1a workflow review, when the command is evaluated, then its public interface is justified by a concrete user need and does not duplicate another command unnecessarily.

### [E1-16] Document the vault threat model and its exclusions

- type: task
- epic: E1
- labels: type:task, area:vault, area:docs, priority:P1
- milestone: H0

State in `docs/en/SECURITY.md` what the vault and integrity modes protect against and what they do not (for example: detects edits outside the tool; does not resist someone who can edit both files and hashes; modes do not replace database permissions).

**Acceptance criteria**
- Given `docs/en/SECURITY.md`, when I read it, then it states the guarantees, the threat model and the exclusions.
- Given the README and CLI help, when I compare them with it, then they make no stronger claim.

---

## [E2] Observe and drift

- type: epic
- labels: type:epic, area:core, area:runtime, priority:P1
- milestone: H1a

A developer can point the tool at a real PostgreSQL database, with read-only access, and learn how it differs from an explicit reference state.

**Stories**

- As a developer, I want to see how my production database differs from a reference schema, so that I notice unexpected changes before they hurt me.
- As a team lead, I want structural differences described clearly and, where evidence permits, classified by risk, so that I know which ones need attention first.
- As an operator, I want observation to need only read-only credentials, so that I can run it safely.
- As a developer without a `.cham` file, I want to take a baseline of my database and compare against it later, or compare two environments, so that I get value before adopting anything else ([Product Definition, section 5.1](../product-definition.md)).

**Done when** developers outside the project run the read-only workflow against real databases, obtain a first report within the defined conditions, and confirm meaningful outcomes.

### [E2-01] Specify the supported PostgreSQL subset

- type: task
- epic: E2
- labels: type:task, area:docs, priority:P1
- milestone: H1a

List which object types, types and constraints are supported, observed but not fully interpreted, or excluded. Consider tables, columns, types, nullability, defaults, primary and foreign keys, unique constraints, indexes and other relevant objects. Informs [ADR-0008](../adr/0008-schema-source-dsl-vs-sql-first.md).

**Acceptance criteria**
- Given a PostgreSQL object, when I read the specification, then I can tell whether it is supported, unsupported/uninterpretable, or explicitly excluded.
- Given an unsupported object, when I read the CLI output of `drift`, then it is reported as such and not silently skipped.
- Given a supported object, when it is introspected, then the specification identifies which properties are compared.
- Given a change to the supported subset, when it is released, then the subset version and documentation are updated.

### [E2-02] Implement faithful introspection for the supported subset

- type: story
- epic: E2
- labels: type:story, area:runtime, priority:P1
- milestone: H1a

**Dependencies:** E1-12 spike and E1-13 canonical model; E2-01 supported-subset specification.

**Acceptance criteria**
- Given a database created from schema S, when I introspect it, then the resulting model compares equal to S for the supported subset.
- Given a database object outside the supported subset, when I introspect, then it is listed as unsupported or uninterpretable.
- Given a read-only database role, when I introspect, then it succeeds and writes nothing.
- Given an object that cannot be faithfully interpreted, when introspection completes, then the limitation is surfaced rather than silently normalized away.
- Given an introspected database within the supported subset, when it is canonicalized, then it uses the canonical model from E1-13 and its digest equals that of an equivalent `.cham` schema.

### [E2-03] Describe and classify schema differences

- type: story
- epic: E2
- labels: type:story, area:core, priority:P1
- milestone: H1a

Describe structural differences between an explicit reference and an observed database. Keep structural classification distinct from risk assessment.

**Acceptance criteria**
- Given a reference and a database with an extra column, when I compare them, then the difference is reported as present in the database but absent from the reference, with the object name.
- Given a database missing a declared table, when I compare, then it is reported as present in the reference but absent from the database.
- Given a column whose type differs, when I compare, then both types are reported.
- Given any reported difference, when I read it, then it identifies the reference and observed state.
- Given any report, when I read it, then it states the supported subset and that ChameleonDB will not change the database automatically.
- Given insufficient context to assess execution risk, when a difference is reported, then risk is marked unknown or not assessed rather than presented as a guarantee.

### [E2-04] Implement `drift` command with text and JSON reports

- type: story
- epic: E2
- labels: type:story, area:runtime, priority:P1
- milestone: H1a

**Dependencies:** E2-01, E2-02, E2-03 and E2-08.

**Acceptance criteria**
- Given a reference (schema file, snapshot or second database) and a database URL, when I run `chameleon drift`, then I get a human-readable report.
- Given no differences, when the command completes, then it returns the documented no-difference exit code.
- Given differences, when the command completes, then it returns the documented differences-found exit code.
- Given a connection failure, invalid reference or inability to evaluate the requested comparison, when the command completes, then it returns a distinct error exit code.
- Given `--output json`, when I run it, then the output validates against the documented, versioned JSON structure.
- Given unsupported objects, when the command completes, then they are explicitly listed in the report.
- Given any execution, when the command completes, then neither the reference nor observed database has been modified.

### [E2-05] Guide: running observation with a read-only role

- type: task
- epic: E2
- labels: type:task, area:docs, priority:P2
- milestone: H1a

Document how to create a least-privilege role for observation in PostgreSQL and verify that the tool works with it.

**Acceptance criteria**
- Given the guide, when I follow it on a clean PostgreSQL 16 instance, then the drift report runs.
- Given the role, when I inspect its grants and effective privileges, then it has only the permissions required for observation.
- Given the observation workflow, when it runs with this role, then no database writes are performed.
- Given the role, when I attempt a write in a separate verification test, then PostgreSQL refuses it.

### [E2-06] Store baseline snapshots of a database

- type: story
- epic: E2
- labels: type:story, area:runtime, area:core, priority:P1
- milestone: H1a

As a developer without a declared schema, I want to store a snapshot of my database's current schema state, so that I can later see what changed.

**Dependencies:** E1-13 and E2-02.

**Acceptance criteria**
- Given a database URL, when I run `chameleon snapshot`, then a local snapshot is stored with its canonical digest, canonical format version, supported-subset version, ChameleonDB version, provenance and timestamp.
- Given two snapshots of an unchanged database, when I compare their schema digests, then they are equal even if timestamps differ.
- Given a snapshot, when I inspect it, then it contains no data rows or credentials.
- Given unsupported objects, when a snapshot is taken, then they are listed as unsupported in it.
- Given a snapshot, when I inspect its metadata, then the timestamp does not affect the semantic schema digest.

### [E2-07] Compare two environments

- type: story
- epic: E2
- labels: type:story, area:runtime, area:core, priority:P1
- milestone: H1a

As a developer, I want to compare two databases (for example production and staging), so that I can see how they differ without declaring a schema.

**Dependencies:** E2-02 and E2-03.

**Acceptance criteria**
- Given two database URLs with read-only roles, when I run the comparison, then differences are reported in both directions (present in A, not in B; present in B, not in A; differs).
- Given objects outside the supported subset in either database, when I run the comparison, then they are listed as unsupported or not compared.
- Given the run, when it ends, then neither database was modified.
- Given a comparison report, when I read it, then I can identify each environment without exposing connection credentials.

### [E2-08] Define and test the drift report contract

- type: task
- epic: E2
- labels: type:task, area:core, area:docs, priority:P1
- milestone: H1a

Define the information that every text and JSON drift report must contain. Keep the contract independent of a particular CLI presentation.

**Acceptance criteria**
- Given a report, when I inspect it, then it identifies the reference state and observed state.
- Given a report, when I inspect it, then it identifies the canonical format and supported-subset version.
- Given unsupported or uninterpretable objects, when I inspect it, then they are explicitly listed.
- Given a structural difference, when I inspect it, then the affected object, property and values are identifiable where available.
- Given insufficient information to classify risk, when I inspect it, then uncertainty is explicit.
- Given a JSON report, when I validate it, then it conforms to a documented versioned schema.
- Given a future report-format change, when compatibility is affected, then the format version is updated.

### [E2-09] Protect credentials and sensitive metadata in observation

- type: task
- epic: E2
- labels: type:task, area:runtime, area:core, priority:P1
- milestone: H1a

Review the observation workflow for accidental disclosure of credentials and sensitive metadata in logs, errors, snapshots and reports.

**Acceptance criteria**
- Given a database connection string containing credentials, when commands run or fail, then credentials are not written to logs or user-facing errors.
- Given a snapshot, when it is stored, then it contains no credentials or database data rows.
- Given an error, when it is displayed, then it does not expose secrets from connection configuration.
- Given report and snapshot contents, when I review them, then the documented metadata scope is clear and unnecessary sensitive information is excluded.
- Given tests for logging and serialization, when they run, then representative credentials and secret values are not present in output.

### [E2-10] Validate the H1a workflow with external developers

- type: task
- epic: E2
- labels: type:task, area:docs, priority:P1
- milestone: H1a

Validate the complete read-only workflow with five developers outside the project using real PostgreSQL databases.

**Dependencies:** A usable end-to-end H1a workflow, including E2-04 and the required reference workflow.

**Acceptance criteria**
- Given five external developers, when they follow the documented workflow with a ready binary, prepared credentials and connectivity, then each can attempt to produce a report.
- Given each session, when it is recorded, then setup time, blockers, completion time and feedback are documented.
- Given the sessions, when results are summarized, then meaningful outcomes are distinguished as discovery of an unexpected difference, confirmation of expected state or reduced effort.
- Given the results, when H1a is evaluated, then at least three developers report a meaningful outcome.
- Given the results, when the milestone is reviewed, then the under-10-minute target is evaluated under the stated prerequisites, not as an unconditional promise.

---

## [E3] Gated apply

- type: epic
- labels: type:epic, area:core, area:runtime, priority:P2
- milestone: H1b

Risk-classified change plans, a gate for destructive and locking changes, and role-based verification of the control level ([ADR-0006](../adr/0006-levels-of-control-and-role-enforcement.md)).

To be refined after E1 and E2 produce evidence. H1b starts only when H1a has demonstrated meaningful value.

### [E3-01] Refine the scope of epic E3

- type: task
- epic: E3
- labels: type:task, area:docs, priority:P2
- milestone: H1b

Write user stories and acceptance criteria for E3 using what E1 and E2 taught.

**Acceptance criteria**
- Given the refinement, when it is complete, then E3 contains child issues with observable acceptance criteria and dependencies.
- Given the Product Definition, when E3 is refined, then the exit signal for H1b is consistent with the product horizon.
- Given ADR-0006, when E3 is refined, then observe, mediated and enforced control are distinguished accurately.
- Given H1a validation, when E3 is scoped, then only validated user needs are included in the initial controlled-change workflow.

---

## [E4] Validation and research

- type: epic
- labels: type:epic, area:docs, priority:P1
- milestone: H1a

Gather evidence to guide product and architectural decisions through developer interviews and time-boxed spikes. Research tasks must not silently become implementation commitments.

### [E4-01] Interview developers about schema-change incidents

- type: task
- epic: E4
- labels: type:task, area:docs, priority:P1
- milestone: H1a

Interview 10–15 developers (indie developers and teams of 2–10 using PostgreSQL) about their last schema-related incident, how they detect differences between environments, and how AI agents touch their databases. Do not present ChameleonDB first.

**Acceptance criteria**
- Given the interviews, when they finish, then notes are summarized in a document that states which hypotheses were supported, contradicted or untested.
- Given each interview, when it is conducted, then questions focus on actual past behavior and incidents rather than hypothetical interest in ChameleonDB.
- Given the summary, when conclusions are drawn, then evidence and interpretation are distinguished.

### [E4-02] Interview developers of offline and field-data applications

- type: task
- epic: E4
- labels: type:task, area:docs, priority:P2
- milestone: H3

Interview 8–10 developers of offline-first or field-collection applications ([ADR-0009](../adr/0009-offline-first-and-distributed-data.md)).

This research is deferred until the initial PostgreSQL observation problem has been validated or a clear opportunity justifies parallel discovery.

**Acceptance criteria**
- Given the interviews, when they finish, then the summary states whether schema evolution across long-offline replicas is a recurring pain.
- Given the findings, when they are reviewed, then they are recorded as evidence for or against ADR-0009 rather than treated as implementation approval.

### [E4-03] Spike: Git as history substrate and lifecycle states on references

- type: spike
- epic: E4
- labels: type:spike, area:vault, priority:P2
- milestone: H1b

Time box: 2 days. [ADR-0004](../adr/0004-git-as-history-substrate.md).

**Acceptance criteria**
- Given the spike report, when I read it, then it shows whether revisions, references and lifecycle states can be represented in Git and what breaks.
- Given the experiment, when it ends, then trade-offs, operational costs and unresolved questions are documented.
- Given the results, when ADR-0004 is updated, then it is accepted, revised or rejected based on evidence.

### [E4-04] Spike: levels of control and role enforcement

- type: spike
- epic: E4
- labels: type:spike, area:vault, priority:P2
- milestone: H1b

Time box: 2 days. [ADR-0006](../adr/0006-levels-of-control-and-role-enforcement.md).

**Acceptance criteria**
- Given the spike report, when I read it, then it shows a working check that distinguishes observe, mediated and enforced control against a real PostgreSQL instance.
- Given the experiment, when it ends, then it documents required database roles, effective privileges and residual bypass paths.
- Given the results, when ADR-0006 is updated, then it is accepted, revised or rejected based on evidence.

### [E4-05] Spike: schema source — DSL, SQL import and round trip

- type: spike
- epic: E4
- labels: type:spike, area:core, priority:P2
- milestone: H1a

Time box: 3 days, after E1-12. [ADR-0008](../adr/0008-schema-source-dsl-vs-sql-first.md).

This spike informs the role of `.cham` and adoption paths. It must not make DSL adoption a prerequisite for the initial database-observation workflow.

**Dependencies:** E1-12.

**Acceptance criteria**
- Given three realistic schemas, when each is authored in the DSL, represented in SQL and imported from a database, then the canonical models are compared and gaps are recorded.
- Given unsupported or non-round-trippable constructs, when the spike completes, then they are listed with their implications.
- Given the results, when ADR-0008 is updated, then the decision and remaining open questions are explicit.

### [E4-06] Spike: SQLite adapter for the portable subset

- type: spike
- epic: E4
- labels: type:spike, area:core, priority:P2
- milestone: H2

Time box: 1 week, after E1-12 and E4-05. [ADR-0010](../adr/0010-backend-strategy-and-capability-declarations.md).

**Acceptance criteria**
- Given a minimal SQLite adapter, when the same conformance checks run against PostgreSQL and SQLite, then capability differences are documented.
- Given unsupported or non-portable constructs, when the spike completes, then they are reported rather than silently treated as equivalent.
- Given the results, when ADR-0010 is updated, then the decision to proceed, defer or reject the adapter is explicit.

### [E4-07] Prototype: compatibility verdict between schema revisions

- type: spike
- epic: E4
- labels: type:spike, area:core, priority:P2
- milestone: H3

Time box: 1 week, after E1-13. [ADR-0009](../adr/0009-offline-first-and-distributed-data.md).

**Acceptance criteria**
- Given real historical schema changes from three to five open-source projects, when a compatibility verdict is computed and reviewed manually, then the classification is evaluated for correctness and explainability.
- Given incorrect or ambiguous verdicts, when the prototype is evaluated, then failure cases are listed.
- Given the results, when ADR-0009 is reviewed, then the evidence is recorded without implying that a production distributed-data capability exists.

---

## Deferred or non-blocking work

The following work is intentionally not part of the H1a critical path:

- Consolidation of mutation logic (E1-11), pending ADR-0007 and ADR-0011.
- Vault hardening (E1-08), unless the vault is part of the observation workflow or its current exposure creates a direct blocker.
- Public `digest` CLI command (E1-15), pending confirmation of user value and H1a interface needs.
- Git history substrate (E4-03), controlled enforcement (E4-04), SQLite adapter (E4-06) and distributed/offline-first compatibility (E4-02, E4-07), according to their horizons.

These items are deferred, not necessarily rejected.

## Initial execution order

The following is the intended order of work, not a requirement that every item be completed serially. Independent documentation and CI work may proceed in parallel.

**Parallel track from the start:** E4-01 developer interviews. They have no code dependency and may change the scope of E2.

1. **E1-01 + E1-05:** reproduce the issue described in the private advisory, develop the fix and its regression tests together, and verify the supported end-to-end paths.
2. **E1-02:** reassess and remediate applicable dependency vulnerabilities.
3. **E1-03:** harden the FFI boundary.
4. **E0-08:** record the actual test baseline and evidence.
5. **E0-01, E0-02, E0-03, E0-04, E0-06, E0-07, E1-09, E1-10:** repository, documentation and CI foundations; parallelize where practical.
6. **E1-12:** run the canonical digest spike and update ADR-0005.
7. **E1-13:** implement the canonical model and digest.
8. **E2-01, E2-02, E2-03, E2-06, E2-07, E2-08, E2-09:** establish the supported subset and observation foundations.
9. **E2-04 and E2-05:** complete and document the read-only workflow.
10. **E2-10:** validate the complete read-only workflow with external developers.
11. **E3-01:** refine controlled change only after H1a evidence.

E1-14 informs architecture and distribution but is not automatically a blocker for H1a. E4-05 informs DSL and SQL ingestion decisions and should be coordinated with E1-12 without making DSL adoption mandatory.

## Definition of done

A task is not complete merely because code was written or a local command succeeded. Each issue should state, as applicable:

- problem and intended outcome;
- scope and non-goals;
- relevant product capability and ADR;
- dependencies and assumptions;
- observable acceptance criteria;
- unit, integration, regression or security tests;
- failure and unsupported-input behavior;
- documentation and public-claim implications;
- evidence used to mark the task complete.

For security-sensitive tasks, include the threat model and exact public paths tested. For schema behavior, include supported-subset and determinism coverage.
