# Review: `chameleon config set schema-paths` Command

## Status: ✅ Enhanced & Hardened

### Security & Access Control

✅ **Paranoid Mode Enforcement**
- Only `privileged` or `emergency` mode allowed
- Others denied with clear error message and mode upgrade path
- All permission checks logged to journal

✅ **Path Validation**
- Paths must exist (prevents arbitrary paths)
- Symlinks rejected (security risk: prevents symlink attacks)
- Each path validated individually before proceeding

✅ **Emergency Mode Extra Protection**
- Extra confirmation required when in `emergency` mode (most dangerous)
- User must type `"I understand the risks"` explicitly
- Protects against accidental schema path changes in ultra-privileged mode

### Journal Logging

✅ **Complete audit trail** (all events logged):
- `attempt` - User requested schema path change (mode + requested paths)
- `denied` - Insufficient privileges (denied reason + required mode)
- `failed` - Various failures (path not found, symlink, vault issues, config save)
- `cancelled` - User cancelled (reason: emergency confirmation or user declined)
- `changed` - SUCCESS (new paths + mode)

All events also logged to vault `SCHEMA_PATH` audit log for integrity tracking.

### Error Messages & User Guidance

✅ **Clear & Actionable**
```
Permission denied: schema path changes require elevated privileges
Changing schema paths requires privileged or emergency mode.
Current mode: readonly
Upgrade with: chameleon config set mode=privileged
```

✅ **Context-aware warnings**
- Normal mode: Standard confirmation
- Emergency mode: Extra 🚨 warning + explicit confirmation required

✅ **Helpful success output**
- Shows new paths
- Reminds user that change is logged in integrity.log
- Instructs to run `chameleon migrate` to apply changes

### Event Flow (Normal Path)

```
1. User: chameleon config set schema-paths=schemas/,legacy/

2. ATTEMPT LOG → journal: "attempt" event with mode + requested paths

3. MODE CHECK → Only privileged/emergency allowed
   ❌ DENIED → journal denied event, clear error, fail
   ✅ ALLOWED → continue

4. PATH VALIDATION → Each path must exist and not be symlink
   ❌ INVALID → journal failed event, clear error, fail
   ✅ VALID → continue

5. EMERGENCY MODE CHECK → If emergency, require extra confirmation
   ❌ USER SAYS NO → journal cancelled event, exit
   ✅ USER CONFIRMS → continue

6. NORMAL CONFIRMATION → Are you sure?
   ❌ USER SAYS NO → journal cancelled event, exit
   ✅ USER CONFIRMS → continue

7. SAVE CHANGES → Update .chameleon.yml
   ❌ SAVE ERROR → journal failed event, fail
   ✅ SAVED → continue

8. SUCCESS LOG → journal: "changed" event + vault: SCHEMA_PATH audit log

9. OUTPUT & NEXT STEPS → Print success, remind to run migrate
```

### Security Considerations

1. **Vault Requirement**: Vault must be initialized (caught early)
2. **Paranoid Mode Only**: No way around mode requirement (enforced at vault level)
3. **Path Existence**: Prevents pointing to non-existent directories
4. **Symlink Rejection**: Prevents symlink attacks / hardlink races
5. **Double Confirmation**: Emergency mode requires explicit risk acknowledgment
6. **Audit Trail**: All attempts (success/fail/cancel) logged for forensics
7. **Vault Integration**: SCHEMA_PATH audit log prevents tampering/untracking

### Testing Scenarios

```bash
# ✅ Normal flow
chameleon config set mode=privileged
chameleon config set schema-paths=schemas/,legacy/
# → SUCCESS logged in both journal & vault

# ❌ Try without privilege
chameleon config set schema-paths=schemas/
# → DENIED, logs to journal, shows error + upgrade path

# ❌ Try with non-existent path
chameleon config set mode=privileged
chameleon config set schema-paths=/nonexistent
# → FAILED, logs to journal, shows error

# ❌ Try symlink (security test)
ln -s /tmp/schemas schemas_link
chameleon config set mode=privileged
chameleon config set schema-paths=schemas_link
# → FAILED, logs security rejection to journal

# ⚠️  Emergency mode (ultra-dangerous)
chameleon config set mode=emergency
chameleon config set schema-paths=schemas/
# → Asks for explicit confirmation "I understand the risks"
# → Only if confirmed: SUCCESS logged with mode=emergency
```

### Code Quality

✅ **Separation of Concerns**
- Mode checking separate from path validation
- Each error case has dedicated logging
- Clear flow with early returns

✅ **Error Context**
- All errors include mode information
- All errors logged before returning (no silent failures)
- File system errors include path reference

✅ **User Experience**
- Warnings are prominent (⚠️ emoji)
- Success is clear (✓)
- Guidance is actionable (specific commands)

### Compliance

✅ **Security Policy**
- ✓ Only privileged/emergency can use
- ✓ All changes logged
- ✓ Symlinks blocked
- ✓ Confirmation required
- ✓ Extra confirmation in emergency

✅ **Audit Trail**
- ✓ Journal: Attempt → Denial/Success/Cancellation
- ✓ Vault: SCHEMA_PATH audit log
- ✓ Timestamp: Automatic via logger
- ✓ Context: Mode + paths + reason

---

## Summary

The `schema-paths` command is **production-ready** with:
- Strong security posture (paranoid mode only, symlink protection)
- Comprehensive audit trail (journal + vault logging)
- Clear user guidance and error messages
- Emergency mode safeguards
- Full compliance with security policy

All events are tracked (attempt/denied/failed/cancelled/changed) providing complete forensic visibility.
