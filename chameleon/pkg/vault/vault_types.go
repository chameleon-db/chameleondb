package vault

import (
	"time"
)

// Vault represents the Schema Vault system
type Vault struct {
	RootPath string    // .chameleon/vault/
	Manifest *Manifest // Current state
}

// Manifest represents the vault's manifest.json
type Manifest struct {
	CurrentVersion string         `json:"current_version"`
	Versions       []VersionEntry `json:"versions"`
	ParanoidMode   string         `json:"paranoid_mode"` // Legacy compatibility field
}

// ModeConfig stores current security/paranoid mode (source of truth)
type ModeConfig struct {
	ParanoidMode string `json:"paranoid_mode"`
}

// ModeAuthConfig stores password verifier for privileged mode changes.
type ModeAuthConfig struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

// VersionEntry represents a single schema version in the vault
type VersionEntry struct {
	Version        string    `json:"version"`         // v001, v002, etc.
	Hash           string    `json:"hash"`            // SHA256 hash
	Timestamp      time.Time `json:"timestamp"`       // When registered
	Author         string    `json:"author"`          // Who registered it
	Parent         *string   `json:"parent"`          // Parent version (null for v001)
	Locked         bool      `json:"locked"`          // Immutability flag
	ChangesSummary string    `json:"changes_summary"` // Human-readable description
	Files          []string  `json:"files"`           // Schema files included
}

// IntegrityLogEntry represents a single entry in integrity.log
type IntegrityLogEntry struct {
	Timestamp time.Time
	Action    string // INIT, REGISTER, MIGRATE, VERIFY, etc.
	Version   string
	Details   map[string]string
}

// VaultStatus represents current vault state
type VaultStatus struct {
	Exists         bool
	CurrentVersion string
	TotalVersions  int
	IntegrityOK    bool
	LastModified   time.Time
}

// VerificationResult represents integrity check results
type VerificationResult struct {
	Valid        bool
	Issues       []string
	VersionsOK   []string
	VersionsFail []string
}
