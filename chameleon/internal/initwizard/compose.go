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

// AgentsBlockMarker opens the block that `init --agents` adds to AGENTS.md.
const AgentsBlockMarker = "# chameleon agents"

// AgentsRules are the rules for coding agents. They are the body of the
// "For coding agents" section of CHAMELEON.md and of the AGENTS.md block, so
// both say the same. Every line has to be true for the installed version
// (ADR-0002); the tests check the commands and exit codes it names.
const AgentsRules = "- Order: `chameleon validate`, then `chameleon migrate --dry-run` to see the plan.\n" +
	"  **Ask a human before `chameleon migrate --apply`**: it changes the database.\n" +
	"- Use `--json` (`validate`, `status`, `verify`, `migrate`, `introspect`). Stdout is then\n" +
	"  one JSON document (`ok`, `data`, `errors[]`) and human text goes to stderr. It never asks\n" +
	"  questions; `--no-input` does the same without JSON.\n" +
	"- Exit codes: 0 ok, 2 usage, 3 invalid schema, 4 blocked by the mode, 5 integrity\n" +
	"  failure, 6 database unreachable, 7 not initialized, 8 migration rejected, 1 other.\n" +
	"  Read `errors[].code` and `hint` before retrying.\n" +
	"- Do not edit `.chameleon/` (the vault and `mode.json`). Changes made outside the\n" +
	"  CLI are detected and stop every command with exit 5.\n" +
	"- The readonly mode after the first apply is expected. It is not an error to work around.\n" +
	"  Changing the mode needs a password that only a human has.\n" +
	"- Never write a real `DATABASE_URL` or password in a file; use the environment.\n" +
	"- There are no backups and no rollback yet. Do not promise either.\n"

// AgentsBlock is the text added to AGENTS.md.
func AgentsBlock() string {
	return AgentsBlockMarker + "\n\nThis project uses ChameleonDB. Rules for coding agents:\n\n" + AgentsRules
}

// AgentsResult says what WriteAgentsBlock did.
type AgentsResult int

const (
	// AgentsCreated: AGENTS.md did not exist and was created.
	AgentsCreated AgentsResult = iota
	// AgentsAppended: the block was added at the end of an existing file.
	AgentsAppended
	// AgentsUnchanged: the file already has the block.
	AgentsUnchanged
)

// WriteAgentsBlock creates AGENTS.md with the block, or appends the block to an
// existing file that does not have it. Existing content is never changed.
func WriteAgentsBlock(dir string) (AgentsResult, error) {
	path := filepath.Join(dir, "AGENTS.md")
	block := AgentsBlock()

	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(block), 0644); err != nil {
			return AgentsUnchanged, fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
		return AgentsCreated, nil
	}
	if err != nil {
		return AgentsUnchanged, fmt.Errorf("failed to read AGENTS.md: %w", err)
	}

	content := string(existing)
	if strings.Contains(content, AgentsBlockMarker) {
		return AgentsUnchanged, nil
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
		return AgentsUnchanged, fmt.Errorf("failed to open AGENTS.md: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(prefix + block); err != nil {
		return AgentsUnchanged, fmt.Errorf("failed to append to AGENTS.md: %w", err)
	}
	return AgentsAppended, nil
}
