package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultParanoidMode = "readonly"

var validParanoidModes = map[string]struct{}{
	"readonly":   {},
	"standard":   {},
	"privileged": {},
	"emergency":  {},
}

func normalizeParanoidMode(mode string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "admin" {
		normalized = "privileged"
	}

	if _, ok := validParanoidModes[normalized]; !ok {
		return "", fmt.Errorf("invalid mode %q (allowed: readonly, standard, privileged, emergency)", mode)
	}

	return normalized, nil
}

func (v *Vault) modeConfigPath() string {
	return filepath.Join(v.RootPath, VaultDirName, ModeFileName)
}

func (v *Vault) saveModeConfig(cfg *ModeConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize mode config: %w", err)
	}

	if err := os.WriteFile(v.modeConfigPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write mode config: %w", err)
	}

	return nil
}

// SetParanoidMode updates the current paranoid mode in mode.json.
func (v *Vault) SetParanoidMode(mode string) error {
	if !v.Exists() {
		return fmt.Errorf("vault not initialized")
	}

	normalized, err := normalizeParanoidMode(mode)
	if err != nil {
		return err
	}

	if err := v.saveModeConfig(&ModeConfig{ParanoidMode: normalized}); err != nil {
		return err
	}

	if err := v.Load(); err == nil {
		v.Manifest.ParanoidMode = normalized
		if err := v.saveManifest(v.Manifest); err != nil {
			return err
		}
	}

	if err := v.AppendLog("MODE", "", map[string]string{
		"action": "mode_updated",
		"mode":   normalized,
	}); err != nil {
		return err
	}

	return nil
}

// GetParanoidMode returns the current paranoid mode.
// Source of truth is mode.json; manifest is used only as backward-compatible fallback.
func (v *Vault) GetParanoidMode() (string, error) {
	if !v.Exists() {
		return DefaultParanoidMode, nil
	}

	data, err := os.ReadFile(v.modeConfigPath())
	if err == nil {
		var cfg ModeConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return "", fmt.Errorf("failed to parse mode config: %w", err)
		}

		return normalizeParanoidMode(cfg.ParanoidMode)
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read mode config: %w", err)
	}

	if loadErr := v.Load(); loadErr == nil && v.Manifest != nil && v.Manifest.ParanoidMode != "" {
		legacyMode, normalizeErr := normalizeParanoidMode(v.Manifest.ParanoidMode)
		if normalizeErr != nil {
			return "", normalizeErr
		}

		if writeErr := v.saveModeConfig(&ModeConfig{ParanoidMode: legacyMode}); writeErr != nil {
			return "", writeErr
		}

		return legacyMode, nil
	}

	if err := v.saveModeConfig(&ModeConfig{ParanoidMode: DefaultParanoidMode}); err != nil {
		return "", err
	}

	return DefaultParanoidMode, nil
}
