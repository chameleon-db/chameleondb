package initwizard

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// ComposeFileName is the file the wizard writes.
const ComposeFileName = "docker-compose.yml"

// DefaultComposeURL is the connection string for the service in composeFile.
const DefaultComposeURL = "postgresql://chameleon:chameleon@localhost:5432/chameleon?sslmode=disable"

// ComposeContent is a PostgreSQL 16 service for local development. It is the
// same service as in the 03-todo-app example.
const ComposeContent = `# PostgreSQL for local development. Start it with: docker compose up -d
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: chameleon
      POSTGRES_PASSWORD: chameleon
      POSTGRES_DB: chameleon
    ports:
      - "5432:5432"
    volumes:
      - chameleon-db-data:/var/lib/postgresql/data

volumes:
  chameleon-db-data:
`

// WriteComposeFile writes docker-compose.yml in dir. It never overwrites: if
// the file (or compose.yml) exists, it returns created=false.
func WriteComposeFile(dir string) (created bool, err error) {
	for _, name := range []string{ComposeFileName, "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if _, statErr := os.Stat(filepath.Join(dir, name)); statErr == nil {
			return false, nil
		}
	}

	if err := os.WriteFile(filepath.Join(dir, ComposeFileName), []byte(ComposeContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", ComposeFileName, err)
	}
	return true, nil
}

// EnvExample returns the .env.example content for rawURL, with the password
// replaced by a placeholder. A real password never reaches a file.
func EnvExample(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return "DATABASE_URL=" + rawURL + "\n"
	}

	if _, hasPassword := u.User.Password(); hasPassword {
		u.User = url.UserPassword(u.User.Username(), "change-me")
	}
	return "DATABASE_URL=" + u.String() + "\n"
}

// WriteEnvExample writes .env.example in dir unless it exists.
func WriteEnvExample(dir, rawURL string) (created bool, err error) {
	path := filepath.Join(dir, ".env.example")
	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(EnvExample(rawURL)), 0644); err != nil {
		return false, fmt.Errorf("failed to write .env.example: %w", err)
	}
	return true, nil
}
