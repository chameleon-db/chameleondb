package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetParanoidModeDefaultAfterInitialize(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	mode, err := v.GetParanoidMode()
	if err != nil {
		t.Fatalf("GetParanoidMode() error = %v", err)
	}

	if mode != "readonly" {
		t.Fatalf("expected mode readonly, got %s", mode)
	}
}

func TestSetParanoidModeAliasAdminToPrivileged(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := v.SetParanoidMode("admin"); err != nil {
		t.Fatalf("SetParanoidMode() error = %v", err)
	}

	mode, err := v.GetParanoidMode()
	if err != nil {
		t.Fatalf("GetParanoidMode() error = %v", err)
	}

	if mode != "privileged" {
		t.Fatalf("expected mode privileged, got %s", mode)
	}
}

func TestGetParanoidModeFallbackFromManifest(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := v.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	v.Manifest.ParanoidMode = "admin"
	if err := v.saveManifest(v.Manifest); err != nil {
		t.Fatalf("saveManifest() error = %v", err)
	}

	modePath := filepath.Join(root, VaultDirName, ModeFileName)
	if err := os.Remove(modePath); err != nil {
		t.Fatalf("Remove(mode.json) error = %v", err)
	}

	mode, err := v.GetParanoidMode()
	if err != nil {
		t.Fatalf("GetParanoidMode() error = %v", err)
	}

	if mode != "privileged" {
		t.Fatalf("expected mode privileged from legacy manifest, got %s", mode)
	}

	if _, err := os.Stat(modePath); err != nil {
		t.Fatalf("expected mode.json to be recreated, stat error = %v", err)
	}
}
