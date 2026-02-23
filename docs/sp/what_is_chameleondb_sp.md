# ChameleonDB

![Chameleon logo](../logo-200x150.png)

## ¿Qué es ChameleonDB?

ChameleonDB es un **data language y runtime orientado al dominio**, diseñado para definir, validar y consultar modelos de datos como **dominios semánticos**, no como tablas SQL crudas.
Permite a los desarrolladores trabajar con backends de datos relacionales, analíticos, cacheados y otros, usando un **único lenguaje tipado**, mientras delega los detalles de ejecución a un runtime que se adapta a los motores de almacenamiento subyacentes.

ChameleonDB se enfoca en la **corrección, claridad y mantenibilidad a largo plazo** de aplicaciones con uso intensivo de datos.

---

## ¿Qué problema resuelve?

Las aplicaciones modernas suelen manejar múltiples tipos de datos:

- datos transaccionales (usuarios, órdenes)
- datos analíticos (métricas, agregados)
- datos cacheados o efímeros
- datos vectoriales o relacionados con ML

Hoy en día, esto suele resultar en:
- lógica SQL duplicada
- joins frágiles
- reglas de negocio dispersas entre consultas
- modelos inconsistentes entre capas
- errores silenciosos en tiempo de ejecución

ChameleonDB aborda esto mediante:
- definir el modelo de datos una vez, a nivel de dominio
- validar relaciones y consultas antes de la ejecución
- proveer un punto de entrada semántico único para diferentes backends de datos
- reducir el acoplamiento entre la lógica de la aplicación y los detalles de almacenamiento

---

## ¿Qué NO es ChameleonDB?

ChameleonDB **no** es:

- un motor de base de datos
- un reemplazo para PostgreSQL u otras bases de datos
- un ORM tradicional
- una herramienta de BI o reporting
- un framework web

ChameleonDB no almacena datos.
**Orquesta el acceso a backends de datos existentes**.

---

## ¿Es ChameleonDB un lenguaje de programación?

ChameleonDB incluye un **domain-specific language (DSL)** con su propia sintaxis, parser, sistema de tipos y reglas de validación.

**No es un lenguaje de propósito general**. Es un lenguaje diseñado específicamente para definir y consultar modelos de datos, compilado en un plan semántico validado que consume el runtime.

---

## ¿Es compilado o interpretado?

Los schemas y queries de ChameleonDB se **compilan en un plan semántico validado**, no en código máquina.

Este paso de compilación:
- valida tipos y relaciones
- detecta queries inválidas tempranamente
- produce un plan de ejecución que consume el runtime

No hay máquina virtual ni runtime de bytecode.

---

## ¿Cómo funciona ChameleonDB?

### 1. Definí tu dominio

Describís entidades, campos, relaciones y anotaciones usando el lenguaje de ChameleonDB:

```go
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    created_at: timestamp default now(),
    orders: [Order] via user_id,
    session: string @cache,
}

entity Order {
    id: uuid primary,
    total: decimal,
    status: string,
    user_id: uuid,
    user: User,
    items: [OrderItem] via order_id,
}
```

### 2. Validá y planeá

El core de ChameleonDB valida schemas y queries en tiempo de compilación:
- las referencias a entidades existen y son consistentes
- los destinos de las relaciones son válidos y las foreign keys coinciden
- cada entidad tiene exactamente una primary key
- las anotaciones de backend se usan correctamente
- no hay dependencias circulares de ownership

Si la validación falla, obtenés mensajes de error claros y contextualizados antes de que cualquier código se ejecute.

### 3. Ejecutá vía runtime

El runtime ejecuta las queries contra los backends configurados:

```go
// Nota: V1.0 usa filtros basados en strings. Ver Query reference para más detalles.
users := db.Users().
    Filter("email", "eq", "ana@mail.com").
    Include("orders").
    Execute()
```

La aplicación expresa la **intención**. El runtime maneja los detalles de ejecución.

---

## ¿Dónde se almacenan los datos?

Los datos se almacenan en **backends externos**, como:
- PostgreSQL (OLTP)
- almacenes analíticos (futuro)
- cachés (futuro)
- vector stores (futuro)

ChameleonDB no posee el almacenamiento.
Define **cómo se accede y valida los datos**, no dónde viven.

---

## ¿Qué son las anotaciones?

Las anotaciones proporcionan **hints semánticos** sobre cómo deberían tratarse los datos:

```rust
session_token: string @cache
monthly_spent: decimal @olap
embedding: vector(384) @vector
```

Las anotaciones:
- no cambian el modelo de dominio lógico
- no fuerzan un backend específico
- se validan en tiempo de compilación (ej., `@vector` requiere tipo `vector(N)`)
- permiten futura especialización del backend sin reescribir schemas

En versiones iniciales, las anotaciones son metadatos declarativos.
El ruteo de ejecución evoluciona con el tiempo a medida que se agregan backends.

---

## ¿Cómo se usa ChameleonDB desde las aplicaciones?

ChameleonDB provee una API runtime (actualmente en Go) que permite a las aplicaciones:
- cargar y validar schemas
- construir queries
- ejecutarlas de forma segura

Ejemplo (Go):
```go
users := db.Users().
    Filter(expr.Field("email").Eq("ana@mail.com")).
    Include("orders").
    Execute()
```

El runtime:
- valida las queries contra el schema
- las traduce a operaciones específicas del backend
- devuelve resultados estructurados

---

## ¿Para quién es ChameleonDB?

ChameleonDB está diseñado para:
- desarrolladores backend que trabajan con modelos de datos complejos
- equipos que mantienen sistemas con uso intensivo de datos
- aplicaciones que combinan datos transaccionales y analíticos
- desarrolladores que buscan garantías más sólidas que SQL crudo o los ORMs tradicionales

También está diseñado para ser **accesible a no expertos en SQL**, como analistas de datos, a través de abstracciones de más alto nivel construidas sobre el core.

---

## Alcance y filosofía del proyecto

ChameleonDB prioriza:
- la corrección sobre la conveniencia
- modelos explícitos sobre magia implícita
- la validación antes de la ejecución
- la evolución a largo plazo sobre hacks a corto plazo

No todas las funcionalidades se implementan de una vez.
La arquitectura está diseñada para **crecer sin romper modelos existentes**.

---

## Estructura del proyecto

```
chameleondb/
├── chameleon-core/          Rust — parser, type checker, AST, FFI
│   ├── src/
│   │   ├── ast/             Estructuras de datos del schema
│   │   ├── parser/          Gramática LALRPOP y parser
│   │   ├── typechecker/     Validación en tiempo de compilación
│   │   └── ffi/             Puente C ABI hacia Go
│   └── tests/               Tests de integración
│
├── chameleon/               Go — runtime, CLI, conectores a backends
│   ├── cmd/chameleon/       Herramienta CLI
│   ├── pkg/engine/          API pública del engine
│   └── internal/ffi/        Bindings CGO
│
├── examples/                Schemas .cham de ejemplo
└── docs/
    ├── en/                  Documentación en inglés
    └── sp/                  Documentación en español
```

- **Rust core** — parsing, validación y optimización
- **Go runtime** — ejecución de queries y conectividad con backends
- **FFI bridge** — interfaz C ABI entre Rust y Go (~100ns de overhead)

---

## Estado Actual

ChameleonDB **v1.0-alpha** ya está disponible.

**Qué funciona hoy:**
- ✅ DSL de schemas con entidades, campos, relaciones, anotaciones de backend
- ✅ Parser con sintaxis completa (Rust core, LALRPOP)
- ✅ Type checker con validación (relaciones, constraints, ciclos)
- ✅ **Schema Vault** - Schemas versionados y verificados por hash
- ✅ **Integrity Modes** - Governanza en runtime (readonly/standard/privileged/emergency)
- ✅ Constructor de queries (Filter, Include, Select)
- ✅ Mutaciones (Insert, Update, Delete)
- ✅ Ejecución con backend PostgreSQL
- ✅ Generación de migraciones
- ✅ **Introspection** (DB → Schema)
- ✅ **IdentityMap** (deduplicación de objetos)
- ✅ Modo debug (visibilidad de SQL)
- ✅ Mapeo de errores completo
- ✅ Herramientas CLI (8 comandos)
- ✅ **204 tests pasando**

**Madurez para producción:**
- ✅ Funcionalidades core estables
- ✅ Modelo de seguridad completo
- ⏳ La API puede tener cambios menores
- ⏳ Recomendado para evaluación y cargas de trabajo no críticas

**Qué viene después (v1.1+):**
- Soporte de transacciones
- Operaciones batch
- Generador de código (boilerplate basado en schema)
- Benchmarks de performance

**Futuro (v2.0 - Premium):**
- Ruteo de backends (múltiples bases de datos)
- Optimización de queries basada en ML
- Editor visual de schemas
- Vault distribuido