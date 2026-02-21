package main

import "testing"

func TestRequiresModeAuth(t *testing.T) {
	tests := []struct {
		name        string
		currentMode string
		targetMode  string
		want        bool
	}{
		{name: "readonly to standard requires auth", currentMode: "readonly", targetMode: "standard", want: true},
		{name: "standard to privileged requires auth", currentMode: "standard", targetMode: "privileged", want: true},
		{name: "privileged to emergency requires auth", currentMode: "privileged", targetMode: "emergency", want: true},
		{name: "standard to readonly does not require auth", currentMode: "standard", targetMode: "readonly", want: false},
		{name: "emergency to privileged does not require auth", currentMode: "emergency", targetMode: "privileged", want: false},
		{name: "same mode does not require auth", currentMode: "standard", targetMode: "standard", want: false},
		{name: "admin alias is treated as privileged", currentMode: "standard", targetMode: "admin", want: true},
		{name: "invalid target mode does not trigger auth check", currentMode: "standard", targetMode: "invalid", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresModeAuth(tt.currentMode, tt.targetMode)
			if got != tt.want {
				t.Fatalf("requiresModeAuth(%q, %q) = %v, want %v", tt.currentMode, tt.targetMode, got, tt.want)
			}
		})
	}
}
