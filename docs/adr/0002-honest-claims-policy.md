# ADR-0002: Honest claims policy

- **Status:** Accepted
- **Date:** 2026-10-24
- **Related:** [ADR-0003](0003-initial-audience-and-claims-boundary.md), [vision-and-foundations §3](../vision-and-foundations.md)

## Context

A review of the project found public statements that the code did not support at the time: a guard described as "compile time" that is a runtime error, an opt-in API that is documented but not callable, performance figures without benchmarks, test counts that disagree across documents, and version numbers that disagree across the README, changelog, tags and website. For a product whose value proposition is trust, this is a credibility risk independent of code quality.

## Decision

1. Every public claim (website, README, docs, release notes) must be backed by one of: a test in the repository, a reproducible benchmark or command, or a link to a document that states the evidence.
2. Capabilities are labelled **Implemented**, **Planned** or **Exploratory** wherever a reader could confuse them. Planned or exploratory features are never described in the present tense.
3. Numbers (tests, performance, coverage) are quoted only with the command or CI job that produces them, and are not copied by hand between documents.
4. Security statements name the threat model they cover (for example "detects accidental edits", not "tamper-proof").
5. Versions are defined in one place; the changelog, tags and website are checked against it in CI.
6. A claims matrix (claim → evidence → status) is maintained alongside the documentation and reviewed before each release.

## Consequences

- Positive: marketing and engineering stay consistent; reviewers and users can verify statements.
- Negative: some current website and README text must be corrected or softened before the next release; features move to "Planned" until proven.
- Competitive comparisons name their date and sources and are re-checked before publication (prices and features change).
