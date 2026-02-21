package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const modePasswordEnvVar = "CHAMELEON_MODE_PASSWORD"

var paranoidModeRank = map[string]int{
	"readonly":   0,
	"standard":   1,
	"privileged": 2,
	"emergency":  3,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local ChameleonDB configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set key=value",
	Short: "Set a local configuration value",
	Long: `Set a local configuration value.

Currently supported:
  mode=readonly|standard|privileged|emergency

Examples:
  chameleon config set mode=standard
  chameleon config set mode=emergency`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		factory := admin.NewManagerFactory(workDir)
		journalLogger, _ := factory.CreateJournalLogger()

		parts := strings.SplitN(args[0], "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: use key=value")
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "mode":
			v := vault.NewVault(workDir)
			if !v.Exists() {
				return fmt.Errorf("vault not initialized. Run 'chameleon migrate' first")
			}

			previousMode, err := v.GetParanoidMode()
			if err != nil {
				return err
			}

			targetMode := strings.ToLower(strings.TrimSpace(value))
			if targetMode == "admin" {
				targetMode = "privileged"
			}

			if requiresModeAuth(previousMode, targetMode) {
				if !v.HasModePassword() {
					if journalLogger != nil {
						_ = journalLogger.Log("config_mode", "denied", map[string]interface{}{
							"reason":      "mode_password_not_configured",
							"from_mode":   previousMode,
							"target_mode": targetMode,
						}, nil)
					}
					_ = v.AppendLog("MODE", "", map[string]string{
						"action": "mode_change_denied",
						"mode":   targetMode,
						"reason": "mode_password_not_configured",
					})
					return fmt.Errorf("mode password not configured. Run 'chameleon config auth set-password' first")
				}

				password, passwordErr := readModePassword()
				if passwordErr != nil {
					return passwordErr
				}

				ok, verifyErr := v.VerifyModePassword(password)
				if verifyErr != nil {
					return verifyErr
				}

				if !ok {
					if journalLogger != nil {
						_ = journalLogger.Log("config_mode", "denied", map[string]interface{}{
							"reason":      "invalid_mode_password",
							"from_mode":   previousMode,
							"target_mode": targetMode,
						}, nil)
					}
					_ = v.AppendLog("MODE", "", map[string]string{
						"action": "mode_change_denied",
						"mode":   targetMode,
						"reason": "invalid_mode_password",
					})
					return fmt.Errorf("invalid mode password")
				}
			}

			if err := v.SetParanoidMode(targetMode); err != nil {
				if journalLogger != nil {
					_ = journalLogger.LogError("config_mode", err, map[string]interface{}{
						"from_mode":   previousMode,
						"target_mode": targetMode,
					})
				}
				return err
			}

			mode, err := v.GetParanoidMode()
			if err != nil {
				return err
			}

			if journalLogger != nil {
				_ = journalLogger.Log("config_mode", "success", map[string]interface{}{
					"from_mode": previousMode,
					"to_mode":   mode,
				}, nil)
			}

			printSuccess("Paranoid Mode updated: %s", mode)
			return nil
		default:
			return fmt.Errorf("unsupported key %q (supported: mode)", key)
		}
	},
}

var configAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage mode authorization settings",
}

var configAuthSetPasswordCmd = &cobra.Command{
	Use:   "set-password",
	Short: "Set or rotate admin password for mode upgrades",
	Long: `Set or rotate the local admin password required for paranoid mode upgrades
(for example: readonly -> standard, standard -> privileged, privileged -> emergency).

Tip: for non-interactive usage, set CHAMELEON_MODE_PASSWORD.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		v := vault.NewVault(workDir)
		if !v.Exists() {
			return fmt.Errorf("vault not initialized. Run 'chameleon migrate' first")
		}

		password, err := readModePasswordForSetup()
		if err != nil {
			return err
		}

		if err := v.SetModePassword(password); err != nil {
			return err
		}

		factory := admin.NewManagerFactory(workDir)
		journalLogger, _ := factory.CreateJournalLogger()
		if journalLogger != nil {
			_ = journalLogger.Log("config_mode_auth", "success", map[string]interface{}{
				"action": "password_configured",
			}, nil)
		}

		printSuccess("Mode password configured")
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get key",
	Short: "Get a local configuration value",
	Long: `Get a local configuration value.

Currently supported:
  mode

Examples:
  chameleon config get mode`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ToLower(strings.TrimSpace(args[0]))

		switch key {
		case "mode":
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			v := vault.NewVault(workDir)
			if !v.Exists() {
				return fmt.Errorf("vault not initialized. Run 'chameleon migrate' first")
			}

			mode, err := v.GetParanoidMode()
			if err != nil {
				return err
			}

			fmt.Println(mode)
			return nil
		default:
			return fmt.Errorf("unsupported key %q (supported: mode)", key)
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configAuthCmd.AddCommand(configAuthSetPasswordCmd)
	configCmd.AddCommand(configAuthCmd)
	rootCmd.AddCommand(configCmd)
}

func canonicalParanoidMode(mode string) string {
	clean := strings.ToLower(strings.TrimSpace(mode))
	if clean == "admin" {
		return "privileged"
	}

	return clean
}

func requiresModeAuth(currentMode, targetMode string) bool {
	currentRank, currentOK := paranoidModeRank[canonicalParanoidMode(currentMode)]
	targetRank, targetOK := paranoidModeRank[canonicalParanoidMode(targetMode)]
	if !currentOK || !targetOK {
		return false
	}

	return targetRank > currentRank
}

func readModePassword() (string, error) {
	if value := strings.TrimSpace(os.Getenv(modePasswordEnvVar)); value != "" {
		return value, nil
	}

	return readHiddenPassword(fmt.Sprintf("Enter mode password (or set %s env var): ", modePasswordEnvVar))
}

func readModePasswordForSetup() (string, error) {
	if value := strings.TrimSpace(os.Getenv(modePasswordEnvVar)); value != "" {
		return value, nil
	}

	return readHiddenPassword(fmt.Sprintf("Choose mode password (min %d chars): ", vault.MinModePasswordLength))
}

func readHiddenPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Println()
		return "", fmt.Errorf("interactive password input is unavailable; set %s env var", modePasswordEnvVar)
	}

	passwordBytes, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	password := strings.TrimSpace(string(passwordBytes))
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	return password, nil
}
