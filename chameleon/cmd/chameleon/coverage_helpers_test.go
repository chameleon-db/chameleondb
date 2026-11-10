package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/fatih/color"
)

func TestLoadConnectorConfigErrorAndDefaults(t *testing.T) {
	t.Run("invalid environment URL", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "not-a-connection-string")
		if _, err := LoadConnectorConfig(); err == nil {
			t.Fatal("expected invalid DATABASE_URL to return an error")
		}
	})

	t.Run("defaults without environment or project config", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(oldDir) })

		got, err := LoadConnectorConfig()
		if err != nil {
			t.Fatalf("LoadConnectorConfig() error = %v", err)
		}
		want := engine.DefaultConfig()
		if got != want {
			t.Fatalf("LoadConnectorConfig() = %+v, want %+v", got, want)
		}
	})
}

func TestConfigPasswordHelpersUseEnvironment(t *testing.T) {
	t.Setenv(modePasswordEnvVar, "test-password")

	for name, fn := range map[string]func() (string, error){
		"mode password":       readModePassword,
		"setup mode password": readModePasswordForSetup,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := fn()
			if err != nil {
				t.Fatalf("password helper returned error: %v", err)
			}
			if got != "test-password" {
				t.Fatalf("password helper returned %q", got)
			}
		})
	}

	if got := canonicalParanoidMode(" ADMIN "); got != "privileged" {
		t.Fatalf("canonicalParanoidMode() = %q, want privileged", got)
	}
}

func TestIntrospectExistingSchemaBranches(t *testing.T) {
	defaultSchema := `entity User {
  id: uuid primary,
  posts: [Post] via user_id,
}
entity Post {
  id: uuid primary,
}
entity Comment {
  id: uuid primary,
}
entity UserProfile {
  user: User via user_id,
}`
	if !isDefaultSchema(defaultSchema) {
		t.Fatal("expected default schema to be recognized")
	}
	if isModifiedSchema(defaultSchema) {
		t.Fatal("default schema must not be considered modified")
	}

	t.Run("empty existing schema can be overwritten", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "schema.cham")
		if err := os.WriteFile(path, []byte("// generated\n"), 0644); err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdin = r
		_, _ = w.WriteString("yes\n")
		_ = w.Close()
		t.Cleanup(func() { os.Stdin = oldStdin })

		got, err := checkExistingSchemaAndGetOutput(path)
		if err != nil || got != path {
			t.Fatalf("checkExistingSchemaAndGetOutput() = %q, %v", got, err)
		}
	})
}

func TestJournalOutputHelpers(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	entries := []*journal.Entry{
		{
			Timestamp: now,
			Action:    "migrate",
			Status:    "success",
			Details:   map[string]interface{}{"version": "v1"},
			Duration:  12,
		},
		{
			Timestamp: now,
			Action:    "query",
			Status:    "failed",
			Error:     "something went wrong",
		},
	}

	out := captureStdout(t, func() {
		printEntriesTable(entries)
		printMigrationsTable(entries)
		printEntriesJSON(entries)
	})
	if !strings.Contains(out, "duration=12ms") ||
		!strings.Contains(out, "error=something went wrong") ||
		!strings.Contains(out, `"duration_ms": 12`) {
		t.Fatalf("journal output omitted expected fields: %s", out)
	}
}

func TestPlaygroundMutationHelpersAndErrors(t *testing.T) {
	eng := engine.NewEngineWithoutSchema()
	if _, err := eng.LoadSchemaFromString(`
entity User {
  id: uuid primary,
  displayName: string,
}`); err != nil {
		t.Fatal(err)
	}
	entity := findSchemaEntity(eng.GetSchema(), "user")
	if entity == nil {
		t.Fatal("expected case-insensitive entity lookup")
	}

	if _, _, err := buildMutationSQL(eng.GetSchema(), nil); err == nil {
		t.Fatal("expected nil mutation error")
	}
	if _, _, err := buildMutationSQL(eng.GetSchema(), &playgroundSQLMutation{Type: "insert"}); err == nil {
		t.Fatal("expected missing entity error")
	}
	if _, _, err := buildMutationSQL(eng.GetSchema(), &playgroundSQLMutation{Type: "unknown", Entity: "User"}); err == nil {
		t.Fatal("expected unsupported mutation error")
	}
	if _, _, err := buildInsertSQL(entity, &playgroundSQLMutation{Type: "insert", Entity: "User"}); err == nil {
		t.Fatal("expected empty insert error")
	}
	if _, _, err := buildUpdateSQL(entity, &playgroundSQLMutation{Type: "update", Entity: "User"}); err == nil {
		t.Fatal("expected empty update error")
	}
	if _, err := buildWhereClauses(entity, []playgroundSQLMutationFilter{{Field: "missing", Op: "eq"}}, new(playgroundParams)); err == nil {
		t.Fatal("expected unknown filter field error")
	}
	if _, err := buildWhereClauses(entity, []playgroundSQLMutationFilter{{Field: "id", Op: "unsupported"}}, new(playgroundParams)); err == nil {
		t.Fatal("expected unsupported operator error")
	}

	for _, op := range []string{"eq", "neq", "ne", "gt", "gte", "lt", "lte", "like"} {
		if _, err := mutationSQLOperator(op); err != nil {
			t.Fatalf("mutationSQLOperator(%q) error = %v", op, err)
		}
	}
	if got := normalizeFilterOp(" lte "); got != "Lte" {
		t.Fatalf("normalizeFilterOp() = %q", got)
	}
	if got := normalizeFilterOp("unknown"); got != "unknown" {
		t.Fatalf("normalizeFilterOp() changed unknown operator to %q", got)
	}

	binary := engine.FilterExpr{Binary: &engine.BinaryExpr{
		Op:    "or",
		Left:  engine.FilterExpr{Condition: &engine.FilterCondition{Op: "gt"}},
		Right: engine.FilterExpr{Condition: &engine.FilterCondition{Op: "neq"}},
	}}
	normalizeFilterExpr(&binary)
	if binary.Binary.Op != "Or" || binary.Binary.Left.Condition.Op != "Gt" || binary.Binary.Right.Condition.Op != "Neq" {
		t.Fatalf("normalizeFilterExpr() did not normalize binary expression: %+v", binary)
	}
	normalizeFilterExpr(nil)

	var params playgroundParams
	if got := params.add(float64(4)); got != "$1" || params[0] != engine.NewSQLParam(int64(4)) {
		t.Fatalf("playgroundParams.add() = %q, %#v", got, params)
	}
	if got := toSnakeCase("DisplayName"); got != "display_name" || pluralize("users") != "users" {
		t.Fatal("string helpers returned unexpected values")
	}
}

func TestReadHiddenPasswordWithoutTerminal(t *testing.T) {
	t.Setenv(modePasswordEnvVar, "")
	oldNoColor := color.NoColor
	t.Cleanup(func() { color.NoColor = oldNoColor })
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	_ = w.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })

	color.NoColor = true
	if _, err := readHiddenPassword("password: "); err == nil {
		t.Fatal("expected non-terminal password input to fail")
	}
}
