package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// RegisterVersion registers a new schema version in the vault
func (v *Vault) RegisterVersion(schemaPath string, author string, changesSummary string) (*VersionEntry, error) {
	// Ensure vault exists
	if !v.Exists() {
		if err := v.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize vault: %w", err)
		}
	}

	// Load current manifest
	if err := v.Load(); err != nil {
		return nil, err
	}

	// Compute schema hash
	hash, err := v.ComputeSchemaHash(schemaPath)
	if err != nil {
		return nil, err
	}

	// Check if schema changed (compare with current version)
	if v.Manifest.CurrentVersion != "" {
		current, err := v.GetCurrentVersion()
		if err == nil && current.Hash == hash {
			// No changes
			return current, nil
		}
	}

	// Generate new version number
	versionNum := len(v.Manifest.Versions) + 1
	version := fmt.Sprintf("v%03d", versionNum)

	// Determine parent
	var parent *string
	if v.Manifest.CurrentVersion != "" {
		parent = &v.Manifest.CurrentVersion
	}

	// Read schema content
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	// Create version entry
	entry := VersionEntry{
		Version:        version,
		Hash:           hash,
		Timestamp:      time.Now().UTC(),
		Author:         author,
		Parent:         parent,
		Locked:         true,
		ChangesSummary: changesSummary,
		Files:          []string{schemaPath},
	}

	// Save version snapshot
	if err := v.SaveVersion(version, schemaContent, hash); err != nil {
		return nil, err
	}

	// Update manifest
	v.Manifest.Versions = append(v.Manifest.Versions, entry)
	v.Manifest.CurrentVersion = version

	if err := v.saveManifest(v.Manifest); err != nil {
		return nil, err
	}

	// Log registration
	if err := v.AppendLog("REGISTER", version, map[string]string{
		"action":  "schema_registered",
		"hash":    hash[:12] + "...",
		"parent":  stringOrNull(parent),
		"changes": changesSummary,
	}); err != nil {
		return nil, err
	}

	return &entry, nil
}

// DetectChanges checks if schema has changed since last version
func (v *Vault) DetectChanges(schemaPath string) (bool, string, error) {
	if !v.Exists() {
		// No vault = first time = changes exist
		return true, "Initial schema", nil
	}

	if err := v.Load(); err != nil {
		return false, "", err
	}

	if v.Manifest.CurrentVersion == "" {
		// No versions yet
		return true, "Initial schema", nil
	}

	// Compute current hash
	currentHash, err := v.ComputeSchemaHash(schemaPath)
	if err != nil {
		return false, "", err
	}

	// Get last version hash
	lastVersion, err := v.GetCurrentVersion()
	if err != nil {
		return false, "", err
	}

	if currentHash == lastVersion.Hash {
		// No changes
		return false, "", nil
	}

	// Changes detected (we don't compute diff yet, just detect)
	return true, "Schema modified", nil
}

// GetVersionHistory returns all versions in chronological order
func (v *Vault) GetVersionHistory() ([]VersionEntry, error) {
	if !v.Exists() {
		return []VersionEntry{}, nil
	}

	if err := v.Load(); err != nil {
		return nil, err
	}

	return v.Manifest.Versions, nil
}

// GetVersionContent reads the schema content for a specific version
func (v *Vault) GetVersionContent(version string) ([]byte, error) {
	vaultPath := v.RootPath + "/.chameleon/vault"
	versionPath := vaultPath + "/versions/" + version + ".json"

	data, err := os.ReadFile(versionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version content: %w", err)
	}

	return data, nil
}

// FormatVersion formats a version entry for display
func FormatVersion(entry *VersionEntry) string {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	parent := stringOrNull(entry.Parent)

	return fmt.Sprintf(
		"%s\n"+
			"├─ Hash: %s\n"+
			"├─ Date: %s\n"+
			"├─ Author: %s\n"+
			"├─ Changes: %s\n"+
			"└─ Parent: %s",
		entry.Version,
		entry.Hash[:12]+"...",
		timestamp,
		entry.Author,
		entry.ChangesSummary,
		parent,
	)
}

// stringOrNull returns string value or "none"
func stringOrNull(s *string) string {
	if s == nil {
		return "none"
	}
	return *s
}

// SerializeSchema converts schema to JSON for storage
func SerializeSchema(schema interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize schema: %w", err)
	}
	return data, nil
}
