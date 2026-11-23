package cliout

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTableIsConsistent(t *testing.T) {
	seenCode := map[Code]bool{}
	seenExit := map[int]bool{}
	for _, e := range Table {
		if e.Code == "" || e.Description == "" {
			t.Errorf("entry %+v is incomplete", e)
		}
		if e.Exit <= ExitOK {
			t.Errorf("%s: failures must exit non-zero, got %d", e.Code, e.Exit)
		}
		if seenCode[e.Code] {
			t.Errorf("duplicate code %s", e.Code)
		}
		if seenExit[e.Exit] {
			t.Errorf("exit code %d is used by two codes", e.Exit)
		}
		seenCode[e.Code], seenExit[e.Exit] = true, true
		if e.Code.ExitCode() != e.Exit {
			t.Errorf("%s.ExitCode() = %d, want %d", e.Code, e.Code.ExitCode(), e.Exit)
		}
	}
}

// The numbers are part of the contract: scripts compare against them.
func TestExitCodesAreStable(t *testing.T) {
	want := map[Code]int{
		CodeInternal:         1,
		CodeUsage:            2,
		CodeSchemaInvalid:    3,
		CodeModeBlocked:      4,
		CodeIntegrityFailed:  5,
		CodeConnectionFailed: 6,
		CodeNotInitialized:   7,
		CodeApplyFailed:      8,
	}
	if len(want) != len(Table) {
		t.Fatalf("table has %d entries, test expects %d", len(Table), len(want))
	}
	for code, exit := range want {
		if got := code.ExitCode(); got != exit {
			t.Errorf("%s exits %d, want %d", code, got, exit)
		}
	}
	if Code("something_new").ExitCode() != ExitInternal {
		t.Error("an unknown code must exit as an internal error")
	}
}

func TestCodeAndExitOf(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code Code
		exit int
	}{
		{"nil", nil, "", 0},
		{"plain error", errors.New("boom"), CodeInternal, 1},
		{"typed", New(CodeModeBlocked, "readonly mode: schema locked"), CodeModeBlocked, 4},
		{"wrapped typed", fmt.Errorf("context: %w", New(CodeIntegrityFailed, "x")), CodeIntegrityFailed, 5},
		{"cobra unknown flag", errors.New("unknown flag: --nope"), CodeUsage, 2},
		{"cobra args", errors.New("accepts 1 arg(s), received 3"), CodeUsage, 2},
		{"cobra unknown command", errors.New(`unknown command "x" for "chameleon"`), CodeUsage, 2},
	}
	for _, c := range cases {
		if got := CodeOf(c.err); got != c.code {
			t.Errorf("%s: code = %q, want %q", c.name, got, c.code)
		}
		if got := ExitCodeOf(c.err); got != c.exit {
			t.Errorf("%s: exit = %d, want %d", c.name, got, c.exit)
		}
	}
}

func TestWrapKeepsExistingClass(t *testing.T) {
	inner := New(CodeSchemaInvalid, "bad schema")
	if got := Wrap(CodeInternal, inner); got != inner {
		t.Error("wrapping a typed error must keep it")
	}
	if Wrap(CodeInternal, nil) != nil {
		t.Error("wrapping nil must return nil")
	}
	cause := errors.New("dial tcp: refused")
	e := Wrap(CodeConnectionFailed, cause)
	if !errors.Is(e, cause) || e.Error() != "dial tcp: refused" {
		t.Errorf("Wrap lost the cause: %v", e)
	}
}

func TestErrorfWrapsCause(t *testing.T) {
	cause := errors.New("root")
	e := Errorf(CodeApplyFailed, "failed to execute migration: %w", cause)
	if !errors.Is(e, cause) || e.Error() != "failed to execute migration: root" {
		t.Errorf("Errorf = %v", e)
	}
}

func TestDetailsOf(t *testing.T) {
	if DetailsOf(nil) != nil {
		t.Error("nil error has no details")
	}

	plain := DetailsOf(errors.New("boom"))
	if len(plain) != 1 || plain[0].Code != CodeInternal || plain[0].Message != "boom" {
		t.Errorf("plain = %+v", plain)
	}

	hinted := DetailsOf(New(CodeModeBlocked, "locked").WithHint("set a password"))
	if len(hinted) != 1 || hinted[0].Hint != "set a password" {
		t.Errorf("hinted = %+v", hinted)
	}

	multi := DetailsOf(New(CodeSchemaInvalid, "schema validation failed").WithDetails(
		Detail{Message: "a", File: "x.cham", Line: 3, Column: 7},
		Detail{Message: "b", Code: CodeSchemaInvalid},
	))
	if len(multi) != 2 || multi[0].Code != CodeSchemaInvalid || multi[0].Line != 3 || multi[1].Message != "b" {
		t.Errorf("multi = %+v", multi)
	}
}

func TestSilent(t *testing.T) {
	if IsSilent(errors.New("x")) || IsSilent(nil) {
		t.Error("plain errors are not silent")
	}
	if !IsSilent(fmt.Errorf("w: %w", New(CodeIntegrityFailed, "x").AsSilent())) {
		t.Error("silent flag must survive wrapping")
	}
}

func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("output is not one JSON document: %v\n%s", err, b)
	}
	return m
}

func TestSuccessEnvelopeShape(t *testing.T) {
	b, err := Marshal(Success("chameleon validate", map[string]any{"valid": true}))
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, b)
	if m["schema_version"] != float64(SchemaVersion) || m["command"] != "chameleon validate" || m["ok"] != true {
		t.Errorf("envelope = %v", m)
	}
	errs, ok := m["errors"].([]any)
	if !ok || len(errs) != 0 {
		t.Errorf("errors must be an empty array, got %#v", m["errors"])
	}
	if !bytes.HasSuffix(b, []byte("\n")) {
		t.Error("output must end with a newline")
	}
}

func TestEnvelopeNeverHasNullDataOrErrors(t *testing.T) {
	b, err := Marshal(Envelope{SchemaVersion: 1, Command: "x"})
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, b)
	if _, ok := m["data"].(map[string]any); !ok {
		t.Errorf("data must be an object, got %#v", m["data"])
	}
	if _, ok := m["errors"].([]any); !ok {
		t.Errorf("errors must be an array, got %#v", m["errors"])
	}
}

func TestFailureEnvelope(t *testing.T) {
	err := New(CodeModeBlocked, "readonly mode: schema locked").WithHint("chameleon config set mode=standard")
	b, mErr := Marshal(Failure("chameleon migrate", err, nil))
	if mErr != nil {
		t.Fatal(mErr)
	}
	m := decode(t, b)
	if m["ok"] != false {
		t.Errorf("ok = %v", m["ok"])
	}
	e := m["errors"].([]any)[0].(map[string]any)
	if e["code"] != "mode_blocked" || e["message"] != "readonly mode: schema locked" || e["hint"] != "chameleon config set mode=standard" {
		t.Errorf("error entry = %v", e)
	}
	if _, has := e["line"]; has {
		t.Error("empty optional fields must be omitted")
	}
}

func TestFailureKeepsPartialData(t *testing.T) {
	b, _ := Marshal(Failure("chameleon verify", New(CodeIntegrityFailed, "2 issues"), map[string]any{"valid": false}))
	m := decode(t, b)
	if m["data"].(map[string]any)["valid"] != false {
		t.Errorf("data = %v", m["data"])
	}
}

func TestMarshalDoesNotEscapeHTML(t *testing.T) {
	b, _ := Marshal(Success("c", map[string]any{"sql": "a < b AND c > d"}))
	if !strings.Contains(string(b), "a < b AND c > d") {
		t.Errorf("SQL was escaped: %s", b)
	}
}

func TestWriteWritesOneDocument(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, Success("c", nil)); err != nil {
		t.Fatal(err)
	}
	decode(t, buf.Bytes())
}

func TestRedact(t *testing.T) {
	cases := []struct{ in, want string }{
		{"postgres://app:s3cret@localhost:5432/db", "postgres://app:***@localhost:5432/db"},
		{"postgresql://app:p%40ss%2Fword@db.example.com/x?sslmode=require", "postgresql://app:***@db.example.com/x?sslmode=require"},
		{"failed: dial postgres://u:pw@h/db: refused", "failed: dial postgres://u:***@h/db: refused"},
		{"host=h user=u password=hunter2 dbname=d", "host=h user=u password=*** dbname=d"},
		{"host=h password='two words' dbname=d", "host=h password=*** dbname=d"},
		{"PGPASSWORD=abc psql", "PGPASSWORD=*** psql"},
		{"postgres://localhost/db", "postgres://localhost/db"},
		{"postgres://user@localhost/db", "postgres://user@localhost/db"},
		{"nothing to hide: 10:30 at user@example.com", "nothing to hide: 10:30 at user@example.com"},
	}
	for _, c := range cases {
		if got := Redact(c.in); got != c.want {
			t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEnvelopeNeverContainsPasswords(t *testing.T) {
	err := Wrap(CodeConnectionFailed, errors.New("cannot reach postgres://app:topsecret@db:5432/x"))
	b, mErr := Marshal(Failure("chameleon migrate", err, map[string]any{"url": "postgres://a:alsosecret@h/d"}))
	if mErr != nil {
		t.Fatal(mErr)
	}
	for _, secret := range []string{"topsecret", "alsosecret"} {
		if strings.Contains(string(b), secret) {
			t.Errorf("password %q leaked: %s", secret, b)
		}
	}
	decode(t, b) // still valid JSON after redaction
}
