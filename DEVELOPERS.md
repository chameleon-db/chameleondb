# Developer guide

How to build, test and change ChameleonDB. For the project's direction see [the vision](docs/vision-and-foundations.md) and the [ADR index](docs/adr/README.md). For contribution rules see [CONTRIBUTING.md](CONTRIBUTING.md).

> The project is alpha. Commands below were verified on Linux. Windows and macOS contributor setups are not yet documented here.

## Repository layout

```
chameleon-core/     Rust core: parser, AST, type checker, SQL / mutation / migration generators, C ABI
chameleon/          Go runtime and CLI (cobra), vault, journal, PostgreSQL connector, introspection
  cmd/chameleon/      CLI commands
  pkg/engine/         query and mutation engine (public Go API)
  pkg/vault/          schema vault and integrity modes
  internal/           config, state, journal, FFI bridge, admin (dashboard)
  tests/integration/  integration tests (need PostgreSQL)
examples/           sample schemas
dist/               install and uninstall scripts
docs/               documentation (English); ADRs in docs/adr
.agents/skills/     task playbooks for coding agents
```

The Go code calls the Rust core through cgo. The Rust library must be built before the Go binary. How long this split stays is under review in [ADR-0011](docs/adr/0011-language-runtime-and-distribution.md).

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Rust | stable toolchain | `cargo`, `clippy`. The README badge says 1.75+; the exact minimum is not verified. |
| Go | 1.25.6 or newer | taken from `chameleon/go.mod` |
| C toolchain | any | needed by cgo |
| Docker with Compose | recent | integration tests start PostgreSQL 16 through `docker compose` |
| make | any | |

Optional quality tools: `cargo audit`, `govulncheck` (`go install golang.org/x/vuln/cmd/govulncheck@latest`).

Podman: the Makefile calls `docker compose`. Running the integration tests under Podman is not yet supported by the project; see the issue on portable integration tests.

## Build

All common tasks run from the repository root. `make help` lists them.

```bash
make build            # builds the Rust core, then chameleon/bin/chameleon
```

The root `Makefile` only delegates: the Go recipes are in `chameleon/Makefile` and the Rust ones are plain `cargo` commands in `chameleon-core`. You can still run those directly.

## Releases

Release notes first, tag last. A release is made from `main`, in this order:

1. In a pull request, add the `## [vX.Y.Z]` section to `CHANGELOG.md` and update `RELEASE.md` (current release, platforms, limits). Merge it.
2. From an up-to-date `main`, run:

   ```bash
   make tag                  # proposes the next patch version; press Enter to accept it
   make tag VERSION=1.2.0    # or choose the version
   ```

The script (`scripts/tag-release.sh`) refuses to continue when:

- the working tree has uncommitted changes, the current branch is not `main`, or `main` is not the same commit as `origin/main`;
- the version is invalid, equal to the current one, or one of its tags already exists;
- `CHANGELOG.md` has no `## [vX.Y.Z]` section (`vX.Y.Z-alpha` is also accepted) or `RELEASE.md` does not mention `[vX.Y.Z-alpha]`.

Otherwise it updates the version in `chameleon-core/Cargo.toml`, runs `cargo check`, and shows the diff. After confirmation it creates the commit `chore: release vX.Y.Z` and two tags: `vX.Y.Z` (used by CI) and `chameleon/vX.Y.Z` (identifies the component). It asks separately before pushing; it pushes the branch first, then both tags, and verifies the three remote references. If you decline the push, it prints the commands to finish by hand. If preparing the release fails, `Cargo.toml` is restored.

The release workflow runs only when a `vX.Y.Z` tag is pushed; commits and pull requests do not start it. It refuses to build when the tag and the Cargo manifest version differ.

## Test

```bash
make test               # Rust unit and FFI tests, then Go unit tests
make test-core          # Rust only
make test-go            # Go unit tests (builds the Rust library first)
make test-integration   # Go integration tests against PostgreSQL 16 (starts and stops a container on port 5433)
make test-all           # Rust tests plus every Go test
```

Integration tests read their connection settings from environment variables, with these defaults:

| Variable | Default |
|---|---|
| `CHAMELEON_TEST_DB_HOST` | `localhost` |
| `CHAMELEON_TEST_DB_PORT` | `5433` |
| `CHAMELEON_TEST_DB_NAME` | `chameleon_test` |
| `CHAMELEON_TEST_DB_USER` | `postgres` |
| `CHAMELEON_TEST_DB_PASS` | `postgres` |

To run one Go test: `cd chameleon && CGO_LDFLAGS="-L$PWD/../chameleon-core/target/release -lchameleon" CGO_CFLAGS="-I$PWD/../chameleon-core/include" LD_LIBRARY_PATH=$PWD/../chameleon-core/target/release go test ./pkg/engine -run TestName -count=1`.

## Quality gates

Before opening a pull request:

```bash
cd chameleon-core && cargo fmt --check && cargo clippy --all-targets
cd ../chameleon && gofmt -l . && go vet ./...
cd .. && make test
```

Known state at the last review: clippy reports warnings and a few default-deny errors that are tracked as issues; new code must not add more. `cargo audit` and `govulncheck` are expected to be clean once the dependency-update issue is closed.

## Working on the Rust core

- Grammar: `chameleon-core/src/parser` (LALRPOP). Type checker: `src/typechecker`. SQL generation: `src/sql`.
- The C header in `chameleon-core/include` is generated by `build.rs` (cbindgen). Change the Rust FFI, not the header by hand.
- Anything crossing the FFI boundary must handle null pointers and invalid input without panicking.
- Generated SQL must be deterministic and must never embed user-supplied values as SQL text. If you touch query or mutation generation, read [SECURITY.md](SECURITY.md) first and use the `security-change` skill.

## Working on the Go runtime

- CLI commands live in `chameleon/cmd/chameleon`, one file per command with its tests.
- Database access goes through the engine and connector packages; do not build SQL strings in command code.
- Vault and integrity-mode changes need tests for the failure paths, not only the success paths.

## Documentation

- Documentation is written in **English**.
- Decisions are recorded as ADRs ([ADR-0001](docs/adr/0001-record-architecture-decisions.md)); use the template in `docs/adr/README.md`.
- Describe what the code does today separately from what is planned. Do not document a capability as available until it is implemented and tested ([ADR-0002](docs/adr/0002-honest-claims-policy.md)).

## Planning

Work is tracked in GitHub issues and the project board. The backlog is organised in epics, with milestones named after horizons (H0, H1, …) instead of dates. See [docs/planning/README.md](docs/planning/README.md).

## Agents

Coding agents should start with [AGENTS.md](AGENTS.md) and the playbooks in `.agents/skills/`.
