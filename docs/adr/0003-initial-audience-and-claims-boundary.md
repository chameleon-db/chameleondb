# ADR-0003: Initial audience and claims boundary

- **Status:** Accepted (decided by the project owner on 2026-10-24; to be revalidated with user research)
- **Date:** 2026-10-24
- **Related:** [ADR-0002](0002-honest-claims-policy.md), landscape and pain study (private study)

## Context

The long-term ambition is a platform that larger organisations could rely on. In the near term the project has one maintainer, one real backend (PostgreSQL) and no production track record. The market study found that governance features (drift detection, approvals, audit, destructive-change linting) are mostly paid or enterprise-oriented in existing tools, while small teams are served by plain migration runners with no governance.

## Options considered

1. Target enterprises and compliance-driven teams from the start.
2. Target indie developers and small teams first; earn trust in production; move upmarket later.
3. Stay generic ("any team").

## Decision

- **Phase 1 audience:** indie developers and small teams (roughly 1–10 engineers) running PostgreSQL, who want safer schema changes without operating a server or paying per seat.
- ChameleonDB is validated in real production use with early adopters before any claim is made about larger organisations.
- **Claims boundary:** no statement that the product supports, enables or certifies compliance (for example SOC 2, ISO 27001, HIPAA, PCI) until independent evidence exists. Documentation may describe the audit and control mechanisms that exist, with their limits, and must not imply certification or regulatory fitness.
- Compliance-oriented capabilities (for example strong separation of duties, tamper-evident external logs, signed approvals) are designed with that future in mind but are not marketed in Phase 1.

## Consequences

- Positive: a focused first user, a smaller scope, and honest positioning.
- Positive: fits the observed gap (free governance for small teams) and avoids head-on competition with enterprise offerings.
- Negative: lower revenue potential early on; business-model options such as managed monitoring or support must be validated separately.
- The decision is revisited when (a) early adopters run ChameleonDB in production for a meaningful period, and (b) user interviews confirm the problem and the audience. The criteria and thresholds are to be defined with the validation plan.
