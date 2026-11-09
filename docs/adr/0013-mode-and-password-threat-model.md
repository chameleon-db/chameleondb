# ADR-0013: Mode and password: what they protect and what they do not

- **Status:** Proposed (H0 is implemented; H1 needs a spike)
- **Date:** 2026-11-09
- **Related:** [ADR-0002](0002-honest-claims-policy.md), [ADR-0004](0004-git-as-history-substrate.md), [ADR-0006](0006-levels-of-control-and-role-enforcement.md), [ADR-0012](0012-config-flags-roadmap.md)

## Context

A project has a *paranoid mode* (`readonly`, `standard`, `privileged`, `emergency`). The first `migrate --apply` leaves it in `readonly`, which blocks further schema changes. Raising it needs a password, set with `chameleon config auth set-password`. The mode lives in `.chameleon/vault/mode.json` and the password verifier in `.chameleon/vault/mode_auth.json` (git-ignored). The CLI, not the configuration file, is the way to change both.

Checking how this behaves (tests against the real `vault` package, v1.1.3-alpha) showed that it is not a barrier against anyone who can write to the project directory:

| Action | Result |
|---|---|
| Edit `mode.json` from `readonly` to `standard` | Accepted. No password is asked |
| Run `set-password` when a password exists | Replaces it without asking for the old one |
| Delete `mode_auth.json` | No password is configured any more; a new one can be set |
| Run `verify` after editing `mode.json` | Reports the vault as valid |

The password was stored as one round of salted SHA-256, which is not a password-hashing function: it can be guessed offline at billions of attempts per second on a GPU.

[ADR-0006](0006-levels-of-control-and-role-enforcement.md) already says the modes "must not be presented as equivalent to enforcement". [ADR-0002](0002-honest-claims-policy.md) asks that security statements name their threat model. This ADR does both for modes and the password.

## Decision

### 1. What a mode is

A mode is a **guard against accidental changes and a stage in a workflow**. It stops `chameleon` from applying a migration by mistake. It is **not access control**: anyone who can run the CLI as the project's user, or write to `.chameleon/`, can change it. Documentation, CLI help and release notes say so. Wording such as "protects", "secures" or "locks" is not used for modes; "guards against accidental schema changes" is.

Real enforcement is database roles ([ADR-0006](0006-levels-of-control-and-role-enforcement.md)). Nothing in this ADR replaces that.

### 2. Password storage (H0, implemented)

- **Argon2id**, stored as a PHC string (`$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>`): the OWASP minimum parameters (19 MiB, 2 iterations, 1 lane), a random 16-byte salt per password, a 32-byte output, comparison in constant time. The parameters live in the string, so they can be raised later.
- A verifier written by an earlier version (SHA-256) is accepted once and replaced by an Argon2id one on the first successful check.
- When a file is read, the parameters are bounded (memory, iterations, lanes), so that editing the file cannot make verification consume unbounded resources.
- Minimum length 8, no composition rules, surrounding whitespace ignored (NIST SP 800-63B).
- `set-password` creates the first password. **Changing it requires the current one.** Interactive input asks twice; non-interactive use takes `CHAMELEON_MODE_PASSWORD` (current or first) and `CHAMELEON_MODE_PASSWORD_NEW` (replacement). Environment variables are visible to other processes of the same user, and the documentation says so.
- Log entries (`password_configured`, `password_rotated`, `password_rotation_denied`, `password_hash_upgraded`) never contain secrets or hashes.

### 3. Detecting edits to the mode (H0, implemented)

The integrity log records the mode when the vault is created and on every successful change. `verify`, `migrate` and the Go engine's start-up compare `mode.json` with the last recorded mode, and report a mismatch as an integrity failure.

This **detects an edit of `mode.json` made outside the CLI**. It does not stop it, and it does not detect an edit of both files, because the log is a plain text file. It does not apply to vaults created before the initial mode was recorded and with no mode change since: there is nothing to compare, and they are not flagged. Because `vault/` is versioned, a change to either file also shows in the diff of a pull request ([ADR-0004](0004-git-as-history-substrate.md)); that review is the practical control.

### 4. What H0 still does not do

- Deleting `mode_auth.json` and setting a new password still works, because a clone of the repository legitimately has no `mode_auth.json` (it is git-ignored).
- Editing `mode.json` and `integrity.log` together is not detected.
- A program that calls the `vault` package directly is not subject to any of this.
- Anyone with database credentials is outside all of it.

### 5. H1: signed mode state (needs a spike)

Direction, not a commitment:

- At password creation, generate a random Ed25519 key pair. Store the **private key encrypted** with a key derived from the password (Argon2id plus an authenticated cipher) in the git-ignored `mode_auth.json`. Commit the **public key**.
- A mode change writes `{mode, sequence, previous signature}` signed with the private key, which needs the password. Reading the mode verifies the signature with the public key and needs no password. An invalid or missing signature falls back to `readonly` and is reported.
- The public key is random, not derived from the password, so committing it gives an attacker nothing to guess offline. Replacing it is visible in the diff.
- Open questions for the spike: how a second machine or a CI runner obtains the ability to change the mode (export and import of the encrypted key, or one key per person); recovery when the password is lost; whether the sequence number is enough against replaying an old signed state.

### 6. H2

Database roles, as in [ADR-0006](0006-levels-of-control-and-role-enforcement.md). Modes then become policy over enforced roles, not a substitute for them.

## Consequences

- Positive: passwords get a standard hash and cannot be replaced without the current one; the cheap tamper (editing `mode.json`) is detected; claims match the threat model.
- Negative: a new dependency (`golang.org/x/crypto`); a tampered `mode.json` now makes `verify`, `migrate` and the engine start-up fail, which is the point but may surprise someone who edited it by hand; the strongest attacks (deleting the password file, editing both files) remain until H1.
- Follow-up: the H1 spike; wording changes in the README, the CLI help, the Documento Maestro and `docs/cli/config`; a decision on whether `CHAMELEON_MODE_PASSWORD` should be accepted at all outside CI.
