package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5"
)

// pingPostgres opens and closes one connection, to tell the user early whether
// a connection string works.
func pingPostgres(rawURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, rawURL)
	if err != nil {
		return err
	}

	return conn.Close(ctx)
}

// runDockerComposeUp starts the services of the compose file in dir.
func runDockerComposeUp(dir string) error {
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// waitForPostgres retries pingPostgres once a second.
func waitForPostgres(rawURL string, attempts int) error {
	for i := 0; i < attempts; i++ {
		if pingPostgres(rawURL) == nil {
			return nil
		}
		time.Sleep(time.Second)
	}

	return errors.New("PostgreSQL did not accept connections in time")
}
