package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLoggerCreatesDirectory(t *testing.T) {
	journalDir := filepath.Join(t.TempDir(), "journal")

	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	if st, err := os.Stat(journalDir); err != nil || !st.IsDir() {
		t.Fatalf("expected journal directory to exist: %v", err)
	}
}

func TestLogAndLast(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.Log("validate", "ok", map[string]interface{}{"entity": "User"}, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if err := logger.Log("migrate", "ok", map[string]interface{}{"version": "v001"}, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	entries, err := logger.Last(10)
	if err != nil {
		t.Fatalf("Last() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Action != "validate" || entries[1].Action != "migrate" {
		t.Fatalf("unexpected actions: %#v", []string{entries[0].Action, entries[1].Action})
	}
}

func TestErrorsFiltersEntries(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.Log("validate", "ok", nil, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if err := logger.LogError("migrate", errors.New("boom"), map[string]interface{}{"step": "apply"}); err != nil {
		t.Fatalf("LogError() error = %v", err)
	}

	errEntries, err := logger.Errors()
	if err != nil {
		t.Fatalf("Errors() error = %v", err)
	}
	if len(errEntries) != 1 {
		t.Fatalf("expected 1 error entry, got %d", len(errEntries))
	}
	if errEntries[0].Action != "migrate" || errEntries[0].Status != "error" {
		t.Fatalf("unexpected error entry: %#v", errEntries[0])
	}
}

func TestLogMigrationAndMigrations(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.LogMigration("v002", "ok", 17, "backup.sql", map[string]interface{}{"from": "v001"}); err != nil {
		t.Fatalf("LogMigration() error = %v", err)
	}
	if err := logger.Log("validate", "ok", nil, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	entries, err := logger.Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 migration entry, got %d", len(entries))
	}
	if entries[0].Action != "migrate" {
		t.Fatalf("expected migrate action, got %s", entries[0].Action)
	}
}

func TestLastAndErrorsWhenNoFile(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entries, err := logger.Last(5)
	if err != nil {
		t.Fatalf("Last() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty entries, got %d", len(entries))
	}

	errEntries, err := logger.Errors()
	if err != nil {
		t.Fatalf("Errors() error = %v", err)
	}
	if len(errEntries) != 0 {
		t.Fatalf("expected empty errors, got %d", len(errEntries))
	}
}

func TestRoundTripKeepsEverything(t *testing.T) {
	logger, err := NewLogger(t.TempDir())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	err = logger.LogMigration("v002", "error", 1200, "", map[string]interface{}{"note": "two words"})
	if err != nil {
		t.Fatalf("LogMigration() error = %v", err)
	}
	if err := logger.LogError("migrate", errors.New("connection refused by host"), nil); err != nil {
		t.Fatalf("LogError() error = %v", err)
	}
	entries, err := logger.Last(10)
	if err != nil || len(entries) != 2 {
		t.Fatalf("Last() = %d entries, err %v", len(entries), err)
	}
	first := entries[0]
	if first.Duration != 1200 {
		t.Errorf("duration = %d, want 1200", first.Duration)
	}
	if first.Details["version"] != "v002" || first.Details["note"] != "two words" {
		t.Errorf("details not read back: %v", first.Details)
	}
	if got := entries[1].Error; got != "connection refused by host" {
		t.Errorf("error = %q, want the full message", got)
	}
}

func TestFormatEntryIsDeterministicAndSingleLine(t *testing.T) {
	logger := &Logger{}
	entry := &Entry{
		Timestamp: time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC),
		Action:    "config_mode",
		Status:    "denied",
		Details: map[string]interface{}{
			"zeta": 1, "alpha": "a b", "mid": "line1\nline2 status=ok",
		},
	}
	first := logger.formatEntry(entry)
	for i := 0; i < 20; i++ {
		if got := logger.formatEntry(entry); got != first {
			t.Fatalf("formatEntry is not deterministic:\n%s\n%s", first, got)
		}
	}
	if strings.Contains(first, "\n") {
		t.Fatalf("an entry must be one line, got %q", first)
	}
	want := `2026-11-06T10:00:00Z [config_mode] status=denied alpha="a b" mid="line1\nline2 status=ok" zeta=1`
	if first != want {
		t.Fatalf("formatEntry =\n%s\nwant\n%s", first, want)
	}
	parsed, err := parseEntry(first)
	if err != nil {
		t.Fatalf("parseEntry() error = %v", err)
	}
	if parsed.Status != "denied" {
		t.Errorf("a value containing status=ok changed the status to %q", parsed.Status)
	}
	if parsed.Details["mid"] != "line1\nline2 status=ok" {
		t.Errorf("mid = %q", parsed.Details["mid"])
	}
}

func TestReservedDetailKeysDoNotOverrideFields(t *testing.T) {
	logger, err := NewLogger(t.TempDir())
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	details := map[string]interface{}{"status": "spoofed", "error": "fake", "duration_ms": 99}
	if err := logger.Log("check", "ok", details, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	entries, _ := logger.Last(1)
	e := entries[0]
	if e.Status != "ok" || e.Error != "" || e.Duration != 0 {
		t.Fatalf("reserved fields were overridden: %+v", e)
	}
	if e.Details["status"] != "spoofed" || e.Details["error"] != "fake" || e.Details["duration_ms"] != "99" {
		t.Fatalf("details were not preserved: %v", e.Details)
	}
}

func writeDay(t *testing.T, dir string, day string, lines ...string) {
	t.Helper()
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, day+".log"), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", day, err)
	}
}

func TestLastReadsBackAcrossDays(t *testing.T) {
	dir := t.TempDir()
	logger, _ := NewLogger(dir)
	writeDay(t, dir, "2026-11-04",
		"2026-11-04T09:00:00Z [validate] status=ok",
		"2026-11-04T09:05:00Z [migrate] status=ok version=v001")
	writeDay(t, dir, "2026-11-05", "2026-11-05T09:00:00Z [introspect] status=ok")
	entries, err := logger.Last(2)
	if err != nil {
		t.Fatalf("Last() error = %v", err)
	}
	if len(entries) != 2 || entries[0].Action != "migrate" || entries[1].Action != "introspect" {
		t.Fatalf("expected [migrate introspect], got %v", actions(entries))
	}
	all, _ := logger.Last(50)
	if len(all) != 3 || all[0].Action != "validate" {
		t.Fatalf("expected all 3 entries oldest first, got %v", actions(all))
	}
}

func TestErrorsIncludeFailedAndDenied(t *testing.T) {
	dir := t.TempDir()
	logger, _ := NewLogger(dir)
	writeDay(t, dir, "2026-11-04",
		"2026-11-04T09:00:00Z [migrate] status=error error=\"boom happened\"",
		"2026-11-04T09:01:00Z [config_mode] status=denied reason=invalid_mode_password",
		"2026-11-04T09:02:00Z [config_schema_paths] status=failed",
		"2026-11-04T09:03:00Z [introspect] status=aborted_readonly",
		"2026-11-04T09:04:00Z [validate] status=ok",
		"2026-11-04T09:05:00Z [config_mode] status=success")
	entries, err := logger.Errors()
	if err != nil {
		t.Fatalf("Errors() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 error entries, got %v", actions(entries))
	}
	if entries[0].Error != "boom happened" {
		t.Errorf("error = %q", entries[0].Error)
	}
}

func TestMigrationsSpanDaysAndKeepVersion(t *testing.T) {
	dir := t.TempDir()
	logger, _ := NewLogger(dir)
	writeDay(t, dir, "2026-11-03", "2026-11-03T09:00:00Z [migrate] status=ok version=v001 backup_path=\"\" duration_ms=40")
	writeDay(t, dir, "2026-11-04", "2026-11-04T09:00:00Z [migrate] status=ok version=v002 duration_ms=55")
	entries, err := logger.Migrations()
	if err != nil || len(entries) != 2 {
		t.Fatalf("Migrations() = %d entries, err %v", len(entries), err)
	}
	if entries[0].Details["version"] != "v001" || entries[0].Duration != 40 {
		t.Errorf("first migration lost its version or duration: %+v", entries[0])
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	logger, _ := NewLogger(dir)
	writeDay(t, dir, "2026-11-04",
		"2026-11-04T09:00:00Z [migrate] status=ok version=v001",
		"2026-11-04T09:01:00Z [config_mode] status=denied reason=invalid_mode_password")
	writeDay(t, dir, "2026-11-05",
		"2026-11-05T09:00:00Z [migrate] status=ok version=v002",
		"2026-11-05T09:01:00Z [migrate] status=error error=\"Boom\"")
	byAction, _ := logger.Search("MIGRATE", 0)
	if len(byAction) != 3 {
		t.Fatalf("expected 3 migrate matches, got %v", actions(byAction))
	}
	limited, _ := logger.Search("migrate", 1)
	if len(limited) != 1 || limited[0].Error != "Boom" {
		t.Fatalf("limit should keep the most recent match, got %+v", limited)
	}
	none, err := logger.Search("nothing-here", 0)
	if err != nil || len(none) != 0 {
		t.Fatalf("expected no matches, got %d, err %v", len(none), err)
	}
}

func TestParseEntryToleratesOldFormat(t *testing.T) {
	entry, err := parseEntry(`2026-02-12T10:15:00Z [migrate] status=error version=v001 note=two words error="bad thing happened"`)
	if err != nil {
		t.Fatalf("parseEntry() error = %v", err)
	}
	if entry.Status != "error" || entry.Error != "bad thing happened" || entry.Details["version"] != "v001" {
		t.Fatalf("unexpected parse result: %+v", entry)
	}
}

func actions(entries []*Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Action)
	}
	return out
}
