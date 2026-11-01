---
name: write-issue
description: Turn a need, bug or idea into a well-formed backlog issue with context, scope, acceptance criteria, labels, epic and milestone, in the backlog format used by this repository.
---

# write-issue

Issues are authored in `docs/planning/backlog.md` (exported to GitHub by `scripts/planning/export_backlog.py`) or directly in GitHub using the templates.

## Structure

- **Title:** imperative, specific, under 80 characters.
- **Context:** why this matters; links to ADRs or the vision.
- **Scope:** what is included and what is explicitly not.
- **Acceptance criteria:** observable, testable statements, preferably Given / When / Then.
- **Notes:** constraints, dependencies, risks.

## Backlog entry format

```
## [E1-07] Title
- type: story | task | bug | spike | epic
- epic: E1
- labels: area:core, priority:P1
- milestone: H0

Body in markdown.
```

## Rules

- One issue = one reviewable change, ideally under two days of work. Larger work becomes an epic with sub-issues.
- Acceptance criteria must be checkable by someone who did not write the issue.
- Security findings are **not** written as public issues; see `security-change`.
- Do not promise dates. Use milestones (H0, H1, …) and priority labels.
- Spikes state a time box and the decision they unblock.
