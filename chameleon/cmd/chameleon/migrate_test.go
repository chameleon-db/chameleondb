package main

import (
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/internal/schema"
)

func TestTryMapErrorToSource(t *testing.T) {
	lineMap := map[int]schema.SourceLine{
		5:  {File: "schema.cham", LineNumber: 10},
		10: {File: "entities/user.cham", LineNumber: 5},
		15: {File: "entities/post.cham", LineNumber: 20},
		25: {File: "relations.cham", LineNumber: 3},
	}

	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "error with 'line N' pattern",
			errMsg:   "syntax error at line 10",
			expected: "Error in entities/user.cham:5",
		},
		{
			name:     "error with '--> file:N:col' pattern",
			errMsg:   "--> schema.cham:15:8 unexpected token",
			expected: "Error in entities/post.cham:20",
		},
		{
			name:     "error with ' N │' pattern",
			errMsg:   "  25 │ invalid syntax here",
			expected: "Error in relations.cham:3",
		},
		{
			name:     "error with nearby line (offset +1)",
			errMsg:   "line 11",
			expected: "Error in entities/user.cham:6",
		},
		{
			name:     "error with nearby line (offset -1)",
			errMsg:   "line 9",
			expected: "Error in entities/user.cham:4",
		},
		{
			name:     "error with no matching line",
			errMsg:   "line 100",
			expected: "",
		},
		{
			name:     "error with no pattern match",
			errMsg:   "random error message",
			expected: "",
		},
		{
			name:     "error at line 5",
			errMsg:   "error at line 5",
			expected: "Error in schema.cham:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tryMapErrorToSource(tt.errMsg, lineMap)
			if got != tt.expected {
				t.Errorf("tryMapErrorToSource(%q, lineMap) = %q, want %q", tt.errMsg, got, tt.expected)
			}
		})
	}
}

func TestTryMapErrorToSourceEmptyMap(t *testing.T) {
	emptyMap := map[int]schema.SourceLine{}
	got := tryMapErrorToSource("line 10", emptyMap)
	if got != "" {
		t.Errorf("tryMapErrorToSource with empty map should return empty string, got %q", got)
	}
}
