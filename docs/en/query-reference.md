/docs/en/query-reference.md

# Query Reference
![Chameleon logo](../logo-200x150.png)

ChameleonDB queries are built using a chainable API.
Every query targets an entity defined in your schema and is
validated against it before execution.

All examples use this schema:
```rust
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

## Basic Queries

### Fetch all

Retrieve all instances from an entity.
```go
users, err := db.Users().Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users;
```

---

### Filter by equality

> **Note:** v0.1 uses string-based filters. The target API
> uses generated typed accessors (see [Code Generation](#code-generation)).

**Current (v0.1):**
```go
users, err := db.Users().
    Filter("email", "eq", "ana@mail.com").
    Execute()
```

**Target API (with code generation):**
```go
users, err := db.Users().
    Filter(User.Email.Eq("ana@mail.com")).
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE email = 'ana@mail.com';
```

---

### Filter by comparison

Supported comparison operators.

| Operator | Meaning | Example |
|----------|---------|---------|
| `eq` | Equal | `Filter("age", "eq", 25)` |
| `neq` | Not equal | `Filter("status", "neq", "deleted")` |
| `gt` | Greater than | `Filter("age", "gt", 18)` |
| `gte` | Greater than or equal | `Filter("age", "gte", 18)` |
| `lt` | Less than | `Filter("total", "lt", 100)` |
| `lte` | Less than or equal | `Filter("total", "lte", 100)` |
| `like` | Contains (pattern) | `Filter("name", "like", "ana")` |
| `in` | In list | `Filter("status", "in", [...])` |

```go
users, err := db.Users().
    Filter("age", "gte", 18).
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE age >= 18;
```

---

### Multiple filters

Multiple `.Filter()` calls are combined with `AND`.
```go
users, err := db.Users().
    Filter("age", "gte", 18).
    Filter("age", "lte", 65).
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE age >= 18 AND age <= 65;
```

---

### Like (pattern matching)

Match strings using `like`. Wildcards (`%`) are added automatically.
```go
users, err := db.Users().
    Filter("name", "like", "ana").
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE name LIKE '%ana%';
```

---

### In (multiple values)

Match against a list of values.
```go
users, err := db.Users().
    Filter("status", "in", []string{"active", "pending"}).
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
WHERE status IN ('active', 'pending');
```

---

## Relations

### Include (eager loading)

Load related entities alongside the main query.
Without `.Include()`, relations are not fetched.
```go
users, err := db.Users().
    Include("orders").
    Execute()
```

Generated SQL:
```sql
-- Main query
SELECT id, email, name, age, created_at
FROM users;

-- Eager load (separate query, matched by foreign key)
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);  -- IDs from main query
```

> ChameleonDB uses separate queries for eager loading
> (not JOINs) to avoid row duplication and keep results clean.

---

### Nested include

Load relations multiple levels deep.
```go
users, err := db.Users().
    Include("orders").
    Include("orders.items").
    Execute()
```

Generated SQL:
```sql
-- 1. Main query
SELECT id, email, name, age, created_at
FROM users;

-- 2. Load orders
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);

-- 3. Load order items
SELECT id, quantity, price, order_id
FROM order_items
WHERE order_id IN (...);  -- IDs from orders query
```

---

### Filter on related entity

Filter the main entity based on a condition on a related entity.
This is different from filtering the included results.
```go
users, err := db.Users().
    Filter("orders.total", "gt", 100).
    Execute()
```

Generated SQL:
```sql
SELECT DISTINCT users.id, users.email, users.name, users.age, users.created_at
FROM users
INNER JOIN orders ON orders.user_id = users.id
WHERE orders.total > 100;
```

> When filtering on a relation, ChameleonDB uses a JOIN
> automatically. `DISTINCT` is added to avoid duplicates
> when a user has multiple matching orders.

---

### Filter on relation + include

You can filter on a relation and also include it.
The filter affects which users are returned;
the include loads all their orders (not just matching ones).
```go
users, err := db.Users().
    Filter("orders.total", "gt", 100).
    Include("orders").
    Execute()
```

Generated SQL:
```sql
-- 1. Main query (filtered via JOIN)
SELECT DISTINCT users.id, users.email, users.name, users.age, users.created_at
FROM users
INNER JOIN orders ON orders.user_id = users.id
WHERE orders.total > 100;

-- 2. Eager load ALL orders for matched users
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);
```

---

## Advanced Queries

### Order by

Sort results by one or more fields.
```go
users, err := db.Users().
    OrderBy("created_at", "desc").
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
ORDER BY created_at DESC;
```

Multiple order clauses:
```go
users, err := db.Users().
    OrderBy("name", "asc").
    OrderBy("created_at", "desc").
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
ORDER BY name ASC, created_at DESC;
```

---

### Limit and offset

Paginate results.
```go
users, err := db.Users().
    OrderBy("created_at", "desc").
    Limit(10).
    Offset(20).
    Execute()
```

Generated SQL:
```sql
SELECT id, email, name, age, created_at
FROM users
ORDER BY created_at DESC
LIMIT 10 OFFSET 20;
```

> **Best practice:** Always use `OrderBy` with `Limit`/`Offset`
> to ensure deterministic pagination.

---

### Combining everything

A realistic query combining multiple features:
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

Generated SQL:
```sql
-- 1. Main query
SELECT DISTINCT users.id, users.email, users.name, users.age, users.created_at
FROM users
INNER JOIN orders ON orders.user_id = users.id
WHERE users.age >= 18
  AND orders.total > 50
ORDER BY users.created_at DESC
LIMIT 10;

-- 2. Eager load orders
SELECT id, total, status, created_at, user_id
FROM orders
WHERE user_id IN (...);

-- 3. Eager load order items
SELECT id, quantity, price, order_id
FROM order_items
WHERE order_id IN (...);
```

---

## Query Validation

All queries are validated before execution.
ChameleonDB will catch these errors at compile time:

| Error | Example |
|-------|---------|
| Unknown entity | `db.Products()` when `Product` is not in schema |
| Unknown field | `.Filter("phone", ...)` when `phone` is not a field |
| Invalid relation path | `.Include("orders.address")` when `address` doesn't exist |
| Type mismatch | `.Filter("age", "eq", "not a number")` |
| Missing order with pagination | `.Limit(10)` without `.OrderBy()` warns |

---

## Limitations (v0.1)

These features are **not supported** in the current version:

- Aggregations (`count`, `sum`, `avg`)
- Group by
- Subqueries
- Raw SQL escape hatch
- Mutations (insert, update, delete)
- Transactions

These are planned for future versions.