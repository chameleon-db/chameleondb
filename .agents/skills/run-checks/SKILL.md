---
name: run-checks
description: Run the full local verification for ChameleonDB (format, lint, unit and integration tests) before opening a pull request, and report results honestly.
---

# run-checks

Use before every pull request and after any change to Rust, Go or SQL generation.

## Steps

1. Rust core:
   ```bash
   cd chameleon-core
   cargo fmt --check
   cargo clippy --all-targets
   ```
2. Go runtime (builds the Rust library first):
   ```bash
   cd chameleon
   gofmt -l .
   go vet ./...
   ```
3. Tests, from the repository root:
   ```bash
   make test
   ```
   If the change touches database behaviour, vault, migrations or introspection, also run `make test-integration` (needs Docker with Compose; PostgreSQL 16 on port 5433).
4. If dependencies changed, run `cargo audit` and `govulncheck ./...` when installed.

## Reporting

- Paste the pass/fail summary per step. State which steps you could **not** run and why.
- Compare clippy output with the baseline: new findings are not acceptable; existing ones are tracked in issues.
- Never claim a check passed without having run it. Never skip or weaken a failing test to get green.
