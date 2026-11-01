# Architecture Decision Records

Short documents that capture one significant decision each: the context, the options considered, the decision (or proposal) and its consequences. See [ADR-0001](0001-record-architecture-decisions.md) for the rules.

## Statuses

| Status | Meaning |
|---|---|
| **Proposed** | A direction the maintainers lean toward; needs validation (often a time-boxed spike) before acceptance. |
| **Open** | A decision that is deliberately not made yet; options and criteria are recorded. |
| **Accepted** | In force. Changing it requires a new ADR that supersedes it. |
| **Superseded** / **Rejected** | Kept for history, with a link to what replaced it or why it was dropped. |

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted |
| [0002](0002-honest-claims-policy.md) | Honest claims policy | Accepted |
| [0003](0003-initial-audience-and-claims-boundary.md) | Initial audience and claims boundary | Accepted |
| [0004](0004-git-as-history-substrate.md) | Git as history substrate; lifecycle states live on references | Proposed |
| [0005](0005-canonical-schema-digest.md) | Canonical schema digest as the first technical deliverable | Proposed |
| [0006](0006-levels-of-control-and-role-enforcement.md) | Levels of control and role-based enforcement | Proposed |
| [0007](0007-data-access-layer-scope.md) | Scope of the data-access layer | Open |
| [0008](0008-schema-source-dsl-vs-sql-first.md) | Schema source and the role of the DSL | Proposed (direction); details Open |
| [0009](0009-offline-first-and-distributed-data.md) | Offline-first and distributed data in production — scope and stance | Proposed |
| [0010](0010-backend-strategy-and-capability-declarations.md) | Backend strategy — capability declarations and a first engine pair | Proposed |
| [0011](0011-language-runtime-and-distribution.md) | Implementation languages, runtime boundary and distribution | Proposed |

## Template

```markdown
# ADR-NNNN: Title

- **Status:** Proposed | Open | Accepted | Superseded by ADR-XXXX | Rejected
- **Date:** YYYY-MM-DD
- **Related:** links to ADRs, docs, issues

## Context
What forces are at play? What evidence do we have, and how strong is it?

## Options considered
1. …

## Decision / Proposal
What we will do (or lean toward), stated plainly.

## Consequences
Positive, negative, and what becomes harder.

## Validation
The spike, test or interview that would confirm or overturn this. Success criteria and time box.
```
