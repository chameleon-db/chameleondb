package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MinModePasswordLength = 8

func (v *Vault) modeAuthPath() string {
	return filepath.Join(v.RootPath, VaultDirName, ModeAuthFileName)
}

// HasModePassword reports whether an admin password was configured for mode escalation.
func (v *Vault) HasModePassword() bool {
	_, err := os.Stat(v.modeAuthPath())
	return err == nil
}

func (v *Vault) saveModeAuthConfig(cfg *ModeAuthConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize mode auth config: %w", err)
	}

	if err := os.WriteFile(v.modeAuthPath(), data, 0600); err != nil {
		return fmt.Errorf("failed to write mode auth config: %w", err)
	}

	return nil
}

func (v *Vault) loadModeAuthConfig() (*ModeAuthConfig, error) {
	data, err := os.ReadFile(v.modeAuthPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("mode password is not configured")
		}
		return nil, fmt.Errorf("failed to read mode auth config: %w", err)
	}

	var cfg ModeAuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse mode auth config: %w", err)
	}

	if cfg.Salt == "" || cfg.Hash == "" {
		return nil, fmt.Errorf("invalid mode auth config")
	}

	return &cfg, nil
}

func randomSaltHex(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	return hex.EncodeToString(b), nil
}

func hashModePassword(password, salt string) string {
	payload := salt + ":" + password
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// SetModePassword configures (or rotates) the admin password for elevated mode changes.
func (v *Vault) SetModePassword(password string) error {
	if !v.Exists() {
		return fmt.Errorf("vault not initialized")
	}

	clean := strings.TrimSpace(password)
	if len(clean) < MinModePasswordLength {
		return fmt.Errorf("password too short (minimum %d characters)", MinModePasswordLength)
	}

	salt, err := randomSaltHex(16)
	if err != nil {
		return err
	}

	cfg := &ModeAuthConfig{
		Salt: salt,
		Hash: hashModePassword(clean, salt),
	}

	if err := v.saveModeAuthConfig(cfg); err != nil {
		return err
	}

	if err := v.AppendLog("MODE_AUTH", "", map[string]string{
		"action": "password_configured",
	}); err != nil {
		return err
	}

	return nil
}

// VerifyModePassword verifies whether password matches configured admin password.
func (v *Vault) VerifyModePassword(password string) (bool, error) {
	cfg, err := v.loadModeAuthConfig()
	if err != nil {
		return false, err
	}

	provided := hashModePassword(strings.TrimSpace(password), cfg.Salt)
	ok := subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.Hash)) == 1
	return ok, nil
}
