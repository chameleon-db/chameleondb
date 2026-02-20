package vault

import "testing"

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
