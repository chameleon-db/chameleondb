# Introspect
![Chameleon logo](../logo-200x150.png)

`chameleon introspect` inspecciona un schema de base de datos existente y genera un archivo de schema de Chameleon (`.cham`).

Este comando está pensado para bootstrapping desde bases de datos heredadas, validar el descubrimiento de tablas y generar un schema base para flujos de trabajo con migraciones.

---

## Sintaxis del Comando

```bash
chameleon introspect <database-url> [--output <file>] [--force]
```

Formas cortas:

```bash
chameleon introspect <database-url> -o schema.cham
chameleon introspect <database-url> -f
```

- `<database-url>` es obligatorio.
- La salida por defecto es `schema.cham`.
- `--force` omite las verificaciones de seguridad de sobreescritura.

---

## Variantes de Connection String

`introspect` soporta URLs directas y referencias a variables de entorno.

### 1) URL directa

```bash
chameleon introspect postgresql://user:pass@localhost:5432/mydb
```

### 2) Referencia a env estilo shell (`$VAR`)

```bash
chameleon introspect $DATABASE_URL
```

### 3) Referencia a env con llaves (`${VAR}`)

```bash
chameleon introspect ${DATABASE_URL}
```

### 4) Referencia a env explícita (`env:VAR`)

```bash
chameleon introspect env:DATABASE_URL
```

Si la variable de entorno referenciada no existe o está vacía, el comando falla con un error explícito.

---

## Verificación del Modo Paranoid

Antes de que comience la introspección, Chameleon verifica el Paranoid Mode del Schema Vault (si está inicializado).

### Comportamiento según el modo

| Modo Paranoid | Comportamiento de introspect |
|---|---|
| `readonly` | Bloqueado |
| `standard` | Permitido |
| `privileged` | Permitido |
| `emergency` | Permitido |

Cuando el modo es `readonly`, la introspección se aborta y te indica que actualices el modo.

### Flujo de upgrade desde `readonly`

Como los upgrades de modo requieren autenticación con contraseña:

```bash
chameleon config auth set-password
chameleon config set mode=standard
chameleon introspect <database-url>
```

Notas:
- Los upgrades requieren contraseña (ejemplo `readonly -> standard`).
- Los downgrades no requieren contraseña.

---

## Verificaciones de Seguridad del Archivo de Salida

Sin `--force`, introspect aplica protecciones de sobreescritura:

1. Valida que la ruta de salida no sea un directorio.
2. Detecta schemas template por defecto generados por `chameleon init`.
3. Detecta schemas modificados/en uso y sugiere un flujo de backup-o-nuevo-archivo.
4. Permite escribir archivos nuevos de forma segura.

Con `--force`, estas verificaciones se omiten.

---

## Ejemplos Completos

### Introspección baseline desde env var estilo Railway

```bash
export DATABASE_URL="postgresql://user:pass@host:5432/dbname"
chameleon introspect $DATABASE_URL -o schema.introspected.cham
```

### Introspección con sintaxis de resolución de env explícita

```bash
chameleon introspect env:DATABASE_URL --output schema.cham
```

### Sobrescribir schema existente intencionalmente

```bash
chameleon introspect $DATABASE_URL --output schema.cham --force
```

---

## Notas Operativas para Desarrolladores

- La introspección actualmente apunta al descubrimiento de metadatos relacionales y la generación de schemas.
- El schema generado debe revisarse manualmente (relaciones, nomenclatura y convenciones).
- Recomendación de seguimiento:

```bash
chameleon validate
```

Para flujos de trabajo basados en migraciones, validá el schema generado y luego continuá con tu ciclo de migraciones estándar.