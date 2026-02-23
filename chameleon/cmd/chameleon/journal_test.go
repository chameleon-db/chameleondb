package main

import "testing"

func TestStatusString(t *testing.T) {
	tests := []struct {
		name     string
		locked   bool
		expected string
	}{
		{name: "locked", locked: true, expected: "locked ✓"},
		{name: "unlocked", locked: false, expected: "unlocked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusString(tt.locked)
			if got != tt.expected {
				t.Errorf("statusString(%v) = %q, want %q", tt.locked, got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{
			name:     "short string (no truncation)",
			s:        "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			s:        "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate long string",
			s:        "this is a very long error message that needs truncation",
			maxLen:   20,
			expected: "this is a very lo...",
		},
		{
			name:     "truncate at 10 chars",
			s:        "0123456789abcdefghij",
			maxLen:   10,
			expected: "0123456...",
		},
		{
			name:     "empty string",
			s:        "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "very short maxLen",
			s:        "hello world",
			maxLen:   5,
			expected: "he...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.expected)
			}
		})
	}
}
