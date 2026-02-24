# Mutations (CRUD) Usage in ChameleonDB

This guide shows how to use `Insert`, `Update`, and `Delete` safely in Go applications, with copy-paste-ready examples.

---

## 1) Minimal setup (quick start)

This example shows the minimum setup: create engine, connect to PostgreSQL, and run one mutation.

```go
package main

import (
	"context"
	"log"
	"os"

	_ "github.com/chameleon-db/chameleondb/chameleon/pkg/engine/mutation"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()

	eng, err := engine.NewEngine() // loads schema from vault
	if err != nil {
		log.Fatal(err)
	}
	defer eng.Close()

	cfg, err := engine.ParseConnectionString(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	if err := eng.Connect(ctx, cfg); err != nil {
		log.Fatal(err)
	}

	_, err = eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "ana@mail.com").
		Set("name", "Ana").
		Debug().
		Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
```

Quick notes:
- `Filter("id", "eq", uuidString)` is the recommended pattern for update/delete by ID.
- `Debug()` prints SQL and values for diagnostics.
- If `Connect()` is missing, mutations fail with a connection error.

---

## 2) Real example (repository.go style)

Example focused on a repository layer, without services/controllers.

```go
package repository

import (
	"context"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/google/uuid"
)

type User struct {
	ID    string
	Email string
	Name  string
}

type UserRepository struct {
	eng *engine.Engine
}

func NewUserRepository(eng *engine.Engine) *UserRepository {
	return &UserRepository{eng: eng}
}

func (r *UserRepository) Create(ctx context.Context, email, name string) (*engine.InsertResult, error) {
	return r.eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", email).
		Set("name", name).
		Execute(ctx)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (map[string]interface{}, error) {
	result, err := r.eng.Query("User").
		Filter("id", "eq", id).
		Limit(1).
		Execute(ctx)
	if err != nil {
		return nil, err
	}
	if len(result.Rows) == 0 {
		return nil, nil
	}
	return result.Rows[0], nil
}

func (r *UserRepository) UpdateName(ctx context.Context, id, name string) (*engine.UpdateResult, error) {
	return r.eng.Update("User").
		Filter("id", "eq", id).
		Set("name", name).
		Execute(ctx)
}

func (r *UserRepository) DeleteByID(ctx context.Context, id string) (int, error) {
	res, err := r.eng.Delete("User").
		Filter("id", "eq", id).
		Execute(ctx)
	if err != nil {
		return 0, err
	}
	return res.Affected, nil
}
```

---

## 3) Typed error pattern (recommended)

To return useful HTTP errors in your E2E example:

```go
func mapMutationError(err error) (status int, msg string) {
	var uniqueErr *engine.UniqueConstraintError
	if errors.As(err, &uniqueErr) {
		return 409, uniqueErr.Error()
	}

	var fkErr *engine.ForeignKeyError
	if errors.As(err, &fkErr) {
		return 400, fkErr.Error()
	}

	var notNullErr *engine.NotNullError
	if errors.As(err, &notNullErr) {
		return 400, notNullErr.Error()
	}

	return 500, "internal server error"
}
```

> Important: this helper requires `import "errors"`.

---

## 4) Anti-500 checklist for mutations

Before blaming the engine, verify:

- Engine created with `engine.NewEngine()` without errors.
- Mutation package registration is included: `_ "github.com/chameleon-db/chameleondb/chameleon/pkg/engine/mutation"`.
- `eng.Connect(ctx, cfg)` is called and `eng.Ping(ctx)` is OK at startup.
- Valid UUIDs are used in ID filters (`id`, `user_id`, etc.).
- `Debug()` is enabled in `Update/Delete` while diagnosing to inspect SQL and args.

With this pattern, your E2E example should handle full CRUD with traceable and controlled errors.
