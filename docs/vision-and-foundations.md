# ChameleonDB — Vision, principles and conceptual architecture

> **Status:** foundational draft · **Document version:** 0.3 · **Date:** 2026-10-24
> **Supersedes:** 0.2 (2026-10-24)
> **Related:** [Product definition](product-definition.md) (scope and priorities take precedence over this document) · [ADR index](adr/README.md) · Landscape and pain study (private study)

This document consolidates the product vision, design principles and conceptual decisions that should guide the evolution of ChameleonDB. It is **not** a technical specification and **not** a roadmap.

Statements carry one of three labels:

- **Agreed** — accepted direction.
- **Proposed** — coherent with the vision, needs validation (often a spike; see the linked ADR).
- **Open** — deliberately undecided.

Descriptions of the current code are kept separate from the future vision. **Including an idea here does not mean it is implemented** ([ADR-0002](adr/0002-honest-claims-policy.md)).

### What changed from 0.2

- Longer-term direction added: **distributed and offline-first systems in production**, not only offline schema authoring ([ADR-0009](adr/0009-offline-first-and-distributed-data.md), section 7A).
- Schema source reframed: a **canonical model with several front-ends**; the DSL is justified by metadata SQL cannot carry, and the project is not tied to it ([ADR-0008](adr/0008-schema-source-dsl-vs-sql-first.md)).
- Backend strategy: adapters with capability declarations; PostgreSQL (server) plus SQLite (device) as the first pair ([ADR-0010](adr/0010-backend-strategy-and-capability-declarations.md), section 12).
- The sync and offline landscape was studied; several claims remain unverified (see the study, section 6A). There is currently no design partner for this direction.
- Horizons updated; decision register updated.

### What changed from 0.1 (kept for history)

- Target audience and claims boundary made explicit: indie developers and small teams first ([ADR-0003](adr/0003-initial-audience-and-claims-boundary.md)).
- Positioning grounded in a market and pain study; the headline moves from "access layer with guards" to **governed, observable schema change**.
- New proposals: git as the history substrate and lifecycle states on references ([ADR-0004](adr/0004-git-as-history-substrate.md)); canonical digest as the first deliverable ([ADR-0005](adr/0005-canonical-schema-digest.md)); enforcement through database roles ([ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md)).
- The data-access layer and the schema source language are now explicit open decisions ([ADR-0007](adr/0007-data-access-layer-scope.md), [ADR-0008](adr/0008-schema-source-dsl-vs-sql-first.md)).
- Automation and AI agents promoted to a section of their own; horizons and candidate epics added; business-model evidence added.

## 1. Executive summary

ChameleonDB aims to be an **adaptive platform for governing the lifecycle of data models**. The schema is a declarative, versioned and auditable definition; databases are concrete operational states that may match that definition or diverge from it.

It is not a new database engine and does not replace PostgreSQL. It provides definition, validation, evolution, governance and observability, making decisions and the limits of control explicit. Essential capabilities work locally and self-hosted, without a ChameleonDB cloud service.

> The schema defines the intent; each database represents a real state; governance decides which changes become official and how they are applied.

Publishing a version does not mean every database has applied it. Detecting a divergence does not authorise fixing it automatically.

**Ambition.** The long-term ambition is large: an operational platform for governed data-model change that organisations — and eventually a major infrastructure vendor — would consider strategic. The path is deliberately incremental: earn trust with small teams on PostgreSQL, then widen (§17).

## 2. Problem, audience and positioning

Data models evolve across time, environments and levels of control. The declared schema, the real state of a database and the changes a team considers legitimate often disagree. Migrations, design decisions, permissions and environment state end up scattered across files, tools, people and procedures. That makes it hard to answer:

- What is the official definition of the model?
- What changed, who proposed it and why?
- What validations and approvals back it?
- Which version is applied in each database?
- Does a database diverge, and is the divergence known, accidental or not yet evaluated?
- What guarantees does the platform actually give about that database?

### 2.1 Primary audience (Agreed — ADR-0003)

**Indie developers and small teams (roughly 1–10 engineers) running PostgreSQL**, who want safer schema changes without running a server or paying per seat. The product is validated in real production use with early adopters. **No compliance claims** (SOC 2 and similar) are made until independent evidence exists; larger organisations and compliance-driven teams come later.

### 2.2 Evidence-based positioning (Proposed)

The study (research (private study)) suggests, as hypotheses to validate:

1. **A free governance floor.** Drift detection, destructive-change gating, audit and approval policies are mostly paid or enterprise-oriented in existing tools; small teams are served by plain migration runners.
2. **A deterministic gate for automated and AI-driven schema changes** that a solo developer can adopt. Destructive commands run by coding agents against real databases are documented; incumbents' agent features target teams and enterprises.
3. **Honest levels of control per database.** No surveyed tool states what level of control it really has.

What is **not** a differentiator by itself: blocking `UPDATE`/`DELETE` without `WHERE` and parameterised queries (both exist in mainstream tools), plain migration execution, immutable hashed migrations, zero-downtime migrations. These may be supporting features, not the headline.

Risks to the thesis: Atlas is a strong incumbent for Go schema management; Prisma's newer contract model points in a similar direction for TypeScript; a new schema language is a high adoption cost; agents that bypass the tool still hold credentials; willingness to pay is not yet evidenced.

### 2.3 Goal

Build a platform centred on the schema that connects definition, evolution, validation, governance, migrations and the real state of databases.

### 2.4 Non-goals

- Building a database engine of our own in the first stage.
- Promising universal compatibility across engines without acknowledging their differences.
- Turning every schema publication into an automatic migration for all environments.
- Letting an AI approve changes or expand its own authority.
- Guaranteeing that an administrator with sufficient privileges cannot alter a database outside ChameleonDB.
- Presenting future capabilities as available.
- Implementing a full distributed-consensus system early on.
- Claiming compliance certification or regulatory fitness before evidence exists.

## 3. Design principles

1. **Schema-first.** The declarative definition is the source of project intent.
2. **Explicit versioning.** Official revisions are immutable; correcting one means creating another.
3. **Observable real state.** Databases are inspected and compared with a declared reference.
4. **No silent changes.** An external modification may originate a proposal; it never alters the official definition by itself.
5. **Publishing and applying are different events.** Each database has its own application state.
6. **Governance before authority.** A proposal must satisfy applicable policies before becoming official.
7. **Divergence is not permission to repair.** Detect and classify first; then decide.
8. **Local-first and self-hosted.** Essential work must not depend on a central ChameleonDB service.
9. **Offline is first-class.** Work, validate and prepare proposals without a connection.
10. **Honest guarantees.** Say what is controlled, what is observed and what cannot be guaranteed.
11. **Useful audit.** Proposals, decisions, application attempts and actual results can be reconstructed.
12. **Bounded automation.** Automatic actions are limited to permissions and policies explicitly delegated.
13. **One core.** CLI and dashboard must not implement different business rules.
14. **Explicit portability.** Common and engine-specific capabilities are distinguished.
15. **Deterministic gate for automation.** Checks that protect data must be deterministic and verifiable, never dependent on a model's judgement.

## 4. Conceptual architecture

Responsibilities are separated; physical layout is pending design.

- **Declarative model.** A canonical, validatable, versionable representation of the schema that yields deterministic results (§5, ADR-0005).
- **Validation and planning core.** Interprets the definition, validates constraints and relations, compares revisions and helps generate change plans. Sensitive operations rely on validations independent of AI.
- **History and project authority.** Keeps revisions, ancestry, proposals, approvals and publications. Authority decides which transitions are legitimate under the project's policy. **Proposed:** use git as the substrate (ADR-0004).
- **Backend adapters.** Connect to concrete engines and environments; each declares its capabilities, semantic differences and the limits of observation, planning and application.
- **Execution and observability.** Inspects databases, evaluates divergence, runs authorised migrations and records results. Observed state is never confused with expected state.
- **Interfaces.** The CLI offers complete control. An optional local dashboard consumes the same core and rules.
- **Data-access layer (scope Open — ADR-0007).** Generated query and mutation code derived from the same schema, if it stays in scope.

## 5. Schema identity and revisions

Content identity is separate from revision identity.

### 5.1 `schema_digest` (Agreed as a concept)

A deterministic cryptographic identifier of the canonical schema content.

- Same canonical content → same digest, regardless of branch, author or history.
- Canonicalisation is deterministic and the format and its version are identified in the digest to avoid ambiguity between specification versions.
- **Proposed (ADR-0005):** digest the *canonical model*, not the source text; ship test vectors; version the format. Algorithm, serialisation and canonical rules are to be specified in the spike.

### 5.2 `revision_id` (Agreed as a concept)

Identifier of an immutable revision in the project history. A revision references a `schema_digest` and keeps lineage and provenance. Two revisions may share a digest with different ancestry. Because a revision can have several ancestors, history is a graph.

### 5.3 Data associated with a revision

At minimum: project identity; `schema_digest`; ancestor references; specification and canonical-format version; provenance and authorship; origin Change Proposal, if any; separately identified validation evidence. Exact fields and their role in cryptographic identity are pending.

### 5.4 Identity and lifecycle (Proposed)

Approval, publication and application are events distinct from a revision's intrinsic identity; the state of a database is never part of it.

An immutable revision cannot change state. **Proposed (ADR-0004):** lifecycle states such as *draft*, *official*, *deprecated*, *archived* are properties of **references** (like branches and tags in git) that point at revisions. Candidate states are not a closed taxonomy. An official revision is never edited; a correction creates a new one; drafts may change until frozen for review.

## 6. Branches and Change Proposals

**Branches** allow developing changes without altering the official reference.

**Change Proposal (CP)** — provisional name — is an identifiable unit of change submitted to validation and governance; it is not merely a diff. It may include: source revision and proposed schema; branch and ancestry; structural differences; validation and test results; risk and estimated impact; justification and owners; required and received approvals; decision history; review state and outcome.

### 6.1 Conceptual flow

1. Create or select a branch.
2. Modify the schema and test locally.
3. Freeze a candidate revision for the CP.
4. Run validations and evaluate impact (including destructive-change checks).
5. Review the proposal and resolve comments.
6. Satisfy governance rules.
7. Publish an immutable official revision.
8. Create or select a migration plan per target database.
9. Apply the plan with the required authority.
10. Verify and record the effective state.

Publishing does not apply the change to every database.

### 6.2 Outdated proposals

If another proposal publishes a revision that makes a CP's base obsolete, the original is kept and marked outdated. Updating or rebasing needs an explicit action, conflict resolution and new validation. The evaluated revision never changes silently because of later edits to the branch.

## 7. Offline work and coordination

A person should be able to create branches, edit schemas, test against a local database, run tests and prepare proposals offline. On reconnecting, changes are synchronised and reconciled with the known project state. Synchronisation is neither approval nor publication.

### 7.1 Project authority

Authority is defined by project and its rules, not by a machine that must stay on or a mandatory cloud service. **Proposed:** start with one designated authority per project, with explicit participants and rules, designed so it can evolve towards more distributed forms without redoing the core concepts. A participant who is offline can validate and prepare work; they cannot simulate a global approval that the policy requires and has not been received.

## 7A. Offline-first and distributed data in production (Proposed — ADR-0009)

Section 7 concerns people working offline on schemas. This section concerns **production data** held on devices or sites that are disconnected for hours, days or weeks and later reconcile with a central database (for example, field data collection without network coverage).

Position (Proposed):

- ChameleonDB does **not** start by building a sync engine. Existing engines are mature in different ways, and several products in this space have been discontinued or changed hands, which shows the cost of owning one.
- What the project can own is the **contract**: which schema revision each replica speaks (digest), a deterministic **compatibility verdict** between two revisions (identical, additive, requires translation, breaking), and **sync policy metadata** in the schema (authority per entity, merge strategy per field, replication scope, deletion retention, maximum offline window), checked for consistency by the type checker.
- Adapters to existing sync engines come after the contract exists; an own protocol only if integration proves insufficient.
- This is a hypothesis. Public evidence that the problem is underserved is moderate for schema evolution across long-disconnected replicas and weak for custom field-app developers as a customer segment. No public claim about offline or distributed support is made before interviews, a prototype and an integration spike.
- Note: the whitepaper places distributed execution out of scope; this section extends beyond it and would need its own paper or a later whitepaper version.

## 8. Governance, identity and permissions

- **Identity:** who or what performs an action.
- **Authority:** what that participant may do in the project or environment.
- **Policy:** which conditions make a transition valid.

Roles must not be confused with operating modes. Nomenclature and the permissions model are pending.

### 8.1 Configurable policies

Policies come from templates adaptable to each project, but ordinary configuration must not let someone without authority turn a destructive operation into an unrestricted one. Some safeguards must remain **system invariants** regardless of configuration (list pending). A dangerous operation (for example `DROP TABLE`) is not universally illegitimate: it may be valid in a sandbox and require reinforced controls in production, depending on environment, impact, permissions and approvals.

### 8.2 Soft and hard rules

- **Soft rules** let work proceed under explicit conditions, possibly with a warning, a justification and a record.
- **Hard rules** block the transition until requirements are met.

Machines may automate deterministic checks and actions within previously delegated authority. Decisions the policy reserves for a person need that person.

### 8.3 Emergencies

An emergency flow must not become a button that bypasses every control. It may have predefined rules: who can invoke it, scope, justification, audit and post-incident review. Essential blocks are preserved.

## 9. Real backend control and limits of guarantee

ChameleonDB must identify and communicate the real level of control it has over each database. **Proposed (ADR-0006):**

1. **Observe.** Inspection and divergence detection, without control over writes.
2. **Mediated.** Operations performed through ChameleonDB follow its rules; other access can bypass them.
3. **Enforced.** Database roles and privileges restrict direct changes and reinforce the guarantees.

For PostgreSQL the enforced level is expected to rely on separate roles — a **migrator** (DDL, used only by an approved plan), an **application/agent** role (data privileges only) and an **observer** (catalog read). A verification command reports which guarantees actually hold. If a person or service keeps privileged credentials, ChameleonDB cannot prevent external changes by itself; the platform verifies what it can and states the limits.

The modes present in the code today (`readonly`, `standard`, `privileged`, `emergency`) are client-side settings and must not be presented as equivalent to this model without technical review.

## 10. Divergence and per-database state

The official reference and the real state of a database are independent dimensions. Provisional states: **synchronised** (observed schema matches the reference within the scope and checks performed); **known or tolerated divergence** (identified and explicitly accepted under conditions); **divergence pending resolution** (requires evaluation or decision).

A matching schema does not prove that data is correct. Detecting a difference does not authorise automatic repair: inspect, classify, evaluate impact, then decide explicitly whether to accept, reconcile or correct. Fidelity of observation depends on the **supported PostgreSQL subset**, which must be declared object by object (ADR-0008).

## 11. Migrations, recovery and operational safety

A published revision expresses an official definition; it is not itself a migration plan. Plans are generated and evaluated for a concrete target; if the target revision, the observed state or other relevant conditions change, the plan becomes stale and must be recalculated or reviewed.

### 11.1 Destructive and locking changes

Destructive or lock-heavy operations are the best-evidenced pain in the study. The planner should classify operations by risk (data loss, table rewrite, long lock), require stronger conditions for high-risk ones, and make the classification visible in the proposal. Classification is deterministic.

### 11.2 Rollback and recovery

Schema rollback does not mean automatic data recovery. Some changes are destructive or irreversible without extra information. Recovery may need real backups, point-in-time recovery, compensating migrations or engine-specific procedures. Instant or universal rollback must not be promised. Roll-forward with tested recovery is the default assumption; verifying that backups can be restored is outside the product but should be surfaced in guidance.

### 11.3 Audit

The record should distinguish: proposal created or modified; revision frozen; validation run and result; approval or rejection; publication; application attempt; effective result; later verification and observed state. An attempt is not a successful application; results reflect what happened, including partial failures where the backend allows.

### 11.4 Integrity and threat model

A hash detects unexpected changes but does not protect against someone who can modify both file and hash. Stronger guarantees may need signatures, protected storage, separation of privileges or external records. Each mechanism states the threat model it covers; detecting accidental errors is not resistance to a privileged attacker.

## 12. Portability and adapters

Portability is described in levels: (1) **data model** — concepts expressible in the common declarative model; (2) **data access** — operations representable portably; (3) **engine-specific capabilities** — functions, types, constraints, semantics. A portable definition does not imply identical semantics across engines; adapters declare what they support and flag differences that affect validation, planning or execution. PostgreSQL is the first real backend; no compatibility is claimed until there is specific implementation and validation.

**Proposed ([ADR-0010](adr/0010-backend-strategy-and-capability-declarations.md)):** each adapter publishes a capability matrix; canonical constructs are classified portable or engine-specific; the canonical model must not leak PostgreSQL types; the first engine pair is PostgreSQL (server) and SQLite (device), chosen because it tests the abstraction on two very different engines and fits the offline scenario. Adding further engines requires demonstrated demand, a conformance suite and a maintainer commitment. Evidence from other tools shows that broad engine support dilutes quality (one major ORM narrowed its first-class set; others split packages per dialect).

**Multiple languages:** the Rust core and C ABI already allow bindings in several languages; the project should state which bindings are supported and tested rather than claim general multi-language support.

## 13. Data-access layer (scope Open — ADR-0007)

The project began as a schema-first query and mutation layer. Evidence suggests it is not the strongest headline: parameterisation and write guards exist in mainstream tools. **Proposed:** treat it as a supporting layer generated from the same canonical schema, with a single implementation, and freeze new features until the positioning is validated. Guarantees it offers (for example that the generated SQL is visible, that writes without conditions are refused) are documented with their exact scope; tests must include adversarial cases.

## 14. Interfaces

The CLI keeps complete control, including advanced and governance flows. A local per-project dashboard may visualise revisions, branches, CPs, divergence, policies and application state, consuming the same core as the CLI without duplicating rules or creating an alternative path that bypasses them. Scope, technology and timing are pending.

## 15. Automation and AI agents

AI can help explain schema differences, summarise risks, propose alternatives, document decisions and guide conflict resolution. It must not turn a recommendation into an approval, expand its own permissions, skip hard rules, pretend a validation ran when it did not, treat silence as consent, or hide uncertainty.

Deterministic checks run through verifiable mechanisms; AI recommendations are distinguished from validation results and are subject to policy and authority.

**Proposed emphasis:** agents are a first-class *client* of the governance model. The practical safeguard is credential separation: an agent receives only an application-level role and can *propose* changes, while applying them requires the migrator path with the required approval. Tool interfaces for agents (for example an MCP server exposing proposal and plan operations, but not apply) are a later epic. Vendors already ship such features for teams; the opportunity is a lightweight, open, deterministic option for small teams.

## 16. Open source, sustainability and business model

The Open Source part of ChameleonDB stays Open Source (Agreed). The core is not to be closed to create a commercial edition. Openness has functional and strategic value: independent inspection and audit, credibility, contributions, and self-hosted utility.

### 16.1 Agreed

- The open component remains open.
- Independent validation and audit are important goals.
- Essential local work, validation, revision identity and self-hosted use do not depend on a commercial service.
- The product is not described as fully auditable if closed critical components prevent verifying the announced guarantees.

### 16.2 Open — options and evidence

| Option | Evidence from the study |
|---|---|
| **Managed/hosted service** | Supabase and Prisma monetise hosting around an open core; hosting is capital-heavy. A *small* hosted offering (for example scheduled drift observation or audit retention) is a lower-cost variant to evaluate. |
| **Open core by seat or resource** | Atlas and Bytebase charge for drift, approvals and audit. Cautionary lesson: an open edition that lacks too much creates friction. |
| **Support and services** | Plausible early; depends on adoption. |
| **Sponsorship / grants** | Plausible for open developer infrastructure; not evidenced here. |
| **Licensing changes** | Liquibase moved its community edition to a source-available licence in 2025; this conflicts with the commitment above and is not proposed. |

No model is decided. Licence details, contribution policy, boundaries of any commercial component and the sustainability strategy are pending. **Open core** is a distribution strategy, not the idea that the business must be private. No critical security guarantee may depend on a private component without explicit justification.

## 17. Horizons and candidate epics

Not a roadmap. A shape for sequencing, to be turned into scope after validation.

| Horizon | Goal | Exit signal |
|---|---|---|
| **H0 — Trustworthy base** | The existing code behaves predictably and its claims match reality: deterministic output, hardened query generation, supported dependencies, honest documentation, CI that blocks bad releases. | Reproducible green build and test pipeline; claims matrix reviewed ([ADR-0002](adr/0002-honest-claims-policy.md)). |
| **H1a — Observe and drift** | Canonical model constructible from the DSL and from introspection; baseline snapshots and environment comparison; drift classification and reports; read-only observation. | Early adopters run drift against real databases and find meaningful differences ([product definition](product-definition.md)). |
| **H1b — Controlled change** | Risk-classified plans, destructive-change gate, control-level verification, audit; branches/CPs start here. | Teams use plans and gates in real change workflows. |
| **H2 — Teams and agents** | Branches/CPs on top of git; approvals; agent-safe interfaces; policies. | Teams of several people use CPs to ship changes. |
| **H2b — Contract for offline and multi-backend** | Compatibility verdict between revisions; SQLite adapter for the portable subset; sync policy metadata; first integration spike with an existing sync engine. | At least one external team runs a real offline workload and confirms the schema-evolution pain and the value of the verdict. |
| **H3 — Platform** | Optional hosted observation/audit; more backends; compliance-oriented capabilities after real-world evidence; ecosystem. | Sustainable business model validated; independent assurance where needed. |

**Candidate first epics (not committed):**

1. **E1 — Trustworthy core:** canonical model and digest ([ADR-0005](adr/0005-canonical-schema-digest.md)), deterministic generation, security hardening, dependency updates, claims cleanup, CI gates.
2. **E2 — Observe and drift:** faithful introspection for the declared subset, classification of divergence, reports. Depends on ADR-0008.
3. **E3 — Gated apply:** risk-classified plans, destructive-change gate, role-based control verification ([ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md)), audit trail.

Epics beyond these (compatibility verdict, SQLite adapter, sync policy) stay candidates until validation ([ADR-0009](adr/0009-offline-first-and-distributed-data.md), [ADR-0010](adr/0010-backend-strategy-and-capability-declarations.md)).

## 18. Current state versus the vision

This section is a dated summary; the detailed baseline is kept privately and re-validated before any public use.

As of the review of 2026-10-23, the repository contains: a schema DSL with parser, AST and type checker (Rust); SQL and migration generation; a Go CLI and runtime bridged to the Rust core; a local vault with schema hashing, integrity modes and a journal; PostgreSQL introspection; and unit and integration tests that pass in the reviewed environment (including integration tests against PostgreSQL 16). PostgreSQL is the only real backend.

The vision of governance is broader than what the current code guarantees. It must not be presented as complete until implementation, tests and evidence exist. Before widening any security or automation promise, the base must be made predictable: query generation, determinism of output, the FFI boundary, vault and migration behaviour, dependency updates, release pipeline and distribution (the current shared-library linkage complicates installation). Specific findings are tracked privately and are published with their fixes.

## 19. Decision register

### 19.1 Agreed

- ChameleonDB is a platform for governing the data-model lifecycle, not a database engine in the first stage.
- The schema is the declarative reference; the real state of each database is observed separately.
- Official revisions are immutable. `schema_digest` identifies canonical content; `revision_id` identifies a revision in history; history may be a graph.
- Approval, publication and application are separate; publishing does not apply.
- Divergences are not repaired automatically.
- Offline work and self-hosting are design requirements; authority does not depend on an always-on machine or a mandatory cloud service.
- Policies vary by project, preserving security invariants still to be defined.
- AI may assist but not take authority or replace required approvals.
- The Open Source part stays Open Source.
- Documentation distinguishes future vision from existing capabilities ([ADR-0002](adr/0002-honest-claims-policy.md)).
- **Phase 1 audience is indie developers and small teams; no compliance claims before evidence** ([ADR-0003](adr/0003-initial-audience-and-claims-boundary.md)).
- Architecture decisions are recorded as ADRs ([ADR-0001](adr/0001-record-architecture-decisions.md)).

### 19.2 Proposed

- Git as history substrate; lifecycle states on references ([ADR-0004](adr/0004-git-as-history-substrate.md)).
- Canonical digest as the first technical deliverable ([ADR-0005](adr/0005-canonical-schema-digest.md)).
- Canonical model with several front-ends; the DSL stays small and justified by metadata SQL cannot carry ([ADR-0008](adr/0008-schema-source-dsl-vs-sql-first.md)).
- Contract layer and adapters for offline-first production data, rather than an own sync engine ([ADR-0009](adr/0009-offline-first-and-distributed-data.md)).
- Adapters with capability declarations; PostgreSQL plus SQLite as the first pair ([ADR-0010](adr/0010-backend-strategy-and-capability-declarations.md)).
- Three control levels, enforced through database roles ([ADR-0006](adr/0006-levels-of-control-and-role-enforcement.md)).
- Change Proposals as the formal unit of reviewable change; one official line per project initially.
- A local dashboard sharing the CLI's core.
- Governed, observable schema change (including for agents) as the headline; the data-access layer as a supporting component.

### 19.3 Open

- Scope of the data-access layer ([ADR-0007](adr/0007-data-access-layer-scope.md)).
- Details of the schema source: supported subset, round-trip fidelity, DSL scope ([ADR-0008](adr/0008-schema-source-dsl-vs-sql-first.md)).
- Exact canonicalisation, algorithm and `revision_id` construction.
- Signatures, trust and tamper-resistant storage.
- Final lifecycle and CP naming; CP schema and CLI.
- Identity, roles, permissions and non-configurable invariants.
- Offline synchronisation and reconciliation semantics, conflict expressiveness and authorisation on partial replicas ([ADR-0009](adr/0009-offline-first-and-distributed-data.md)).
- Per-backend guarantees and adapter capabilities; supported PostgreSQL subset.
- Transactional strategy, recovery and destructive operations.
- Dashboard scope and technology.
- Core technology and distribution (language boundary, linkage, packaging).
- Licence details, contribution policy, commercial boundaries, business model and sustainability.

## 20. Conceptual risks to avoid

- Confusing the declared schema with the real state of a database.
- Confusing a published revision with an applied migration.
- Treating a hash as complete security.
- Claiming a database is protected while privileged access bypasses ChameleonDB.
- Treating a local test as a production guarantee.
- Treating an AI recommendation as validation or approval.
- Promising data rollback when only structural reversal exists.
- Promising total portability without describing engine differences.
- Presenting the business model as solved before validating it.
- Presenting conceptual capabilities as implemented.
- Building a large platform before any user confirms the problem.

## 21. Glossary

| Term | Meaning |
|---|---|
| **Schema** | Declarative definition of the data model. |
| **`schema_digest`** | Deterministic identity of the canonical schema content. |
| **`revision_id`** | Identity of an immutable revision in the project history. |
| **Official revision** | A revision that met the required governance and was published as a reference. |
| **Reference** | A named pointer to a revision (for example "official"), carrying lifecycle state. |
| **Branch** | A line of work where changes develop without altering the official reference. |
| **Change Proposal (CP)** | A unit of change submitted to validation, review and governance. |
| **Publication** | The event that makes a revision an official reference under policy. |
| **Application** | Running a change plan against a specific database. |
| **Divergence** | Difference between a database's observed state and the reference compared with it. |
| **Control level** | Observe, mediated or enforced: how much control ChameleonDB really has over a database. |
| **Authority** | Recognised power to perform an action or approve a transition. |
| **Policy** | Conditions that govern actions and transitions. |
| **Backend** | The engine or system ChameleonDB connects to. |
| **Open core** | A possible strategy where an open core coexists with clearly delimited commercial components. |

## 22. Evolving this document

Important decisions update the register and keep traceability through ADRs. Before turning these ideas into epics or a roadmap, validate the concepts against the current code, resolve the fundamental technical risks (H0) and turn pending points into concrete decisions. The aim is not to freeze the design prematurely, but to establish an honest and coherent base so that the product does not depend on implicit assumptions.
