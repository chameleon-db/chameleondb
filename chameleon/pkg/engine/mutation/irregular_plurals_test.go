package mutation

import "testing"

func TestSingularizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "regular", in: "Users", want: "User"},
		{name: "irregular", in: "People", want: "Person"},
		{name: "irregular lowercase", in: "analyses", want: "analysis"},
		{name: "unchanged", in: "Profile", want: "Profile"},
		{name: "invariant irregular", in: "Fish", want: "Fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SingularizeName(tt.in)
			if got != tt.want {
				t.Fatalf("SingularizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
