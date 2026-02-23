package main

import (
	"testing"
	"time"
)

func TestFormatTimeSince(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		t        time.Time
		expected string
	}{
		{
			name:     "just now (30 seconds ago)",
			t:        now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			t:        now.Add(-1 * time.Minute),
			expected: "1 minutes ago",
		},
		{
			name:     "5 minutes ago",
			t:        now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "59 minutes ago",
			t:        now.Add(-59 * time.Minute),
			expected: "59 minutes ago",
		},
		{
			name:     "1 hour ago",
			t:        now.Add(-1 * time.Hour),
			expected: "1 hours ago",
		},
		{
			name:     "3 hours ago",
			t:        now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "23 hours ago",
			t:        now.Add(-23 * time.Hour),
			expected: "23 hours ago",
		},
		{
			name:     "1 day ago",
			t:        now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			t:        now.Add(-72 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "30 days ago",
			t:        now.Add(-720 * time.Hour),
			expected: "30 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeSince(tt.t)
			if got != tt.expected {
				t.Errorf("formatTimeSince(%v) = %q, want %q", tt.t, got, tt.expected)
			}
		})
	}
}

func TestGetModeIcon(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{name: "readonly mode", mode: "readonly", expected: "🛡️"},
		{name: "standard mode", mode: "standard", expected: "⚙️"},
		{name: "privileged mode", mode: "privileged", expected: "👑"},
		{name: "emergency mode", mode: "emergency", expected: "🚨"},
		{name: "unknown mode", mode: "unknown", expected: "❓"},
		{name: "empty mode", mode: "", expected: "❓"},
		{name: "invalid mode", mode: "invalid", expected: "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getModeIcon(tt.mode)
			if got != tt.expected {
				t.Errorf("getModeIcon(%q) = %q, want %q", tt.mode, got, tt.expected)
			}
		})
	}
}
