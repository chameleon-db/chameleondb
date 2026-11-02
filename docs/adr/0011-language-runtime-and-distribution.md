# ADR-0011: Implementation languages, runtime boundary and distribution

- **Status:** Proposed (needs a spike)
- **Date:** 2026-10-24
- **Related:** [ADR-0005](0005-canonical-schema-digest.md), [ADR-0007](0007-data-access-layer-scope.md), [ADR-0009](0009-offline-first-and-distributed-data.md), [ADR-0010](0010-backend-strategy-and-capability-declarations.md)

## Context

ChameleonDB today is two programs joined by a native boundary:

- **Rust core** (`chameleon-core`): parser (LALRPOP), AST, type checker, SQL / mutation / migration generators, and a C ABI that exchanges JSON. It builds as `rlib`, `staticlib` and `cdylib`.
- **Go runtime and CLI** (`chameleon`): command-line interface, vault, journal, PostgreSQL connector (pgx), introspection and the engine. It calls the Rust core through cgo.

Observed consequences in the repository (review of 2026-10-23):

- Building the Go binary requires building the Rust library first and passing linker and include paths; the Makefile exports `CGO_LDFLAGS`, `CGO_CFLAGS` and `LD_LIBRARY_PATH` for every test target.
- The FFI boundary has had fragile spots (raw pointer handling flagged by the linter; JSON as the only contract).
- Mutation logic exists twice: an unused Rust implementation and the Go implementation actually executed.
- Toolchain versions in the README badges (Go 1.21+, Rust 1.75+) do not match `go.mod` (Go 1.25.6).
- Installation is a shell script that downloads release artefacts; the shared library must be present next to the binary.

Forces:

- The valuable new logic of the vision — canonical model and digest ([ADR-0005](0005-canonical-schema-digest.md)), compatibility verdict ([ADR-0009](0009-offline-first-and-distributed-data.md)), per-engine adapters and capability declarations ([ADR-0010](0010-backend-strategy-and-capability-declarations.md)) — is **core logic**, not CLI logic. It currently belongs in the Rust core.
- Future use on devices (embedded SQLite), single-binary installation, Windows support and bindings for other languages favour a core with a stable ABI or a WASM build.
- The Go code works, is tested (unit and integration) and is what the CLI users run today. Rewriting it has a cost and a regression risk.
- The project has one maintainer; every extra language or boundary multiplies the maintenance load.

## Options considered

- **A. Keep as is** — Rust core, Go runtime and CLI, linked through cgo.
- **B. Rust owns the core and the CLI; Go becomes a binding** — the CLI, vault, journal, adapters and engine move to Rust over time; Go remains available as a client library over the stable ABI.
- **C. Everything in Go** — considered and **not pursued** (decision of the project owner, 2026-10-24): it would require rewriting the parser, type checker and generators, and offers the weakest fit for embedded use, WASM and a stable ABI for other languages.
- **D. Rust core without cgo for Go consumers** — keep the Go CLI but call the core as a separate process or as a WASM module executed by a pure-Go runtime, removing the native link step from Go builds.

## Proposal

Lean towards **B**, with **D** as a possible transitional path, and decide only after a short spike. **Do not rewrite anything before the first epic of trustworthy-core work**.

Principles regardless of the final choice:

1. **New core logic goes in Rust** (canonical model, digest, compatibility verdict, adapter traits). It is needed under every remaining option.
2. **Exactly one implementation of each behaviour.** Mutation logic is consolidated on the side that survives the decision; until then it is frozen ([ADR-0007](0007-data-access-layer-scope.md)).
3. **The boundary is a documented contract**: versioned, tested from both sides, with explicit error handling and no panics crossing it.
4. **Installation is a first-class requirement:** a user obtains a working binary with one command and no extra libraries, on Linux, macOS and Windows.
5. Supported language bindings are stated explicitly; no generic multi-language claim ([ADR-0002](0002-honest-claims-policy.md)).

## Consequences

- If B: one toolchain for the product, a single static binary, natural path to SQLite on devices and to WASM; cost is migrating the CLI, vault, journal, introspection and connectors, and re-testing them; contributors need Rust for most changes.
- If D: smaller migration, still two languages and a process or WASM boundary to maintain.
- If A: no migration cost; installation friction and boundary risk remain, and new core logic still lands in Rust, so most contributors need both languages.

## Decision criteria

- Installation steps and time for a new user on each platform.
- Cross-compilation and release complexity (Linux, macOS, Windows; amd64 and arm64).
- Size and stability of the FFI surface; ability to test it from both sides.
- Ability to ship the core in an embedded or WASM form.
- Contributor effort: languages needed for a typical change.
- Migration cost and regression risk, measured by the existing test suites.

## Validation (spike, time box: 3 days, after the canonical-digest spike)

1. In Rust, build a minimal CLI crate that implements `check` and a `digest` command for an existing `.cham` file, reusing the core crate.
2. Cross-compile it for the three target platforms in CI and record binary size and build time.
3. Build the current Go CLI in the same CI and record the same figures plus the extra steps needed to install the shared library.
4. Prototype option D with the existing Go tests: call the core as a WASM module from Go and run the Go unit tests that touch the core.
5. Write down the results and the migration cost estimate for the CLI commands listed in [the CLI reference](../cli-reference.md). Accept, reject or revise this ADR.
