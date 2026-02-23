# Query Reference
![Chameleon logo](../logo-200x150.png)

Las queries de ChameleonDB se construyen usando una API encadenable.
Cada query apunta a una entidad definida en tu esquema y es
validada contra ese esquema antes de ejecutarse.

Todos los ejemplos usan este esquema:
```go
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    age: int nullable,
    created_at: timestamp default now(),
    orders: [Order] via user_id,
}

entity Order {
    id: uuid primary,
    total: decimal,
    status: string,
    created_at: timestamp default now(),
    user_id: uuid,
    user: User,
    items: [OrderItem] via order_id,
}

entity OrderItem {
    id: uuid primary,
    quantity: int,
    price: decimal,
    order_id: uuid,
    order: Order,
}
```

---

## Queries Básicas

### Obtener todos

Recupera todas las instancias de una entidad.
```go
users, err := db.Users().Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users;
```

---

### Filtrar por igualdad

> **Nota:** v0.1 usa filtros basados en strings. La API futura
> usará accessors tipados generados (ver [Code Generation](#code-generation)).

**Actual (v0.1):**
```go
users, err := db.Users().
    Filter("email", "eq", "ana@mail.com").
    Execute()
```

**API futura (con code generation):**
```go
users, err := db.Users().
    Filter(User.Email.Eq("ana@mail.com")).
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE email = 'ana@mail.com';
```

---

### Filtrar por comparación

Operadores de comparación soportados.

| Operator | Significado | Ejemplo |
|----------|-------------|---------|
| `eq` | Igual | `Filter("age", "eq", 25)` |
| `neq` | Distinto | `Filter("status", "neq", "deleted")` |
| `gt` | Mayor que | `Filter("age", "gt", 18)` |
| `gte` | Mayor o igual que | `Filter("age", "gte", 18)` |
| `lt` | Menor que | `Filter("total", "lt", 100)` |
| `lte` | Menor o igual que | `Filter("total", "lte", 100)` |
| `like` | Contiene (patrón) | `Filter("name", "like", "ana")` |
| `in` | En lista | `Filter("status", "in", [...])` |

```go
users, err := db.Users().
    Filter("age", "gte", 18).
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE age >= 18;
```

---

### Múltiples filtros

Múltiples llamadas a `.Filter()` se combinan con `AND`.
```go
users, err := db.Users().
    Filter("age", "gte", 18).
    Filter("age", "lte", 65).
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE age >= 18 AND age <= 65;
```

---

### Like (coincidencia de patrones)

Busca strings usando `like`. Los wildcards (`%`) se agregan automáticamente.
```go
users, err := db.Users().
    Filter("name", "like", "ana").
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE name LIKE '%ana%';
```

---

### In (múltiples valores)

Coincide contra una lista de valores.
```go
users, err := db.Users().
    Filter("status", "in", []string{"active", "pending"}).
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE status IN ('active', 'pending');
```

---

## Relaciones

### Include (eager loading)

Carga entidades relacionadas junto con la query principal.
Sin `.Include()`, las relaciones no se traen.
```go
users, err := db.Users().
    Include("orders").
    Execute()
```

SQL generado:
```sql
-- Query principal
SELECT id, email, name, age, created_at
FROM users;

-- Eager load (query separada, matcheada por foreign key)
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);  -- IDs de la query principal
```

> ChameleonDB usa queries separadas para eager loading
> (no JOINs) para evitar duplicación de filas y mantener los resultados limpios.

---

### Nested include

Carga relaciones de varios niveles de profundidad.
```go
users, err := db.Users().
    Include("orders").
    Include("orders.items").
    Execute()
```

SQL generado:
```sql
-- 1. Query principal
SELECT id, email, name, age, created_at
FROM users;

-- 2. Carga de orders
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);

-- 3. Carga de order items
SELECT id, quantity, price, order_id
FROM order_items
WHERE order_id IN (...);  -- IDs de la query de orders
```

---

### Filtrar sobre entidad relacionada

Filtra la entidad principal basándose en una condición sobre una entidad relacionada.
Esto es diferente a filtrar los resultados incluidos.
```go
users, err := db.Users().
    Filter("orders.total", "gt", 100).
    Execute()
```

SQL generado:
```sql
SELECT DISTINCT users.id, users.email, users.name, users.age, users.created_at
FROM users
INNER JOIN orders ON orders.user_id = users.id
WHERE orders.total > 100;
```

> Al filtrar sobre una relación, ChameleonDB usa un JOIN
> automáticamente. Se agrega `DISTINCT` para evitar duplicados
> cuando un usuario tiene múltiples órdenes que coinciden.

---

### Filtrar sobre relación + include

Podés filtrar sobre una relación y también incluirla.
El filtro afecta qué usuarios se devuelven;
el include carga todas sus órdenes (no solo las que coinciden).
```go
users, err := db.Users().
    Filter("orders.total", "gt", 100).
    Include("orders").
    Execute()
```

SQL generado:
```sql
-- 1. Query principal (filtrada vía JOIN)
SELECT DISTINCT users.id, users.email, users.name, users.age, users.created_at
FROM users
INNER JOIN orders ON orders.user_id = users.id
WHERE orders.total > 100;

-- 2. Eager load de TODAS las órdenes de los usuarios que coincidieron
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);
```

---

## Queries Avanzadas

### Order by

Ordena los resultados por uno o más campos.
```go
users, err := db.Users().
    OrderBy("created_at", "desc").
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
ORDER BY created_at DESC;
```

Múltiples cláusulas de orden:
```go
users, err := db.Users().
    OrderBy("name", "asc").
    OrderBy("created_at", "desc").
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
ORDER BY name ASC, created_at DESC;
```

---

### Limit y offset

Paginar resultados.
```go
users, err := db.Users().
    OrderBy("created_at", "desc").
    Limit(10).
    Offset(20).
    Execute()
```

SQL generado:
```sql
SELECT id, email, name, age, created_at
FROM users
ORDER BY created_at DESC
LIMIT 10 OFFSET 20;
```

> **Buenas prácticas:** Usá siempre `OrderBy` con `Limit`/`Offset`
> para asegurar una paginación determinística.

---

### Combinando todo

Una query realista que combina múltiples features:
```go
users, err := db.Users().
    Filter("age", "gte", 18).
    Filter("orders.total", "gt", 50).
    Include("orders").
    Include("orders.items").
    OrderBy("created_at", "desc").
    Limit(10).
    Execute()
```

SQL generado:
```sql
-- 1. Query principal
SELECT DISTINCT users.id, users.email, users.name, users.age, users.created_at
FROM users
INNER JOIN orders ON orders.user_id = users.id
WHERE users.age >= 18
  AND orders.total > 50
ORDER BY users.created_at DESC
LIMIT 10;

-- 2. Eager load de orders
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);

-- 3. Eager load de order items
SELECT id, quantity, price, order_id
FROM order_items
WHERE order_id IN (...);
```

---

## Validación de Queries

Todas las queries se validan antes de ejecutarse.
ChameleonDB va a detectar estos errores en tiempo de compilación:

| Error | Ejemplo |
|-------|---------|
| Entidad desconocida | `db.Products()` cuando `Product` no está en el schema |
| Campo desconocido | `.Filter("phone", ...)` cuando `phone` no es un campo |
| Ruta de relación inválida | `.Include("orders.address")` cuando `address` no existe |
| Type mismatch | `.Filter("age", "eq", "not a number")` |
| Falta order con paginación | `.Limit(10)` sin `.OrderBy()` genera un warning |

---

## Limitaciones (v0.1)

Estas funcionalidades **no están soportadas** en la versión actual:

- Agregaciones (`count`, `sum`, `avg`)
- Group by
- Subqueries
- Transacciones

Están planificadas para versiones futuras.