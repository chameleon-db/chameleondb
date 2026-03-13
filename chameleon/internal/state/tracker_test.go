package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewTrackerCreatesDirectory(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	tracker, err := NewTracker(stateDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}
}

func TestLoadCurrentDefaultWhenMissing(t *testing.T) {
	tracker, err := NewTracker(t.TempDir())
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	state, err := tracker.LoadCurrent()
	if err != nil {
		t.Fatalf("LoadCurrent() error = %v", err)
	}
	if state.Status != "pending_migration" {
		t.Fatalf("expected pending_migration, got %s", state.Status)
	}
	if state.Validation.Status != "valid" {
		t.Fatalf("expected valid validation status, got %s", state.Validation.Status)
	}
}

func TestSaveAndLoadCurrent(t *testing.T) {
	tracker, err := NewTracker(t.TempDir())
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	initial := &CurrentState{Status: "in_sync"}
	if err := tracker.SaveCurrent(initial); err != nil {
		t.Fatalf("SaveCurrent() error = %v", err)
	}

	loaded, err := tracker.LoadCurrent()
	if err != nil {
		t.Fatalf("LoadCurrent() error = %v", err)
	}
	if loaded.Status != "in_sync" {
		t.Fatalf("expected in_sync, got %s", loaded.Status)
	}
	if loaded.Version != "0.1.4" {
		t.Fatalf("expected version 0.1.4, got %s", loaded.Version)
	}
	if loaded.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestManifestLifecycle(t *testing.T) {
	tracker, err := NewTracker(t.TempDir())
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	manifest, err := tracker.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if len(manifest.Migrations) != 0 {
		t.Fatalf("expected empty migrations, got %d", len(manifest.Migrations))
	}

	mig1 := &Migration{Version: "v001", Status: "pending", AppliedAt: time.Now()}
	mig2 := &Migration{Version: "v002", Status: "applied", AppliedAt: time.Now()}

	if err := tracker.AddMigration(mig1); err != nil {
		t.Fatalf("AddMigration() error = %v", err)
	}
	if err := tracker.AddMigration(mig2); err != nil {
		t.Fatalf("AddMigration() error = %v", err)
	}

	last, err := tracker.GetLastMigration()
	if err != nil {
		t.Fatalf("GetLastMigration() error = %v", err)
	}
	if last == nil || last.Version != "v002" {
		t.Fatalf("expected last applied migration v002, got %#v", last)
	}
}

func TestHashHelpers(t *testing.T) {
	h1 := HashSchema("entity User { id: uuid primary, }")
	h2 := HashSchema("entity User { id: uuid primary, }")
	if h1 != h2 {
		t.Fatal("expected deterministic schema hash")
	}
	if len(h1) != 64 {
		t.Fatalf("expected sha256 hex len 64, got %d", len(h1))
	}

	d1 := HashDDL("CREATE TABLE users (id UUID PRIMARY KEY);")
	d2 := HashDDL("CREATE TABLE users (id UUID PRIMARY KEY);")
	if d1 != d2 {
		t.Fatal("expected deterministic ddl hash")
	}
	if len(d1) != 64 {
		t.Fatalf("expected sha256 hex len 64, got %d", len(d1))
	}
}
