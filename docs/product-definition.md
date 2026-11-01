# ChameleonDB — Product Definition

> **Status:** Working draft  
> **Version:** 0.2  
> **Date:** 2026-11-01  
> **Purpose:** Product direction, scope and development reference  
> **Precedence:** ADRs → this document → [Vision and Foundations](vision-and-foundations.md). If they disagree, the ADR wins; this document defines scope and priorities; the vision gives long-term context.  
> **Related documents:** Vision and Foundations · ADRs · Backlog · Security Policy

### What changed from 0.1

- A concrete near-term positioning sentence was added next to the long-term definition (section 1).
- Defined what drift is measured **against** when a project starts without a `.cham` file: the *reference model* (section 5.1).
- Added three concrete scenarios (section 4A) to guide interviews, demos and priorities.
- Horizon 1 was split into **H1a — Observe and drift** and **H1b — Controlled change**, each with its own exit signal (section 7).
- Success criteria now include measurable signals (section 12).
- Added "Why not just use X?" (section 9.6).
- Statements carry status labels (Agreed / Proposed / Open); pricing language moved out of the differentiators (sections 9.4 and 10).

### Status labels

- **Agreed** — accepted direction (recorded in an accepted ADR or explicitly confirmed by the project owner).
- **Proposed** — coherent with the direction; needs validation (see the linked ADR).
- **Open** — deliberately undecided.

Unlabelled text is descriptive or explanatory. Including a capability in this document does not mean it is implemented ([ADR-0002](adr/0002-honest-claims-policy.md)).

---

## 1. Product definition

### Near term (Horizon 1)

> **ChameleonDB helps you know, before it hurts, how your PostgreSQL database differs from what you think it has — and lets you review every schema change together with its risk.**

You point it at a database you already run, with read-only access, and it tells you what is there, what changed, what it does not understand, and which differences matter. You do not need to rewrite your schema or your tooling first.

### Long term

ChameleonDB aims to become an adaptive platform for managing the lifecycle of data models across database engines, environments and development workflows.

Its purpose is to connect the definition of a data model with its evolution, validation, governance and the actual state of the databases that implement it.

ChameleonDB is not a database engine. It is an infrastructure layer that helps developers and teams understand what their data model is, how it changes, whether those changes are legitimate, and how the resulting state relates to the intended definition.

### Adaptation as a design principle (Proposed)

The developer should not have to adapt their workflow to ChameleonDB; ChameleonDB should adapt to the developer's workflow, database engine and operational context. In practice this is a rule about **order of development and minimum adoption cost**: the first useful capability needs no changes to the existing stack, and each further capability asks for one more step the user chooses to take.

This is not a claim that every integration or environment is supported today. "Adaptive" describes how capabilities are prioritised, not what the product already does.

## 2. The problem

Data models evolve continuously. They change as applications grow, requirements shift, teams introduce new features and systems interact with one another.

In practice, the definition of a schema, the actual state of a database and the changes considered legitimate by a team are often distributed across different tools and processes.

Migrations may describe intended changes, but they do not necessarily establish whether every environment is consistent. Database introspection can reveal the current structure, but does not explain whether a difference is intentional. Version control tracks files, but does not inherently establish which schema revision is official or what has actually been applied.

This creates several questions that are difficult to answer reliably:

- What is the authoritative definition of this data model?
- Which changes were made, by whom and for what reason?
- Which changes were reviewed and approved?
- What schema version is present in each database?
- Does a database differ from the intended definition?
- Are those differences known, accepted, accidental or unexplained?
- What guarantees exist around changes made by developers, applications or automated agents?

ChameleonDB aims to provide a coherent way to answer these questions.

The problem is not simply executing migrations. It is maintaining a reliable relationship between **declared intent, legitimate change and actual database state**.

## 3. Product principles

These principles should influence architecture, UX, integrations and prioritization.

### 3.1 Adaptation over imposition (Proposed)

ChameleonDB should integrate with existing environments and workflows.

- Existing databases should be observable without requiring a complete schema rewrite.
- SQL and database introspection are valid entry points.
- The DSL should provide value, not become a mandatory adoption tax.
- Backend-specific capabilities should be explicit rather than hidden behind false portability.
- Integrations should preserve the developer's existing tools where practical.

The product progressively adds capabilities around an existing system instead of requiring a complete migration before it becomes useful. Governance does require some workflow change (roles, reviews, approvals); the principle is that this change is **opt-in and incremental**, not a precondition for the first result.

### 3.2 Schema as a first-class concept (Agreed)

The data model is more than a collection of migration files.

ChameleonDB should provide a structured representation that can be validated, compared, versioned and interpreted consistently.

### 3.3 Separate intent from reality (Agreed)

The declared schema and the actual state of a database are different things.

A published revision does not mean every database has applied it. A database that differs from the declared model should not be silently modified to make it match.

### 3.4 Explicit authority (Agreed)

Changes should have identifiable origins, validation results and governance requirements.

Automation may perform actions within explicitly delegated permissions. It should not acquire authority simply because it can generate a technically valid change.

### 3.5 Local-first and self-hostable (Agreed)

Essential capabilities should work locally and should not depend on a mandatory ChameleonDB cloud service.

### 3.6 Verifiable guarantees (Agreed)

The platform must communicate what it actually observes, controls and enforces.

A hash is not a complete security mechanism. A migration tool does not automatically control every database connection. A local permission setting does not replace database-level permissions.

### 3.7 One coherent core (Agreed)

Different interfaces and integrations should use the same validation, planning and governance rules. The CLI, a future dashboard and automation clients should not implement independent versions of product logic.

### 3.8 Open Source foundation (Agreed)

The Open Source core remains a foundational commitment. Commercial sustainability should not depend on closing the core or withholding essential capabilities from the community.

### 3.9 Time to first value (Proposed)

A new user should obtain a first meaningful result — a report about their own database — within minutes, on Linux, macOS and Windows, with a single installation step. Installation and distribution are product requirements, not packaging details ([ADR-0011](adr/0011-language-runtime-and-distribution.md)).

## 4. Intended users

### Initial audience — Phase 1 (Agreed — [ADR-0003](adr/0003-initial-audience-and-claims-boundary.md))

Independent developers and small development teams, approximately 1–10 engineers, using PostgreSQL.

Their needs include:

- Understanding differences between declared and actual schemas.
- Detecting unexpected changes across environments.
- Avoiding accidental destructive or high-impact changes.
- Keeping a traceable history of schema evolution.
- Introducing governance without adopting expensive enterprise infrastructure.
- Integrating safety checks into existing development workflows.

The initial audience is a starting hypothesis, not a permanent limitation of the product. No compliance or certification claims are made until after real-world evidence ([ADR-0003](adr/0003-initial-audience-and-claims-boundary.md)).

### Future audiences

Potential future users include:

- Teams operating several database engines.
- Organizations with multiple environments and distributed development teams.
- Developers building offline-first applications.
- Teams using automated coding agents that can interact with databases.
- Organizations that need stronger governance and traceability around data-model changes.

These audiences require additional validation and capabilities. They should not be treated as current customers or supported use cases by default.

## 4A. Scenarios (Proposed — to be validated in interviews)

The scenarios are hypotheses about real situations. They guide interviews, demos and priorities; they are not testimonials.

### Scenario 1 — The hotfix that only exists in production

**Before.** During an incident, someone adds an index and changes a column default directly in production. Nobody records it. Weeks later staging behaves differently, a migration fails on production, and the team spends hours working out what is different and why.

**With ChameleonDB.** A scheduled or manual `drift` run, with a read-only role, compares production with staging (or with last week's baseline) and reports "present in production, not in staging: index `idx_orders_created`; default of `orders.status` differs", along with any objects it could not interpret. The team decides whether to adopt, revert or ignore each difference. Nothing is changed automatically.

### Scenario 2 — The agent that proposes a destructive change

**Before.** A coding agent, asked to "clean up unused columns", generates a migration that drops a column still read by a reporting job. It looks valid and passes review at a glance.

**With ChameleonDB.** The proposed change is turned into a plan and classified by a deterministic rule set: "destructive: drops column with data; requires explicit approval". Under the project's policy the plan cannot be applied by the agent. A human sees the classification and the exact SQL before deciding.

### Scenario 3 — The migration pull request

**Before.** A reviewer opens a pull request with a migration file. Is it locking? Does it rewrite the table? Does it match what production looks like today? The reviewer guesses.

**With ChameleonDB.** The pull request includes a plan with a risk classification (additive, requires attention, destructive, locking) and the observed difference between the intended and the current production state, so the review is about the change, not about reconstructing context.

## 5. Product architecture — conceptual model

ChameleonDB should be organized around a canonical data model and a set of interchangeable entry points and integrations.

The canonical model is the semantic reference used for validation, comparison, versioning and governance.

A conceptual architecture:

**Inputs and authoring**
- ChameleonDB DSL.
- SQL / DDL.
- Existing database introspection.
- Future integrations with frameworks and development tools.

↓

**Canonical model**
- Normalized representation.
- Semantic validation.
- Deterministic identity.
- Explicit specification and capability versions.

↓

**Schema intelligence**
- Comparison between revisions.
- Compatibility evaluation.
- Difference and drift detection.
- Change planning.
- Risk classification.

↓

**Governance and history**
- Change proposals.
- Validation evidence.
- Policies and approvals.
- Immutable revisions.
- Publication and authority.

↓

**Backend adapters and execution**
- Database introspection.
- Capability verification.
- Authorized migrations.
- Application-state verification.
- Audit evidence.

The physical implementation of these components remains subject to technical decisions and ADRs.

### 5.1 The reference model: what drift is measured against (Proposed)

A drift report compares an **observed** state with a **reference**. A project may not have a declared schema at the start, so the reference can be any of:

| Reference | Typical use | Needs a `.cham` file? |
|---|---|---|
| **Baseline snapshot** — the introspected state of a database at a chosen moment, stored locally and identified by its canonical digest | "Tell me what changed since last week / since the last release" | No |
| **Another environment** — the observed state of a second database | "Why does staging differ from production?" | No |
| **Declared schema** — a `.cham` file, SQL/DDL files or the result of applying the project's migrations to an empty database | "Does reality match what we declared?" | No (SQL) or yes (DSL) |

Consequences:

- The canonical model and its digest must be constructible **from introspection**, not only from the DSL. Introspection is a first-class front-end from the first horizon, not a convenience.
- The first useful workflow is "take a baseline, compare later" or "compare two databases". Declaring a schema is a later step the user chooses to take.
- Every report states its reference and its supported subset, and lists the objects it could not interpret ([ADR-0002](adr/0002-honest-claims-policy.md)).

### Adaptation through multiple frontends (Proposed — [ADR-0008](adr/0008-schema-source-dsl-vs-sql-first.md))

The canonical model should not depend on a single authoring language.

The `.cham` DSL is one frontend. It may provide a concise way to express schema metadata that SQL alone cannot represent.

However, developers should be able to obtain useful functionality without first rewriting an existing schema into `.cham`.

A PostgreSQL project should be able to begin with introspection or SQL import, use the supported subset and progressively adopt additional ChameleonDB capabilities.

If the DSL creates more adoption and maintenance cost than value, its scope must be reconsidered.

### Adaptation through backend adapters (Proposed — [ADR-0010](adr/0010-backend-strategy-and-capability-declarations.md))

Each backend adapter should declare what it supports and what guarantees it can provide.

Portability does not mean pretending that PostgreSQL, SQLite and other engines behave identically.

ChameleonDB should distinguish:

- Portable schema concepts.
- Backend-specific capabilities.
- Unsupported constructs.
- Differences in introspection, planning and execution guarantees.

A capability must not be advertised as portable until it passes the corresponding conformance tests.

## 6. Core product capabilities

The following capabilities define the intended product. Their presence in this document does not imply that they are already implemented.

### 6.1 Schema definition and ingestion

Provide ways to represent a data model through the canonical model.

Possible inputs include the DSL, SQL/DDL and database introspection.

### 6.2 Canonical identity and versioning (Proposed — [ADR-0005](adr/0005-canonical-schema-digest.md))

Identify schema content independently from its position in project history.

A canonical digest identifies semantic content. A revision identifier identifies an immutable point in the project's history.

This distinction enables deterministic comparison, provenance and traceability.

### 6.3 Schema comparison and drift detection

Compare an observed database state with a reference model (section 5.1) or compare two schema revisions.

Report differences explicitly, including unsupported or unobserved objects where possible.

Detection must not trigger automatic correction (Agreed).

### 6.4 Change proposals and validation (Proposed)

Treat a schema change as a reviewable unit that can contain:

- The original and proposed revisions.
- Structural differences.
- Validation results.
- Compatibility and risk information.
- Justification and authorship.
- Required approvals and decision history.

The final workflow and terminology remain open to refinement.

### 6.5 Governance and permissions (Proposed — [ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md))

Define which changes are allowed, under what conditions and with which authority.

Governance should distinguish between observation, mediated operations and database-level enforcement.

The actual level of control must be verified against the target database.

### 6.6 Migration planning and application

Generate and evaluate changes for a specific target database.

A published schema revision is not itself an instruction to modify every database.

Application should be explicit, authorized and followed by verification.

### 6.7 Audit and operational evidence

Maintain enough information to reconstruct relevant events:

- What was proposed.
- What was validated.
- What was approved or rejected.
- What was published.
- What was applied.
- What state was observed afterward.

The integrity and security guarantees of this evidence must be specified separately.

### 6.8 Automation and AI integration

Automated tools and AI agents should be able to inspect schemas, propose changes and run deterministic validations.

They should not independently grant themselves authority, bypass required approvals or redefine governance policies.

The platform's safety-critical decisions must rely on verifiable rules rather than a model's judgment.

## 7. Development horizons

The product is ambitious by design. Its development must be incremental and evidence-driven. Horizons are ordered by dependency and evidence, **not by date**.

### Horizon 0 — Trustworthy foundation

**Objective:** Make the existing project reliable enough to serve as a foundation for further development.

Focus:

- Security hardening.
- Deterministic schema and query generation.
- Canonical schema representation and digest, constructible from the DSL **and from introspection**.
- Clear technical boundaries and error handling.
- Consolidation of duplicated logic.
- Reliable build, test and release pipelines.
- Honest documentation and security policy.
- Reproducible installation and distribution.

**Exit condition:** The current core behaves predictably, critical security issues are addressed, and public claims correspond to tested behavior.

This horizon is not about adding a large number of features. It is about earning the right to build on the existing foundation.

### Horizon 1a — Observe and drift

**Objective:** Deliver a first useful product for developers using PostgreSQL, with read-only access and no changes to their stack.

Focus:

- Canonical model and a documented supported PostgreSQL subset.
- PostgreSQL introspection as a first-class front-end.
- Baseline snapshots and comparison between environments (section 5.1).
- Drift classification and reports (text and machine-readable).
- Explicit reporting of unsupported and unobserved objects.
- Read-only observation with least-privilege roles.
- Usability in real projects.

**Exit condition:** Early adopters run the drift report against real databases, understand the differences it finds, and report at least some of them as meaningful, without rebuilding their development workflow.

This is the first major product validation milestone.

### Horizon 1b — Controlled change

**Objective:** Make schema changes reviewable and gated.

Focus:

- Change planning with deterministic risk classification (additive, requires attention, destructive, locking).
- Gate for destructive and locking changes.
- Verification of the real level of control against database roles.
- Initial governance and controlled application, followed by verification.
- Traceable revisions and validation evidence.

**Exit condition:** Teams use plans and gates in real change workflows (including with coding agents), and the classification and controls are confirmed useful and trustworthy.

H1b starts only when H1a has produced evidence about what users value.

### Horizon 2 — Adaptable multi-environment platform

**Objective:** Extend the platform beyond the initial PostgreSQL workflow.

Candidate capabilities:

- Additional database adapters.
- SQL and framework integrations.
- Local dashboard sharing the same core as the CLI.
- More complete change proposal and review workflows.
- Stronger identity, permissions and governance.
- Improved collaboration and offline schema development.
- Integrations with automated development tools.

Each capability requires demand, technical feasibility and conformance evidence.

**Exit condition:** ChameleonDB can operate consistently across multiple environments and workflows without fragmenting its core semantics.

### Horizon 3 — Distributed and offline-first data systems

**Objective:** Explore governance of data models in systems whose production data is distributed across connected and disconnected environments ([ADR-0009](adr/0009-offline-first-and-distributed-data.md)).

Potential capabilities:

- Schema compatibility between revisions.
- Replica-aware schema identity.
- Explicit synchronization and conflict policies.
- Integration with existing synchronization engines.
- Governance across devices, sites and central databases.
- Compatibility evaluation for long-disconnected replicas.

This horizon represents a substantial expansion of the product.

It may require new architecture, research, integration work and dedicated resources. It is not part of the initial product promise.

**Exit condition:** A validated contract and integration model can support real distributed applications with explainable compatibility and reconciliation behavior.

ChameleonDB should not initially attempt to build and maintain its own synchronization engine unless evidence demonstrates that existing solutions cannot satisfy the requirements.

## 8. Current scope and boundaries

The distinction between product direction and implementation status is mandatory.

### Current project

ChameleonDB currently has an alpha-stage implementation based on a Rust core and Go runtime/CLI, with PostgreSQL as its current database target and a dedicated schema language.

The existing project originated with schema-driven data access and query/mutation capabilities.

The current implementation is not yet the complete governance platform described in this document.

Its architecture, security guarantees, distribution and existing functionality must continue to be evaluated against the repository and its tests.

### Near-term scope

The immediate development effort should prioritize:

1. Making the existing implementation trustworthy.
2. Defining and implementing the canonical model and deterministic digest, from both the DSL and database introspection.
3. Establishing a clearly documented supported schema subset.
4. Supporting useful PostgreSQL introspection, baseline snapshots and drift detection.
5. Validating the product with developers using real databases.
6. Testing the actual guarantees of database-level control.
7. Resolving architectural decisions that materially affect these capabilities.

### Explicitly outside the initial scope

- Building a database engine.
- Replacing every ORM or migration framework.
- Universal compatibility across database engines.
- A mandatory hosted cloud service.
- A complete distributed synchronization engine.
- Compliance certification.
- Automatic repair of every detected divergence.
- AI agents with unrestricted database authority.

## 9. Differentiation and product hypotheses

The product should not differentiate itself through features that are already common or easily replicated.

Potential differentiators to validate:

### 9.1 Adaptation without forced migration

A developer can begin with an existing database and workflow, then adopt more advanced capabilities progressively.

### 9.2 A clear relationship between intent and actual state

The platform distinguishes declared schema, official revision, observed database state and applied changes.

### 9.3 Verifiable governance

The product communicates the real control level and the conditions under which changes can be performed.

### 9.4 Governance accessible to small teams

Useful schema governance should be reachable by individuals and small teams without enterprise infrastructure. How this is sustained commercially is an open question (section 10).

### 9.5 Deterministic safety for automation

Automated tools can propose and validate changes through explicit rules and permissions.

These are product hypotheses, not established competitive advantages. Interviews, prototypes and real adoption must determine which ones matter to users.

### 9.6 Why not just use X? (internal; re-verify before any public use)

Based on the landscape study (kept private), as of October 2026. Product features and tiers change; every row must be re-checked before it is used outside the project.

| Alternative | What it already does well | Where a gap appears for our audience (hypothesis) | What ChameleonDB intends to do differently |
|---|---|---|---|
| **Migration tools** (Flyway, Liquibase, Atlas, and similar) | Versioned, repeatable migrations; mature ecosystems | Governance features such as drift detection, linting and approvals often sit in paid tiers; migrations describe intended changes, not the observed state of every environment | Free, local drift and classification built on the same model as planning |
| **ORMs and schema DSLs** (Prisma, Drizzle, and similar) | Developer experience; type-safe access; some have a contract model that checks the database layout | They assume you author the schema in their tool; they are not governance layers; drift across environments is not their focus | Start from an existing database; treat the DSL as optional |
| **Database change platforms** (for example Bytebase) | Review workflows, approvals, multi-database | Heavier to adopt and operate; strongest features target larger teams and often paid plans | Lightweight, local-first, usable by one developer |
| **Plain tooling** (git, SQL files, CI checks, `pg_dump --schema-only` and `diff`) | Free, familiar, flexible | Manual, brittle, no semantic comparison, no risk classification, no record of intent | Semantic comparison, explicit unsupported-object reporting, risk classification |

If interviews show that the plain-tooling row is already good enough for the audience, the product thesis is weaker than assumed and must be revisited.

## 10. Business model (Open)

The business model remains open.

The Open Source core is a fixed commitment. Potential commercial directions may include services, support, managed collaboration or additional operational capabilities.

However, no commercial boundary should be designed before understanding:

- Which capabilities users consider essential.
- Which problems they would pay to solve.
- What can be sustained by a small team.
- Whether commercial services can coexist with local-first and self-hosted operation.
- How the project can remain sustainable without fragmenting the product.

Pricing structure (for example per-seat versus usage-based) is not decided and should not be inferred from the product principles.

The business model should support the product's principles, not dictate an artificial limitation of the core.

## 11. Product validation

Development must be accompanied by user research.

Technical feasibility is necessary, but it does not prove that a product solves a sufficiently important problem.

Initial validation should investigate:

- How developers currently manage schema changes.
- What happens when environments diverge.
- Whether unexpected database changes are a recurring problem.
- How destructive changes are reviewed.
- How much effort is required to understand the state of a database.
- Which existing tools developers would retain.
- Whether adoption without a DSL rewrite is a meaningful advantage.
- Whether teams would use or pay for a governance layer.
- Whether the scenarios in section 4A match situations interviewees have actually lived.
- Whether plain tooling (section 9.6) is already good enough for them.

Interviews should begin with real incidents and existing workflows, not a presentation of ChameleonDB's proposed solution.

Technical spikes should answer bounded questions and produce evidence that can change an ADR.

## 12. Success criteria

Success should be measured through product outcomes, not only code volume.

The numeric signals below are **initial targets to calibrate** after the first real users; they are not commitments.

### Foundation

- Reproducible builds and tests.
- Documented and tested security boundaries.
- Deterministic canonical representation: the same schema yields the same digest across runs and formatting changes.
- Accurate public documentation: every behavioural claim has a test or is removed.

### Adoption (H1a)

- A new user obtains a first drift or baseline report on their own PostgreSQL database in under 10 minutes, with a single installation step.
- At least 5 developers outside the project run it against real (non-toy) databases.
- No user needs to rewrite their schema into `.cham` to get the first report.
- Users understand and trust the output: in follow-up conversations, they can explain what a reported difference means.

### Product value (H1a and H1b)

- At least 3 of those users report finding a real, previously unknown difference.
- Reports state the reference and the supported subset, and list unsupported objects; none are silently ignored.
- Change plans are understandable and reviewable: reviewers can say why a plan is classified as it is.
- Actual control guarantees are verifiable against a real database.
- Users can reconstruct relevant schema-change decisions.

### Long-term expansion

- Additional backends pass conformance tests.
- Integrations preserve common product semantics.
- Distributed use cases are validated before implementation becomes a commitment.

## 13. Product decision rules

When evaluating a feature or architectural change, ask:

1. Does it help manage the lifecycle of data models?
2. Does it preserve the distinction between declared intent and actual state?
3. Does it allow developers to use their existing environment?
4. Does it introduce an unnecessary adoption barrier?
5. Can its behavior and guarantees be verified?
6. Does it belong in the current horizon?
7. Is there evidence of a real user need?
8. Does it preserve the Open Source foundation?
9. Does it create a second implementation of existing business rules?
10. Is the complexity justified by the value it provides?

A technically interesting feature is not automatically a product priority.

## 14. Open decisions

The following topics require explicit decisions or validation:

- Final scope of the data-access layer ([ADR-0007](adr/0007-data-access-layer-scope.md)).
- Canonical model semantics and serialization ([ADR-0005](adr/0005-canonical-schema-digest.md)).
- Exact DSL responsibilities and supported subset ([ADR-0008](adr/0008-schema-source-dsl-vs-sql-first.md)).
- Form of the baseline snapshot and of environment-to-environment comparison (section 5.1).
- Git and revision-history implementation ([ADR-0004](adr/0004-git-as-history-substrate.md)).
- Change Proposal format and lifecycle.
- Database identity, roles and permission model ([ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md)).
- Trust and integrity of audit evidence.
- Rust/Go boundary and distribution strategy ([ADR-0011](adr/0011-language-runtime-and-distribution.md)).
- First additional backend and adapter capabilities ([ADR-0010](adr/0010-backend-strategy-and-capability-declarations.md)).
- Dashboard scope.
- Commercial sustainability.
- Evidence for offline-first production data as a target market ([ADR-0009](adr/0009-offline-first-and-distributed-data.md)).

These decisions should be recorded in ADRs rather than silently settled in implementation.

---

## Product statement

**ChameleonDB aims to become an adaptive platform for governing the lifecycle of data models.**

It should connect schema definition, evolution, validation, governance and real database state while adapting to the developer's existing tools, engines and workflows.

The ambition is to build a comprehensive platform. The execution begins with a trustworthy, useful and verifiable solution for a clearly defined problem: knowing how your PostgreSQL database differs from what you think it has, and reviewing schema changes with their risk.

The long-term vision establishes the direction. The current implementation, evidence and available resources establish what can honestly be promised at each stage.
