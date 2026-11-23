package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/internal/cliout"
)

func TestExampleSchemaContainsCoreEntities(t *testing.T) {
	schema := exampleSchema()
	if !strings.Contains(schema, "entity User") {
		t.Fatal("example schema must contain User entity")
	}
	if !strings.Contains(schema, "entity Post") {
		t.Fatal("example schema must contain Post entity")
	}
	if !strings.Contains(schema, "published: bool") {
		t.Fatal("example schema must contain a bool field")
	}
}

func TestExampleReadmeContainsProjectNameAndCommands(t *testing.T) {
	readme := exampleReadme("myproj")
	if !strings.Contains(readme, "# myproj") {
		t.Fatal("readme must include project title")
	}
	if !strings.Contains(readme, "chameleon migrate --check") {
		t.Fatal("readme must include migrate check command")
	}
	if !strings.Contains(readme, "DATABASE_URL") {
		t.Fatal("readme must include DATABASE_URL setup")
	}
}

func TestExampleReadmeNeverClaimsBackupsOrRollback(t *testing.T) {
	readme := exampleReadme("myproj")
	for _, claim := range []string{"backed up before applying", "Supports rollback"} {
		if strings.Contains(readme, claim) {
			t.Errorf("the guide must not claim %q", claim)
		}
	}
	if !strings.Contains(readme, "does not create backups") {
		t.Error("the guide must say that no backup is created")
	}
	if strings.Contains(readme, "README.md               This file") {
		t.Error("the guide is CHAMELEON.md, not README.md")
	}
}

func TestInitNeverOverwritesExistingFiles(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	t.Setenv("CI", "1")

	project := filepath.Join(dir, "existing")
	if err := os.MkdirAll(filepath.Join(project, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	mine := map[string]string{
		"README.md":          "my readme\n",
		"CHAMELEON.md":       "my guide\n",
		"docker-compose.yml": "services: {}\n",
		"schemas/mine.cham":  "entity Mine { id: uuid primary default uuid(), }\n",
		".env.example":       "OTHER=1\n",
	}
	for name, content := range mine {
		if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := initCmd.RunE(nil, []string{project}); err != nil {
		t.Fatalf("init command: %v", err)
	}

	for name, content := range mine {
		if name == ".env.example" {
			continue // may get an appended block; checked below
		}
		got, err := os.ReadFile(filepath.Join(project, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Errorf("%s was modified: %q", name, got)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "schemas", "example.cham")); err == nil {
		t.Error("example.cham must not be created when schemas/ already has a .cham file")
	}
	env, err := os.ReadFile(filepath.Join(project, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(env), "OTHER=1\n") {
		t.Errorf(".env.example lost its original content: %q", env)
	}
}

func TestGuideHasSectionForCodingAgents(t *testing.T) {
	guide := exampleReadme("myproj")
	i := strings.Index(guide, "## For coding agents")
	if i < 0 {
		t.Fatal("CHAMELEON.md needs a section for coding agents")
	}
	section := guide[i:]
	if j := strings.Index(section[3:], "\n## "); j >= 0 {
		section = section[:j+3]
	}

	// It must say the things an agent gets wrong without them.
	for _, want := range []string{"Ask a human before", "migrate --apply", "--json", "--no-input", ".chameleon/", "readonly mode", "DATABASE_URL", "no backups and no rollback"} {
		if !strings.Contains(section, want) {
			t.Errorf("the section must mention %q", want)
		}
	}
	// It must not promise what the tool does not do (ADR-0002).
	lower := strings.ToLower(section)
	for _, claim := range []string{"approval workflow", "will ask", "is secure", "compliant", "backed up before"} {
		if strings.Contains(lower, claim) {
			t.Errorf("the section must not claim %q", claim)
		}
	}
}

// Every command the section names with --json must exist and support it, and
// every exit code in the section must be one the CLI defines.
func TestAgentsSectionMatchesTheCLI(t *testing.T) {
	section := agentsSection()
	for _, name := range []string{"validate", "status", "verify", "migrate", "introspect"} {
		if !strings.Contains(section, "`"+name+"`") {
			t.Errorf("section does not list %q", name)
		}
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Fatalf("command %q does not exist", name)
		}
		if cmd.Annotations[jsonAnnotation] != "true" {
			t.Errorf("%q is listed as supporting --json but does not", name)
		}
	}
	for _, e := range cliout.Table {
		if e.Exit == 0 {
			continue
		}
		if !strings.Contains(section, fmt.Sprintf("%d ", e.Exit)) {
			t.Errorf("section does not mention exit code %d (%s)", e.Exit, e.Code)
		}
	}
}
