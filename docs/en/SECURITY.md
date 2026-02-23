# Security Model

ChameleonDB implements a **defense-in-depth security model** with multiple layers protecting schema integrity and access control.

---

## Overview

```
┌───────────────────────────────────────┐
│  Application Code (Restricted)        │
│  - Can only load from vault           │
│  - Mode enforcement at runtime        │
└───────────────────────────────────────┘
                  ↓
┌───────────────────────────────────────┐
│  Schema Vault (Source of Truth)       │
│  - Versioned schemas                  │
│  - SHA256 integrity                   │
│  - Immutable snapshots                │
└───────────────────────────────────────┘
                  ↑
┌───────────────────────────────────────┐
│  CLI (Trusted)                        │
│  - Merge schemas                      │
│  - Verify integrity                   │
│  - Register versions                  │
└───────────────────────────────────────┘
```

---

## Security Layers

### Layer 1: File Permissions (OS-level)

```bash
# Recommended permissions
chmod 700 .chameleon/              # Owner only
chmod 700 .chameleon/vault/
chmod 600 .chameleon/vault/auth/   # Passwords
chmod 644 .chameleon.yml           # Readable config
```

**Purpose:** Prevent unauthorized file system access

---

### Layer 2: Hash Integrity (Vault)

Every schema version is cryptographically hashed:

```
.chameleon/vault/
├── manifest.json          # Version metadata
├── versions/
│   ├── v001.json         # Schema snapshot
│   └── v002.json
└── hashes/
    ├── v001.hash         # SHA256 verification
    └── v002.hash
```

**How it works:**
1. Schema is saved to `versions/v001.json`
2. SHA256 hash computed and saved to `hashes/v001.hash`
3. On every load, hash is verified
4. If mismatch → integrity violation detected

**Purpose:** Tamper detection

---

### Layer 3: Integrity Modes (Access Control)

Four operational modes control schema modifications:

| Mode | Ring | Access | Schema Changes |
|------|------|--------|----------------|
| **readonly** | R3 | Production default | ❌ Blocked |
| **standard** | R2 | Development teams | ✅ Controlled |
| **privileged** | R1 | DBAs | ✅ Direct (logged) |
| **emergency** | R0 | Incident recovery | ✅ No checks (audited) |

**Mode enforcement:**
- Application code checks mode before operations
- Mode upgrades require password authentication
- All mode changes logged in audit trail

**Example:**
```bash
# Try to modify in readonly mode
$ chameleon migrate --apply
❌ readonly mode: schema modifications blocked

# Upgrade with password
$ chameleon config set mode=standard
🔐 Enter password: ****
✅ Mode upgraded

# Now modification allowed
$ chameleon migrate --apply
✅ Migration applied
```

**Purpose:** Runtime access control

---

### Layer 4: Vault-Enforced Loading

Application code **cannot bypass** vault:

```go
// ✅ SECURE (default)
eng, err := engine.NewEngine()
// ↑ Loads ONLY from .chameleon/state/schema.merged.cham
// ↑ Verifies integrity automatically
// ↑ Enforces mode restrictions

// ❌ INSECURE (blocked by mode)
eng.LoadSchemaFromFile("untrusted.cham")
// → Error: blocked by readonly mode
```

**CLI tools** have explicit bypass:
```go
// CLI context only
eng := engine.NewEngineForCLI()
eng.LoadSchemaFromFile("schemas/user.cham")
```

**Purpose:** Prevent schema bypass attacks

---

### Layer 5: Audit Trail

Complete event logging:

**integrity.log (append-only):**
```
2026-02-23T10:30:00Z [INIT] vault_created version=v001
2026-02-23T10:30:00Z [REGISTER] schema_registered version=v001 hash=3f2a8b9c...
2026-02-23T10:35:00Z [MIGRATE] migration_applied version=v001 tables_created=3
2026-02-23T15:45:00Z [MODE_CHANGE] from=readonly to=privileged type=upgrade
2026-02-23T15:50:00Z [SCHEMA_PATH] action=schema_paths_changed new_paths=schemas/ mode=privileged
```

**journal (structured):**
```json
{
  "timestamp": "2026-02-23T10:30:00Z",
  "action": "migrate",
  "status": "applied",
  "details": {
    "version": "v001",
    "duration_ms": 45
  }
}
```

**Purpose:** Forensics and compliance

---

## Threat Model

### What ChameleonDB Protects Against

✅ **Schema tampering**
- Hashes detect file modifications
- Integrity check runs on every operation

✅ **Unauthorized schema changes**
- Mode enforcement blocks operations
- Password required for mode upgrades

✅ **Schema bypass attacks**
- Application code cannot load arbitrary schemas
- Vault is the only trusted source

✅ **Privilege escalation**
- Mode upgrades require password
- All escalations logged

✅ **Audit trail tampering**
- integrity.log is append-only
- Deletion/modification detected by monitoring

---

### What ChameleonDB Does NOT Protect Against

❌ **Root/admin access**
- OS-level root can modify anything
- Solution: Use OS access controls (sudoers, SELinux)

❌ **Database compromise**
- ChameleonDB doesn't secure the database itself
- Solution: Use database security (SSL, auth, encryption at rest)

❌ **Memory attacks**
- Passwords in memory during operation
- Solution: Use memory protection (ASLR, DEP)

❌ **Social engineering**
- User gives away password
- Solution: Security training, MFA for production

---

## Best Practices

### 1. File Permissions

```bash
# Set once after init
chmod 700 .chameleon/
chmod 600 .chameleon/vault/auth/mode.key
```

### 2. Password Management

```bash
# Set strong password
chameleon config auth set-password

# Use environment variable for CI/CD
export CHAMELEON_MODE_PASSWORD="strong-password"
```

### 3. Mode Strategy

```
Development:  standard (controlled changes)
Staging:      readonly (verify before prod)
Production:   readonly (locked)
Maintenance:  privileged (temporary, logged)
Emergency:    emergency (rare, fully audited)
```

### 4. Git Strategy

**DO commit:**
```gitignore
✅ .chameleon.yml (no secrets)
✅ vault/manifest.json (public metadata)
✅ schemas/*.cham (source schemas)
```

**DON'T commit:**
```gitignore
❌ vault/auth/ (passwords)
❌ .env (secrets)
❌ state/schema.merged.cham (generated)
```

### 5. Secrets Management

**Never in config files:**
```yaml
# ❌ BAD
database:
  password: "hardcoded123"

# ✅ GOOD
database:
  connection_string: "${DATABASE_URL}"
```

**Use environment variables:**
```bash
export DATABASE_URL="postgresql://user:pass@host:5432/db"
```

---

## Configuration Security

### .chameleon.yml

```yaml
# No secrets in this file!
database:
  connection_string: "${DATABASE_URL}"  # ← From env

security:
  directory_permissions: "0700"
  verify_on_startup: true
  log_mode_changes: true

paranoia:
  mode: readonly
  require_password: true
```

### Environment Variables

```bash
# .env (gitignored)
DATABASE_URL=postgresql://user:pass@host:5432/db
CHAMELEON_MODE_PASSWORD=strong-password
```

Load with:
```bash
export $(cat .env | xargs)
```

---

## Compliance

### Audit Requirements

ChameleonDB provides:
- ✅ Complete audit trail (who, what, when)
- ✅ Tamper detection (hash verification)
- ✅ Access control (mode enforcement)
- ✅ Non-repudiation (all actions logged)

**View audit trail:**
```bash
# Integrity log
cat .chameleon/vault/integrity.log

# Journal
chameleon journal last 100

# Schema history
chameleon journal schema
```

---

## Security Checklist

Before deploying to production:

```
Security Configuration:
[ ] File permissions set (700 for .chameleon/)
[ ] Mode password configured
[ ] Mode set to readonly
[ ] DATABASE_URL in environment (not config)
[ ] .env file gitignored

Verification:
[ ] chameleon verify passes
[ ] No secrets in .chameleon.yml
[ ] Audit logs working
[ ] Mode upgrades require password

Monitoring:
[ ] integrity.log monitored for violations
[ ] journal reviewed regularly
[ ] Unexpected mode changes alerted
```

---

## Incident Response

### Integrity Violation Detected

```bash
$ chameleon verify
❌ INTEGRITY VIOLATION
   v001.json: hash mismatch

# Response steps:
1. Stop all migrations immediately
2. Review integrity.log for tampering
3. Restore from backup if available
4. Investigate access logs
5. Rotate passwords
6. Document incident
```

### Unauthorized Mode Change

```bash
# Check journal
$ chameleon journal last 50 | grep mode

# If unauthorized:
1. Change mode password immediately
2. Review who has access
3. Audit recent schema changes
4. Check for unexpected migrations
```

---

## Summary

ChameleonDB security model:
- ✅ Multi-layered defense (OS + vault + modes + audit)
- ✅ Tamper detection (SHA256 hashing)
- ✅ Access control (password-protected modes)
- ✅ Complete audit trail (append-only logs)
- ✅ Fail-safe defaults (readonly mode)

**Security is not optional** — it's built into the core design.