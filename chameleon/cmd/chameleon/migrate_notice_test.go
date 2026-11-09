package main

import (
	"strings"
	"testing"
)

func TestReadonlyNotice_Readonly(t *testing.T) {
	out := readonlyNotice("readonly")

	for _, want := range []string{
		"Paranoid mode: readonly",
		"another 'migrate --apply' will be refused",
		"chameleon config auth set-password",
		"chameleon config set mode=standard",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("notice should contain %q, got:\n%s", want, out)
		}
	}

	if strings.Index(out, "set-password") > strings.Index(out, "mode=standard") {
		t.Error("the password step must come before the mode change")
	}
}

func TestReadonlyNotice_OtherModes(t *testing.T) {
	for _, mode := range []string{"standard", "", "unknown"} {
		if out := readonlyNotice(mode); out != "" {
			t.Errorf("mode %q should print nothing, got:\n%s", mode, out)
		}
	}
}
