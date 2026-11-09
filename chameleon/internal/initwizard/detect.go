package initwizard

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Candidate is a PostgreSQL connection found in the repository or in the
// environment.
type Candidate struct {
	// Source says where it was found, for display.
	Source string
	// URL is the full connection string. It may contain a password: it is only
	// ever printed to the terminal, never written to a file (see EnvExample).
	URL string
}

// Redacted returns the URL with the password hidden, for display.
func (c Candidate) Redacted() string {
	u, err := url.Parse(c.URL)
	if err != nil {
		return "(unparsable)"
	}
	return u.Redacted()
}

// ValidatePostgresURL checks that raw is a postgres:// or postgresql:// URL
// with a host.
func ValidatePostgresURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("not a valid URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("scheme must be postgresql:// or postgres://, got %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("the URL has no host")
	}
	return nil
}

// Detect looks for PostgreSQL connections, in a fixed order: the DATABASE_URL
// environment variable, .env, .env.example, then postgres services in
// docker-compose*.yml / compose*.yml (sorted by file name). Duplicates are
// dropped. Nothing is written.
func Detect(dir string, getenv func(string) string) []Candidate {
	var found []Candidate
	seen := map[string]bool{}
	add := func(source, raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] || ValidatePostgresURL(raw) != nil {
			return
		}
		seen[raw] = true
		found = append(found, Candidate{Source: source, URL: raw})
	}

	if getenv != nil {
		add("DATABASE_URL in the environment", getenv("DATABASE_URL"))
	}

	for _, name := range []string{".env", ".env.example"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		add("DATABASE_URL in "+name, ParseDotenv(string(data))["DATABASE_URL"])
	}

	for _, name := range composeFileNames(dir) {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		for _, svc := range composePostgresServices(data, getenv) {
			add(fmt.Sprintf("service %q in %s", svc.name, name), svc.url)
		}
	}

	return found
}

func composeFileNames(dir string) []string {
	var names []string
	for _, pattern := range []string{"docker-compose*.yml", "docker-compose*.yaml", "compose*.yml", "compose*.yaml"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {
			names = append(names, filepath.Base(m))
		}
	}
	sort.Strings(names)
	return names
}

type composeService struct {
	name string
	url  string
}

type composeDoc struct {
	Services map[string]struct {
		Image       string      `yaml:"image"`
		Environment interface{} `yaml:"environment"`
		Ports       []any       `yaml:"ports"`
	} `yaml:"services"`
}

var varPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?:(:?-)([^}]*))?\}`)

// expandVars resolves ${VAR}, ${VAR:-default} and ${VAR-default}.
func expandVars(s string, getenv func(string) string) string {
	return varPattern.ReplaceAllStringFunc(s, func(m string) string {
		parts := varPattern.FindStringSubmatch(m)
		value := ""
		if getenv != nil {
			value = getenv(parts[1])
		}
		if value == "" && parts[2] != "" {
			return parts[3]
		}
		return value
	})
}

func composeEnv(raw interface{}, getenv func(string) string) map[string]string {
	env := map[string]string{}
	switch v := raw.(type) {
	case map[string]interface{}:
		for k, val := range v {
			env[k] = expandVars(fmt.Sprint(val), getenv)
		}
	case []interface{}:
		for _, item := range v {
			if k, val, ok := strings.Cut(fmt.Sprint(item), "="); ok {
				env[k] = expandVars(val, getenv)
			}
		}
	}
	return env
}

// publishedPort returns the host port that maps to container port 5432.
func publishedPort(ports []any, getenv func(string) string) (int, bool) {
	for _, p := range ports {
		switch v := p.(type) {
		case string:
			parts := strings.Split(expandVars(v, getenv), ":")
			// "5433:5432", "127.0.0.1:5433:5432", "5432" (no host port)
			if len(parts) < 2 {
				continue
			}
			container := strings.Split(parts[len(parts)-1], "/")[0]
			if container != "5432" {
				continue
			}
			if host, err := strconv.Atoi(parts[len(parts)-2]); err == nil {
				return host, true
			}
		case map[string]interface{}:
			target, _ := strconv.Atoi(expandVars(fmt.Sprint(v["target"]), getenv))
			published, _ := strconv.Atoi(expandVars(fmt.Sprint(v["published"]), getenv))
			if target == 5432 && published > 0 {
				return published, true
			}
		}
	}
	return 0, false
}

func composePostgresServices(data []byte, getenv func(string) string) []composeService {
	var doc composeDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}

	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []composeService
	for _, name := range names {
		svc := doc.Services[name]
		image := strings.ToLower(svc.Image)
		if !strings.Contains(image, "postgres") && !strings.Contains(image, "postgis") {
			continue
		}

		port, ok := publishedPort(svc.Ports, getenv)
		if !ok {
			continue // not reachable from the host
		}

		env := composeEnv(svc.Environment, getenv)
		user := env["POSTGRES_USER"]
		if user == "" {
			user = "postgres"
		}
		db := env["POSTGRES_DB"]
		if db == "" {
			db = user
		}

		u := url.URL{
			Scheme:   "postgresql",
			User:     url.UserPassword(user, env["POSTGRES_PASSWORD"]),
			Host:     net.JoinHostPort("localhost", strconv.Itoa(port)),
			Path:     "/" + db,
			RawQuery: "sslmode=disable",
		}
		out = append(out, composeService{name: name, url: u.String()})
	}

	return out
}
