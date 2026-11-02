# ADR-0010: Backend strategy — capability declarations and a first engine pair

- **Status:** Proposed (needs a spike)
- **Date:** 2026-10-24
- **Related:** [ADR-0005](0005-canonical-schema-digest.md), [ADR-0008](0008-schema-source-dsl-vs-sql-first.md), [ADR-0009](0009-offline-first-and-distributed-data.md)

## Context

The project does not want to be tied to a single database. PostgreSQL was chosen as the first backend for ease of use, documentation and familiarity to coding agents, not as a permanent commitment. The offline-first direction ([ADR-0009](0009-offline-first-and-distributed-data.md)) additionally implies an embedded engine on devices.

Evidence from the study:

- Prisma narrowed its "first-class" databases to six and cited more than 3,000 open issues in its 2024 manifesto; its feature matrix is uneven across engines, with explicit escape hatches **[R]**.
- Drizzle uses separate packages per dialect; Hasura v2 offers full support only on PostgreSQL; Atlas keeps per-engine depth behind a paid tier **[R]**.
- Hibernate-style portability breaks as soon as engine-specific functions are used **[R]**.
- Demand appears to be for specific engines (SQLite, PostgreSQL, MongoDB), not for abstract portability across them; this is weak evidence **[R]**.

## Options considered

- **A. Single engine (PostgreSQL) indefinitely.**
- **B. Many engines at a lowest-common-denominator subset.**
- **C. Adapters that declare their capabilities; add engines one at a time.**

## Proposal

Adopt **C**, with **PostgreSQL (server) and SQLite (device/embedded)** as the first engine pair.

1. Each adapter publishes a **capability matrix**: types, constraints, index kinds, transactional DDL, introspection fidelity, concurrency model, and behaviour on schema change.
2. Constructs of the canonical model are classified **portable** or **engine-specific**. Engine-specific constructs live in declared blocks and are rejected by adapters that lack them.
3. **No "compatible with X" claim** is made until a conformance suite passes for that adapter ([ADR-0002](0002-honest-claims-policy.md)).
4. The canonical model **must not leak PostgreSQL types or behaviour**; engine-specific logic lives in adapters.
5. Adding an engine requires evidence of user demand, a conformance suite and a maintainer commitment.
6. Raw escape hatches are explicit and visible in the audit trail.

## Consequences

- Positive: honest portability; a concrete test of the abstraction with two engines of very different capabilities; fits the offline scenario.
- Negative: doubles the test matrix early; SQLite's limited in-place `ALTER TABLE` support affects migration planning (table rebuilds); some PostgreSQL features have no SQLite equivalent and must be declared engine-specific.

## Validation (spike, time box: 1 week, after the ADR-0005 and ADR-0008 spikes)

Implement a minimal SQLite adapter for the canonical subset (create, introspect, compare); run the same conformance suite against PostgreSQL and SQLite; document the capability differences and the constructs that had to be classified engine-specific.
