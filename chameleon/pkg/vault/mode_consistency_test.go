package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newInitializedVault(t *testing.T) *Vault {
	t.Helper()
	v := NewVault(t.TempDir())
	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	return v
}

func writeModeFile(t *testing.T, v *Vault, content string) {
	t.Helper()
	if err := os.WriteFile(v.modeConfigPath(), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func verifyIssues(t *testing.T, v *Vault) (bool, []string) {
	t.Helper()
	res, err := v.VerifyIntegrity()
	if err != nil {
		t.Fatalf("VerifyIntegrity() error = %v", err)
	}
	return res.Valid, res.Issues
}

func TestVerifyIntegrity_FreshVaultIsConsistent(t *testing.T) {
	v := newInitializedVault(t)
	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("a fresh vault should verify, got %v", issues)
	}
}

func TestVerifyIntegrity_DetectsModeEditedAfterInit(t *testing.T) {
	v := newInitializedVault(t)
	writeModeFile(t, v, `{"paranoid_mode": "standard"}`)

	valid, issues := verifyIssues(t, v)
	if valid {
		t.Fatal("editing mode.json by hand must be reported")
	}
	if len(issues) != 1 || !strings.Contains(issues[0], `"standard"`) || !strings.Contains(issues[0], `"readonly"`) {
		t.Errorf("issue should name both modes, got %v", issues)
	}
}

func TestVerifyIntegrity_DetectsModeEditedAfterChange(t *testing.T) {
	v := newInitializedVault(t)
	if err := v.SetParanoidMode("standard"); err != nil {
		t.Fatal(err)
	}
	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("a mode set through the API must verify, got %v", issues)
	}

	writeModeFile(t, v, `{"paranoid_mode": "emergency"}`)
	if valid, _ := verifyIssues(t, v); valid {
		t.Fatal("editing mode.json after a recorded change must be reported")
	}

	if err := v.SetParanoidMode("readonly"); err != nil {
		t.Fatal(err)
	}
	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("going back through the API must verify, got %v", issues)
	}
}

func TestVerifyIntegrity_DeniedChangesDoNotCount(t *testing.T) {
	v := newInitializedVault(t)
	if err := v.AppendLog("MODE", "", map[string]string{
		"action": "mode_change_denied",
		"mode":   "emergency",
		"reason": "invalid_mode_password",
	}); err != nil {
		t.Fatal(err)
	}

	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("a denied attempt did not change the mode, got %v", issues)
	}
}

func TestVerifyIntegrity_ModeJSONAliasIsNormalized(t *testing.T) {
	v := newInitializedVault(t)
	if err := v.SetParanoidMode("admin"); err != nil {
		t.Fatal(err)
	}
	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("the admin alias is recorded as privileged, got %v", issues)
	}
}

func TestVerifyIntegrity_CorruptModeFile(t *testing.T) {
	v := newInitializedVault(t)
	writeModeFile(t, v, `{not json`)
	if valid, _ := verifyIssues(t, v); valid {
		t.Fatal("an unparsable mode.json must be reported")
	}

	writeModeFile(t, v, `{"paranoid_mode": "godmode"}`)
	if valid, _ := verifyIssues(t, v); valid {
		t.Fatal("an invalid mode must be reported")
	}
}

func TestVerifyIntegrity_LegacyVaultWithoutModeEventsIsNotFlagged(t *testing.T) {
	v := newInitializedVault(t)

	// A vault created before the initial mode was recorded: no mode event.
	logPath := filepath.Join(v.RootPath, VaultDirName, IntegrityLogName)
	legacy := "2026-01-01T00:00:00Z [INIT] action=vault_created\n"
	if err := os.WriteFile(logPath, []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	writeModeFile(t, v, `{"paranoid_mode": "standard"}`)

	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("with nothing to compare against it must not flag, got %v", issues)
	}
}

func TestVerifyIntegrity_MissingModeFileFallsBackToManifest(t *testing.T) {
	v := newInitializedVault(t)
	if err := os.Remove(v.modeConfigPath()); err != nil {
		t.Fatal(err)
	}
	if valid, issues := verifyIssues(t, v); !valid {
		t.Fatalf("the legacy layout without mode.json must still verify, got %v", issues)
	}
}
