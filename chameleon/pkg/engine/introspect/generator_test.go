package introspect

import "testing"

func TestToEntityName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		want      string
	}{
		{name: "regular plural", tableName: "users", want: "User"},
		{name: "snake case plural", tableName: "user_posts", want: "UserPost"},
		{name: "irregular plural", tableName: "people", want: "Person"},
		{name: "irregular plural in last segment", tableName: "user_people", want: "UserPerson"},
		{name: "irregular ending with es", tableName: "statuses", want: "Status"},
		{name: "already singular", tableName: "profile", want: "Profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toEntityName(tt.tableName)
			if got != tt.want {
				t.Fatalf("toEntityName(%q) = %q, want %q", tt.tableName, got, tt.want)
			}
		})
	}
}
