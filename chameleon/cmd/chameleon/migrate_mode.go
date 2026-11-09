package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/internal/initwizard"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

// afterFirstApply runs once, after the migrate --apply that created the vault,
// while the project is in readonly mode. With a terminal it offers a password
// and standard mode; with --set-mode standard it does the same from
// CHAMELEON_MODE_PASSWORD; otherwise it does nothing, and the readonly notice
// is the only output.
func afterFirstApply(v *vault.Vault, journalLogger *journal.Logger, noInput bool, setMode string) error {
	mode, err := v.GetParanoidMode()
	if err != nil || mode != "readonly" {
		return nil
	}

	logStep := func(action string, details map[string]interface{}) {
		if journalLogger != nil {
			_ = journalLogger.Log(action, "success", details, nil)
		}
	}

	switchToStandard := func(password string) error {
		if err := v.SetModePassword(password); err != nil {
			return err
		}
		logStep("config_mode_auth", map[string]interface{}{"action": "password_configured", "source": "migrate_first_apply"})

		if err := v.SetParanoidMode("standard"); err != nil {
			return fmt.Errorf("the password was set, but the mode could not be changed: %w", err)
		}
		logStep("config_mode", map[string]interface{}{"from_mode": "readonly", "target_mode": "standard", "source": "migrate_first_apply"})

		return nil
	}

	if setMode != "" {
		if strings.ToLower(strings.TrimSpace(setMode)) != "standard" {
			return fmt.Errorf("--set-mode only supports \"standard\" (got %q); the migration was applied", setMode)
		}

		password := strings.TrimSpace(os.Getenv(modePasswordEnvVar))
		if password == "" {
			return fmt.Errorf("--set-mode needs %s; the migration was applied and the project is still in readonly mode", modePasswordEnvVar)
		}

		if err := switchToStandard(password); err != nil {
			return fmt.Errorf("%w; the migration was applied", err)
		}

		printSuccess("Mode password configured. Mode: standard")
		return nil
	}

	if !interactiveTerminal(noInput) {
		return nil
	}

	fmt.Println()
	enabled, err := initwizard.OfferStandardMode(initwizard.ModeOffer{
		Prompter:     initwizard.NewPrompter(os.Stdin, os.Stdout),
		ReadPassword: readHiddenPassword,
		MinLength:    vault.MinModePasswordLength,
		SetPassword: func(password string) error {
			return v.SetModePassword(password)
		},
		SetStandard: func() error {
			if err := v.SetParanoidMode("standard"); err != nil {
				return err
			}
			logStep("config_mode_auth", map[string]interface{}{"action": "password_configured", "source": "migrate_first_apply"})
			logStep("config_mode", map[string]interface{}{"from_mode": "readonly", "target_mode": "standard", "source": "migrate_first_apply"})
			return nil
		},
	})
	if err != nil {
		return err
	}

	if enabled {
		printSuccess("Mode password configured. Mode: standard")
	}

	return nil
}
