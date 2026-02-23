# Guía de Inicio Rápido

Empezá con ChameleonDB en 5 minutos.

---

## Prerrequisitos

- Go 1.21+
- PostgreSQL 14+
- CLI de ChameleonDB instalado

**Instalar ChameleonDB:**
```bash
curl -sSL https://chameleondb.dev/install | sh
```

O compilar desde el fuente:
```bash
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make build
make install
```

**Verificar la instalación:**
```bash
chameleon --version
# Salida: chameleon v1.0-alpha
```

---

## Paso 1: Inicializar Proyecto

```bash
mkdir my-app
cd my-app
chameleon init
```

**Qué sucede:**
- Crea el directorio `.chameleon/`
- Inicializa el Schema Vault
- Crea el archivo `schema.cham` por defecto
- Establece el modo en `readonly`

**Salida:**
```
✅ Directorio .chameleon/ creado
✅ Schema Vault inicializado
✅ schema.cham creado
ℹ️  Paranoid Mode: readonly
💡 Tip: Establecé la contraseña del modo con 'chameleon config auth set-password'
```

---

## Paso 2: Definir tu Schema

Editá `schema.cham`:

```go
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    created_at: timestamp default now(),
    posts: [Post] via author_id,
}

entity Post {
    id: uuid primary,
    title: string,
    content: string,
    published: bool default false,
    created_at: timestamp default now(),
    author_id: uuid,
    author: User,
}
```

**Validar:**
```bash
chameleon validate
```

**Salida:**
```
✅ Schema validado exitosamente
   Entidades: 2 (User, Post)
   Relaciones: 2 (users.posts, posts.author)
```

---

## Paso 3: Ejecutar Migración

**Configurar DATABASE_URL:**
```bash
export DATABASE_URL="postgresql://usuario:contraseña@localhost:5432/mydb"
```

**Ejecutar migración:**
```bash
chameleon migrate --apply
```

**Salida:**
```
📦 Inicializando Schema Vault...
   ✓ .chameleon/vault/ creado
   ✓ Schema registrado como v001
   ✓ Hash: 3f2a8b9c...

📋 Vista Previa de la Migración:
─────────────────────────────────────────────────
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE posts (
    id UUID PRIMARY KEY,
    title VARCHAR NOT NULL,
    content TEXT,
    published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    author_id UUID REFERENCES users(id)
);
─────────────────────────────────────────────────

✅ Migración aplicada exitosamente
✅ Schema v001 bloqueado en vault
```

---

## Paso 4: Usar en tu Aplicación

**Inicializar módulo Go:**
```bash
go mod init my-app
go get github.com/chameleon-db/chameleondb/chameleon
```

**Crear `main.go`:**
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
    "github.com/google/uuid"
)

func main() {
    ctx := context.Background()
    
    // Conectar (carga el schema desde vault automáticamente)
    eng, err := engine.NewEngine()
    if err != nil {
        log.Fatal(err)
    }
    defer eng.Close()
    
    // Insertar usuario
    result, err := eng.Insert("User").
        Set("id", uuid.New().String()).
        Set("email", "ana@mail.com").
        Set("name", "Ana García").
        Execute(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Usuario creado: %v\n", result.ID)
    
    // Consultar usuarios
    users, err := eng.Query("User").
        Filter("email", "eq", "ana@mail.com").
        Execute(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    for _, user := range users.Rows {
        fmt.Printf("Usuario: %s <%s>\n", user["name"], user["email"])
    }
}
```

**Ejecutar:**
```bash
go run main.go
```

**Salida:**
```
Usuario creado: 550e8400-e29b-41d4-a716-446655440000
Usuario: Ana García <ana@mail.com>
```

---

## Paso 5: Consultar con Relaciones

```go
// Consultar usuarios con sus posts
result, err := eng.Query("User").
    Select("id", "name", "email").
    Include("posts").
    Execute(ctx)

if err != nil {
    log.Fatal(err)
}

for _, user := range result.Rows {
    fmt.Printf("Usuario: %s\n", user["name"])
    
    if posts, ok := result.Relations["posts"]; ok {
        fmt.Printf("  Posts: %d\n", len(posts))
        for _, post := range posts {
            fmt.Printf("  - %s\n", post["title"])
        }
    }
}
```

---

## Paso 6: Modo Debug

Ver el SQL generado:

```go
result, err := eng.Query("User").
    Filter("email", "like", "ana").
    Debug().  // ← Muestra el SQL
    Execute(ctx)
```

**Salida:**
```
[SQL] Query User
SELECT * FROM users WHERE email LIKE '%ana%'

[TRACE] Query on User: 2.3ms, 1 rows
```

---

## Próximos Pasos

### Explorar Funcionalidades

**Mutaciones:**
```bash
# Ver ejemplos
cat examples/mutations/
```

**Schema Vault:**
```bash
# Ver historial de versiones
chameleon journal schema

# Verificar integridad
chameleon verify

# Ver estado
chameleon status
```

**Introspección:**
```bash
# Generar schema desde DB existente
chameleon introspect $DATABASE_URL
```

### Aprender Más

- [Arquitectura](arquitectura.md) - Diseño del sistema
- [Query Reference](query_reference_sp.md) - API completa
- [Modelo de Seguridad](SECURITY_sp.md) - Vault y modos
- [Introspection](introspect_sp.md) - DB → Schema

---

## Problemas Comunes

### "vault not initialized"

```bash
# Solución: Ejecutar init
chameleon init
```

### "readonly mode: blocked"

```bash
# Solución: Actualizar modo
chameleon config auth set-password
chameleon config set mode=standard
```

### "integrity violation"

```bash
# Verificar qué cambió
chameleon verify

# Ver log de auditoría
cat .chameleon/vault/integrity.log
```

### "DATABASE_URL not set"

```bash
# Configurar variable de entorno
export DATABASE_URL="postgresql://usuario:contraseña@host:5432/db"
```

---

## Proyectos de Ejemplo

**TODO App:**
```bash
cd examples/todo-app
./setup.sh
make run
```

**Blog Platform:**
```bash
cd examples/blog
chameleon migrate --apply
go run main.go
```

---

## Obtener Ayuda

- **Documentación:** https://chameleondb.dev/docs
- **GitHub Issues:** https://github.com/chameleon-db/chameleondb/issues
- **Discord:** https://chameleondb.dev/discord

---

## ¿Qué Sigue?

Ya sabés:
- ✅ Cómo inicializar proyectos
- ✅ Cómo definir schemas
- ✅ Cómo ejecutar migraciones
- ✅ Cómo consultar datos
- ✅ Cómo usar el modo Debug

**Seguí aprendiendo:**
- [Query Reference](query_reference_sp.md) - Queries avanzadas
- [Modelo de Seguridad](SECURITY_sp.md) - Despliegue en producción
- [Ejemplos](../examples/) - Aplicaciones reales