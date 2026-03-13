package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPlaygroundInputFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.json")
	content := `{"schema":"entity User { id: uuid primary, }","query":{"entity":"User"}}`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	data, err := readPlaygroundInput(file)
	if err != nil {
		t.Fatalf("readPlaygroundInput(file) error = %v", err)
	}
	if string(data) != content {
		t.Fatalf("unexpected file content: %s", string(data))
	}
}

func TestReadPlaygroundInputFromStdinPipe(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	payload := `{"schema":"entity User { id: uuid primary, }","query":{"entity":"User"}}`
	_, _ = w.WriteString(payload)
	_ = w.Close()
	os.Stdin = r

	data, err := readPlaygroundInput("")
	if err != nil {
		t.Fatalf("readPlaygroundInput(stdin) error = %v", err)
	}
	if string(data) != payload {
		t.Fatalf("unexpected stdin content: %s", string(data))
	}
}

func TestWritePlaygroundResponse(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w

	resp := playgroundSQLResponse{OK: true, MainQuery: "SELECT 1"}
	if err := writePlaygroundResponse(resp); err != nil {
		t.Fatalf("writePlaygroundResponse() error = %v", err)
	}

	_ = w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)

	var parsed playgroundSQLResponse
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal response output: %v\nout=%s", err, string(out))
	}
	if !parsed.OK || parsed.MainQuery != "SELECT 1" {
		t.Fatalf("unexpected parsed output: %+v", parsed)
	}
}

func TestWritePlaygroundHTTPResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writePlaygroundHTTPResponse(rec, http.StatusAccepted, playgroundSQLResponse{OK: true})

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected application/json content type, got %q", rec.Header().Get("Content-Type"))
	}
	if !strings.HasSuffix(rec.Body.String(), "\n") {
		t.Fatalf("expected response body newline suffix, got %q", rec.Body.String())
	}
}
