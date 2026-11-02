# ADR-0008: Schema source and the role of the DSL

- **Status:** Proposed (direction); details Open
- **Date:** 2026-10-24
- **Related:** [ADR-0005](0005-canonical-schema-digest.md), [ADR-0007](0007-data-access-layer-scope.md), [ADR-0009](0009-offline-first-and-distributed-data.md), [ADR-0010](0010-backend-strategy-and-capability-declarations.md)

## Context

ChameleonDB defines its own schema language (`.cham`). The reasons to keep one are real:

- **Abstraction.** An engine-neutral, entity-oriented definition does not tie the project to one SQL dialect. Many developers think in entities rather than DDL, and coding agents can generate a small, well-documented language reliably. The popularity of Prisma shows demand for entity-style schemas (though that popularity has other causes too, such as the TypeScript ecosystem).
- **Data semantics that DDL cannot carry.** The direction in [ADR-0009](0009-offline-first-and-distributed-data.md) needs per-field merge policy, authority per entity, replication scope, retention of deletions and similar metadata. SQL has no place to express them; a schema language does.

The costs are real too: a new language is the highest adoption cost among the tools surveyed in the landscape study (private study); every engine feature must be added to it; it risks becoming a second SQL. SQL-first tools (`sqlc`, `pgschema`, Atlas) and DSL-first tools (Prisma, Ent) both succeed, so neither approach is dominated.

As reviewed on 2026-10-23, the DSL covers entities, fields, common types, `primary`/`unique`/`nullable`/`default` modifiers and relations, and parses backend annotations that have no runtime yet. Coverage of indexes, check constraints, enums, views, functions, triggers and row-level security is to be confirmed.

## Options considered

- **A. DSL only.** Single authoring path; the DSL must keep growing with each engine.
- **B. SQL only.** Lowest adoption cost; no home for data semantics; harder to stay engine-neutral.
- **C. Canonical model with several front-ends.** The DSL and SQL/DDL (and live-database introspection) compile to the same canonical model.

## Proposal

Adopt **C**.

1. The **canonical model** ([ADR-0005](0005-canonical-schema-digest.md)) is the source of truth for digests, planning, drift detection and compatibility checks, independent of how the schema was authored.
2. The **`.cham` DSL** is the primary authoring front-end, justified by engine-neutral definitions, by metadata that DDL cannot express, and by being easy to generate and review under clear rules.
3. **SQL/DDL import and database introspection** are front-ends that let existing databases be adopted without rewriting them; round-trip fidelity is defined for the supported subset.
4. The DSL **stays small**: portable core constructs, explicit semantic annotations, and declared engine-specific blocks validated against adapter capabilities ([ADR-0010](0010-backend-strategy-and-capability-declarations.md)). Anything outside the supported subset fails loudly instead of being ignored.
5. The project is **not married to the DSL or to PostgreSQL**. If validation shows the DSL is not worth its cost, the canonical model remains valid and the DSL can be reduced or dropped.

## Consequences

- Positive: governance, drift and compatibility work on one model regardless of authoring; adoption of existing databases is possible; the DSL has a defined purpose beyond syntax preference.
- Negative: more than one front-end to maintain; requires a conformance suite so front-ends cannot diverge; the DSL surface needs to be reviewed against this scope.

## Decision criteria (kept from the earlier draft)

- Which PostgreSQL and SQLite features target users actually need governed.
- Round-trip fidelity: declared → canonical → introspected → canonical compares equal for the supported subset.
- Time to first useful result for a solo developer.
- Maintenance cost of parsers and grammar.
- Ability to state, per object type, what is governed and what is explicitly out of scope.

## Validation

After the canonical-digest spike ([ADR-0005](0005-canonical-schema-digest.md)): define the supported subset; write three realistic schemas (a plain web application, a multi-tenant one, and an offline field-collection schema with sync annotations); author each in the DSL and in SQL; import each from a live database; compare the canonical models; record gaps and effort. Time box: 3 days.
