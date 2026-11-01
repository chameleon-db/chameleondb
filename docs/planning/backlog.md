# Backlog (source for the initial GitHub export)

This file is the **starting point** of the backlog. `scripts/planning/export_backlog.py` creates the issues in GitHub; after that **GitHub is the source of truth** and this file is no longer edited (new work is created as issues). See [README](README.md) for the format and the process.

Security findings are deliberately **not** listed here in detail; they are tracked in a private advisory.

---

## [E0] Repository hygiene and project setup
- type: epic
- labels: type:epic, area:docs, priority:P1
- milestone: H0

Make the repository easy to understand, build and contribute to, for people and for coding agents.

**Stories**

- As a new contributor, I want a clear guide to build, test and propose changes, so that I can contribute without asking the maintainer.
- As a user, I want the README and docs to describe what the tool really does today, so that I can decide whether to use it.
- As a coding agent, I want explicit rules and playbooks, so that my changes are safe and reviewable.

**Done when** all child issues are closed.

## [E0-01] Remove stray and generated files from version control
- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

**Context.** Coverage outputs (`coverage.out`, `cmd_cov.out`, `introspect_cov.out`), `chameleon-core/test_sql_output.rs` and duplicate diagram images are tracked.

**Scope.** Remove them from the index, extend `.gitignore`, keep one image per diagram. Do not rewrite history.

**Acceptance criteria**
- Given a fresh clone, when I run the test suites, then no tracked file changes.
- Given the repository, when I list tracked files, then no coverage output or scratch file is present.

## [E0-02] Add community, contributor and agent files
- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

Add `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `DEVELOPERS.md`, `AGENTS.md`, `.agents/skills/*`, issue and pull request templates.

**Acceptance criteria**
- Given the repository root, when I open the GitHub "Community" page, then contributing, security, conduct, issue and PR templates are detected.
- Given a contributor with Rust, Go and Docker, when they follow `DEVELOPERS.md`, then they build and run the unit tests without other help.
- Given the issue chooser, when a user starts a vulnerability report, then they are directed to the private channel.

## [E0-03] Adopt ADRs, vision and architecture documents in the repository
- type: task
- epic: E0
- labels: type:task, area:docs, priority:P1
- milestone: H0

Add `docs/adr`, `docs/product-definition.md`, `docs/vision-and-foundations.md`, `docs/architecture/overview.md`, `docs/cli-reference.md` and link them from the README.

**Acceptance criteria**
- Given the README, when I follow the documentation links, then I reach the product definition, the vision, the ADR index, the architecture overview and the CLI reference.
- Given any document, when I read it, then current behaviour and future proposals are clearly separated.

## [E0-04] Correct overclaims, dates and inconsistencies in README and docs
- type: task
- epic: E0
- labels: type:task, area:docs, priority:P0
- milestone: H0

Run a claims review ([ADR-0002](../adr/0002-honest-claims-policy.md)) over the README and `docs/en`.

**Scope.** Remove calendar dates from the roadmap; align Go and Rust versions with `go.mod` and the build; describe integrity guarantees with their real scope (detects changes made outside the tool; not resistant to a privileged local user); remove references to methods that do not exist; add a short notice that the alpha is not recommended for untrusted input or production use until hardening work is finished.

**Acceptance criteria**
- Given each behavioural statement in the README, when I look for the code and test that support it, then I find them or the statement has been softened or removed.
- Given the README roadmap, when I read it, then it contains no calendar dates.
- Given the README, when I read the status section, then it states that the project is alpha and links to `SECURITY.md`.

## [E0-05] Decide the fate of the Spanish documentation
- type: task
- epic: E0
- labels: type:task, area:docs, priority:P2
- milestone: H1a

`docs/sp` duplicates `docs/en`. Decide to archive it, remove it, or keep it as a translation generated from the English source.

**Acceptance criteria**
- Given `docs/`, when I look at the Spanish content, then it is either removed, clearly marked as an outdated archive, or generated from the English source with a note on how.

## [E0-06] Reconcile versions and the changelog
- type: task
- epic: E0
- labels: type:task, area:ci, priority:P1
- milestone: H0

The Rust crate says `1.0.0-alpha`, tags go up to `v1.0.1-alpha`, and the changelog stops at `0.2.6-alpha`.

**Acceptance criteria**
- Given a release, when I compare the crate version, the Go version output, the tag and the changelog, then they agree.
- Given the release process, when a version is cut, then a changelog entry exists.

---

## [E1] Trustworthy core
- type: epic
- labels: type:epic, area:core, area:runtime, priority:P0
- milestone: H0

The existing code behaves predictably, its claims match reality, and releases are protected by automated gates. Nothing new is promised until this is done.

**Stories**

- As a user, I want the tool to refuse malformed or hostile input safely, so that using it does not expose my database.
- As a maintainer, I want every pull request to build, lint, test and audit automatically, so that regressions are caught before release.
- As an integrator, I want the same schema to always produce the same SQL and the same digest, so that I can compare and review changes.
- As an installer, I want to verify what I download, so that I can trust the binary I run.

**Done when** all child issues are closed and the claims review (E0-04) is complete.

## [E1-01] Harden query generation (see private advisory)
- type: bug
- epic: E1
- labels: type:bug, area:core, area:runtime, priority:P0
- milestone: H0

Query generation must never place user-supplied values into SQL text, and must validate every identifier. Details are in a private security advisory and will be published with the fix.

**Acceptance criteria**
- Given any query built through the public engine API, when SQL is generated, then values are passed as parameters and do not appear in the SQL text.
- Given a field, entity or ordering name that is not in the schema, when a query is built, then it is rejected with an error before any SQL is produced.
- Given the existing test suites, when they run, then they pass; and adversarial tests for this behaviour exist (see E1-05).

## [E1-02] Update vulnerable dependencies
- type: task
- epic: E1
- labels: type:task, area:runtime, priority:P0
- milestone: H0

`govulncheck` reports vulnerable versions of `pgx` and `golang.org/x/text`. Upgrade to fixed versions and re-run the audit.

**Acceptance criteria**
- Given the Go module, when I run `govulncheck ./...`, then it reports no vulnerabilities.
- Given the upgrade, when unit and integration tests run, then they pass.

## [E1-03] Fix default-deny lint findings and harden the FFI boundary
- type: task
- epic: E1
- labels: type:task, area:core, priority:P1
- milestone: H0

`cargo clippy` reports default-deny errors on exported functions that dereference raw pointers, plus unwraps and unreachable branches on paths reachable from FFI.

**Acceptance criteria**
- Given the Rust crate, when I run `cargo clippy --all-targets`, then there are no errors.
- Given a null pointer or invalid JSON passed to any exported function, when it is called, then it returns an error value and does not panic or crash.
- Given the Go bridge, when the core returns an error, then it is surfaced as a Go error.

## [E1-04] Make generated output deterministic
- type: task
- epic: E1
- labels: type:task, area:core, priority:P1
- milestone: H0

Generated SQL, column lists and listings must not depend on hash-map iteration order.

**Acceptance criteria**
- Given the same schema and query, when SQL is generated 100 times, then the output is byte-identical.
- Given the same schema, when DDL is generated, then entities and fields appear in a stable, documented order.

## [E1-05] Add adversarial tests for query and mutation generation
- type: task
- epic: E1
- labels: type:task, area:core, area:runtime, priority:P0
- milestone: H0

**Scope.** Tests assert safe properties (for example, a value never appears in SQL text; unknown identifiers are rejected), at the Rust level and end to end through the Go engine against PostgreSQL. Land with or after E1-01.

**Acceptance criteria**
- Given a set of hostile strings as values, when queries and mutations are built and executed against PostgreSQL, then they are treated as data and the database state is unchanged.
- Given a build with the E1-01 fix reverted, when the tests run, then they fail.

## [E1-06] Restrict developer servers to loopback and require a token
- type: task
- epic: E1
- labels: type:task, area:runtime, priority:P1
- milestone: H0

The playground and dashboard HTTP servers must not be reachable from the network by default.

**Acceptance criteria**
- Given the default configuration, when a server starts, then it binds to `127.0.0.1` only.
- Given a request without the generated token, when it reaches a server endpoint, then it is rejected.

## [E1-07] Verify downloads in the install script and publish checksums
- type: task
- epic: E1
- labels: type:task, area:ci, priority:P1
- milestone: H0

**Acceptance criteria**
- Given a release, when it is published, then a checksums file is published with it.
- Given the install script, when the checksum of a downloaded artefact does not match, then installation stops with a clear error.

## [E1-08] Review the vault threat model and password hashing
- type: task
- epic: E1
- labels: type:task, area:vault, priority:P1
- milestone: H0

Document what the vault protects against and what it does not; replace single-round salted hashing of the admin password with a standard password-hashing function.

**Acceptance criteria**
- Given `docs/en/SECURITY.md`, when I read it, then it states the threat model and what is out of scope.
- Given a newly set admin password, when it is stored, then it uses a memory-hard or iterated password-hashing function with documented parameters, and existing hashes are migrated or rejected with a clear message.

## [E1-09] CI on every pull request: build, lint, test, audit
- type: task
- epic: E1
- labels: type:task, area:ci, priority:P0
- milestone: H0

**Acceptance criteria**
- Given a pull request, when CI runs, then it builds the Rust core and the Go CLI, runs `cargo fmt --check`, `cargo clippy`, `cargo test`, `gofmt`, `go vet`, Go unit tests and Go integration tests against PostgreSQL 16, and runs `cargo audit` and `govulncheck`.
- Given a failing step, when the pull request is evaluated, then merging is blocked.

## [E1-10] Portable integration test setup
- type: task
- epic: E1
- labels: type:task, area:ci, priority:P1
- milestone: H0

Integration tests must not depend on one developer's machine: connection settings come from environment variables, and the docs cover Docker and Podman.

**Acceptance criteria**
- Given a machine with Docker, when I follow `DEVELOPERS.md`, then `make test-integration` passes.
- Given a machine with Podman, when I follow the documented steps, then the integration tests pass, or the documentation states the limitation.
- Given an existing PostgreSQL instance, when I set the documented environment variable, then the tests run against it without starting a container.

## [E1-11] Consolidate duplicated mutation logic
- type: task
- epic: E1
- labels: type:task, area:core, area:runtime, priority:P2
- milestone: H1a

Mutation SQL is generated in Rust (unused) and in Go (used). Keep one implementation. Blocked by the decision in ADR-0011 (E1-14).

**Acceptance criteria**
- Given the repository, when I search for mutation SQL generation, then exactly one implementation exists and it is the one the engine uses.
- Given the integration tests, when they run, then they pass unchanged.

## [E1-12] Spike: canonical schema digest
- type: spike
- epic: E1
- labels: type:spike, area:core, priority:P1
- milestone: H0

Time box: 2–3 days. Implements the experiment of [ADR-0005](../adr/0005-canonical-schema-digest.md) and unblocks E1-13. The canonical model must be constructible both from the DSL and from database introspection ([product definition, section 5.1](../product-definition.md)).

**Acceptance criteria**
- Given two schemas that differ only in formatting, comments or declaration order of independent items, when their digests are computed, then they are equal.
- Given two schemas that differ semantically, when their digests are computed, then they differ.
- Given a database created from a schema S (supported subset), when it is introspected and its digest computed, then the digest equals the digest of S computed from the DSL, or the differences are listed and explained.
- Given the results, when the spike ends, then ADR-0005 is updated to Accepted, revised or rejected.

## [E1-13] Canonical model and `digest` command
- type: story
- epic: E1
- labels: type:story, area:core, area:runtime, priority:P1
- milestone: H0

As a developer, I want to print a stable identifier for a schema or for a database, so that I can tell whether two are the same regardless of formatting.

**Acceptance criteria**
- Given a `.cham` file, when I run `chameleon digest file.cham`, then it prints a prefixed digest and exits with code 0.
- Given the same file formatted differently, when I run the command again, then the output is identical.
- Given an invalid schema, when I run the command, then it prints diagnostics and exits with a non-zero code.
- Given a database URL, when I run the command against it with a read-only role, then it prints the digest of the introspected supported subset and lists unsupported objects.

## [E1-14] Spike: implementation languages, boundary and distribution
- type: spike
- epic: E1
- labels: type:spike, area:ci, priority:P1
- milestone: H0

Time box: 3 days. Implements the spike of [ADR-0011](../adr/0011-language-runtime-and-distribution.md).

**Acceptance criteria**
- Given the spike report, when I read it, then it contains installation steps, binary size, build time and cross-compilation results for each option tried.
- Given the results, when the spike ends, then ADR-0011 is accepted, revised or rejected, with a migration estimate for the CLI commands.

---

## [E2] Observe and drift
- type: epic
- labels: type:epic, area:core, area:runtime, priority:P1
- milestone: H1a

A developer can point the tool at a real PostgreSQL database, with read-only access, and learn how it differs from the declared schema.

**Stories**

- As a developer, I want to see how my production database differs from my schema, so that I notice unexpected changes before they hurt me.
- As a team lead, I want differences classified by risk, so that I know which ones need attention first.
- As an operator, I want observation to need only read-only credentials, so that I can run it safely.
- As a developer without a `.cham` file, I want to take a baseline of my database and compare against it later, or compare two environments, so that I get value before adopting anything else ([product definition, section 5.1](../product-definition.md)).

**Done when** early adopters run the drift report against real databases, get a first report in under 10 minutes, and confirm that at least some reported differences are meaningful.

## [E2-01] Specify the supported PostgreSQL subset
- type: task
- epic: E2
- labels: type:task, area:docs, priority:P1
- milestone: H1a

List which object types, types and constraints are governed (tables, columns, types, nullability, defaults, primary and foreign keys, unique constraints, indexes, and so on) and which are explicitly out of scope. Informs [ADR-0008](../adr/0008-schema-source-dsl-vs-sql-first.md).

**Acceptance criteria**
- Given a PostgreSQL object, when I read the specification, then I can tell whether it is governed, observed only, or ignored.
- Given the specification, when I read the CLI output of `drift`, then unsupported objects are reported as such and not silently skipped.

## [E2-02] Faithful introspection for the supported subset
- type: story
- epic: E2
- labels: type:story, area:runtime, priority:P1
- milestone: H1a

**Acceptance criteria**
- Given a database created from schema S, when I introspect it, then the resulting model compares equal to S for the supported subset.
- Given a database object outside the subset, when I introspect, then it is listed as unsupported.
- Given a read-only database role, when I introspect, then it succeeds and writes nothing.

## [E2-06] Baseline snapshots of a database
- type: story
- epic: E2
- labels: type:story, area:runtime, area:core, priority:P1
- milestone: H1a

As a developer without a declared schema, I want to store a snapshot of my database's current state, so that I can later see what changed since.

**Acceptance criteria**
- Given a database URL, when I run `chameleon snapshot`, then a local snapshot is stored with its canonical digest, the supported-subset version and a timestamp.
- Given two snapshots of an unchanged database, when I compare their digests, then they are equal.
- Given a snapshot, when I inspect it, then it contains no data rows and no credentials.
- Given unsupported objects, when the snapshot is taken, then they are listed as unsupported in it.

## [E2-07] Compare two environments
- type: story
- epic: E2
- labels: type:story, area:runtime, area:core, priority:P1
- milestone: H1a

As a developer, I want to compare two databases (for example production and staging), so that I can see how they differ without declaring a schema.

**Acceptance criteria**
- Given two database URLs with read-only roles, when I run the comparison, then differences are reported in both directions ("present in A, not in B", "present in B, not in A", "differs").
- Given objects outside the supported subset in either database, when I run the comparison, then they are listed as not compared.
- Given the run, when it ends, then neither database was modified.

## [E2-03] Classify differences between a reference and a database
- type: story
- epic: E2
- labels: type:story, area:core, priority:P1
- milestone: H1a

**Acceptance criteria**
- Given a reference (declared schema, baseline snapshot or another environment) and a database with an extra column, when I compare them, then the difference is reported as "present in database, not declared" with the object name.
- Given a database missing a declared table, when I compare, then it is reported as "declared, not present".
- Given a column whose type differs, when I compare, then it is reported with both types.
- Given any reported difference, when I read it, then it says the tool will not change the database automatically.
- Given any report, when I read it, then it states which reference was used and which supported subset.

## [E2-04] `drift` command with text and JSON reports
- type: story
- epic: E2
- labels: type:story, area:runtime, priority:P1
- milestone: H1a

**Acceptance criteria**
- Given a reference (schema file, snapshot or second database) and a database URL, when I run `chameleon drift`, then I get a human-readable report and exit code 0 if there are no differences.
- Given differences, when I run it, then the exit code is non-zero and documented.
- Given `--output json`, when I run it, then the output validates against a documented JSON structure.

## [E2-05] Guide: running observation with a read-only role
- type: task
- epic: E2
- labels: type:task, area:docs, priority:P2
- milestone: H1a

Document how to create a least-privilege role for observation in PostgreSQL and verify that the tool works with it and cannot write.

**Acceptance criteria**
- Given the guide, when I follow it on a clean PostgreSQL 16, then the drift report runs.
- Given the same role, when the tool attempts a write, then the database refuses it.

---

## [E3] Gated apply
- type: epic
- labels: type:epic, area:core, area:runtime, priority:P2
- milestone: H1b

Risk-classified change plans, a gate for destructive and locking changes, and role-based verification of the control level ([ADR-0006](../adr/0006-levels-of-control-and-role-enforcement.md)). To be refined after E1 and E2 produce evidence. Horizon H1b starts only when H1a has shown what users value.

## [E3-01] Refine the scope of epic E3
- type: task
- epic: E3
- labels: type:task, area:docs, priority:P2
- milestone: H1b

Write user stories and acceptance criteria for E3 using what E1 and E2 taught.

**Acceptance criteria**
- Given the refinement, when it is complete, then E3 contains child issues with acceptance criteria and the exit signal of horizon H1 is restated.

---

## [E4] Validation and research
- type: epic
- labels: type:epic, area:docs, priority:P1
- milestone: H1a

Evidence that guides the next decisions: interviews with developers and time-boxed spikes for the open ADRs.

## [E4-01] Interview developers about schema-change incidents
- type: task
- epic: E4
- labels: type:task, area:docs, priority:P1
- milestone: H1a

Interview 10–15 developers (indie developers and teams of 2–10 using PostgreSQL) about their last schema-related incident, how they detect differences between environments, and how AI agents touch their databases. Do not present ChameleonDB first.

**Acceptance criteria**
- Given the interviews, when they finish, then notes are summarised in a document that states which hypotheses were supported, contradicted or untested.

## [E4-02] Interview developers of offline and field-data applications
- type: task
- epic: E4
- labels: type:task, area:docs, priority:P2
- milestone: H1a

Interview 8–10 developers of offline-first or field-collection applications ([ADR-0009](../adr/0009-offline-first-and-distributed-data.md)).

**Acceptance criteria**
- Given the interviews, when they finish, then the summary states whether schema evolution across long-offline replicas is a recurring pain.

## [E4-03] Spike: git as history substrate and lifecycle states on references
- type: spike
- epic: E4
- labels: type:spike, area:vault, priority:P2
- milestone: H1b

Time box: 2 days. [ADR-0004](../adr/0004-git-as-history-substrate.md).

**Acceptance criteria**
- Given the spike report, when I read it, then it shows whether revisions, references and lifecycle states can be represented in git and what breaks.

## [E4-04] Spike: levels of control and role enforcement
- type: spike
- epic: E4
- labels: type:spike, area:vault, priority:P2
- milestone: H1b

Time box: 2 days. [ADR-0006](../adr/0006-levels-of-control-and-role-enforcement.md).

**Acceptance criteria**
- Given the spike report, when I read it, then it shows a working check that distinguishes observe, mediated and enforced control against a real PostgreSQL.

## [E4-05] Spike: schema source — DSL, SQL import and round trip
- type: spike
- epic: E4
- labels: type:spike, area:core, priority:P2
- milestone: H1a

Time box: 3 days, after E1-12. [ADR-0008](../adr/0008-schema-source-dsl-vs-sql-first.md).

**Acceptance criteria**
- Given three realistic schemas, when each is authored in the DSL and in SQL and imported from a database, then the canonical models are compared and gaps are recorded.

## [E4-06] Spike: SQLite adapter for the portable subset
- type: spike
- epic: E4
- labels: type:spike, area:core, priority:P2
- milestone: H2

Time box: 1 week, after E1-12 and E4-05. [ADR-0010](../adr/0010-backend-strategy-and-capability-declarations.md).

**Acceptance criteria**
- Given a minimal SQLite adapter, when the same conformance checks run against PostgreSQL and SQLite, then the capability differences are documented.

## [E4-07] Prototype: compatibility verdict between schema revisions
- type: spike
- epic: E4
- labels: type:spike, area:core, priority:P2
- milestone: H3

Time box: 1 week, after E1-13. [ADR-0009](../adr/0009-offline-first-and-distributed-data.md).

**Acceptance criteria**
- Given the real historical schema changes of three to five open-source projects, when the verdict is computed and reviewed manually, then the classification is judged correct with explainable output in most cases, and the failures are listed.
