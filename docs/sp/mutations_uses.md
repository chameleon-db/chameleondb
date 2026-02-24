# Uso de Mutations (CRUD) en ChameleonDB

Esta guía resume cómo usar `Insert`, `Update` y `Delete` de forma segura en aplicaciones Go, con ejemplos listos para usar.

---

## 1) Setup mínimo (arranque rápido)

Este ejemplo muestra el setup base: crear engine, conectar a PostgreSQL y ejecutar una mutación.

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

	eng, err := engine.NewEngine() // carga schema desde vault
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

Notas rápidas:
- `Filter("id", "eq", uuidString)` es el patrón recomendado para updates/deletes por ID.
- `Debug()` imprime SQL y valores para diagnóstico.
- Si no hay `Connect()`, las mutaciones fallan con error de conexión.

---

## 2) Ejemplo real (estilo repository.go)

Ejemplo orientado a una capa `repository` sin servicios/controladores.

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

## 3) Patrón para errores tipados (recomendado)

Para devolver errores HTTP útiles en tu ejemplo E2E:

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

> Importante: para ese helper hace falta `import "errors"`.

---

## 4) Checklist anti-500 en mutaciones

Antes de culpar al engine, validar:

- Engine creado con `engine.NewEngine()` sin error.
- Registro de mutaciones incluido: `_ "github.com/chameleon-db/chameleondb/chameleon/pkg/engine/mutation"`.
- `eng.Connect(ctx, cfg)` ejecutado y `eng.Ping(ctx)` OK al iniciar.
- UUIDs válidos en filtros por ID (`id`, `user_id`, etc.).
- Logs con `Debug()` en `Update/Delete` para ver SQL y argumentos.

Con este patrón, el ejemplo E2E debería cubrir CRUD completo con errores controlados y trazables.
