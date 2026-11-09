package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

const MinModePasswordLength = 8

// ErrModePasswordConfigured is returned by SetModePassword when a password
// already exists. Changing it goes through RotateModePassword, which proves
// knowledge of the current one.
var ErrModePasswordConfigured = errors.New("mode password is already configured; rotate it with the current password")

// Argon2id parameters: the OWASP Password Storage Cheat Sheet minimum
// (19 MiB, 2 iterations, 1 lane). They are stored in each verifier, so they can
// be raised later without invalidating existing passwords.
const (
	argon2Memory  = 19 * 1024 // KiB
	argon2Time    = 2
	argon2Threads = 1
	argon2KeyLen  = 32
	argon2SaltLen = 16

	// Upper bounds accepted when reading a verifier from disk, so that an
	// edited file cannot make verification consume unbounded memory or time.
	argon2MaxMemory  = 1 << 20 // 1 GiB, in KiB
	argon2MaxTime    = 10
	argon2MaxThreads = 16
)

const argon2idPrefix = "$argon2id$"

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

	if cfg.Hash == "" || (!isArgon2idHash(cfg.Hash) && cfg.Salt == "") {
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

// hashModePassword is the legacy verifier: one round of salted SHA-256. It is
// kept only to verify (and then upgrade) files written by earlier versions.
func hashModePassword(password, salt string) string {
	payload := salt + ":" + password
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func isArgon2idHash(hash string) bool {
	return strings.HasPrefix(hash, argon2idPrefix)
}

// hashArgon2id returns an Argon2id verifier in the PHC string format:
// $argon2id$v=19$m=<KiB>,t=<iterations>,p=<lanes>$<salt>$<hash>
func hashArgon2id(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// verifyArgon2id checks password against a PHC string, using the parameters
// stored in it.
func verifyArgon2id(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid mode auth config: malformed argon2id hash")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, fmt.Errorf("invalid mode auth config: unsupported argon2 version")
	}

	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, fmt.Errorf("invalid mode auth config: malformed argon2id parameters")
	}
	if memory == 0 || memory > argon2MaxMemory ||
		iterations == 0 || iterations > argon2MaxTime ||
		threads == 0 || threads > argon2MaxThreads {
		return false, fmt.Errorf("invalid mode auth config: argon2id parameters out of range")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false, fmt.Errorf("invalid mode auth config: malformed salt")
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, fmt.Errorf("invalid mode auth config: malformed hash")
	}

	got := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func cleanModePassword(password string) (string, error) {
	clean := strings.TrimSpace(password)
	if len(clean) < MinModePasswordLength {
		return "", fmt.Errorf("password too short (minimum %d characters)", MinModePasswordLength)
	}

	return clean, nil
}

func (v *Vault) writeModePassword(clean string) error {
	hash, err := hashArgon2id(clean)
	if err != nil {
		return err
	}

	return v.saveModeAuthConfig(&ModeAuthConfig{Hash: hash})
}

// SetModePassword configures the admin password for elevated mode changes.
// It only creates the first password: once one exists, changing it requires
// RotateModePassword, so that running this command is not a way around it.
func (v *Vault) SetModePassword(password string) error {
	if !v.Exists() {
		return fmt.Errorf("vault not initialized")
	}

	if v.HasModePassword() {
		return ErrModePasswordConfigured
	}

	clean, err := cleanModePassword(password)
	if err != nil {
		return err
	}

	if err := v.writeModePassword(clean); err != nil {
		return err
	}

	return v.AppendLog("MODE_AUTH", "", map[string]string{
		"action": "password_configured",
	})
}

// RotateModePassword replaces the admin password. The current one must be
// provided and correct.
func (v *Vault) RotateModePassword(current, next string) error {
	if !v.Exists() {
		return fmt.Errorf("vault not initialized")
	}

	ok, err := v.VerifyModePassword(current)
	if err != nil {
		return err
	}
	if !ok {
		_ = v.AppendLog("MODE_AUTH", "", map[string]string{
			"action": "password_rotation_denied",
			"reason": "invalid_current_password",
		})
		return fmt.Errorf("invalid current mode password")
	}

	clean, err := cleanModePassword(next)
	if err != nil {
		return err
	}

	if err := v.writeModePassword(clean); err != nil {
		return err
	}

	return v.AppendLog("MODE_AUTH", "", map[string]string{
		"action": "password_rotated",
	})
}

// VerifyModePassword verifies whether password matches configured admin password.
// A verifier written by an earlier version (salted SHA-256) is accepted once
// and replaced by an Argon2id one.
func (v *Vault) VerifyModePassword(password string) (bool, error) {
	cfg, err := v.loadModeAuthConfig()
	if err != nil {
		return false, err
	}

	provided := strings.TrimSpace(password)

	if isArgon2idHash(cfg.Hash) {
		return verifyArgon2id(provided, cfg.Hash)
	}

	legacy := hashModePassword(provided, cfg.Salt)
	if subtle.ConstantTimeCompare([]byte(legacy), []byte(cfg.Hash)) != 1 {
		return false, nil
	}

	// Upgrade on success. The password is valid either way, so a failure to
	// write the new verifier does not fail the check.
	if clean, cleanErr := cleanModePassword(provided); cleanErr == nil {
		if upgradeErr := v.writeModePassword(clean); upgradeErr == nil {
			_ = v.AppendLog("MODE_AUTH", "", map[string]string{
				"action": "password_hash_upgraded",
			})
		}
	}

	return true, nil
}
