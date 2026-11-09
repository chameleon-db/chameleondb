package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetAndVerifyModePassword(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := v.SetModePassword("supersecure123"); err != nil {
		t.Fatalf("SetModePassword() error = %v", err)
	}

	if !v.HasModePassword() {
		t.Fatalf("expected HasModePassword() to be true")
	}

	ok, err := v.VerifyModePassword("supersecure123")
	if err != nil {
		t.Fatalf("VerifyModePassword() error = %v", err)
	}

	if !ok {
		t.Fatalf("expected password verification to succeed")
	}
}

func TestVerifyModePasswordFailsWithWrongPassword(t *testing.T) {
	root := t.TempDir()
	v := NewVault(root)

	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := v.SetModePassword("supersecure123"); err != nil {
		t.Fatalf("SetModePassword() error = %v", err)
	}

	ok, err := v.VerifyModePassword("wrongpass")
	if err != nil {
		t.Fatalf("VerifyModePassword() error = %v", err)
	}

	if ok {
		t.Fatalf("expected password verification to fail")
	}
}

func newVaultWithPassword(t *testing.T, password string) *Vault {
	t.Helper()
	v := NewVault(t.TempDir())
	if err := v.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if password != "" {
		if err := v.SetModePassword(password); err != nil {
			t.Fatalf("SetModePassword() error = %v", err)
		}
	}
	return v
}

func readAuthFile(t *testing.T, v *Vault) ModeAuthConfig {
	t.Helper()
	data, err := os.ReadFile(v.modeAuthPath())
	if err != nil {
		t.Fatal(err)
	}
	var cfg ModeAuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestNewPasswordIsStoredAsArgon2id(t *testing.T) {
	v := newVaultWithPassword(t, "supersecure123")
	cfg := readAuthFile(t, v)

	if !strings.HasPrefix(cfg.Hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("expected an argon2id PHC string with the OWASP parameters, got %q", cfg.Hash)
	}
	if cfg.Salt != "" {
		t.Errorf("the salt lives inside the PHC string, got separate salt %q", cfg.Salt)
	}
	if strings.Contains(cfg.Hash, "supersecure123") {
		t.Error("the verifier must not contain the password")
	}

	info, err := os.Stat(v.modeAuthPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode_auth.json should be 0600, got %v", info.Mode().Perm())
	}
}

func TestSamePasswordGivesDifferentVerifiers(t *testing.T) {
	a := readAuthFile(t, newVaultWithPassword(t, "supersecure123"))
	b := readAuthFile(t, newVaultWithPassword(t, "supersecure123"))
	if a.Hash == b.Hash {
		t.Error("each verifier needs its own random salt")
	}
}

func TestVerifyIgnoresSurroundingWhitespace(t *testing.T) {
	v := newVaultWithPassword(t, "  supersecure123 ")
	ok, err := v.VerifyModePassword("supersecure123\n")
	if err != nil || !ok {
		t.Fatalf("expected success, got ok=%v err=%v", ok, err)
	}
}

func TestSetModePasswordRefusesToOverwrite(t *testing.T) {
	v := newVaultWithPassword(t, "supersecure123")

	err := v.SetModePassword("attacker-password")
	if !errors.Is(err, ErrModePasswordConfigured) {
		t.Fatalf("expected ErrModePasswordConfigured, got %v", err)
	}

	if ok, _ := v.VerifyModePassword("supersecure123"); !ok {
		t.Error("the original password must still work")
	}
	if ok, _ := v.VerifyModePassword("attacker-password"); ok {
		t.Error("the refused password must not work")
	}
}

func TestRotateModePassword(t *testing.T) {
	v := newVaultWithPassword(t, "supersecure123")

	if err := v.RotateModePassword("wrong-current", "another-secret-1"); err == nil {
		t.Fatal("rotation with a wrong current password must fail")
	}
	if ok, _ := v.VerifyModePassword("supersecure123"); !ok {
		t.Fatal("a failed rotation must keep the old password")
	}

	if err := v.RotateModePassword("supersecure123", "short"); err == nil {
		t.Fatal("a too-short new password must be rejected")
	}
	if ok, _ := v.VerifyModePassword("supersecure123"); !ok {
		t.Fatal("a rejected rotation must keep the old password")
	}

	if err := v.RotateModePassword("supersecure123", "another-secret-1"); err != nil {
		t.Fatalf("rotation failed: %v", err)
	}
	if ok, _ := v.VerifyModePassword("another-secret-1"); !ok {
		t.Error("new password should work")
	}
	if ok, _ := v.VerifyModePassword("supersecure123"); ok {
		t.Error("old password should stop working")
	}
}

func TestRotateWithoutPasswordConfigured(t *testing.T) {
	v := newVaultWithPassword(t, "")
	if err := v.RotateModePassword("anything-long", "another-secret-1"); err == nil {
		t.Fatal("there is nothing to rotate")
	}
}

func TestPasswordEventsAreLoggedWithoutSecrets(t *testing.T) {
	v := newVaultWithPassword(t, "supersecure123")
	_ = v.RotateModePassword("wrong-current", "another-secret-1")
	if err := v.RotateModePassword("supersecure123", "another-secret-1"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(v.RootPath, VaultDirName, IntegrityLogName))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	for _, want := range []string{"password_configured", "password_rotation_denied", "password_rotated"} {
		if !strings.Contains(log, want) {
			t.Errorf("log should contain %s:\n%s", want, log)
		}
	}
	for _, secret := range []string{"supersecure123", "another-secret-1", "wrong-current", "argon2id"} {
		if strings.Contains(log, secret) {
			t.Errorf("log must not contain %q:\n%s", secret, log)
		}
	}
}

func writeLegacyAuth(t *testing.T, v *Vault, password string) {
	t.Helper()
	salt := "3e1e3e9dba17cdd0da809e4e1ce828fb"
	if err := v.saveModeAuthConfig(&ModeAuthConfig{Salt: salt, Hash: hashModePassword(password, salt)}); err != nil {
		t.Fatal(err)
	}
}

func TestLegacySHA256VerifierIsUpgradedOnSuccess(t *testing.T) {
	v := newVaultWithPassword(t, "")
	writeLegacyAuth(t, v, "supersecure123")

	if ok, err := v.VerifyModePassword("wrong-password"); err != nil || ok {
		t.Fatalf("wrong password: ok=%v err=%v", ok, err)
	}
	if cfg := readAuthFile(t, v); isArgon2idHash(cfg.Hash) {
		t.Fatal("a failed verification must not upgrade the verifier")
	}

	if ok, err := v.VerifyModePassword("supersecure123"); err != nil || !ok {
		t.Fatalf("legacy password should verify: ok=%v err=%v", ok, err)
	}

	cfg := readAuthFile(t, v)
	if !isArgon2idHash(cfg.Hash) || cfg.Salt != "" {
		t.Fatalf("verifier should now be argon2id, got %+v", cfg)
	}
	if ok, _ := v.VerifyModePassword("supersecure123"); !ok {
		t.Error("the password must keep working after the upgrade")
	}
}

func TestLegacyVerifierCanBeRotated(t *testing.T) {
	v := newVaultWithPassword(t, "")
	writeLegacyAuth(t, v, "supersecure123")

	if err := v.RotateModePassword("supersecure123", "another-secret-1"); err != nil {
		t.Fatalf("rotation of a legacy verifier failed: %v", err)
	}
	if ok, _ := v.VerifyModePassword("another-secret-1"); !ok {
		t.Error("new password should work")
	}
}

func TestMalformedOrExtremeArgon2idVerifiersAreRejected(t *testing.T) {
	v := newVaultWithPassword(t, "")

	cases := map[string]string{
		"not a PHC string":    "$argon2id$garbage",
		"memory too large":    "$argon2id$v=19$m=4194304,t=2,p=1$c2FsdHNhbHQ$aGFzaGhhc2g",
		"zero iterations":     "$argon2id$v=19$m=19456,t=0,p=1$c2FsdHNhbHQ$aGFzaGhhc2g",
		"too many iterations": "$argon2id$v=19$m=19456,t=99,p=1$c2FsdHNhbHQ$aGFzaGhhc2g",
		"wrong version":       "$argon2id$v=16$m=19456,t=2,p=1$c2FsdHNhbHQ$aGFzaGhhc2g",
		"bad salt encoding":   "$argon2id$v=19$m=19456,t=2,p=1$***$aGFzaGhhc2g",
	}
	for name, hash := range cases {
		if err := v.saveModeAuthConfig(&ModeAuthConfig{Hash: hash}); err != nil {
			t.Fatal(err)
		}
		ok, err := v.VerifyModePassword("supersecure123")
		if ok || err == nil {
			t.Errorf("%s: expected an error and no match, got ok=%v err=%v", name, ok, err)
		}
	}
}

func TestEmptyOrIncompleteAuthFileIsInvalid(t *testing.T) {
	v := newVaultWithPassword(t, "")
	for _, cfg := range []ModeAuthConfig{{}, {Hash: "abc"}, {Salt: "abc"}} {
		if err := v.saveModeAuthConfig(&cfg); err != nil {
			t.Fatal(err)
		}
		if _, err := v.VerifyModePassword("supersecure123"); err == nil {
			t.Errorf("config %+v should be invalid", cfg)
		}
	}
}
