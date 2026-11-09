package initwizard

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newPrompter(input string) (*Prompter, *bytes.Buffer) {
	var out bytes.Buffer
	return NewPrompter(strings.NewReader(input), &out), &out
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func noEnv(string) string { return "" }

// ---- prompts

func TestChoose(t *testing.T) {
	opts := []string{"a", "b", "c"}
	cases := []struct {
		input string
		want  int
	}{
		{"\n", 0}, {"2\n", 1}, {"3\n", 2}, {"", 0}, {"x\n9\n0\n", 0}, {"x\n2\n", 1},
	}
	for _, c := range cases {
		p, _ := newPrompter(c.input)
		if got := p.Choose("q", opts, 0); got != c.want {
			t.Errorf("input %q: want %d, got %d", c.input, c.want, got)
		}
	}
}

func TestConfirm(t *testing.T) {
	cases := []struct {
		input string
		def   bool
		want  bool
	}{
		{"\n", false, false}, {"\n", true, true}, {"y\n", false, true}, {"YES\n", false, true},
		{"n\n", true, false}, {"", true, true}, {"maybe\ny\n", false, true}, {"a\nb\nc\n", true, true},
	}
	for _, c := range cases {
		p, _ := newPrompter(c.input)
		if got := p.Confirm("q", c.def); got != c.want {
			t.Errorf("input %q def %v: want %v, got %v", c.input, c.def, c.want, got)
		}
	}
}

// ---- dotenv

func TestParseDotenv(t *testing.T) {
	got := ParseDotenv("# comment\n\nA=1\nexport B=\"two words\"\nC='x'\nD=plain # trailing\nE=\nbad line\r\nF=a=b\n")
	want := map[string]string{"A": "1", "B": "two words", "C": "x", "D": "plain", "E": "", "F": "a=b"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: want %q, got %q", k, v, got[k])
		}
	}
	if _, ok := got["bad line"]; ok {
		t.Error("lines without = are ignored")
	}
}

// ---- detection

func TestValidatePostgresURL(t *testing.T) {
	for _, ok := range []string{"postgresql://u:p@localhost/db", "postgres://localhost:5433/db?sslmode=disable"} {
		if err := ValidatePostgresURL(ok); err != nil {
			t.Errorf("%s should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"mysql://u@h/db", "postgresql:///db", "not a url", "", "${DATABASE_URL}"} {
		if err := ValidatePostgresURL(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestDetect_OrderAndSources(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "DATABASE_URL=postgresql://app:secret@localhost:5440/appdb\n")
	write(t, dir, ".env.example", "DATABASE_URL=postgresql://app:change-me@localhost:5440/appdb\n")
	write(t, dir, "docker-compose.yml", `
services:
  web:
    image: nginx
    ports: ["80:80"]
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: shop
      POSTGRES_PASSWORD: shoppw
      POSTGRES_DB: shopdb
    ports:
      - "5433:5432"
`)

	env := func(k string) string {
		if k == "DATABASE_URL" {
			return "postgresql://fromenv:pw@db.internal:5432/x"
		}
		return ""
	}
	got := Detect(dir, env)
	if len(got) != 4 {
		t.Fatalf("want 4 candidates, got %d: %+v", len(got), got)
	}

	order := []string{"environment", ".env", ".env.example", `service "db" in docker-compose.yml`}
	for i, frag := range order {
		if !strings.Contains(got[i].Source, frag) {
			t.Errorf("candidate %d should come from %q, got %q", i, frag, got[i].Source)
		}
	}
	if got[3].URL != "postgresql://shop:shoppw@localhost:5433/shopdb?sslmode=disable" {
		t.Errorf("compose URL: %s", got[3].URL)
	}
	if strings.Contains(got[1].Redacted(), "secret") || !strings.Contains(got[1].Redacted(), "xxxxx") {
		t.Errorf("display form must hide the password: %s", got[1].Redacted())
	}
}

func TestDetect_DropsDuplicatesAndInvalid(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "DATABASE_URL=postgresql://a:b@localhost/x\n")
	write(t, dir, ".env.example", "DATABASE_URL=postgresql://a:b@localhost/x\n")
	write(t, dir, "compose.yml", "services: [this is not a map\n")
	got := Detect(dir, noEnv)
	if len(got) != 1 {
		t.Fatalf("want 1 candidate, got %+v", got)
	}

	write(t, dir, ".env", "DATABASE_URL=${DATABASE_URL}\n")
	write(t, dir, ".env.example", "DATABASE_URL=mysql://u@h/db\n")
	if got := Detect(dir, noEnv); len(got) != 0 {
		t.Errorf("placeholders and other engines are not candidates: %+v", got)
	}
}

func TestDetect_ComposeVariants(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "docker-compose.dev.yml", `
services:
  pg:
    image: postgis/postgis:16-3.4
    environment:
      - POSTGRES_USER=${PGUSER:-dev}
      - POSTGRES_PASSWORD=${PGPASS}
    ports:
      - "127.0.0.1:${PGPORT:-6543}:5432/tcp"
  longform:
    image: bitnami/postgresql
    ports:
      - target: 5432
        published: 7000
  internal:
    image: postgres:16
    ports: ["5432"]
  exporter:
    image: prom/node-exporter
    ports: ["9100:9100"]
`)
	env := func(k string) string {
		if k == "PGPASS" {
			return "s3cret"
		}
		return ""
	}
	got := Detect(dir, env)
	if len(got) != 2 {
		t.Fatalf("want 2 candidates (internal-only and non-postgres are skipped), got %+v", got)
	}
	if got[1].URL != "postgresql://dev:s3cret@localhost:6543/dev?sslmode=disable" {
		t.Errorf("variables and defaults should expand: %s", got[1].URL)
	}
	if !strings.Contains(got[0].URL, "localhost:7000") {
		t.Errorf("long-form ports: %s", got[0].URL)
	}
}

func TestDetect_WritesNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "DATABASE_URL=postgresql://a:b@localhost/x\n")
	before, _ := os.ReadDir(dir)
	Detect(dir, noEnv)
	after, _ := os.ReadDir(dir)
	if len(before) != len(after) {
		t.Error("detection must not write files")
	}
}

// ---- compose and .env.example

func TestWriteComposeFile(t *testing.T) {
	dir := t.TempDir()
	created, err := WriteComposeFile(dir)
	if err != nil || !created {
		t.Fatalf("first write: created=%v err=%v", created, err)
	}
	if got := Detect(dir, noEnv); len(got) != 1 || got[0].URL != DefaultComposeURL {
		t.Errorf("the generated file must be detected as DefaultComposeURL, got %+v", got)
	}

	write(t, dir, ComposeFileName, "# mine\n")
	created, err = WriteComposeFile(dir)
	if err != nil || created {
		t.Fatalf("an existing file must be left alone: created=%v err=%v", created, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ComposeFileName))
	if string(data) != "# mine\n" {
		t.Error("existing docker-compose.yml was overwritten")
	}

	other := t.TempDir()
	write(t, other, "compose.yml", "services: {}\n")
	if created, _ := WriteComposeFile(other); created {
		t.Error("a compose.yml also counts as an existing file")
	}
}

func TestEnvExample_NeverContainsThePassword(t *testing.T) {
	out := EnvExample("postgresql://app:s3cret@localhost:5432/db?sslmode=disable")
	if strings.Contains(out, "s3cret") {
		t.Fatalf("password leaked: %s", out)
	}
	if out != "DATABASE_URL=postgresql://app:change-me@localhost:5432/db?sslmode=disable\n" {
		t.Errorf("unexpected content: %s", out)
	}
	if out := EnvExample("postgresql://localhost/db"); out != "DATABASE_URL=postgresql://localhost/db\n" {
		t.Errorf("a URL without credentials is kept as is: %s", out)
	}

	dir := t.TempDir()
	if created, err := WriteEnvExample(dir, "postgresql://a:b@h/d"); err != nil || !created {
		t.Fatalf("create: %v %v", created, err)
	}
	if created, _ := WriteEnvExample(dir, "postgresql://a:other@h/d"); created {
		t.Error("an existing .env.example must not be overwritten")
	}
}

// ---- Resolve

func TestResolve_DefaultsWithoutTerminal(t *testing.T) {
	p, out := newPrompter("")
	res, err := Resolve(p, Flags{}, Env{Dir: t.TempDir(), Getenv: noEnv, Interactive: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "" || res.WriteCompose || res.RunCompose || !res.CreateExample || len(res.Notes) != 0 {
		t.Errorf("defaults should equal today's init: %+v", res)
	}
	if out.Len() != 0 {
		t.Errorf("no prompt may be printed without a terminal, got %q", out.String())
	}
}

func TestResolve_EnterOnEveryQuestionIsToday(t *testing.T) {
	p, _ := newPrompter("\n\n")
	res, err := Resolve(p, Flags{}, Env{Dir: t.TempDir(), Getenv: noEnv, Interactive: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "" || res.WriteCompose || !res.CreateExample {
		t.Errorf("Enter on everything must give today's result: %+v", res)
	}
}

func TestResolve_FlagsAreMutuallyExclusive(t *testing.T) {
	p, _ := newPrompter("")
	_, err := Resolve(p, Flags{DBURL: "postgresql://h/d", Compose: true}, Env{Getenv: noEnv})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestResolve_DBURLFlag(t *testing.T) {
	var pinged string
	env := Env{Dir: t.TempDir(), Getenv: noEnv, Ping: func(u string) error { pinged = u; return errors.New("connection refused to " + u) }}

	p, out := newPrompter("")
	res, err := Resolve(p, Flags{DBURL: "postgresql://a:pw@localhost/d", NoExample: true}, env)
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "postgresql://a:pw@localhost/d" || res.CreateExample {
		t.Errorf("unexpected result: %+v", res)
	}
	if pinged == "" || len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "Could not connect") {
		t.Errorf("a failed connection is reported, not fatal: %+v", res.Notes)
	}
	if out.Len() != 0 {
		t.Errorf("flags never prompt: %q", out.String())
	}

	if _, err := Resolve(p, Flags{DBURL: "mysql://h/d"}, env); err == nil {
		t.Error("a non-PostgreSQL URL must be rejected")
	}
}

func TestResolve_DetectFlagPicksTheFirstWithoutPrompting(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "DATABASE_URL=postgresql://a:pw@localhost:5440/x\n")

	p, out := newPrompter("")
	res, err := Resolve(p, Flags{DetectDB: true}, Env{Dir: dir, Getenv: noEnv, Ping: func(string) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "postgresql://a:pw@localhost:5440/x" {
		t.Errorf("unexpected URL %q", res.URL)
	}
	if strings.Contains(out.String(), "pw") {
		t.Errorf("the displayed form must hide the password: %q", out.String())
	}

	empty, _ := Resolve(p, Flags{DetectDB: true}, Env{Dir: t.TempDir(), Getenv: noEnv})
	if empty.URL != "" || len(empty.Notes) != 1 {
		t.Errorf("nothing found keeps ${DATABASE_URL} and says so: %+v", empty)
	}
}

func TestResolve_MenuDetectWithSeveralAndFailingConnection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".env", "DATABASE_URL=postgresql://a:pw@localhost:5440/x\n")
	write(t, dir, ".env.example", "DATABASE_URL=postgresql://a:pw@localhost:5441/y\n")

	failing := func(string) error { return errors.New("dial tcp: connection refused") }

	// Menu 2 (detect), pick candidate 2, refuse to use it, then answer the example question.
	p, out := newPrompter("2\n2\nn\nn\n")
	res, err := Resolve(p, Flags{}, Env{Dir: dir, Getenv: noEnv, Interactive: true, Ping: failing})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "" || res.CreateExample {
		t.Errorf("a refused connection falls back to ${DATABASE_URL}: %+v", res)
	}
	if !strings.Contains(out.String(), "connection refused") || strings.Contains(out.String(), "pw@") {
		t.Errorf("the reason is shown and the password is not: %q", out.String())
	}

	// Same, but accept the failing connection.
	p, _ = newPrompter("2\n1\ny\n\n")
	res, _ = Resolve(p, Flags{}, Env{Dir: dir, Getenv: noEnv, Interactive: true, Ping: failing})
	if res.URL != "postgresql://a:pw@localhost:5440/x" || !res.CreateExample {
		t.Errorf("accepting keeps the URL: %+v", res)
	}
}

func TestResolve_MenuCompose(t *testing.T) {
	p, _ := newPrompter("3\ny\n\n")
	res, _ := Resolve(p, Flags{}, Env{Dir: t.TempDir(), Getenv: noEnv, Interactive: true})
	if !res.WriteCompose || !res.RunCompose || res.URL != DefaultComposeURL {
		t.Errorf("unexpected: %+v", res)
	}

	p, _ = newPrompter("3\n\n\n")
	res, _ = Resolve(p, Flags{}, Env{Dir: t.TempDir(), Getenv: noEnv, Interactive: true})
	if res.RunCompose {
		t.Error("docker compose is never run without a yes")
	}

	p, _ = newPrompter("")
	res, _ = Resolve(p, Flags{Compose: true}, Env{Dir: t.TempDir(), Getenv: noEnv})
	if !res.WriteCompose || res.RunCompose {
		t.Errorf("the flag writes the file and never runs docker: %+v", res)
	}
}

func TestResolve_MenuEnterAConnectionString(t *testing.T) {
	p, _ := newPrompter("4\npostgresql://u:p@localhost/d\n\n")
	res, _ := Resolve(p, Flags{}, Env{Dir: t.TempDir(), Getenv: noEnv, Interactive: true})
	if res.URL != "postgresql://u:p@localhost/d" {
		t.Errorf("unexpected: %+v", res)
	}

	p, _ = newPrompter("4\nmysql://u@h/d\n\n")
	res, _ = Resolve(p, Flags{}, Env{Dir: t.TempDir(), Getenv: noEnv, Interactive: true})
	if res.URL != "" || len(res.Notes) != 1 {
		t.Errorf("an invalid string is not used and is explained: %+v", res)
	}
}

// ---- mode offer

type modeRecorder struct {
	passwords []string
	reads     int
	standard  bool
}

func (m *modeRecorder) offer(input string, answers ...string) (ModeOffer, *bytes.Buffer) {
	p, out := newPrompter(input)
	return ModeOffer{
		Prompter:  p,
		MinLength: 8,
		ReadPassword: func(string) (string, error) {
			a := answers[m.reads]
			m.reads++
			return a, nil
		},
		SetPassword: func(pw string) error { m.passwords = append(m.passwords, pw); return nil },
		SetStandard: func() error { m.standard = true; return nil },
	}, out
}

func TestOfferStandardMode_EnterKeepsReadonly(t *testing.T) {
	m := &modeRecorder{}
	o, _ := m.offer("\n")
	ok, err := OfferStandardMode(o)
	if err != nil || ok || m.reads != 0 || m.standard {
		t.Fatalf("Enter must change nothing: ok=%v err=%v %+v", ok, err, m)
	}
}

func TestOfferStandardMode_Accept(t *testing.T) {
	m := &modeRecorder{}
	o, _ := m.offer("y\n", "supersecure1", "supersecure1")
	ok, err := OfferStandardMode(o)
	if err != nil || !ok || !m.standard || len(m.passwords) != 1 || m.passwords[0] != "supersecure1" {
		t.Fatalf("unexpected: ok=%v err=%v %+v", ok, err, m)
	}
}

func TestOfferStandardMode_MismatchThenShortThenOK(t *testing.T) {
	m := &modeRecorder{}
	o, out := m.offer("y\n", "supersecure1", "different-one", "short", "short", "supersecure2", "supersecure2")
	ok, err := OfferStandardMode(o)
	if err != nil || !ok || m.passwords[0] != "supersecure2" {
		t.Fatalf("unexpected: ok=%v err=%v %+v", ok, err, m)
	}
	if !strings.Contains(out.String(), "do not match") || !strings.Contains(out.String(), "too short") {
		t.Errorf("each failure is explained: %q", out.String())
	}
}

func TestOfferStandardMode_ThreeFailuresFallBackToReadonly(t *testing.T) {
	m := &modeRecorder{}
	o, out := m.offer("y\n", "aaaaaaaa", "bbbbbbbb", "aaaaaaaa", "bbbbbbbb", "aaaaaaaa", "bbbbbbbb")
	ok, err := OfferStandardMode(o)
	if err != nil || ok || m.standard || len(m.passwords) != 0 {
		t.Fatalf("must fall back without error: ok=%v err=%v %+v", ok, err, m)
	}
	if !strings.Contains(out.String(), "Staying in readonly") {
		t.Errorf("the fallback is stated: %q", out.String())
	}
}

func TestOfferStandardMode_ErrorsAreReported(t *testing.T) {
	m := &modeRecorder{}
	o, _ := m.offer("y\n", "supersecure1", "supersecure1")
	o.SetStandard = func() error { return errors.New("boom") }
	ok, err := OfferStandardMode(o)
	if ok || err == nil || !strings.Contains(err.Error(), "password was set") {
		t.Fatalf("a failed mode change after the password must say so: ok=%v err=%v", ok, err)
	}
}
