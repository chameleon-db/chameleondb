package mutation

import (
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

func TestFormatDebugParamsMasksSensitiveColumns(t *testing.T) {
	out := formatDebugParams(
		[]string{"email", "password_hash", "api_token"},
		[]interface{}{"a@mail.com", "s3cret", "tok-123"},
	)

	if !strings.Contains(out, "$1 = a@mail.com") {
		t.Fatalf("regular value should be printed, got:\n%s", out)
	}
	for _, leaked := range []string{"s3cret", "tok-123"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("sensitive value %q leaked:\n%s", leaked, out)
		}
	}
	if !strings.Contains(out, "$2 = [REDACTED]") || !strings.Contains(out, "$3 = [REDACTED]") {
		t.Fatalf("sensitive values should be masked, got:\n%s", out)
	}
}

func TestFormatDebugParamsMasksEverythingWhenLabelsDoNotAlign(t *testing.T) {
	out := formatDebugParams([]string{"email"}, []interface{}{"a@mail.com", "b@mail.com"})

	if strings.Contains(out, "mail.com") {
		t.Fatalf("misaligned labels must not print values, got:\n%s", out)
	}
}

func TestFormatDebugParamsEmpty(t *testing.T) {
	if out := formatDebugParams(nil, nil); out != "" {
		t.Fatalf("no params should print nothing, got %q", out)
	}
}

func TestFilterLabels(t *testing.T) {
	labels := filterLabels(map[string]interface{}{"name:like": "x", "id": 1, "age:gte": 2})
	want := []string{"age", "id", "name"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("want %v, got %v", want, labels)
	}
}

func TestEffectiveDebugLevel(t *testing.T) {
	t.Setenv("CHAMELEON_DEBUG", "trace")
	if got := effectiveDebugLevel(nil); got != engine.DebugTrace {
		t.Fatalf("env should apply without override, got %v", got)
	}

	level := engine.DebugNone
	if got := effectiveDebugLevel(&level); got != engine.DebugNone {
		t.Fatalf("override should win over env, got %v", got)
	}
}
