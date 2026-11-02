---
name: claims-review
description: Review documentation, README text, CLI help and release notes for statements that overstate what the code does; propose precise wording.
---

# claims-review

Applies [ADR-0002](../../../docs/adr/0002-honest-claims-policy.md) and [ADR-0003](../../../docs/adr/0003-initial-audience-and-claims-boundary.md).

## Steps

1. List every statement about behaviour in the files under review: guarantees ("prevents", "ensures", "immutable", "secure", "compliant"), availability ("supports", "available now"), and dates or versions.
2. For each statement, locate the code and the test that demonstrates it. If neither exists, mark it **unsupported**.
3. Check the scope: does the guarantee hold only for operations mediated by the tool? Against which threat (accidental change, another user, a privileged local user)?
4. Check consistency: versions, toolchain requirements, command names and flags against the code.
5. Output a table: statement, location, evidence (file or test), verdict (supported / partly / unsupported), proposed wording.

## Rules

- No compliance, certification or production-readiness claims.
- Prefer "detects" over "prevents" unless prevention is enforced and tested.
- Roadmap items are labelled as planned and carry no dates.
