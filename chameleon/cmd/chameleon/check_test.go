package main

import (
	"encoding/json"
	"testing"
)

func TestCheckErrorStructure(t *testing.T) {
	// Test that CheckError struct can be marshaled/unmarshaled correctly
	snippet := "entity User {"
	suggestion := "Add closing brace"

	err := CheckError{
		Message:    "Syntax error",
		Line:       10,
		Column:     5,
		File:       "schema.cham",
		Severity:   "error",
		Snippet:    &snippet,
		Suggestion: &suggestion,
	}

	// Marshal to JSON
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal CheckError: %v", marshalErr)
	}

	// Unmarshal back
	var unmarshaled CheckError
	if unmarshalErr := json.Unmarshal(data, &unmarshaled); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal CheckError: %v", unmarshalErr)
	}

	// Verify fields
	if unmarshaled.Message != err.Message {
		t.Errorf("Message mismatch: got %q, want %q", unmarshaled.Message, err.Message)
	}
	if unmarshaled.Line != err.Line {
		t.Errorf("Line mismatch: got %d, want %d", unmarshaled.Line, err.Line)
	}
	if unmarshaled.Column != err.Column {
		t.Errorf("Column mismatch: got %d, want %d", unmarshaled.Column, err.Column)
	}
	if unmarshaled.File != err.File {
		t.Errorf("File mismatch: got %q, want %q", unmarshaled.File, err.File)
	}
	if unmarshaled.Severity != err.Severity {
		t.Errorf("Severity mismatch: got %q, want %q", unmarshaled.Severity, err.Severity)
	}
	if unmarshaled.Snippet == nil || *unmarshaled.Snippet != *err.Snippet {
		t.Errorf("Snippet mismatch")
	}
	if unmarshaled.Suggestion == nil || *unmarshaled.Suggestion != *err.Suggestion {
		t.Errorf("Suggestion mismatch")
	}
}

func TestCheckResultStructure(t *testing.T) {
	// Test CheckResult with valid schema
	validResult := CheckResult{
		Valid:  true,
		Errors: []CheckError{},
	}

	data, err := json.Marshal(validResult)
	if err != nil {
		t.Fatalf("Failed to marshal valid CheckResult: %v", err)
	}

	var unmarshaled CheckResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal valid CheckResult: %v", err)
	}

	if !unmarshaled.Valid {
		t.Error("Expected Valid to be true")
	}
	if len(unmarshaled.Errors) != 0 {
		t.Errorf("Expected empty errors array, got %d errors", len(unmarshaled.Errors))
	}
}

func TestCheckResultWithErrors(t *testing.T) {
	// Test CheckResult with errors
	result := CheckResult{
		Valid: false,
		Errors: []CheckError{
			{
				Message:  "Missing primary key",
				Line:     5,
				Column:   1,
				File:     "schema.cham",
				Severity: "error",
			},
			{
				Message:  "Unknown type",
				Line:     10,
				Column:   15,
				File:     "schema.cham",
				Severity: "error",
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CheckResult with errors: %v", err)
	}

	var unmarshaled CheckResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CheckResult with errors: %v", err)
	}

	if unmarshaled.Valid {
		t.Error("Expected Valid to be false")
	}
	if len(unmarshaled.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(unmarshaled.Errors))
	}

	// Verify first error
	if unmarshaled.Errors[0].Message != "Missing primary key" {
		t.Errorf("First error message mismatch: got %q", unmarshaled.Errors[0].Message)
	}
	if unmarshaled.Errors[0].Line != 5 {
		t.Errorf("First error line mismatch: got %d", unmarshaled.Errors[0].Line)
	}

	// Verify second error
	if unmarshaled.Errors[1].Message != "Unknown type" {
		t.Errorf("Second error message mismatch: got %q", unmarshaled.Errors[1].Message)
	}
	if unmarshaled.Errors[1].Line != 10 {
		t.Errorf("Second error line mismatch: got %d", unmarshaled.Errors[1].Line)
	}
}

func TestCheckErrorOmitEmptyFields(t *testing.T) {
	// Test that nil snippet/suggestion are omitted from JSON
	err := CheckError{
		Message:  "Error",
		Line:     1,
		Column:   1,
		File:     "test.cham",
		Severity: "error",
	}

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal CheckError: %v", marshalErr)
	}

	// Check that snippet and suggestion are not in JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, hasSnippet := raw["snippet"]; hasSnippet {
		t.Error("Expected snippet to be omitted from JSON")
	}
	if _, hasSuggestion := raw["suggestion"]; hasSuggestion {
		t.Error("Expected suggestion to be omitted from JSON")
	}
}
