package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	VaultDirName     = ".chameleon/vault"
	ManifestFileName = "manifest.json"
	ModeFileName     = "mode.json"
	ModeAuthFileName = "mode_auth.json"
	IntegrityLogName = "integrity.log"
	VersionsDirName  = "versions"
	HashesDirName    = "hashes"
)

// NewVault creates a vault instance (does not initialize on disk)
func NewVault(rootPath string) *Vault {
	return &Vault{
		RootPath: rootPath,
	}
}

// Exists checks if vault exists on disk
func (v *Vault) Exists() bool {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)
	manifestPath := filepath.Join(vaultPath, ManifestFileName)

	_, err := os.Stat(manifestPath)
	return err == nil
}

// Initialize creates vault structure on disk
func (v *Vault) Initialize() error {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)

	// Create directories
	dirs := []string{
		vaultPath,
		filepath.Join(vaultPath, VersionsDirName),
		filepath.Join(vaultPath, HashesDirName),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create initial manifest
	manifest := &Manifest{
		CurrentVersion: "",
		Versions:       []VersionEntry{},
		ParanoidMode:   "readonly", // Legacy compatibility; mode source of truth is mode.json
	}

	if err := v.saveManifest(manifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	v.Manifest = manifest

	if err := v.saveModeConfig(&ModeConfig{ParanoidMode: "readonly"}); err != nil {
		return fmt.Errorf("failed to save mode config: %w", err)
	}

	// Log initialization
	if err := v.AppendLog("INIT", "", map[string]string{
		"action": "vault_created",
	}); err != nil {
		return fmt.Errorf("failed to log initialization: %w", err)
	}

	return nil
}

// Load reads the manifest from disk
func (v *Vault) Load() error {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)
	manifestPath := filepath.Join(vaultPath, ManifestFileName)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	v.Manifest = &manifest
	return nil
}

// saveManifest writes manifest to disk
func (v *Vault) saveManifest(manifest *Manifest) error {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)
	manifestPath := filepath.Join(vaultPath, ManifestFileName)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// GetStatus returns current vault status
func (v *Vault) GetStatus() (*VaultStatus, error) {
	if !v.Exists() {
		return &VaultStatus{Exists: false}, nil
	}

	if err := v.Load(); err != nil {
		return nil, err
	}

	status := &VaultStatus{
		Exists:         true,
		CurrentVersion: v.Manifest.CurrentVersion,
		TotalVersions:  len(v.Manifest.Versions),
		IntegrityOK:    true, // Will be checked by Verify()
	}

	// Get last modified time
	vaultPath := filepath.Join(v.RootPath, VaultDirName)
	manifestPath := filepath.Join(vaultPath, ManifestFileName)

	info, err := os.Stat(manifestPath)
	if err == nil {
		status.LastModified = info.ModTime()
	}

	return status, nil
}

// GetVersion retrieves a specific version entry
func (v *Vault) GetVersion(version string) (*VersionEntry, error) {
	if v.Manifest == nil {
		if err := v.Load(); err != nil {
			return nil, err
		}
	}

	for _, entry := range v.Manifest.Versions {
		if entry.Version == version {
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("version %s not found", version)
}

// GetCurrentVersion returns the current version entry
func (v *Vault) GetCurrentVersion() (*VersionEntry, error) {
	if v.Manifest == nil {
		if err := v.Load(); err != nil {
			return nil, err
		}
	}

	if v.Manifest.CurrentVersion == "" {
		return nil, fmt.Errorf("no current version set")
	}

	return v.GetVersion(v.Manifest.CurrentVersion)
}

// appendLog appends an entry to integrity.log
func (v *Vault) AppendLog(action, version string, details map[string]string) error {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)
	logPath := filepath.Join(vaultPath, IntegrityLogName)

	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Build log line
	logLine := fmt.Sprintf("%s [%s]", timestamp, action)

	if version != "" {
		logLine += fmt.Sprintf(" version=%s", version)
	}

	for key, value := range details {
		logLine += fmt.Sprintf(" %s=%s", key, value)
	}

	logLine += "\n"

	// Append to file (create if doesn't exist)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(logLine); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// ReadLog reads the integrity log
func (v *Vault) ReadLog() ([]string, error) {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)
	logPath := filepath.Join(vaultPath, IntegrityLogName)

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read log: %w", err)
	}

	lines := []string{}
	for _, line := range splitLines(string(data)) {
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines, nil
}

// splitLines splits text into lines
func splitLines(text string) []string {
	lines := []string{}
	current := ""

	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}
