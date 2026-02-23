package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset verbose flag
	oldVerbose := verbose
	verbose = false
	defer func() { verbose = oldVerbose }()

	// Execute version command
	versionCmd.Run(&cobra.Command{}, []string{})

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains version
	if !strings.Contains(output, "ChameleonDB v") {
		t.Errorf("Expected output to contain 'ChameleonDB v', got: %s", output)
	}
}

func TestVersionCommandVerbose(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set verbose flag
	oldVerbose := verbose
	verbose = true
	defer func() { verbose = oldVerbose }()

	// Execute version command
	versionCmd.Run(&cobra.Command{}, []string{})

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains version and components
	if !strings.Contains(output, "ChameleonDB v") {
		t.Errorf("Expected output to contain 'ChameleonDB v', got: %s", output)
	}
	if !strings.Contains(output, "Components:") {
		t.Errorf("Expected verbose output to contain 'Components:', got: %s", output)
	}
	if !strings.Contains(output, "CLI:") {
		t.Errorf("Expected verbose output to contain 'CLI:', got: %s", output)
	}
	if !strings.Contains(output, "Core:") {
		t.Errorf("Expected verbose output to contain 'Core:', got: %s", output)
	}
}
