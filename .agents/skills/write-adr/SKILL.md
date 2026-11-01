---
name: write-adr
description: Record an architectural decision as an ADR in docs/adr following the project template, status rules and honest-claims policy.
---

# write-adr

Use when a change commits the project to a direction that is hard to reverse or affects several components.

## Steps

1. Read `docs/adr/README.md` (statuses, index, template) and the ADRs related to the topic.
2. Use the next free number: `docs/adr/NNNN-short-title.md`.
3. Fill the template: Status, Date, Related, Context (with the strength of the evidence), Options considered, Decision or Proposal, Consequences, Validation (spike, success criteria, time box).
4. Status rules: **Proposed** needs validation; **Open** is deliberately undecided; **Accepted** is in force and is changed only by a superseding ADR.
5. Add the ADR to the index table in `docs/adr/README.md`.
6. Write in English. Separate what exists today from what is proposed. Do not state a capability as available unless it is implemented and tested.
7. Do not include vulnerability details or private market research.

## Quality bar

An ADR a new contributor can read in five minutes and understand why the decision was taken, what was rejected, and how it would be revisited.
