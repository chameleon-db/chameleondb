package main

import (
	"os"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

func TestLoadConnectorConfigFromDATABASE_URL(t *testing.T) {
	// Set DATABASE_URL
	os.Setenv("DATABASE_URL", "postgresql://user:pass@example.com:5433/mydb")
	defer os.Unsetenv("DATABASE_URL")

	config, err := LoadConnectorConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.Host != "example.com" {
		t.Errorf("expected host=example.com, got %s", config.Host)
	}
	if config.Port != 5433 {
		t.Errorf("expected port=5433, got %d", config.Port)
	}
	if config.Database != "mydb" {
		t.Errorf("expected database=mydb, got %s", config.Database)
	}
	if config.User != "user" {
		t.Errorf("expected user=user, got %s", config.User)
	}
	if config.Password != "pass" {
		t.Errorf("expected password=pass, got %s", config.Password)
	}
}

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
		want    engine.ConnectorConfig
		wantErr bool
	}{
		{
			name:    "valid postgresql URL",
			connStr: "postgresql://user:pass@localhost:5432/testdb",
			want: engine.ConnectorConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name:    "postgres scheme (alias)",
			connStr: "postgres://user:pass@localhost:5432/testdb",
			want: engine.ConnectorConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name:    "without password",
			connStr: "postgresql://user@localhost:5432/testdb",
			want: engine.ConnectorConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ParseConnectionString(tt.connStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConnectionString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Host != tt.want.Host || got.Port != tt.want.Port ||
				got.Database != tt.want.Database || got.User != tt.want.User ||
				got.Password != tt.want.Password {
				t.Errorf("ParseConnectionString() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
