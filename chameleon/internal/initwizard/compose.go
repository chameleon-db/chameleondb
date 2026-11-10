package initwizard

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

// EnvBlockMarker opens the block that `init` adds to .env.example.
const EnvBlockMarker = "# chameleon env"

// EnvExample returns the block for rawURL, with the password replaced by a
// placeholder. A real password never reaches a file.
func EnvExample(rawURL string) string {
	line := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), "change-me")
		}
		line = u.String()
	}

	return EnvBlockMarker + "\nDATABASE_URL=" + line + "\n"
}

// EnvResult says what WriteEnvExample did.
type EnvResult int

const (
	// EnvCreated: .env.example did not exist and was created.
	EnvCreated EnvResult = iota
	// EnvAppended: the block was added at the end of an existing file.
	EnvAppended
	// EnvUnchanged: the file already has the block or a DATABASE_URL.
	EnvUnchanged
)

// WriteEnvExample creates .env.example with the block, or appends the block to
// an existing file when that file has neither the marker nor a DATABASE_URL.
// Existing content is never changed.
func WriteEnvExample(dir, rawURL string) (EnvResult, error) {
	path := filepath.Join(dir, ".env.example")
	block := EnvExample(rawURL)

	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(block), 0644); err != nil {
			return EnvUnchanged, fmt.Errorf("failed to write .env.example: %w", err)
		}
		return EnvCreated, nil
	}
	if err != nil {
		return EnvUnchanged, fmt.Errorf("failed to read .env.example: %w", err)
	}

	content := string(existing)
	if strings.Contains(content, EnvBlockMarker) {
		return EnvUnchanged, nil
	}
	if _, has := ParseDotenv(content)["DATABASE_URL"]; has {
		return EnvUnchanged, nil
	}

	prefix := ""
	switch {
	case content == "":
	case strings.HasSuffix(content, "\n\n"):
	case strings.HasSuffix(content, "\n"):
		prefix = "\n"
	default:
		prefix = "\n\n"
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return EnvUnchanged, fmt.Errorf("failed to open .env.example: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(prefix + block); err != nil {
		return EnvUnchanged, fmt.Errorf("failed to append to .env.example: %w", err)
	}
	return EnvAppended, nil
}
