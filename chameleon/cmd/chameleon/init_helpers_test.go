package main

import (
	"strings"
	"testing"
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
