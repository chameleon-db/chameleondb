package main

import (
	"os"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

func TestSetPasswordRotationRequiresTheCurrentPassword(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	v := vault.NewVault(dir)
	if err := v.Initialize(); err != nil {
		t.Fatal(err)
	}

	// First password: only the new one is needed.
	t.Setenv(modePasswordEnvVar, "first-password")
	t.Setenv(modePasswordNewEnvVar, "")
	if err := configAuthSetPasswordCmd.RunE(nil, nil); err != nil {
		t.Fatalf("first password: %v", err)
	}

	// Running the command again with a different password must not replace it.
	t.Setenv(modePasswordEnvVar, "attacker-password")
	t.Setenv(modePasswordNewEnvVar, "attacker-password-2")
	if err := configAuthSetPasswordCmd.RunE(nil, nil); err == nil {
		t.Fatal("rotation with a wrong current password must fail")
	}
	if ok, _ := v.VerifyModePassword("first-password"); !ok {
		t.Fatal("the original password must still work")
	}

	// Rotation with the current password.
	t.Setenv(modePasswordEnvVar, "first-password")
	t.Setenv(modePasswordNewEnvVar, "second-password")
	if err := configAuthSetPasswordCmd.RunE(nil, nil); err != nil {
		t.Fatalf("rotation with the current password: %v", err)
	}
	if ok, _ := v.VerifyModePassword("second-password"); !ok {
		t.Fatal("the new password should work")
	}
	if ok, _ := v.VerifyModePassword("first-password"); ok {
		t.Fatal("the old password should stop working")
	}
}
