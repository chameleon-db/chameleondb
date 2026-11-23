package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests run the real CLI: the test binary re-executes itself as the
// chameleon command (see TestMain), so exit codes and the stdout/stderr split
// are the ones a script or an agent sees. None of them needs a database.

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata/json")

func TestMain(m *testing.M) {
	if os.Getenv("CHAMELEON_E2E_CHILD") == "1" {
		Execute() // exits
	}
	os.Exit(m.Run())
}

type runResult struct {
	stdout, stderr string
	exit           int
}

// runCLI runs `chameleon args...` in dir.
func runCLI(t *testing.T, dir string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"CHAMELEON_E2E_CHILD=1", "CI=1", "NO_COLOR=1",
		"CHAMELEON_MODE_PASSWORD=", "PGPASSWORD=")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		code := 0
		if err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("running %v: %v", args, err)
			}
			code = ee.ExitCode()
		}
		return runResult{out.String(), errb.String(), code}
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("timeout running %v", args)
		return runResult{}
	}
}

type envelope struct {
	SchemaVersion int            `json:"schema_version"`
	Command       string         `json:"command"`
	OK            bool           `json:"ok"`
	Data          map[string]any `json:"data"`
	Errors        []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
		File    string `json:"file"`
		Line    int    `json:"line"`
		Column  int    `json:"column"`
		Snippet string `json:"snippet"`
	} `json:"errors"`
}

// parseEnvelope requires stdout to be exactly one JSON document.
func parseEnvelope(t *testing.T, r runResult) envelope {
	t.Helper()
	var env envelope
	dec := json.NewDecoder(strings.NewReader(r.stdout))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		t.Fatalf("stdout is not a JSON envelope: %v\nstdout:\n%s\nstderr:\n%s", err, r.stdout, r.stderr)
	}
	if dec.More() {
		t.Fatalf("stdout has more than one document:\n%s", r.stdout)
	}
	if env.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", env.SchemaVersion)
	}
	if env.Data == nil {
		t.Fatalf("data must be an object, never null:\n%s", r.stdout)
	}
	if env.OK != (len(env.Errors) == 0) {
		t.Fatalf("ok=%v with %d errors", env.OK, len(env.Errors))
	}
	return env
}

func wantExit(t *testing.T, r runResult, code int) {
	t.Helper()
	if r.exit != code {
		t.Fatalf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", r.exit, code, r.stdout, r.stderr)
	}
}

func wantError(t *testing.T, env envelope, code string) {
	t.Helper()
	if env.OK || len(env.Errors) == 0 || env.Errors[0].Code != code {
		t.Fatalf("want first error %q, got ok=%v errors=%+v", code, env.OK, env.Errors)
	}
}

// newProject creates a project with `chameleon init`.
func newProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	r := runCLI(t, dir, "init", "--no-input")
	wantExit(t, r, 0)
	return dir
}

func golden(t *testing.T, name, dir, got string) {
	t.Helper()
	got = strings.ReplaceAll(got, dir, "<DIR>")
	path := filepath.Join("testdata", "json", name+".golden.json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run: go test ./cmd/chameleon -run %s -update): %v", path, t.Name(), err)
	}
	if string(want) != got {
		t.Fatalf("output differs from %s (run with -update if the change is intended)\n--- want\n%s\n--- got\n%s", path, want, got)
	}
}

func TestJSONNotInitialized(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"validate"}, {"migrate", "--dry-run"}} {
		r := runCLI(t, dir, append(args, "--json")...)
		wantExit(t, r, 7)
		env := parseEnvelope(t, r)
		wantError(t, env, "not_initialized")
		if env.Errors[0].Hint == "" {
			t.Errorf("%v: not_initialized needs a hint", args)
		}
	}
	r := runCLI(t, dir, "verify", "--json")
	wantExit(t, r, 7)
	wantError(t, parseEnvelope(t, r), "not_initialized")
}

func TestJSONValidate(t *testing.T) {
	dir := newProject(t)

	r := runCLI(t, dir, "validate", "--json")
	wantExit(t, r, 0)
	env := parseEnvelope(t, r)
	if !env.OK || env.Command != "chameleon validate" || env.Data["valid"] != true {
		t.Fatalf("unexpected envelope: %+v", env)
	}

	if err := os.WriteFile(filepath.Join(dir, "bad.cham"), []byte("entity {"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = runCLI(t, dir, "validate", "bad.cham", "--json")
	wantExit(t, r, 3)
	env = parseEnvelope(t, r)
	wantError(t, env, "schema_invalid")
	e := env.Errors[0]
	if e.File != "bad.cham" || e.Line != 1 || e.Column == 0 || e.Snippet == "" {
		t.Errorf("error should carry file, line, column and snippet: %+v", e)
	}
	if env.Data["valid"] != false {
		t.Errorf("data.valid = %v, want false", env.Data["valid"])
	}
	golden(t, "validate_invalid", dir, r.stdout)

	// A file that does not exist is a usage error, not a schema error.
	r = runCLI(t, dir, "validate", "nope.cham", "--json")
	wantExit(t, r, 2)
	wantError(t, parseEnvelope(t, r), "usage_error")
}

func TestJSONStdoutOnlyHoldsTheDocument(t *testing.T) {
	dir := newProject(t)
	r := runCLI(t, dir, "validate", "--json")
	parseEnvelope(t, r) // fails on any extra byte
	if strings.TrimSpace(r.stderr) == "" {
		t.Errorf("human output should go to stderr under --json, stderr is empty")
	}
	if strings.Contains(r.stdout, "\x1b[") {
		t.Errorf("no color codes on stdout")
	}
}

func TestExitCodesDoNotDependOnJSON(t *testing.T) {
	dir := newProject(t)
	if err := os.WriteFile(filepath.Join(dir, "bad.cham"), []byte("entity {"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := [][]string{
		{"validate"},
		{"validate", "bad.cham"},
		{"validate", "nope.cham"},
		{"verify"},
		{"status"},
		{"migrate", "--dry-run"},
		{"migrate", "extra-arg"},
	}
	for _, args := range cases {
		plain := runCLI(t, dir, args...)
		js := runCLI(t, dir, append(append([]string{}, args...), "--json")...)
		if plain.exit != js.exit {
			t.Errorf("%v: exit %d without --json, %d with it", args, plain.exit, js.exit)
		}
		parseEnvelope(t, js)
	}
}

func TestJSONStatusAndVerify(t *testing.T) {
	dir := newProject(t)

	// Fresh project: no vault yet (the first migrate creates it).
	r := runCLI(t, dir, "status", "--json")
	wantExit(t, r, 0)
	env := parseEnvelope(t, r)
	vault, _ := env.Data["vault"].(map[string]any)
	if vault["initialized"] != false {
		t.Fatalf("unexpected vault status: %v", vault)
	}
	golden(t, "status_fresh", dir, r.stdout)

	wantExit(t, runCLI(t, dir, "migrate", "--dry-run"), 0)

	r = runCLI(t, dir, "status", "--json")
	wantExit(t, r, 0)
	env = parseEnvelope(t, r)
	vault, _ = env.Data["vault"].(map[string]any)
	if vault["initialized"] != true || vault["integrity"] != "ok" {
		t.Fatalf("unexpected vault status: %v", vault)
	}
	golden(t, "status_vault", dir, r.stdout)

	r = runCLI(t, dir, "verify", "--json")
	wantExit(t, r, 0)
	env = parseEnvelope(t, r)
	if env.Data["valid"] != true {
		t.Fatalf("verify data: %v", env.Data)
	}
	golden(t, "verify_ok", dir, r.stdout)
}

func TestJSONTamperedModeFile(t *testing.T) {
	dir := newProject(t)
	// The vault is created by the first migrate.
	wantExit(t, runCLI(t, dir, "migrate", "--dry-run"), 0)
	modePath := filepath.Join(dir, ".chameleon", "vault", "mode.json")
	b, err := os.ReadFile(modePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modePath, bytes.ReplaceAll(b, []byte("readonly"), []byte("privileged")), 0o644); err != nil {
		t.Fatal(err)
	}

	r := runCLI(t, dir, "verify", "--json")
	wantExit(t, r, 5)
	env := parseEnvelope(t, r)
	wantError(t, env, "integrity_failed")
	if env.Data["valid"] != false {
		t.Errorf("verify data.valid = %v, want false", env.Data["valid"])
	}

	r = runCLI(t, dir, "migrate", "--dry-run", "--json")
	wantExit(t, r, 5)
	wantError(t, parseEnvelope(t, r), "integrity_failed")

	// status reports the problem instead of failing: it is the diagnostic command.
	r = runCLI(t, dir, "status", "--json")
	wantExit(t, r, 0)
	env = parseEnvelope(t, r)
	vault, _ := env.Data["vault"].(map[string]any)
	if vault["integrity"] != "failed" {
		t.Errorf("status vault.integrity = %v, want failed", vault["integrity"])
	}
}

func TestJSONMigrateDryRun(t *testing.T) {
	dir := newProject(t)
	r := runCLI(t, dir, "migrate", "--dry-run", "--json")
	wantExit(t, r, 0)
	env := parseEnvelope(t, r)
	if env.Data["mode"] != "dry_run" || env.Data["applied"] != false || env.Data["changed"] != true {
		t.Fatalf("unexpected data: %v", env.Data)
	}
	sql, _ := env.Data["sql"].(string)
	if !strings.Contains(sql, "CREATE TABLE") {
		t.Errorf("data.sql should hold the plan, got %q", sql)
	}
	golden(t, "migrate_dry_run", dir, r.stdout)

	// Dry run must not touch the database or register a version.
	r = runCLI(t, dir, "status", "--json")
	env = parseEnvelope(t, r)
	if v, _ := env.Data["vault"].(map[string]any); v["versions"] != float64(0) {
		t.Errorf("dry-run registered a version: %v", v)
	}
}

func TestJSONUsageErrors(t *testing.T) {
	dir := newProject(t)

	r := runCLI(t, dir, "init", "--json")
	wantExit(t, r, 2)
	env := parseEnvelope(t, r)
	wantError(t, env, "usage_error")
	golden(t, "usage_unsupported", dir, r.stdout)

	r = runCLI(t, dir, "validate", "--nope", "--json")
	wantExit(t, r, 2)
	wantError(t, parseEnvelope(t, r), "usage_error")

	r = runCLI(t, dir, "bogus", "--json")
	wantExit(t, r, 2)

	r = runCLI(t, dir, "bogus")
	wantExit(t, r, 2)
	if !strings.Contains(r.stderr, "Error:") || r.stdout != "" {
		t.Errorf("without --json an error goes to stderr only; stdout=%q", r.stdout)
	}
}

func TestJSONIntrospectNeverPromptsAndNeverLeaksPasswords(t *testing.T) {
	// Inside a project the vault starts in readonly mode, which blocks introspect.
	dir := newProject(t)
	wantExit(t, runCLI(t, dir, "migrate", "--dry-run"), 0) // creates the vault
	r := runCLI(t, dir, "introspect", "postgresql://u:topsecret@127.0.0.1:1/db", "--json")
	wantExit(t, r, 4)
	wantError(t, parseEnvelope(t, r), "mode_blocked")
	if strings.Contains(r.stdout+r.stderr, "topsecret") {
		t.Errorf("password leaked:\n%s\n%s", r.stdout, r.stderr)
	}

	// Without a vault: an existing default schema would need a confirmation.
	plain := t.TempDir()
	if err := os.MkdirAll(filepath.Join(plain, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := []byte("entity User {\n    id: uuid primary,\n}\n")
	if err := os.WriteFile(filepath.Join(plain, "schemas", "schema.cham"), existing, 0o644); err != nil {
		t.Fatal(err)
	}
	r = runCLI(t, plain, "introspect", "postgresql://u:topsecret@127.0.0.1:1/db", "--json")
	wantExit(t, r, 2)
	env := parseEnvelope(t, r)
	wantError(t, env, "usage_error")
	if !strings.Contains(env.Errors[0].Hint, "--force") {
		t.Errorf("hint should mention --force: %q", env.Errors[0].Hint)
	}

	// With --force it reaches the connection, which fails (nothing listens on port 1).
	r = runCLI(t, plain, "introspect", "postgresql://u:topsecret@127.0.0.1:1/db", "--force", "--json")
	wantExit(t, r, 6)
	wantError(t, parseEnvelope(t, r), "connection_failed")
	if strings.Contains(r.stdout+r.stderr, "topsecret") {
		t.Errorf("password leaked:\n%s\n%s", r.stdout, r.stderr)
	}

	// Same without --json: the password is redacted on stderr too.
	r = runCLI(t, plain, "introspect", "postgresql://u:topsecret@127.0.0.1:1/db", "--force")
	wantExit(t, r, 6)
	if strings.Contains(r.stdout+r.stderr, "topsecret") {
		t.Errorf("password leaked without --json:\n%s\n%s", r.stdout, r.stderr)
	}

	r = runCLI(t, plain, "introspect", "$CHAMELEON_TEST_UNSET_URL", "--json")
	wantExit(t, r, 2)
	wantError(t, parseEnvelope(t, r), "usage_error")
}

func TestExitCodeTableMatchesDocs(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "cli-reference.md"))
	if err != nil {
		t.Skipf("docs not available: %v", err)
	}
	for _, code := range []string{"internal_error", "usage_error", "schema_invalid", "mode_blocked",
		"integrity_failed", "connection_failed", "not_initialized", "apply_failed"} {
		if !strings.Contains(string(doc), "`"+code+"`") {
			t.Errorf("docs/cli-reference.md does not document %s", code)
		}
	}
}

func TestInitAgentsFlag(t *testing.T) {
	// Without the flag: the guide has the section and nothing else is created.
	plain := t.TempDir()
	wantExit(t, runCLI(t, plain, "init", "--no-input"), 0)
	if _, err := os.Stat(filepath.Join(plain, "AGENTS.md")); err == nil {
		t.Fatal("init without --agents must not create AGENTS.md")
	}
	guide, err := os.ReadFile(filepath.Join(plain, "CHAMELEON.md"))
	if err != nil || !strings.Contains(string(guide), "## For coding agents") {
		t.Fatalf("CHAMELEON.md needs the agents section: %v", err)
	}

	// With the flag over an existing AGENTS.md: content intact, block last, once.
	dir := t.TempDir()
	original := "# Team rules\nUse tabs.\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	wantExit(t, runCLI(t, dir, "init", "--no-input", "--agents"), 0)
	// On an initialized project, --agents only touches AGENTS.md (and is idempotent).
	before, _ := os.ReadFile(filepath.Join(dir, ".chameleon.yml"))
	wantExit(t, runCLI(t, dir, "init", "--agents"), 0)
	after, _ := os.ReadFile(filepath.Join(dir, ".chameleon.yml"))
	if string(before) != string(after) {
		t.Error(".chameleon.yml must not change")
	}
	// Without the flag, init still refuses an initialized project.
	wantExit(t, runCLI(t, dir, "init", "--no-input"), 1)
	got, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), original) {
		t.Errorf("the original content must stay intact:\n%s", got)
	}
	if strings.Count(string(got), "# chameleon agents") != 1 {
		t.Errorf("the block must appear once:\n%s", got)
	}
}
