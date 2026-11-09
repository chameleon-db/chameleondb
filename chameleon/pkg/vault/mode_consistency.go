package vault

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// lastRecordedMode returns the mode set by the most recent mode event in the
// integrity log: the creation of the vault (which records its initial mode) or
// a successful mode change. found is false when the log has no such event, as
// in vaults created before the initial mode was recorded.
func (v *Vault) lastRecordedMode() (mode string, found bool, err error) {
	f, err := os.Open(filepath.Join(v.RootPath, VaultDirName, IntegrityLogName))
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read integrity log: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		var action, lineMode string
		for _, field := range strings.Fields(line) {
			switch {
			case strings.HasPrefix(field, "action="):
				action = strings.TrimPrefix(field, "action=")
			case strings.HasPrefix(field, "mode="):
				lineMode = strings.TrimPrefix(field, "mode=")
			}
		}

		isModeEvent := (strings.Contains(line, "[INIT]") && action == "vault_created") ||
			(strings.Contains(line, "[MODE]") && action == "mode_updated")
		if isModeEvent && lineMode != "" {
			mode, found = lineMode, true
		}
	}

	return mode, found, scanner.Err()
}

// checkModeConsistency compares mode.json with the last mode the log recorded.
// It returns a description of the mismatch, or "" when they agree or when
// there is nothing to compare.
//
// This detects an edit of mode.json made without going through the CLI. It does
// not stop someone who also edits the log: both are plain files (see ADR-0013).
func (v *Vault) checkModeConsistency() string {
	data, err := os.ReadFile(v.modeConfigPath())
	if err != nil {
		return "" // legacy layout: the mode is read from the manifest
	}

	var cfg ModeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Sprintf("mode.json cannot be parsed: %v", err)
	}

	current, err := normalizeParanoidMode(cfg.ParanoidMode)
	if err != nil {
		return fmt.Sprintf("mode.json holds an invalid mode: %v", err)
	}

	recorded, found, err := v.lastRecordedMode()
	if err != nil || !found {
		return ""
	}

	if current != recorded {
		return fmt.Sprintf("mode.json says %q but the last recorded mode change set %q; "+
			"it was edited outside the CLI", current, recorded)
	}

	return ""
}
