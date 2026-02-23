# Raw SQL Escape Hatch

Para consultas complejas que van más allá del constructor de consultas de ChameleonDB, usá raw SQL directamente.

## Acceder al pool de conexiones pgx

```go
// Accedé directamente al pool de pgx
pool := engine.Connector().Pool()

// Ejecutá cualquier SQL
rows, err := pool.Query(ctx, `
    SELECT u.name, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON p.author_id = u.id
    GROUP BY u.id, u.name
    HAVING COUNT(p.id) > 5
    ORDER BY post_count DESC
`)
```

## Cuándo usar raw SQL

**Usá ChameleonDB para:**
- ✅ Operaciones CRUD
- ✅ Filtros simples y paginación
- ✅ Carga de relaciones (`.Include`)
- ✅ Selección parcial de campos (`.Select`)

**Usá raw SQL para:**
- ✅ Agregaciones complejas (GROUP BY, HAVING)
- ✅ Subconsultas y CTEs
- ✅ Funcionalidades específicas de la base de datos (búsqueda de texto completo, operadores JSON)
- ✅ Consultas críticas para el rendimiento con índices personalizados

## Ejemplos

### Análisis Complejo
```go
rows, err := pool.Query(ctx, `
    WITH monthly_sales AS (
        SELECT 
            DATE_TRUNC('month', created_at) as month,
            SUM(total) as sales
        FROM orders
        GROUP BY month
    )
    SELECT * FROM monthly_sales
    WHERE sales > 10000
`)
```

### Búsqueda de Texto Completo
```go
rows, err := pool.Query(ctx, `
    SELECT * FROM posts
    WHERE to_tsvector('english', content) 
        @@ plainto_tsquery('english', $1)
    ORDER BY ts_rank(to_tsvector('english', content), plainto_tsquery('english', $1)) DESC
    LIMIT 20
`, searchTerm)
```