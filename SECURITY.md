# Security policy

## Status

ChameleonDB is **alpha software**. It has not been independently audited and makes no compliance or certification claims. Hardening work on query generation, the native boundary, the vault and the installation path is in progress and is being tracked privately until fixes are released. **Do not use the current alpha with untrusted input or for production data.**

## Reporting a vulnerability

Please **do not** open a public issue, pull request or discussion for a suspected vulnerability.

Report it privately through GitHub: **Security → Report a vulnerability** on this repository (private vulnerability reporting). Include:

- the affected version or commit,
- what you observed and why you believe it is a vulnerability,
- steps to reproduce, if possible,
- the impact you foresee.

You can expect an acknowledgement in a reasonable time. This is a small project maintained by a single person, so we cannot offer a fixed response SLA yet; we will tell you what we decide and credit you if you wish.

## Scope

In scope: the Rust core, the Go runtime and CLI, the vault and its integrity modes, the installation scripts and released artefacts.

Out of scope: vulnerabilities in PostgreSQL, in your own application code, or in third-party dependencies that are already publicly known and tracked upstream (report those upstream; tell us if our pinned version is affected).

## What the tool does and does not guarantee

The security model is documented in [docs/en/SECURITY.md](docs/en/SECURITY.md). Guarantees are stated with their scope and threat model; in particular, integrity checks detect unexpected changes made outside the tool but are **not** resistant to a user who can modify both the data and the checks. Statements in other documents that go beyond this are bugs; please report them.

## Supported versions

Only the latest release on `main` receives fixes while the project is in alpha.
