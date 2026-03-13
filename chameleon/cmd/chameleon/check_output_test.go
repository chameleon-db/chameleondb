package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	oldColorOut := color.Output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	color.Output = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	color.Output = oldColorOut
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout error: %v", err)
	}
	return string(out)
}

func TestPrintJSONSuccess(t *testing.T) {
	out := captureStdout(t, func() {
		printJSONSuccess()
	})

	var result CheckResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal output: %v\nout=%s", err, out)
	}
	if !result.Valid || len(result.Errors) != 0 {
		t.Fatalf("unexpected success result: %+v", result)
	}
}

func TestPrintJSONError(t *testing.T) {
	out := captureStdout(t, func() {
		_ = printJSONError("schema.cham", "broken schema")
	})

	var result CheckResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal output: %v\nout=%s", err, out)
	}
	if result.Valid {
		t.Fatal("expected valid=false")
	}
	if len(result.Errors) != 1 || result.Errors[0].Message != "broken schema" {
		t.Fatalf("unexpected error payload: %+v", result)
	}
}

func TestPrintCheckErrorsJSONAndFallback(t *testing.T) {
	jsonErr := `{"valid":false,"errors":[{"kind":"ParseError","message":"bad token","line":3,"column":7,"snippet":"x","suggestion":"fix it"}]}`
	outJSON := captureStdout(t, func() {
		_ = printCheckErrors("schema.cham", jsonErr)
	})

	var parsed CheckResult
	if err := json.Unmarshal([]byte(outJSON), &parsed); err != nil {
		t.Fatalf("unmarshal json output: %v\nout=%s", err, outJSON)
	}
	if parsed.Valid || len(parsed.Errors) != 1 {
		t.Fatalf("unexpected parsed result: %+v", parsed)
	}
	if parsed.Errors[0].Line != 3 || parsed.Errors[0].Column != 7 {
		t.Fatalf("expected line/column from payload, got %+v", parsed.Errors[0])
	}

	outFallback := captureStdout(t, func() {
		_ = printCheckErrors("schema.cham", "raw plain error")
	})
	if !strings.Contains(outFallback, "raw plain error") {
		t.Fatalf("expected fallback message in output, got: %s", outFallback)
	}
}
