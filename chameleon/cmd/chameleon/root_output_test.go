package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	oldStderr := os.Stderr
	oldColorErr := color.Error
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stderr = w
	color.Error = w

	fn()

	_ = w.Close()
	os.Stderr = oldStderr
	color.Error = oldColorErr
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr error: %v", err)
	}
	return string(out)
}

func TestRootPrintHelpers(t *testing.T) {
	oldColorOut := color.Output
	oldColorErr := color.Error
	defer func() {
		color.Output = oldColorOut
		color.Error = oldColorErr
	}()

	outSuccess := captureStdout(t, func() { printSuccess("ok %s", "done") })
	if !strings.Contains(outSuccess, "ok done") {
		t.Fatalf("expected success output, got %q", outSuccess)
	}

	outWarning := captureStdout(t, func() { printWarning("warn %d", 7) })
	if !strings.Contains(outWarning, "warn 7") {
		t.Fatalf("expected warning output, got %q", outWarning)
	}

	outInfo := captureStdout(t, func() { printInfo("info") })
	if !strings.Contains(outInfo, "info") {
		t.Fatalf("expected info output, got %q", outInfo)
	}

	outError := captureStderr(t, func() { printError("bad %s", "news") })
	if !strings.Contains(outError, "bad news") {
		t.Fatalf("expected error output, got %q", outError)
	}
}
