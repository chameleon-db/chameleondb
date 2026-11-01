# ADR-0004: Git as history substrate; lifecycle states live on references

- **Status:** Proposed (requires a spike)
- **Date:** 2026-10-24
- **Related:** [ADR-0005](0005-canonical-schema-digest.md), [vision-and-foundations §5–§7](../vision-and-foundations.md)

## Context

The vision calls for immutable revisions, a lineage graph with one or more ancestors, branches, change proposals, offline work and an authority that does not depend on a central service. These are the properties of git: content-addressed objects, a commit graph, references, remotes and offline operation. Building an equivalent store and synchronisation protocol is large and already solved.

The draft lifecycle states (`Draft`, `Published`, `Deprecated`, `Archived`) conflict with immutability if they are fields of a revision: an immutable revision cannot change state.

## Options considered

1. **Own store:** ChameleonDB implements its own revision DAG, branches and synchronisation.
2. **Git as substrate:** schema sources and metadata live in a git repository; revisions are commits; official references are signed tags; change proposals are branches or pull requests plus a metadata file; ChameleonDB adds canonicalisation, digests, validation, planning, drift observation, application and audit.
3. **Hybrid:** git for source history, a small local store for evidence (validation results, application attempts) keyed by `schema_digest` / revision.

## Proposal

Lean toward option 3 (git substrate plus a local evidence store).

- A **revision** is immutable and identified by the commit plus the `schema_digest` of its canonical content ([ADR-0005](0005-canonical-schema-digest.md)).
- **Lifecycle states are properties of references** (for example "official", "deprecated"), not of revisions. Moving a state moves or annotates a reference; history is preserved.
- **Official references** are signed tags or protected refs; the trust model states what the signature does and does not prove.
- **Change proposals** reuse branches and pull/merge requests where the team has them, with a ChameleonDB metadata file for validation evidence and decisions.
- Evidence about *database state* (observations, application attempts, drift classification) is stored separately from history and keyed by revision and database identity, because it is not part of a revision's identity.

## Consequences

- Positive: offline and self-hosted by construction; far less code to build and maintain; familiar workflows.
- Negative: users without git or with unusual workflows are excluded; authority partly depends on the git host's controls (branch protection, signing); history rewriting must be detected, not assumed impossible.
- Open questions: where to store metadata (files in the repository, git notes, or dedicated refs); how to represent the "single official line" without constraining future multiple lines; how approvals map to signatures.

## Validation (spike, time box: 2 days)

Prototype a project where revisions are commits and official references are signed tags. Success criteria: (a) the full flow — branch, edit, validate, freeze a candidate, publish an official ref — works offline; (b) a rewritten history is detected by a verification command; (c) the metadata layout survives merges without conflicts in common cases; (d) the model does not require a network service.
