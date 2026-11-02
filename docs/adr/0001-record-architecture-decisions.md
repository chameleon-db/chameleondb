# ADR-0001: Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-10-24
- **Related:** [vision-and-foundations](../vision-and-foundations.md)

## Context

Design intent for ChameleonDB currently lives in a whitepaper, a vision draft, the README and chat discussions. Several decisions were made implicitly, and public documents have drifted from the code. The vision draft already classifies statements as *agreed*, *proposed* and *open*; that classification needs a durable, reviewable home.

## Options considered

1. Keep decisions inside the vision document only.
2. Lightweight ADRs in the repository (one file per decision).
3. An external wiki or issue tracker.

## Decision

Use lightweight ADRs under `docs/adr/`, written in English, numbered sequentially, one decision per file, following the template in [README](README.md).

- A decision needs an ADR when it changes a public claim, an on-disk or wire format, a security guarantee, the choice of a core technology, or the product scope.
- ADRs are not rewritten once **Accepted**. A changed decision is a new ADR that supersedes the old one; the old one is only updated to point to it.
- **Proposed** and **Open** ADRs are encouraged: recording options and evaluation criteria before deciding is part of the point.
- The vision document links to ADRs instead of restating their reasoning.

## Consequences

- Positive: decisions are traceable, reviewable in pull requests, and visible to contributors and auditors.
- Negative: a small writing overhead per decision.
- Security-sensitive findings are not written into public ADRs until fixes are released; ADRs reference "tracked privately" in that case.
