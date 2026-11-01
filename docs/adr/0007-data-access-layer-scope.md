# ADR-0007: Scope of the data-access layer

- **Status:** Open
- **Date:** 2026-10-24
- **Related:** [ADR-0008](0008-schema-source-dsl-vs-sql-first.md), landscape and pain study §4.1 (private study)

## Context

The project began as a schema-first query and mutation layer (typed builders, generated SQL, visible SQL, guarded writes). The revised vision centres on governance of the schema lifecycle. The data-access layer is most of the existing code, so whether it stays in scope shapes the product.

Evidence from the market study:

- Parameterised queries are the default in mainstream ORMs; `sqlc` is parameterised by construction.
- The "no UPDATE/DELETE without WHERE" guard already exists in popular tools (GORM's default error; `pg-safeupdate`; MySQL `--safe-updates`).
- Recent ORM CVEs concern identifiers, aliases and escape hatches — a point where a generated, schema-aware layer can add value.
- No evidence was found that developers pay for secure data access as a category.
- Mutation logic currently exists twice (core and runtime), with different safety properties; this duplication is a risk in a schema-first product.

## Options considered

1. **Headline:** the data-access layer is the main product.
2. **Supporting layer:** keep it, generated from the same canonical schema, with a single implementation; governance is the headline.
3. **Extract or integrate:** drop our own layer and integrate with existing ones (for example generate inputs for `sqlc` or GORM).
4. **Freeze:** no new features; maintenance and hardening only, decide later.

## Proposal (non-binding)

Treat the layer as **supporting (option 2)** and **freeze new features (option 4)** until [ADR-0008](0008-schema-source-dsl-vs-sql-first.md) and the validation interviews inform the decision. In the meantime: consolidate duplicated logic into one implementation, and keep security hardening of the existing code on the critical path (tracked privately until fixes ship).

## Decision criteria

- Does a target user, in interviews, name data-access safety as a pain we can solve better than existing tools?
- Can the layer be generated from the canonical schema without a second source of truth?
- What is the maintenance cost relative to the value it adds to the governance story?
- Does it help the main wedge (governed schema changes and observation), or distract from it?
