package engine

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDebugContextFromEnv(t *testing.T) {
	testCases := []struct {
		envVal string
		want   DebugLevel
	}{
		{"", DebugNone},
		{"0", DebugNone},
		{"1", DebugSQL},
		{"trace", DebugTrace},
		{"explain", DebugExplain},
		{"sql", DebugSQL},
		{"TRUE", DebugSQL},
		{" Trace ", DebugTrace},
		{"nonsense", DebugNone},
	}

	for _, testCase := range testCases {
		t.Run("CHAMELEON_DEBUG="+testCase.envVal, func(t *testing.T) {
			old := os.Getenv("CHAMELEON_DEBUG")
			defer func() {
				if old == "" {
					_ = os.Unsetenv("CHAMELEON_DEBUG")
				} else {
					_ = os.Setenv("CHAMELEON_DEBUG", old)
				}
			}()

			if testCase.envVal == "" {
				_ = os.Unsetenv("CHAMELEON_DEBUG")
			} else {
				_ = os.Setenv("CHAMELEON_DEBUG", testCase.envVal)
			}

			dc := DebugContextFromEnv()
			if dc.Level != testCase.want {
				t.Fatalf("env=%q: want level %d, got %d", testCase.envVal, testCase.want, dc.Level)
			}
		})
	}
}

func TestSchemaParseJSONRoundtrip(t *testing.T) {
	src := &Schema{
		Entities: []*Entity{
			{
				Name: "User",
				Fields: map[string]*Field{
					"id": {Name: "id", Type: FieldTypeUUID, PrimaryKey: true},
				},
				Relations: map[string]*Relation{},
			},
		},
	}

	jsonStr, err := src.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	if !strings.Contains(jsonStr, "User") {
		t.Fatalf("JSON missing User: %s", jsonStr)
	}

	parsed, err := ParseSchemaJSON(jsonStr)
	if err != nil {
		t.Fatalf("ParseSchemaJSON() error = %v", err)
	}
	if len(parsed.Entities) != 1 || parsed.Entities[0].Name != "User" {
		t.Fatalf("unexpected parsed schema: %+v", parsed)
	}
}

func TestParseSchemaJSONInvalid(t *testing.T) {
	if _, err := ParseSchemaJSON("not json"); err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

func TestFieldTypeString(t *testing.T) {
	if s := FieldTypeUUID.String(); s != "UUID" {
		t.Fatalf("expected UUID, got %q", s)
	}
	complex := FieldType{Kind: "Vector", Param: 1536}
	if s := complex.String(); s != "Vector(1536)" {
		t.Fatalf("expected Vector(1536), got %q", s)
	}
}

func TestInvalidMutationsReturnError(t *testing.T) {
	sentinelErr := fmt.Errorf("schema not loaded")
	ctx := context.Background()

	insert := newInvalidInsertMutation(sentinelErr)
	insert = insert.Set("field", "value")
	insert = insert.Debug()
	if _, err := insert.Execute(ctx); err != sentinelErr {
		t.Fatalf("expected sentinel error from invalid insert, got %v", err)
	}

	update := newInvalidUpdateMutation(sentinelErr)
	update = update.Set("field", "value")
	update = update.Filter("id", "=", "1")
	update = update.Debug()
	if _, err := update.Execute(ctx); err != sentinelErr {
		t.Fatalf("expected sentinel error from invalid update, got %v", err)
	}

	del := newInvalidDeleteMutation(sentinelErr)
	del = del.Filter("id", "=", "1")
	del = del.Debug()
	if _, err := del.Execute(ctx); err != sentinelErr {
		t.Fatalf("expected sentinel error from invalid delete, got %v", err)
	}
}
