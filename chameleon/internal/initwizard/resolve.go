package initwizard

import (
	"fmt"
	"strings"
)

// Flags are the non-interactive form of every question.
type Flags struct {
	DBURL     string // --db-url
	Compose   bool   // --compose
	DetectDB  bool   // --detect-db
	NoExample bool   // --no-example
}

// Env is what Resolve needs from the outside world.
type Env struct {
	Dir         string
	Getenv      func(string) string
	Interactive bool               // a terminal is attached and --no-input / CI are not set
	Ping        func(string) error // tests a connection; nil skips the test
}

// Result is what `init` does with the answers.
type Result struct {
	// URL is a connection string to show to the user as `export DATABASE_URL=...`.
	// Empty means "keep ${DATABASE_URL}" with nothing to export.
	URL string
	// WriteCompose asks init to write docker-compose.yml.
	WriteCompose bool
	// RunCompose asks init to run `docker compose up -d` (only ever set after a
	// confirmation).
	RunCompose bool
	// CreateExample says whether to create schemas/example.cham.
	CreateExample bool
	// Notes are lines to print at the end (warnings, what was skipped).
	Notes []string
}

const (
	optEnvVar = iota
	optDetect
	optCompose
	optURL
)

// Resolve turns flags, detection and answers into a Result. Without a terminal
// every question takes its default, and the output stays deterministic.
func Resolve(p *Prompter, fl Flags, env Env) (Result, error) {
	res := Result{CreateExample: !fl.NoExample}

	chosen := 0
	for _, set := range []bool{fl.DBURL != "", fl.Compose, fl.DetectDB} {
		if set {
			chosen++
		}
	}
	if chosen > 1 {
		return res, fmt.Errorf("--db-url, --compose and --detect-db are mutually exclusive")
	}

	option := optEnvVar
	fromMenu := false
	switch {
	case fl.DBURL != "":
		option = optURL
	case fl.Compose:
		option = optCompose
	case fl.DetectDB:
		option = optDetect
	case env.Interactive:
		fromMenu = true
		option = p.Choose("Database connection:", []string{
			"Use ${DATABASE_URL} (default)",
			"Detect from this repository",
			"Create a docker-compose.yml with PostgreSQL 16",
			"Enter a connection string",
		}, optEnvVar)
	}

	switch option {
	case optURL:
		raw := strings.TrimSpace(fl.DBURL)
		if fromMenu {
			raw = strings.TrimSpace(p.Ask("Connection string", ""))
		}
		if err := ValidatePostgresURL(raw); err != nil {
			if !fromMenu {
				return res, fmt.Errorf("--db-url: %w", err)
			}
			res.Notes = append(res.Notes, "Not used: "+err.Error()+". Keeping ${DATABASE_URL}.")
			break
		}
		res.URL = raw
		res.Notes = append(res.Notes, checkConnection(env, raw, "")...)

	case optDetect:
		url, notes := resolveDetect(p, env)
		res.URL = url
		res.Notes = append(res.Notes, notes...)

	case optCompose:
		res.WriteCompose = true
		res.URL = DefaultComposeURL
		if fromMenu {
			res.RunCompose = p.Confirm("Start it now with 'docker compose up -d'?", false)
		}
	}

	if fromMenu && !fl.NoExample {
		res.CreateExample = p.Confirm("Create schemas/example.cham?", true)
	}

	return res, nil
}

// resolveDetect runs detection and lets the user pick when there are several.
func resolveDetect(p *Prompter, env Env) (string, []string) {
	candidates := Detect(env.Dir, env.Getenv)
	if len(candidates) == 0 {
		return "", []string{"Nothing found in the environment, .env files or compose files. Keeping ${DATABASE_URL}."}
	}

	pick := candidates[0]
	if env.Interactive && len(candidates) > 1 {
		options := make([]string, len(candidates))
		for i, c := range candidates {
			options[i] = fmt.Sprintf("%s  (%s)", c.Redacted(), c.Source)
		}
		pick = candidates[p.Choose("Found several connections:", options, 0)]
	} else {
		fmt.Fprintf(p.Out, "Found %s  (%s)\n", pick.Redacted(), pick.Source)
	}

	notes := checkConnection(env, pick.URL, pick.Redacted())
	if len(notes) > 0 && env.Interactive {
		for _, n := range notes {
			fmt.Fprintln(p.Out, n)
		}
		if !p.Confirm("Use it anyway?", false) {
			return "", []string{"Connection not used. Keeping ${DATABASE_URL}."}
		}
		notes = nil
	}

	return pick.URL, notes
}

// checkConnection returns a note when the connection cannot be opened. The
// error text can echo the URL, so the display form is used when provided.
func checkConnection(env Env, raw, display string) []string {
	if env.Ping == nil {
		return nil
	}
	if err := env.Ping(raw); err != nil {
		reason := err.Error()
		if display != "" {
			reason = strings.ReplaceAll(reason, raw, display)
		}
		return []string{"Could not connect: " + reason}
	}
	return nil
}
