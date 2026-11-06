package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/internal/state"
)

// Directory manages the .chameleon/ directory structure
type Directory struct {
	rootDir string // .chameleon/
}

// NewDirectory creates a new directory manager
func NewDirectory(workDir string) *Directory {
	return &Directory{
		rootDir: filepath.Join(workDir, ".chameleon"),
	}
}

// Initialize creates the .chameleon/ directory structure
func (d *Directory) Initialize() error {
	// Create main directory
	if err := os.MkdirAll(d.rootDir, 0755); err != nil {
		return fmt.Errorf("failed to create .chameleon directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{
		"state",
		"state/migrations",
		"state/debug",
		"journal",
		"backups",
	}

	for _, subdir := range subdirs {
		path := filepath.Join(d.rootDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", subdir, err)
		}
	}

	// Create .gitignore
	if err := d.createGitignore(); err != nil {
		return err
	}

	return nil
}

// gitignoreRules are the entries .chameleon/.gitignore must always contain.
// Paths are relative to .chameleon/.
var gitignoreRules = []string{
	"state/",
	"journal/",
	"backups/",
	// Verifier of the mode password: credential material that must not be
	// committed. The rest of vault/ stays versioned (integrity history).
	"vault/mode_auth.json",
}

// createGitignore creates .chameleon/.gitignore
func (d *Directory) createGitignore() error {
	return d.EnsureGitignore()
}

// EnsureGitignore creates .chameleon/.gitignore, or appends the rules an
// existing one is missing, so projects created by an older version also stop
// tracking credential files. Lines already present are left untouched.
// Note that git keeps tracking a file that was committed before it was ignored.
func (d *Directory) EnsureGitignore() error {
	if err := os.MkdirAll(d.rootDir, 0755); err != nil {
		return fmt.Errorf("failed to create .chameleon directory: %w", err)
	}

	gitignorePath := filepath.Join(d.rootDir, ".gitignore")
	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	present := make(map[string]bool)
	for _, line := range strings.Split(existing, "\n") {
		present[strings.TrimSpace(line)] = true
	}

	var missing []string
	for _, rule := range gitignoreRules {
		if !present[rule] {
			missing = append(missing, rule)
		}
	}
	if existing != "" && len(missing) == 0 {
		return nil
	}

	var out strings.Builder
	if existing == "" {
		out.WriteString("# ChameleonDB Administrative Files\n")
		out.WriteString("# State and journal are local to each developer\n")
	} else {
		out.WriteString(existing)
		if !strings.HasSuffix(existing, "\n") {
			out.WriteString("\n")
		}
	}
	for _, rule := range missing {
		out.WriteString(rule + "\n")
	}

	return os.WriteFile(gitignorePath, []byte(out.String()), 0644)
}

// GetPaths returns all directory paths
func (d *Directory) GetPaths() DirectoryPaths {
	return DirectoryPaths{
		Root:       d.rootDir,
		Config:     filepath.Join(d.rootDir, "config.yml"),
		State:      filepath.Join(d.rootDir, "state"),
		Migrations: filepath.Join(d.rootDir, "state", "migrations"),
		Journal:    filepath.Join(d.rootDir, "journal"),
		Backups:    filepath.Join(d.rootDir, "backups"),
	}
}

// DirectoryPaths holds all important paths
type DirectoryPaths struct {
	Root       string
	Config     string
	State      string
	Migrations string
	Journal    string
	Backups    string
}

// ManagerFactory creates and initializes all managers
type ManagerFactory struct {
	workDir string
	dir     *Directory
}

// NewManagerFactory creates a new manager factory
func NewManagerFactory(workDir string) *ManagerFactory {
	return &ManagerFactory{
		workDir: workDir,
		dir:     NewDirectory(workDir),
	}
}

// Initialize initializes the entire .chameleon/ structure
func (mf *ManagerFactory) Initialize() error {
	// Create directory structure
	if err := mf.dir.Initialize(); err != nil {
		return err
	}

	return nil
}

// CreateConfigLoader creates a config loader
func (mf *ManagerFactory) CreateConfigLoader() *config.Loader {
	return config.NewLoader(mf.workDir)
}

// CreateJournalLogger creates a journal logger
func (mf *ManagerFactory) CreateJournalLogger() (*journal.Logger, error) {
	paths := mf.dir.GetPaths()
	return journal.NewLogger(paths.Journal)
}

// CreateStateTracker creates a state tracker
func (mf *ManagerFactory) CreateStateTracker() (*state.Tracker, error) {
	paths := mf.dir.GetPaths()
	return state.NewTracker(paths.State)
}

// Status returns the current directory structure status
func (mf *ManagerFactory) Status() (string, error) {
	paths := mf.dir.GetPaths()

	// Check if root exists
	if _, err := os.Stat(paths.Root); err != nil {
		return "not_initialized", nil
	}

	// Check subdirectories
	status := "initialized\n"
	status += fmt.Sprintf("  Config: %s\n", paths.Config)
	status += fmt.Sprintf("  State: %s\n", paths.State)
	status += fmt.Sprintf("  Journal: %s\n", paths.Journal)
	status += fmt.Sprintf("  Backups: %s\n", paths.Backups)

	// Check if config exists
	if _, err := os.Stat(paths.Config); err == nil {
		status += "  Config loaded: yes\n"
	} else {
		status += "  Config loaded: no\n"
	}

	return status, nil
}
