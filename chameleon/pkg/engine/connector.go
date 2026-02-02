package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectorConfig holds PostgreSQL connection settings
type ConnectorConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	// Pool settings
	MaxConns    int32
	MinConns    int32
	MaxIdleTime time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() ConnectorConfig {
	return ConnectorConfig{
		Host:        "localhost",
		Port:        5432,
		Database:    "chameleon",
		User:        "postgres",
		Password:    "",
		MaxConns:    10,
		MinConns:    2,
		MaxIdleTime: 5 * time.Minute,
	}
}

// ConnectionString builds the pgx connection string
func (c ConnectorConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		c.Host, c.Port, c.Database, c.User, c.Password,
	)
}

// Connector manages the PostgreSQL connection pool
type Connector struct {
	pool   *pgxpool.Pool
	config ConnectorConfig
}

// NewConnector creates a new connector (does not connect yet)
func NewConnector(config ConnectorConfig) *Connector {
	return &Connector{config: config}
}

// Connect establishes the connection pool
func (c *Connector) Connect(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(c.config.ConnectionString())
	if err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
	}

	poolConfig.MaxConns = c.config.MaxConns
	poolConfig.MinConns = c.config.MinConns
	poolConfig.MaxConnIdleTime = c.config.MaxIdleTime

	pool, err := pgxpool.New(ctx, poolConfig.ConnString())
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	c.pool = pool
	return nil
}

// Pool returns the underlying connection pool
// Returns nil if not connected
func (c *Connector) Pool() *pgxpool.Pool {
	return c.pool
}

// IsConnected returns true if the pool is active
func (c *Connector) IsConnected() bool {
	return c.pool != nil
}

// Ping verifies the connection is alive
func (c *Connector) Ping(ctx context.Context) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	return c.pool.Ping(ctx)
}

// Close closes the connection pool
func (c *Connector) Close() {
	if c.pool != nil {
		c.pool.Close()
		c.pool = nil
	}
}
