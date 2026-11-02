# ADR-0009: Offline-first and distributed data in production — scope and stance

- **Status:** Proposed (needs interviews and a prototype; there is currently no design partner)
- **Date:** 2026-10-24
- **Related:** [ADR-0004](0004-git-as-history-substrate.md), [ADR-0005](0005-canonical-schema-digest.md), [ADR-0008](0008-schema-source-dsl-vs-sql-first.md), [ADR-0010](0010-backend-strategy-and-capability-declarations.md), landscape study §6A (private study)

## Context

The project owner's longer-term ambition is a data platform for **distributed and offline-first systems in production**, not only in development. Example scenario: devices collecting field data in areas with no network coverage for days or weeks, later reconciling with a central database.

The whitepaper explicitly places distributed execution outside its scope and lists extension to distributed systems as an open question. The "offline work" in the vision document concerns schema authoring (branches, proposals), not production data. This ADR defines a stance for the production-data case.

Evidence from the study (evidence tags as in the study):

- Zero does not support offline writes and is "not designed for long periods offline" **[V]**.
- Jazz is the only tool found with a principled model for schema versions across replicas: hashed schema versions, bidirectional translation ("lenses") between adjacent versions, old clients can still write, and deploys fail when a migration path is missing. It is alpha **[V]**.
- Electric invalidates local shapes on column changes; PowerSync uses manually versioned streams; schemaless engines (Couchbase, Firestore, Ditto, CouchDB) leave schema meaning to the application **[R]**.
- Sync products have an unstable record: Atlas Device Sync reached end of life on 30 Sep 2025, InstantDB is sunsetting, Triplit's team joined Supabase, Replicache is in maintenance mode **[R]**. ODK's own entity-conflict logic needed corrections **[R]**.
- Documented hard problems: business-level conflicts, referential integrity across replicas, deletes and tombstone retention (a device offline longer than the purge window may never learn of a delete), authorisation on partial replicas, and clocks **[R]**.
- Public evidence that custom field-app developers lack tooling is weak **[R]**. Form builders (ODK, KoboToolbox, CommCare, Survey123) already serve the form-based niche.

## Options considered

- **A. Build our own sync engine and protocol.**
- **B. Contract layer plus adapters to existing sync engines.** ChameleonDB owns the schema contract, compatibility and sync policy; established engines move the data.
- **C. Stay out of production sync.**

## Proposal

Adopt **B**, staged, and do not build a sync engine unless integration is shown to be insufficient.

1. **Replica schema identity.** Each replica can report the digest of the revision it speaks ([ADR-0005](0005-canonical-schema-digest.md)).
2. **Compatibility verdict.** Between any two revisions, a deterministic classification — identical, backward-compatible additive, requires translation, breaking — with an explanation of which changes cause it. Usable in CI and by sync servers.
3. **Sync policy metadata in the schema.** Authority per entity, merge strategy per field, replication scope, retention of deletions and a declared maximum offline window; validated by the type checker. A deterministic check can reject a configuration whose tombstone retention is shorter than the declared offline window.
4. **Adapters to existing engines**, chosen by explicit criteria: licence, backend support, maturity, behaviour on schema change, and independence from a single vendor's hosted service.
5. **Own protocol** only after 1–4 exist and integration has proven inadequate.

## Consequences

- Positive: builds on the project's actual differentiator (schema identity and governance); keeps scope realistic for one maintainer; avoids the correctness risk and maintenance cost of a sync engine.
- Negative: depends on third-party engines whose licences (for example a source-available server licence) and futures must be evaluated; the guarantees ChameleonDB can state are limited to what it verifies (compatibility and policy consistency), not to end-to-end convergence.
- No public claim about offline or distributed support is made before the validation below ([ADR-0002](0002-honest-claims-policy.md)).

## Validation

1. **Interviews:** 8–10 developers who build offline or field-data applications, focused on schema changes while devices are offline, conflict handling and what they use today. (No design partner exists; the earlier informal conversation is not treated as validation.)
2. **Prototype:** implement the compatibility verdict on the canonical model and run it against the real historical schema changes of 3–5 open-source projects; review the classification manually.
3. **Integration spike:** evaluate two candidate sync engines against the criteria in point 4 with a time box of one week each.
4. **Success criteria:** at least some interviewees confirm schema evolution across long-offline replicas as a recurring pain; the verdict classifies historical changes correctly with explainable output; at least one engine can be driven from the schema's policy metadata without forking it.

## Open questions

Which conflict semantics the schema can express without becoming a programming language; how authorisation on partial replicas is modelled; how the compatibility verdict treats renames and data migrations; whether the git-based history ([ADR-0004](0004-git-as-history-substrate.md)) is also the distribution channel for revisions to devices.
