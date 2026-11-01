---
name: security-change
description: Procedure for changes to security-sensitive code (query and mutation generation, FFI, vault, authentication, install scripts) and for handling suspected vulnerabilities privately.
---

# security-change

## If you suspect a vulnerability

1. **Stop.** Do not open a public issue, pull request, commit or test containing exploit details or payloads.
2. Report it through the repository's private security advisory (see `SECURITY.md`) or tell the maintainer privately.
3. Public work may refer to it only generically, for example "harden query generation (see private advisory)".

## When changing security-sensitive code

1. Identify the trust boundary: where does untrusted input enter, and where does it reach SQL, the filesystem or the FFI?
2. Rules: values are parameters, never SQL text; identifiers are validated against the schema; errors are returned, not unwrapped; nothing panics across the FFI; secrets never appear in logs or journals.
3. Add tests for failure and adversarial inputs. Prefer tests that assert the *safe property* (for example "the value never appears in the SQL text") over tests that embed attack strings.
4. Check determinism: output must not depend on map iteration order.
5. Run `cargo clippy`, `cargo audit`, `govulncheck ./...` and the full test suite (`run-checks`).
6. State in the pull request which guarantee the change provides, its scope, and what it does not cover.
7. Update `SECURITY.md` or the documentation if the guarantee or its scope changed.
