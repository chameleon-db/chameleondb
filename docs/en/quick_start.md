# Quick Start Guide

Get started with ChameleonDB in 5 minutes.

---

## Prerequisites

- Go 1.21+
- PostgreSQL 14+
- ChameleonDB CLI installed

**Install ChameleonDB:**
```bash
curl -sSL https://chameleondb.dev/install | sh
```

Or build from source:
```bash
git clone https://github.com/chameleon-db/chameleondb.git
cd chameleondb/chameleon
make install
```

**Verify installation:**
```bash
chameleon --version
# Output: chameleon v1.0-alpha
```

---

## Step 1: Initialize Project

```bash
mkdir my-app
cd my-app
chameleon init
```

**What happens:**
- Creates `.chameleon/` directory
- Initializes Schema Vault
- Creates default `schema.cham`
- Sets mode to `readonly`

**Output:**
```
✅ Created .chameleon/ directory
✅ Schema Vault initialized
✅ Created schema.cham
ℹ️  Paranoid Mode: readonly
💡 Tip: Set mode password with 'chameleon config auth set-password'
```

---

## Step 2: Define Your Schema

Edit `schema.cham`:

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

**Validate:**
```bash
chameleon validate
```

**Output:**
```
✅ Schema validated successfully
   Entities: 2 (User, Post)
   Relations: 2 (users.posts, posts.author)
```

---

## Step 3: Run Migration

**Set DATABASE_URL:**
```bash
export DATABASE_URL="postgresql://user:password@localhost:5432/mydb"
```

**Run migration:**
```bash
chameleon migrate --apply
```

**Output:**
```
📦 Initializing Schema Vault...
   ✓ Created .chameleon/vault/
   ✓ Registered schema as v001
   ✓ Hash: 3f2a8b9c...

📋 Migration Preview:
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

✅ Migration applied successfully
✅ Schema v001 locked in vault
```

---

## Step 4: Use in Your Application

**Initialize Go module:**
```bash
go mod init my-app
go get github.com/chameleon-db/chameleondb/chameleon
```

**Create `main.go`:**
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
    
    // Connect (loads schema from vault automatically)
    eng, err := engine.NewEngine()
    if err != nil {
        log.Fatal(err)
    }
    defer eng.Close()
    
    // Insert user
    result, err := eng.Insert("User").
        Set("id", uuid.New().String()).
        Set("email", "ana@mail.com").
        Set("name", "Ana Garcia").
        Execute(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User created: %v\n", result.ID)
    
    // Query users
    users, err := eng.Query("User").
        Filter("email", "eq", "ana@mail.com").
        Execute(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    for _, user := range users.Rows {
        fmt.Printf("User: %s <%s>\n", user["name"], user["email"])
    }
}
```

**Run:**
```bash
go run main.go
```

**Output:**
```
User created: 550e8400-e29b-41d4-a716-446655440000
User: Ana Garcia <ana@mail.com>
```

---

## Step 5: Query with Relations

```go
// Query users with their posts
result, err := eng.Query("User").
    Select("id", "name", "email").
    Include("posts").
    Execute(ctx)

if err != nil {
    log.Fatal(err)
}

for _, user := range result.Rows {
    fmt.Printf("User: %s\n", user["name"])
    
    if posts, ok := result.Relations["posts"]; ok {
        fmt.Printf("  Posts: %d\n", len(posts))
        for _, post := range posts {
            fmt.Printf("  - %s\n", post["title"])
        }
    }
}
```

---

## Step 6: Debug Mode

See generated SQL:

```go
result, err := eng.Query("User").
    Filter("email", "like", "ana").
    Debug().  // ← Shows SQL
    Execute(ctx)
```

**Output:**
```
[SQL] Query User
SELECT * FROM users WHERE email LIKE '%ana%'

[TRACE] Query on User: 2.3ms, 1 rows
```

---

## Next Steps

### Explore Features

**Mutations:**
```bash
# See examples
cat examples/mutations/
```

**Schema Vault:**
```bash
# View version history
chameleon journal schema

# Verify integrity
chameleon verify

# Check status
chameleon status
```

**Introspection:**
```bash
# Generate schema from existing DB
chameleon introspect $DATABASE_URL
```

### Learn More

- [Architecture](architecture.md) - System design
- [Query Reference](query-reference.md) - Complete API
- [Security Model](SECURITY.md) - Vault & modes
- [Introspection](introspection.md) - DB → Schema

---

## Common Issues

### "vault not initialized"

```bash
# Solution: Run init
chameleon init
```

### "readonly mode: blocked"

```bash
# Solution: Upgrade mode
chameleon config auth set-password
chameleon config set mode=standard
```

### "integrity violation"

```bash
# Check what changed
chameleon verify

# View audit log
cat .chameleon/vault/integrity.log
```

### "DATABASE_URL not set"

```bash
# Set environment variable
export DATABASE_URL="postgresql://user:pass@host:5432/db"
```

---

## Example Projects

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

## Getting Help

- **Documentation:** https://chameleondb.dev/docs
- **GitHub Issues:** https://github.com/chameleon-db/chameleondb/issues
- **Discord:** https://chameleondb.dev/discord
- **Show HN:** https://news.ycombinator.com/...

---

## What's Next?

You now know:
- ✅ How to initialize projects
- ✅ How to define schemas
- ✅ How to run migrations
- ✅ How to query data
- ✅ How to use Debug mode

**Continue learning:**
- [Query Reference](query-reference.md) - Advanced queries
- [Security Model](SECURITY.md) - Production deployment
- [Examples](../examples/) - Real applications