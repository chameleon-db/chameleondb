package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// LoadConnectorConfig loads config from:
// 1. DATABASE_URL environment variable (priority)
// 2. .chameleon file in current directory
func LoadConnectorConfig() (engine.ConnectorConfig, error) {
	// 1. Try DATABASE_URL env var (Heroku, Railway, etc.)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		config, err := engine.ParseConnectionString(databaseURL)
		if err != nil {
			return engine.ConnectorConfig{}, fmt.Errorf("invalid DATABASE_URL: %w", err)
		}
		if verbose {
			printInfo("Using DATABASE_URL from environment")
		}
		return config, nil
	}

	// 2. Try .chameleon file
	configPath := ".chameleon"
	if _, err := os.Stat(configPath); err == nil {
		fileConfig := struct {
			Database struct {
				Host     string `toml:"host"`
				Port     int    `toml:"port"`
				Database string `toml:"database"`
				User     string `toml:"user"`
				Password string `toml:"password"`
				MaxConns int32  `toml:"max_conns"`
				MinConns int32  `toml:"min_conns"`
			} `toml:"database"`
		}{}

		if _, err := toml.DecodeFile(configPath, &fileConfig); err != nil {
			return engine.ConnectorConfig{}, fmt.Errorf("failed to parse .chameleon: %w", err)
		}

		config := engine.DefaultConfig()
		if fileConfig.Database.Host != "" {
			config.Host = fileConfig.Database.Host
		}
		if fileConfig.Database.Port != 0 {
			config.Port = fileConfig.Database.Port
		}
		if fileConfig.Database.Database != "" {
			config.Database = fileConfig.Database.Database
		}
		if fileConfig.Database.User != "" {
			config.User = fileConfig.Database.User
		}
		if fileConfig.Database.Password != "" {
			config.Password = fileConfig.Database.Password
		}
		if fileConfig.Database.MaxConns != 0 {
			config.MaxConns = fileConfig.Database.MaxConns
		}
		if fileConfig.Database.MinConns != 0 {
			config.MinConns = fileConfig.Database.MinConns
		}

		if verbose {
			printInfo("Using .chameleon configuration file")
		}
		return config, nil
	}

	// 3. Return defaults
	if verbose {
		printInfo("Using default configuration (localhost:5432)")
	}
	return engine.DefaultConfig(), nil
}
