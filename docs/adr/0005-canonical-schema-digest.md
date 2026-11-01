# ADR-0005: Canonical schema digest as the first technical deliverable

- **Status:** Proposed (requires a spike)
- **Date:** 2026-10-24
- **Related:** [ADR-0004](0004-git-as-history-substrate.md), [vision-and-foundations §5](../vision-and-foundations.md)

## Context

Schema identity (`schema_digest`) underpins versioning, drift comparison, audit and any future signature scheme. Today:

- the vault hashes the raw schema text with SHA-256, so comments, whitespace or reordering change the hash while the meaning stays the same;
- the core represents schemas with unordered maps, so serialised output and generated SQL are not guaranteed to be deterministic between runs;
- there is no identifier for the canonical format, so a future change to hashing rules would be ambiguous.

Without determinism, nothing built on the digest can be trusted.

## Options considered

1. Hash the **source text**.
2. Hash a **canonical form of the parsed model** (ordered, normalised, versioned).
3. Hash a **normalised catalog model** obtained by introspecting a live database (useful for drift, not for declared schemas).

## Proposal

Adopt option 2 for declared schemas, and make the canonical model the common target for option 3 so that declared and observed schemas can be compared.

Requirements:

- Identical meaning produces an identical digest regardless of comments, whitespace, declaration order where order is not semantic, and process run.
- The digest string carries a **format version and algorithm identifier** (for example a prefix such as `chm1:sha256:`), so formats can evolve without ambiguity.
- A published set of **test vectors** (inputs and expected digests) and golden tests guards the format.
- A deterministic serialisation is used for hashing (a standard canonical-JSON scheme such as RFC 8785 is a candidate to evaluate).

Open design questions to settle during the spike: which orderings are semantic (for example column order), identifier case and quoting rules, normalisation of defaults and types, and how unsupported features are represented without being silently ignored.

## Consequences

- Positive: a stable base for identity, drift and audit; deterministic SQL generation as a side benefit.
- Negative: changing the hashing scheme invalidates existing vault hashes. A migration path is needed: re-hash existing versions under a new format identifier and record the old digest.
- The canonical model should be the single source for SQL planning, avoiding duplicated representations.

## Validation (spike, time box: 2–3 days)

Implement the canonical form and digest for the current model; add test vectors; demonstrate that (a) reordering and comment changes keep the digest, (b) semantic changes alter it, (c) output is identical across runs and platforms, (d) a documented migration of existing vault hashes works.
