# Contributing to ChameleonDB

Thank you for your interest. ChameleonDB is an early-stage project; contributions of code, tests, documentation, bug reports and design feedback are welcome.

## Before you start

- Read the [README](README.md), the [vision](docs/vision-and-foundations.md) and the [ADR index](docs/adr/README.md). The project is **alpha** and the direction is still being validated.
- Look at the open issues and the project board. Issues labelled `good first issue` are a gentle start; `help wanted` marks work where outside help is especially useful.
- For anything larger than a small fix, comment on the issue (or open one) before writing code, so we can agree on the approach.
- **Security problems are not reported in public issues.** See [SECURITY.md](SECURITY.md).

## Development setup

See [DEVELOPERS.md](DEVELOPERS.md).

## Making a change

1. Fork the repository and create a branch from `main`, for example `fix/short-description` or `feat/short-description`.
2. Keep the change focused. One pull request, one purpose.
3. Add or update tests, including failure cases.
4. Run the checks described in [DEVELOPERS.md](DEVELOPERS.md#quality-gates).
5. Update documentation in English in the same change. Do not describe behaviour that is not implemented ([ADR-0002](docs/adr/0002-honest-claims-policy.md)).
6. Open a pull request using the template and link the issue.

## Commit messages

Short imperative subject (about 70 characters), blank line, then the reason for the change. Reference issues with `Refs #123` or `Fixes #123`.

## Architecture decisions

Significant decisions are recorded as ADRs in `docs/adr`. If your change needs one, propose it in the pull request or in an issue first.

## Using coding agents

Agents are welcome as tools. The contributor is responsible for the change: review it, run the tests, and make sure the pull request description is accurate. Agents should follow [AGENTS.md](AGENTS.md).

## Licence

By contributing you agree that your contributions are licensed under the project's licence ([Apache License 2.0](LICENSE)).

## Conduct

Participation is governed by the [Code of Conduct](CODE_OF_CONDUCT.md).
