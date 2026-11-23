package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// These tests run the real `chameleon` binary with --json against the test
// database: the paths that need PostgreSQL (apply, retry after a failed
// connection, a rejected migration, introspect). The unit-level contract of
// --json is covered in cmd/chameleon/e2e_json_test.go.

const cliModePassword = "it-mode-password"

var (
	cliBuildOnce sync.Once
	cliBinary    string
	cliBuildErr  error
)

// buildCLI compiles cmd/chameleon once per test run, with the environment of
// the test (so the same CGO settings apply).
func buildCLI(t *testing.T) string {
	t.Helper()
	cliBuildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "chameleon-cli-it-")
		if err != nil {
			cliBuildErr = err
			return
		}
		cliBinary = filepath.Join(dir, "chameleon")
		cmd := exec.Command("go", "build", "-o", cliBinary, "../../cmd/chameleon")
		out, err := cmd.CombinedOutput()
		if err != nil {
			cliBuildErr = fmt.Errorf("go build failed: %v\n%s", err, out)
		}
	})
	if cliBuildErr != nil {
		t.Fatalf("cannot build the CLI: %v", cliBuildErr)
	}
	return cliBinary
}

type cliResult struct {
	stdout, stderr string
	exit           int
}

func (r cliResult) String() string {
	return fmt.Sprintf("exit=%d\nstdout:\n%s\nstderr:\n%s", r.exit, r.stdout, r.stderr)
}

type cliEnvelope struct {
	SchemaVersion int            `json:"schema_version"`
	Command       string         `json:"command"`
	OK            bool           `json:"ok"`
	Data          map[string]any `json:"data"`
	Errors        []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"errors"`
}

// cliProject is a project directory plus the database it points to.
type cliProject struct {
	t     *testing.T
	dir   string
	dbURL string
}

// runCLI runs the binary in the project. Extra env entries are KEY=VALUE.
func (p *cliProject) run(extraEnv []string, args ...string) cliResult {
	p.t.Helper()
	cmd := exec.Command(buildCLI(p.t), args...)
	cmd.Dir = p.dir
	cmd.Env = append(os.Environ(), "CI=1", "NO_COLOR=1", "DATABASE_URL="+p.dbURL, "CHAMELEON_MODE_PASSWORD=")
	cmd.Env = append(cmd.Env, extraEnv...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Start(); err != nil {
		p.t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		code := 0
		if err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				p.t.Fatalf("running %v: %v", args, err)
			}
			code = ee.ExitCode()
		}
		return cliResult{out.String(), errb.String(), code}
	case <-time.After(45 * time.Second):
		_ = cmd.Process.Kill()
		p.t.Fatalf("timeout running %v", args)
		return cliResult{}
	}
}

// envelope requires stdout to be exactly one JSON envelope.
func (p *cliProject) envelope(r cliResult) cliEnvelope {
	p.t.Helper()
	var env cliEnvelope
	dec := json.NewDecoder(strings.NewReader(r.stdout))
	if err := dec.Decode(&env); err != nil {
		p.t.Fatalf("stdout is not a JSON envelope: %v\n%s", err, r)
	}
	if dec.More() {
		p.t.Fatalf("stdout has more than one document\n%s", r)
	}
	if env.SchemaVersion != 1 || env.Data == nil || env.OK != (len(env.Errors) == 0) {
		p.t.Fatalf("malformed envelope: %+v\n%s", env, r)
	}
	return env
}

func (p *cliProject) want(r cliResult, exit int) cliEnvelope {
	p.t.Helper()
	if r.exit != exit {
		p.t.Fatalf("want exit %d\n%s", exit, r)
	}
	env := p.envelope(r)
	if exit != 0 && (env.OK || len(env.Errors) == 0) {
		p.t.Fatalf("a failing run must say so in the envelope\n%s", r)
	}
	return env
}

func (p *cliProject) writeSchema(content string) {
	p.t.Helper()
	path := filepath.Join(p.dir, "schemas", "app.cham")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		p.t.Fatal(err)
	}
}

const cliSchemaV1 = `entity Widget {
    id: uuid primary default uuid(),
    name: string,
}
`

const cliSchemaV2 = cliSchemaV1 + `
entity Gadget {
    id: uuid primary default uuid(),
    label: string,
}
`

// newCLIProject creates a clean database and a project with one schema.
func newCLIProject(t *testing.T) *cliProject {
	t.Helper()
	skipIfNoDocker(t)

	ctx := context.Background()
	cleanupDatabase(t, ctx, testConfig())
	t.Cleanup(func() { cleanupDatabase(t, context.Background(), testConfig()) })

	p := &cliProject{t: t, dir: t.TempDir(), dbURL: testConfig().ConnectionString()}
	if r := p.run(nil, "init", "--no-input"); r.exit != 0 {
		t.Fatalf("init failed\n%s", r)
	}
	// init writes an example schema; replace it with a small one.
	_ = os.Remove(filepath.Join(p.dir, "schemas", "example.cham"))
	p.writeSchema(cliSchemaV1)
	return p
}

func tableExists(t *testing.T, name string) bool {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	var exists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`, name).Scan(&exists)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	return exists
}

// TestCLIDryRunDoesNotTouchTheDatabase: the plan is in data.sql and nothing is
// created.
func TestCLIDryRunDoesNotTouchTheDatabase(t *testing.T) {
	p := newCLIProject(t)
	env := p.want(p.run(nil, "migrate", "--dry-run", "--json"), 0)
	if env.Data["mode"] != "dry_run" || env.Data["applied"] != false {
		t.Fatalf("dry run data: %v", env.Data)
	}
	if sql, _ := env.Data["sql"].(string); !strings.Contains(sql, "widgets") {
		t.Fatalf("data.sql should hold the plan, got %q", sql)
	}
	if tableExists(t, "widgets") {
		t.Fatal("--dry-run created a table")
	}
}

// TestCLIApplyJSON: first apply (with the switch to standard mode), no-op
// apply, a second version, and the plan-only modes not touching the database.
func TestCLIApplyJSON(t *testing.T) {
	p := newCLIProject(t)
	modeEnv := []string{"CHAMELEON_MODE_PASSWORD=" + cliModePassword}

	// First apply.
	r := p.run(modeEnv, "migrate", "--apply", "--json", "--no-input", "--set-mode", "standard")
	env := p.want(r, 0)
	if env.Data["applied"] != true || env.Data["version"] != "v001" || env.Data["mode"] != "apply" {
		t.Fatalf("first apply data: %v\n%s", env.Data, r)
	}
	if h, _ := env.Data["hash"].(string); len(h) < 16 {
		t.Errorf("hash missing: %v", env.Data["hash"])
	}
	if !tableExists(t, "widgets") {
		t.Fatal("table widgets was not created")
	}

	// Nothing changed: up to date, nothing applied.
	env = p.want(p.run(nil, "migrate", "--apply", "--json", "--no-input"), 0)
	if env.Data["up_to_date"] != true || env.Data["applied"] != false {
		t.Fatalf("second apply data: %v", env.Data)
	}

	// A change becomes the next version.
	p.writeSchema(cliSchemaV2)
	env = p.want(p.run(nil, "migrate", "--apply", "--json", "--no-input"), 0)
	if env.Data["applied"] != true || env.Data["version"] != "v002" {
		t.Fatalf("third apply data: %v", env.Data)
	}
	if !tableExists(t, "gadgets") {
		t.Fatal("table gadgets was not created")
	}

	// status and verify agree.
	env = p.want(p.run(nil, "status", "--json"), 0)
	vault, _ := env.Data["vault"].(map[string]any)
	if vault["integrity"] != "ok" || vault["mode"] != "standard" || vault["versions"] != float64(2) {
		t.Fatalf("status vault: %v", vault)
	}
	env = p.want(p.run(nil, "verify", "--json"), 0)
	if env.Data["valid"] != true {
		t.Fatalf("verify: %v", env.Data)
	}
}

// TestCLIReadonlyBlocksSecondApply: without the switch to standard, a second
// apply is refused with mode_blocked, and the first apply keeps stdout clean.
func TestCLIReadonlyBlocksSecondApply(t *testing.T) {
	p := newCLIProject(t)

	r := p.run(nil, "migrate", "--apply", "--json", "--no-input")
	p.want(r, 0)
	if !strings.Contains(r.stderr, "readonly") {
		t.Errorf("the readonly notice should be on stderr:\n%s", r.stderr)
	}

	p.writeSchema(cliSchemaV2)
	env := p.want(p.run(nil, "migrate", "--apply", "--json", "--no-input"), 4)
	if env.Errors[0].Code != "mode_blocked" || env.Errors[0].Hint == "" {
		t.Fatalf("want mode_blocked with a hint, got %+v", env.Errors)
	}
	if tableExists(t, "gadgets") {
		t.Fatal("a blocked apply changed the database")
	}
}

// TestCLIConnectionFailedThenRetry: an unreachable database exits 6 without
// leaking the password; the version stays registered and the next apply, with
// the database back, applies it. Standard mode is set first because in
// readonly mode even the retry is refused (see the PR notes).
func TestCLIConnectionFailedThenRetry(t *testing.T) {
	p := newCLIProject(t)
	good := p.dbURL

	// v001 applied, project switched to standard mode.
	p.want(p.run([]string{"CHAMELEON_MODE_PASSWORD=" + cliModePassword},
		"migrate", "--apply", "--json", "--no-input", "--set-mode", "standard"), 0)

	// A change, but the database is unreachable.
	p.writeSchema(cliSchemaV2)
	p.dbURL = "postgresql://postgres:it-secret-pw@127.0.0.1:1/chameleon_test"
	r := p.run(nil, "migrate", "--apply", "--json", "--no-input")
	env := p.want(r, 6)
	if env.Errors[0].Code != "connection_failed" {
		t.Fatalf("want connection_failed, got %+v", env.Errors)
	}
	if strings.Contains(r.stdout+r.stderr, "it-secret-pw") {
		t.Fatalf("password leaked\n%s", r)
	}

	// Database back: the registered version is pending.
	p.dbURL = good
	env = p.want(p.run(nil, "migrate", "--json"), 0)
	if env.Data["pending_unapplied"] != true {
		t.Fatalf("expected pending_unapplied after a failed connection: %v", env.Data)
	}

	env = p.want(p.run(nil, "migrate", "--apply", "--json", "--no-input"), 0)
	if env.Data["applied"] != true {
		t.Fatalf("retry did not apply: %v", env.Data)
	}
	if !tableExists(t, "gadgets") {
		t.Fatal("table gadgets was not created by the retry")
	}
}

// TestCLIApplyFailed: PostgreSQL rejects the SQL (a type named like the table
// already exists), which is exit 8 with the database error in the message.
func TestCLIApplyFailed(t *testing.T) {
	p := newCLIProject(t)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(ctx, `CREATE TYPE widgets AS ENUM ('a')`)
	conn.Close(ctx)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	env := p.want(p.run(nil, "migrate", "--apply", "--json", "--no-input"), 8)
	if env.Errors[0].Code != "apply_failed" || !strings.Contains(env.Errors[0].Message, "widgets") {
		t.Fatalf("want apply_failed mentioning the database error, got %+v", env.Errors)
	}
}

// TestCLIIntrospectJSON: introspect of a migrated database writes the schema
// and reports the table count.
func TestCLIIntrospectJSON(t *testing.T) {
	skipIfNoDocker(t)
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()
	runMigration(t, eng, ctx)

	// A directory without a vault: introspect does not need a project.
	p := &cliProject{t: t, dir: t.TempDir(), dbURL: testConfig().ConnectionString()}
	if err := os.MkdirAll(filepath.Join(p.dir, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := p.run(nil, "introspect", p.dbURL, "-o", "out.cham", "--json")
	env := p.want(r, 0)
	if env.Data["tables"] != float64(4) || env.Data["output"] != "schemas/out.cham" {
		t.Fatalf("introspect data: %v\n%s", env.Data, r)
	}
	content, err := os.ReadFile(filepath.Join(p.dir, "schemas", "out.cham"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "entity User {") {
		t.Fatalf("generated schema has no User entity:\n%s", content)
	}

	// A second run over the same file would need a confirmation: it must
	// refuse instead of asking.
	r = p.run(nil, "introspect", p.dbURL, "-o", "out.cham", "--json")
	env = p.want(r, 2)
	if env.Errors[0].Code != "usage_error" {
		t.Fatalf("want usage_error, got %+v", env.Errors)
	}
	r = p.run(nil, "introspect", p.dbURL, "-o", "out.cham", "--json", "--force")
	p.want(r, 0)
}
