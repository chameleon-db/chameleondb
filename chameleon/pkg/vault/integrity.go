package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ComputeSchemaHash computes SHA256 hash of schema file(s)
func (v *Vault) ComputeSchemaHash(schemaPath string) (string, error) {
	// Open schema file
	f, err := os.Open(schemaPath)
	if err != nil {
		return "", fmt.Errorf("failed to open schema: %w", err)
	}
	defer f.Close()

	// Compute SHA256
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to hash schema: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash, nil
}

// VerifyIntegrity checks vault integrity
func (v *Vault) VerifyIntegrity() (*VerificationResult, error) {
	if !v.Exists() {
		return &VerificationResult{
			Valid:  false,
			Issues: []string{"vault does not exist"},
		}, nil
	}

	if err := v.Load(); err != nil {
		return &VerificationResult{
			Valid:  false,
			Issues: []string{fmt.Sprintf("failed to load manifest: %v", err)},
		}, nil
	}

	result := &VerificationResult{
		Valid:        true,
		Issues:       []string{},
		VersionsOK:   []string{},
		VersionsFail: []string{},
	}

	// Verify each version
	for _, entry := range v.Manifest.Versions {
		if err := v.verifyVersion(&entry); err != nil {
			result.Valid = false
			result.VersionsFail = append(result.VersionsFail, entry.Version)
			result.Issues = append(result.Issues, fmt.Sprintf("%s: %v", entry.Version, err))
		} else {
			result.VersionsOK = append(result.VersionsOK, entry.Version)
		}
	}

	return result, nil
}

// verifyVersion verifies a single version's integrity
func (v *Vault) verifyVersion(entry *VersionEntry) error {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)

	// Read stored hash
	hashPath := filepath.Join(vaultPath, HashesDirName, entry.Version+".hash")
	storedHash, err := os.ReadFile(hashPath)
	if err != nil {
		return fmt.Errorf("hash file missing: %w", err)
	}

	// Read version file
	versionPath := filepath.Join(vaultPath, VersionsDirName, entry.Version+".json")
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return fmt.Errorf("version file missing: %w", err)
	}

	// Compute current hash
	hasher := sha256.New()
	hasher.Write(versionData)
	currentHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare
	if currentHash != string(storedHash) {
		return fmt.Errorf("hash mismatch (expected: %s, got: %s)", storedHash, currentHash)
	}

	// Also verify against entry.Hash
	if currentHash != entry.Hash {
		return fmt.Errorf("manifest hash mismatch")
	}

	return nil
}

// SaveVersion saves a version snapshot to vault
func (v *Vault) SaveVersion(version string, schemaContent []byte, hash string) error {
	vaultPath := filepath.Join(v.RootPath, VaultDirName)

	// Save version file
	versionPath := filepath.Join(vaultPath, VersionsDirName, version+".json")
	if err := os.WriteFile(versionPath, schemaContent, 0644); err != nil {
		return fmt.Errorf("failed to save version: %w", err)
	}

	// Save hash file
	hashPath := filepath.Join(vaultPath, HashesDirName, version+".hash")
	if err := os.WriteFile(hashPath, []byte(hash), 0644); err != nil {
		return fmt.Errorf("failed to save hash: %w", err)
	}

	return nil
}
