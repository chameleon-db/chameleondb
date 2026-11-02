# ADR-0006: Levels of control and role-based enforcement

- **Status:** Proposed (requires a spike)
- **Date:** 2026-10-24
- **Related:** [vision-and-foundations §8–§9](../vision-and-foundations.md), landscape and pain study §4.2 (private study)

## Context

ChameleonDB can only guarantee what it controls. If any person, service or agent holds privileged database credentials, changes can bypass ChameleonDB. The existing integrity modes (`readonly`, `standard`, `privileged`, `emergency`) are local, file-based settings enforced by the CLI; they do not constrain what the database allows. The market study found no surveyed tool that states, per database, what level of control it actually has. Reported incidents involving AI agents and destructive schema commands show that credential separation is the control that matters in practice.

## Options considered

1. Keep relying on CLI-level modes only.
2. Declare and verify a **control level per database**, with the strongest level enforced by database roles.
3. Run a proxy in front of the database that inspects and blocks statements.

## Proposal

Define three levels, declared per database and verified by a command (for example `chameleon verify-control`):

1. **Observe:** ChameleonDB reads the catalog and reports divergence; it has no control over writes.
2. **Mediated:** changes made through ChameleonDB follow its rules; other access can bypass them.
3. **Enforced:** database roles restrict direct changes. Suggested role split for PostgreSQL:
   - a **migrator** role that owns the schema and can run DDL, used only by ChameleonDB when applying an approved plan;
   - an **application** role with data privileges only (no DDL), used by services and by AI agents;
   - an **observer** role with read-only access to the catalog, used for drift checks.

Verification inspects the actual privileges and reports which guarantees hold. Statements about guarantees always state the residual risk (for example a superuser can bypass any role-based control).

Out-of-band DDL detection starts with portable catalog comparison; event triggers can add real-time detection where the platform allows them. Support for event triggers on managed PostgreSQL offerings is an item to verify.

## Consequences

- Positive: a concrete, testable guarantee that does not depend on trusting the client; directly addresses the "agent with production credentials" scenario.
- Negative: requires cooperation from whoever administers the database; setup steps must be clear for small teams; managed-platform restrictions may limit some checks.
- Existing modes are re-examined: they may map to policy or to workflow stage, but must not be presented as equivalent to enforcement.

## Validation (spike, time box: 2 days)

On a local PostgreSQL instance, create the three roles and show that: the application role cannot alter the schema; the migrator role can only be used through the plan-apply path; a verification command correctly reports privileges for good and bad configurations; an out-of-band `ALTER TABLE` is detected by the next observation. Document what differs on at least one managed PostgreSQL provider.
