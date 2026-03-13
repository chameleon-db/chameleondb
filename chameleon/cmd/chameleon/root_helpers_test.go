package main

import (
	"os"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

func TestGetConfigFromEnvFallbackToDefaultOnInvalidDatabaseURL(t *testing.T) {
	old := os.Getenv("DATABASE_URL")
	defer func() {
		if old == "" {
			_ = os.Unsetenv("DATABASE_URL")
		} else {
			_ = os.Setenv("DATABASE_URL", old)
		}
	}()

	_ = os.Setenv("DATABASE_URL", "not-a-valid-connection-string")
	cfg := getConfigFromEnv()
	def := engine.DefaultConfig()

	if cfg.Host != def.Host || cfg.Port != def.Port || cfg.Database != def.Database {
		t.Fatalf("expected default config fallback, got %+v (default %+v)", cfg, def)
	}
}

func TestGetConfigFromEnvUsesDatabaseURL(t *testing.T) {
	old := os.Getenv("DATABASE_URL")
	defer func() {
		if old == "" {
			_ = os.Unsetenv("DATABASE_URL")
		} else {
			_ = os.Setenv("DATABASE_URL", old)
		}
	}()

	_ = os.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5432/mydb")
	cfg := getConfigFromEnv()

	if cfg.Host != "localhost" || cfg.Port != 5432 || cfg.Database != "mydb" || cfg.User != "u" {
		t.Fatalf("unexpected parsed config from DATABASE_URL: %+v", cfg)
	}
}
