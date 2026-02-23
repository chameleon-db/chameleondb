# Modelo de Seguridad

ChameleonDB implementa un **modelo de seguridad en profundidad** con múltiples capas que protegen la integridad del schema y el control de acceso.

---

## Visión General

```
┌────────────────────────────────────────┐
│  Código de Aplicación (Restringido)    │
│  - Solo puede cargar desde vault       │
│  - Aplicación de modo en runtime       │
└────────────────────────────────────────┘
                  ↓
┌────────────────────────────────────────┐
│  Schema Vault (Fuente de Verdad)       │
│  - Schemas versionados                 │
│  - Integridad SHA256                   │
│  - Snapshots inmutables                │
└────────────────────────────────────────┘
                  ↑
┌────────────────────────────────────────┐
│  CLI (Confiado)                        │
│  - Merge de schemas                    │
│  - Verificar integridad                │
│  - Registrar versiones                 │
└────────────────────────────────────────┘
```

---

## Capas de Seguridad

### Capa 1: Permisos de Archivos (Nivel SO)

```bash
# Permisos recomendados
chmod 700 .chameleon/              # Solo el dueño
chmod 700 .chameleon/vault/
chmod 600 .chameleon/vault/auth/   # Contraseñas
chmod 644 .chameleon.yml           # Config legible
```

**Propósito:** Prevenir acceso no autorizado al sistema de archivos

---

### Capa 2: Integridad de Hash (Vault)

Cada versión del schema tiene un hash criptográfico:

```
.chameleon/vault/
├── manifest.json          # Metadatos de versiones
├── versions/
│   ├── v001.json         # Snapshot del schema
│   └── v002.json
└── hashes/
    ├── v001.hash         # Verificación SHA256
    └── v002.hash
```

**Cómo funciona:**
1. El schema se guarda en `versions/v001.json`
2. Se calcula el hash SHA256 y se guarda en `hashes/v001.hash`
3. En cada carga, se verifica el hash
4. Si hay discrepancia → se detecta violación de integridad

**Propósito:** Detección de manipulaciones

---

### Capa 3: Modos de Integridad (Control de Acceso)

Cuatro modos operativos controlan las modificaciones del schema:

| Modo | Anillo | Acceso | Cambios de Schema |
|------|--------|--------|-------------------|
| **readonly** | R3 | Producción por defecto | ❌ Bloqueado |
| **standard** | R2 | Equipos de desarrollo | ✅ Controlado |
| **privileged** | R1 | DBAs | ✅ Directo (logueado) |
| **emergency** | R0 | Recuperación de incidentes | ✅ Sin controles (auditado) |

**Aplicación de modo:**
- El código de la aplicación verifica el modo antes de las operaciones
- Los upgrades de modo requieren autenticación con contraseña
- Todos los cambios de modo se registran en la traza de auditoría

**Ejemplo:**
```bash
# Intentar modificar en modo readonly
$ chameleon migrate --apply
❌ readonly mode: modificaciones de schema bloqueadas

# Upgrade con contraseña
$ chameleon config set mode=standard
🔐 Ingresá contraseña: ****
✅ Modo actualizado

# Ahora la modificación está permitida
$ chameleon migrate --apply
✅ Migración aplicada
```

**Propósito:** Control de acceso en runtime

---

### Capa 4: Carga Aplicada por Vault

El código de la aplicación **no puede bypassear** el vault:

```go
// ✅ SEGURO (por defecto)
eng, err := engine.NewEngine()
// ↑ Carga SOLO desde .chameleon/state/schema.merged.cham
// ↑ Verifica integridad automáticamente
// ↑ Aplica restricciones de modo

// ❌ INSEGURO (bloqueado por modo)
eng.LoadSchemaFromFile("untrusted.cham")
// → Error: bloqueado por readonly mode
```

**Propósito:** Prevenir ataques de bypass del schema

---

### Capa 5: Traza de Auditoría

Registro completo de eventos:

**integrity.log (append-only):**
```
2026-02-23T10:30:00Z [INIT] vault_created version=v001
2026-02-23T10:30:00Z [REGISTER] schema_registered version=v001 hash=3f2a8b9c...
2026-02-23T10:35:00Z [MIGRATE] migration_applied version=v001 tables_created=3
2026-02-23T15:45:00Z [MODE_CHANGE] from=readonly to=privileged type=upgrade
2026-02-23T15:50:00Z [SCHEMA_PATH] action=schema_paths_changed new_paths=schemas/ mode=privileged
```

**journal (estructurado):**
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

**Propósito:** Forense y cumplimiento

---

## Modelo de Amenazas

### Contra qué protege ChameleonDB

✅ **Manipulación del schema**
- Los hashes detectan modificaciones de archivos
- La verificación de integridad se ejecuta en cada operación

✅ **Cambios no autorizados en el schema**
- La aplicación de modo bloquea operaciones
- Se requiere contraseña para upgrades de modo

✅ **Ataques de bypass del schema**
- El código de la aplicación no puede cargar schemas arbitrarios
- El vault es la única fuente confiable

✅ **Escalamiento de privilegios**
- Los upgrades de modo requieren contraseña
- Todos los escalamientos se registran

✅ **Manipulación de la traza de auditoría**
- integrity.log es append-only
- La eliminación/modificación se detecta mediante monitoreo

---

### Contra qué NO protege ChameleonDB

❌ **Acceso root/admin**
- El root a nivel SO puede modificar cualquier cosa
- Solución: Usar controles de acceso del SO (sudoers, SELinux)

❌ **Compromiso de la base de datos**
- ChameleonDB no asegura la base de datos en sí
- Solución: Usar seguridad de la base de datos (SSL, autenticación, encriptación en reposo)

❌ **Ataques a memoria**
- Las contraseñas en memoria durante la operación
- Solución: Usar protección de memoria (ASLR, DEP)

❌ **Ingeniería social**
- El usuario revela la contraseña
- Solución: Capacitación en seguridad, MFA para producción

---

## Buenas Prácticas

### 1. Permisos de Archivos

```bash
# Configurar una vez después del init
chmod 700 .chameleon/
chmod 600 .chameleon/vault/auth/mode.key
```

### 2. Gestión de Contraseñas

```bash
# Establecer contraseña fuerte
chameleon config auth set-password

# Usar variable de entorno para CI/CD
export CHAMELEON_MODE_PASSWORD="contraseña-fuerte"
```

### 3. Estrategia de Modos

```
Desarrollo:  standard (cambios controlados)
Staging:     readonly (verificar antes de producción)
Producción:  readonly (bloqueado)
Mantenimiento: privileged (temporal, logueado)
Emergencia:  emergency (raro, completamente auditado)
```

### 4. Estrategia de Git

**SÍ commitear:**
```gitignore
✅ .chameleon.yml (sin secretos)
✅ vault/manifest.json (metadatos públicos)
✅ schemas/*.cham (schemas fuente)
```

**NO commitear:**
```gitignore
❌ vault/auth/ (contraseñas)
❌ .env (secretos)
❌ state/schema.merged.cham (generado)
```

### 5. Gestión de Secretos

**Nunca en archivos de configuración:**
```yaml
# ❌ MAL
database:
  password: "hardcoded123"

# ✅ BIEN
database:
  connection_string: "${DATABASE_URL}"
```

**Usar variables de entorno:**
```bash
export DATABASE_URL="postgresql://usuario:contraseña@host:5432/db"
```

---

## Seguridad de Configuración

### .chameleon.yml

```yaml
# ¡Sin secretos en este archivo!
database:
  connection_string: "${DATABASE_URL}"  # ← Desde env

security:
  directory_permissions: "0700"
  verify_on_startup: true
  log_mode_changes: true

paranoia:
  mode: readonly
  require_password: true
```

### Variables de Entorno

```bash
# .env (gitignored)
DATABASE_URL=postgresql://usuario:contraseña@host:5432/db
CHAMELEON_MODE_PASSWORD=contraseña-fuerte
```

Cargar con:
```bash
export $(cat .env | xargs)
```

---

## Cumplimiento

### Requisitos de Auditoría

ChameleonDB proporciona:
- ✅ Traza de auditoría completa (quién, qué, cuándo)
- ✅ Detección de manipulaciones (verificación de hash)
- ✅ Control de acceso (aplicación de modo)
- ✅ No repudio (todas las acciones registradas)

**Ver traza de auditoría:**
```bash
# Integrity log
cat .chameleon/vault/integrity.log

# Journal
chameleon journal last 100

# Historial de schema
chameleon journal schema
```

---

## Checklist de Seguridad

Antes de desplegar a producción:

```
Configuración de Seguridad:
[ ] Permisos de archivos establecidos (700 para .chameleon/)
[ ] Contraseña de modo configurada
[ ] Modo establecido en readonly
[ ] DATABASE_URL en entorno (no en config)
[ ] Archivo .env en gitignore

Verificación:
[ ] chameleon verify pasa
[ ] Sin secretos en .chameleon.yml
[ ] Logs de auditoría funcionando
[ ] Upgrades de modo requieren contraseña

Monitoreo:
[ ] integrity.log monitoreado por violaciones
[ ] journal revisado regularmente
[ ] Alertas por cambios de modo inesperados
```

---

## Respuesta a Incidentes

### Violación de Integridad Detectada

```bash
$ chameleon verify
❌ VIOLACIÓN DE INTEGRIDAD
   v001.json: hash mismatch

# Pasos de respuesta:
1. Detener todas las migraciones inmediatamente
2. Revisar integrity.log por manipulaciones
3. Restaurar desde backup si está disponible
4. Investigar logs de acceso
5. Rotar contraseñas
6. Documentar el incidente
```

### Cambio de Modo No Autorizado

```bash
# Verificar journal
$ chameleon journal last 50 | grep mode

# Si no está autorizado:
1. Cambiar la contraseña de modo inmediatamente
2. Revisar quién tiene acceso
3. Auditar cambios recientes en el schema
4. Verificar migraciones inesperadas
```

---

## Resumen

Modelo de seguridad de ChameleonDB:
- ✅ Defensa multicapa (SO + vault + modos + auditoría)
- ✅ Detección de manipulaciones (hashing SHA256)
- ✅ Control de acceso (modos protegidos con contraseña)
- ✅ Traza de auditoría completa (logs append-only)
- ✅ Valores por defecto a prueba de fallos (modo readonly)

**La seguridad no es opcional** — está incorporada en el diseño central.