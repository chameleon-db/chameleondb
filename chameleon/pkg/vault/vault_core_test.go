package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVaultInitializeExistsAndStatus(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	if v.Exists() {
		t.Fatal("vault should not exist before initialize")
	}

	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if !v.Exists() {
		t.Fatal("vault should exist after initialize")
	}

	status, err := v.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if !status.Exists {
		t.Fatal("expected status.Exists=true")
	}
	if status.TotalVersions != 0 {
		t.Fatalf("expected no versions, got %d", status.TotalVersions)
	}
}

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\n\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d (%#v)", len(lines), lines)
	}
	if lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestAppendAndReadLog(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)
	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := v.AppendLog("TEST", "v001", map[string]string{"key": "value"}); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}

	lines, err := v.ReadLog()
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one log line")
	}

	last := lines[len(lines)-1]
	if !strings.Contains(last, "[TEST]") || !strings.Contains(last, "version=v001") {
		t.Fatalf("unexpected log line: %s", last)
	}
}

func TestReadLogWhenMissing(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	lines, err := v.ReadLog()
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected empty log lines, got %d", len(lines))
	}
}

func TestRegisterVersionAndVersionQueries(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	schemaPath := filepath.Join(root, "schema.cham")
	schemaV1 := "entity User {\n  id: uuid primary,\n  email: string,\n}\n"
	if err := os.WriteFile(schemaPath, []byte(schemaV1), 0644); err != nil {
		t.Fatalf("write schema v1: %v", err)
	}

	entry1, err := v.RegisterVersion(schemaPath, "tester", "init")
	if err != nil {
		t.Fatalf("RegisterVersion(v1) error = %v", err)
	}
	if entry1.Version != "v001" {
		t.Fatalf("expected v001, got %s", entry1.Version)
	}

	// Register unchanged schema: should return same current version
	entry1Again, err := v.RegisterVersion(schemaPath, "tester", "same")
	if err != nil {
		t.Fatalf("RegisterVersion(same) error = %v", err)
	}
	if entry1Again.Version != "v001" {
		t.Fatalf("expected unchanged version v001, got %s", entry1Again.Version)
	}

	schemaV2 := "entity User {\n  id: uuid primary,\n  email: string,\n  active: bool,\n}\n"
	if err := os.WriteFile(schemaPath, []byte(schemaV2), 0644); err != nil {
		t.Fatalf("write schema v2: %v", err)
	}

	entry2, err := v.RegisterVersion(schemaPath, "tester", "add active")
	if err != nil {
		t.Fatalf("RegisterVersion(v2) error = %v", err)
	}
	if entry2.Version != "v002" {
		t.Fatalf("expected v002, got %s", entry2.Version)
	}
	if entry2.Parent == nil || *entry2.Parent != "v001" {
		t.Fatalf("expected parent v001, got %#v", entry2.Parent)
	}

	history, err := v.GetVersionHistory()
	if err != nil {
		t.Fatalf("GetVersionHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}

	current, err := v.GetCurrentVersion()
	if err != nil {
		t.Fatalf("GetCurrentVersion() error = %v", err)
	}
	if current.Version != "v002" {
		t.Fatalf("expected current v002, got %s", current.Version)
	}

	content, err := v.GetVersionContent("v001")
	if err != nil {
		t.Fatalf("GetVersionContent() error = %v", err)
	}
	if len(content) == 0 {
		t.Fatal("expected non-empty stored version content")
	}
}

func TestDetectChanges(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)
	schemaPath := filepath.Join(root, "schema.cham")

	if err := os.WriteFile(schemaPath, []byte("entity User { id: uuid primary, }\n"), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	changed, reason, err := v.DetectChanges(schemaPath)
	if err != nil {
		t.Fatalf("DetectChanges(initial) error = %v", err)
	}
	if !changed || reason == "" {
		t.Fatalf("expected initial changes detected, got changed=%v reason=%q", changed, reason)
	}

	if _, err := v.RegisterVersion(schemaPath, "tester", "init"); err != nil {
		t.Fatalf("RegisterVersion() error = %v", err)
	}

	changed, reason, err = v.DetectChanges(schemaPath)
	if err != nil {
		t.Fatalf("DetectChanges(same) error = %v", err)
	}
	if changed || reason != "" {
		t.Fatalf("expected no changes, got changed=%v reason=%q", changed, reason)
	}

	if err := os.WriteFile(schemaPath, []byte("entity User { id: uuid primary, email: string }\n"), 0644); err != nil {
		t.Fatalf("write schema updated: %v", err)
	}
	changed, reason, err = v.DetectChanges(schemaPath)
	if err != nil {
		t.Fatalf("DetectChanges(updated) error = %v", err)
	}
	if !changed || reason == "" {
		t.Fatalf("expected changes after update, got changed=%v reason=%q", changed, reason)
	}
}

func TestIntegrityVerification(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)
	schemaPath := filepath.Join(root, "schema.cham")
	if err := os.WriteFile(schemaPath, []byte("entity User { id: uuid primary, }\n"), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	res, err := v.VerifyIntegrity()
	if err != nil {
		t.Fatalf("VerifyIntegrity(no-vault) error = %v", err)
	}
	if res.Valid {
		t.Fatal("expected invalid integrity when vault does not exist")
	}

	if _, err := v.RegisterVersion(schemaPath, "tester", "init"); err != nil {
		t.Fatalf("RegisterVersion() error = %v", err)
	}

	res, err = v.VerifyIntegrity()
	if err != nil {
		t.Fatalf("VerifyIntegrity(valid) error = %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid integrity, issues=%v", res.Issues)
	}
	if len(res.VersionsOK) == 0 {
		t.Fatal("expected verified versions in VersionsOK")
	}

	// Corrupt stored hash file and verify failure
	hashPath := filepath.Join(root, VaultDirName, HashesDirName, "v001.hash")
	if err := os.WriteFile(hashPath, []byte("bad-hash"), 0644); err != nil {
		t.Fatalf("corrupt hash file: %v", err)
	}
	res, err = v.VerifyIntegrity()
	if err != nil {
		t.Fatalf("VerifyIntegrity(corrupt) error = %v", err)
	}
	if res.Valid {
		t.Fatal("expected integrity failure after hash corruption")
	}
}

func TestHelpersFormatSerializeAndStringOrNull(t *testing.T) {
	entry := &VersionEntry{
		Version:        "v001",
		Hash:           strings.Repeat("a", 64),
		Timestamp:      time.Now().UTC(),
		Author:         "tester",
		ChangesSummary: "init",
	}

	formatted := FormatVersion(entry)
	if !strings.Contains(formatted, "v001") || !strings.Contains(formatted, "Changes: init") {
		t.Fatalf("unexpected formatted version output: %s", formatted)
	}

	if stringOrNull(nil) != "none" {
		t.Fatalf("expected stringOrNull(nil)=none")
	}
	parent := "v000"
	if stringOrNull(&parent) != "v000" {
		t.Fatalf("expected stringOrNull(parent)=v000")
	}

	data, err := SerializeSchema(map[string]interface{}{"entity": "User"})
	if err != nil {
		t.Fatalf("SerializeSchema() error = %v", err)
	}
	if !strings.Contains(string(data), "User") {
		t.Fatalf("unexpected serialized schema: %s", string(data))
	}
}
