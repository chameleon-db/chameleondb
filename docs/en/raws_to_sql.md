# Raw SQL Escape Hatch

For complex queries beyond ChameleonDB's query builder, use raw SQL directly.

## Access the pgx connection pool

\`\`\`go
// Get direct access to pgx pool
pool := engine.Connector().Pool()

// Execute any SQL
rows, err := pool.Query(ctx, `
    SELECT u.name, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON p.author_id = u.id
    GROUP BY u.id, u.name
    HAVING COUNT(p.id) > 5
    ORDER BY post_count DESC
`)
\`\`\`

## When to use Raw SQL

**Use ChameleonDB for:**
- ✅ CRUD operations
- ✅ Simple filters and pagination
- ✅ Relation loading (`.Include`)
- ✅ Partial field selection (`.Select`)

**Use Raw SQL for:**
- ✅ Complex aggregations (GROUP BY, HAVING)
- ✅ Subqueries and CTEs
- ✅ Database-specific features (full-text search, JSON operators)
- ✅ Performance-critical queries with custom indexes

## Examples

### Complex Analytics
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

### Full-Text Search
```go
rows, err := pool.Query(ctx, `
    SELECT * FROM posts
    WHERE to_tsvector('english', content) 
        @@ plainto_tsquery('english', $1)
    ORDER BY ts_rank(to_tsvector('english', content), plainto_tsquery('english', $1)) DESC
    LIMIT 20
`, searchTerm)
```