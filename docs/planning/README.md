# Planning

How work is organised in this repository.

## Principles

- **Issues are the unit of work.** Each has context, scope and observable acceptance criteria (Given / When / Then where it helps).
- **Epics group issues.** An epic is an issue labelled `type:epic` that lists its child issues.
- **No calendar dates.** The roadmap is ordered by horizon and priority, not by date. Horizons are defined in the [product definition](../product-definition.md) and described in the [vision](../vision-and-foundations.md#17-horizons-and-candidate-epics).
- **Evidence before scope.** Epics beyond E1 and E2 stay coarse until earlier work produces evidence. Open questions become spikes with a time box.
- **Security findings are private.** They are tracked in a private security advisory and referenced publicly only in generic terms.

## Board

A GitHub Project (board view) with columns:

| Column | Meaning |
|---|---|
| **Now** | being worked on or next up; keep it small |
| **Next** | refined and ready |
| **Later** | accepted, not refined |
| **Done** | closed |

A roadmap view groups issues by milestone. Milestones are named after horizons:

| Milestone | Meaning |
|---|---|
| H0 - Trustworthy base | the existing code is predictable and claims match reality |
| H1a - Observe and drift | canonical model, introspection, baseline snapshots, drift reports |
| H1b - Controlled change | risk-classified plans, gates, verification of the control level |
| H2 - Multi-environment platform | more adapters, integrations, collaboration |
| H3 - Distributed and offline-first | compatibility between revisions, sync-engine integration |

## Labels

`type:*` (epic, story, task, bug, feature, spike), `area:*` (core, runtime, vault, ci, docs), `priority:P0|P1|P2`, `needs-triage`, `good first issue`, `help wanted`. Use the existing labels; the GitHub label list has the descriptions.

## Definition of ready

Context and scope are written; acceptance criteria are observable; dependencies and relevant ADRs are linked; size is small enough to review (about two days or less), otherwise it is split.

## Definition of done

Acceptance criteria are met; tests pass (including failure cases); docs are updated in English and match behaviour; the pull request is reviewed and merged.

## Initial backlog

[`backlog.md`](backlog.md) is the initial set of epics and issues. It was exported to GitHub on 2026-11-07 with a one-off script that is not kept in the repository. **GitHub issues and the project board are the source of truth** from then on: file new work as issues (templates in `.github/ISSUE_TEMPLATE`) and do not edit `backlog.md`. Each exported issue carries a hidden `backlog-id` marker (for example `E1-07`) in its body, which links it back to the snapshot.
