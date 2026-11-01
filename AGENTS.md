# AGENTS.md

Instructions for coding agents (and a quick orientation for humans) working in this repository. Keep changes small, tested and honest.

## What this project is

ChameleonDB is a schema-first tool for governing database schema change, written in Rust (core) and Go (runtime and CLI), with PostgreSQL as the only backend today. It is **alpha**. The direction is described in [docs/vision-and-foundations.md](docs/vision-and-foundations.md); decisions are in [docs/adr](docs/adr/README.md). The code does not yet implement most of the vision.

## Read first

1. [DEVELOPERS.md](DEVELOPERS.md) — build, test, layout.
2. [docs/architecture/overview.md](docs/architecture/overview.md) — current architecture and the target.
3. The ADRs referenced by the issue you are working on.
4. The issue itself: its acceptance criteria define "done".

## Rules

1. **Work from an issue.** If none exists, propose one (see `.agents/skills/write-issue`) before large changes.
2. **Honest claims.** Never describe a feature in docs, comments, CLI output or commit messages as working unless it is implemented and tested. If you find an existing overclaim, report it or fix the text ([ADR-0002](docs/adr/0002-honest-claims-policy.md)).
3. **Security-sensitive changes** (query or mutation generation, the FFI boundary, the vault, authentication, install scripts) follow `.agents/skills/security-change`. **Never put vulnerability details, proof-of-concept payloads or exploit steps in issues, pull requests, commit messages or tests that are committed**; use a private security advisory.
4. **One implementation per behaviour.** Do not add a second copy of logic that exists in the other language. Mutation and query-generation features are frozen except for fixes ([ADR-0007](docs/adr/0007-data-access-layer-scope.md)).
5. **No panics across the FFI.** Handle null pointers and malformed input; return errors.
6. **Deterministic output.** Generated SQL, migrations, hashes and listings must not depend on map iteration order or timestamps.
7. **Parameterise.** Values never become SQL text. Identifiers are validated against the schema.
8. **Tests accompany changes**, including failure and adversarial cases. Do not weaken or delete a test to make a change pass; explain why if a test must change.
9. **Documentation in English.** Update docs in the same change as the code. New architectural decisions need an ADR (`.agents/skills/write-adr`).
10. **Do not touch** `chameleon-core/include/*` by hand (generated), release workflows or install scripts without an issue that says so.
11. **No secrets, no real database URLs** in code, tests or logs.
12. **Do not push to `main`.** Work on a branch and open a pull request using the template.

## Commands

See [DEVELOPERS.md](DEVELOPERS.md). Full check: `.agents/skills/run-checks`.

## Definition of done

- Acceptance criteria in the issue are met and verifiable.
- Tests pass locally (unit, and integration when the change touches database behaviour).
- `cargo fmt`, `cargo clippy` (no new findings), `gofmt`, `go vet` are clean.
- Docs updated; claims match behaviour.
- The pull request description links the issue and states what was **not** tested.

## When unsure

Prefer a small, reversible change and a clear question in the pull request over a broad guess. If a requirement conflicts with an ADR, stop and say so.

## Skills

Playbooks live in `.agents/skills/<name>/SKILL.md`:

| Skill | Use it when |
|---|---|
| `run-checks` | verifying a change before a pull request |
| `write-adr` | recording an architectural decision |
| `claims-review` | reviewing docs or CLI text for overclaims |
| `security-change` | touching security-sensitive code |
| `write-issue` | turning a need into a well-formed issue |
