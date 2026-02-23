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
  schema-paths=path1,path2,...  (requires privileged mode)

Examples:
  chameleon config set mode=standard
  chameleon config set schema-paths=schemas/
  chameleon config set schema-paths=schemas/,legacy/schemas/`,
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

		case "schema-paths":
			v := vault.NewVault(workDir)
			if !v.Exists() {
				if journalLogger != nil {
					_ = journalLogger.Log("config_schema_paths", "failed", map[string]interface{}{
						"reason": "vault_not_initialized",
						"error":  "vault not initialized",
					}, nil)
				}
				return fmt.Errorf("vault not initialized. Run 'chameleon migrate' first")
			}

			// Check current mode
			currentMode, err := v.GetParanoidMode()
			if err != nil {
				if journalLogger != nil {
					_ = journalLogger.LogError("config_schema_paths", err, map[string]interface{}{
						"action": "read_paranoid_mode",
					})
				}
				return err
			}

			// Log attempt to change schema paths
			if journalLogger != nil {
				_ = journalLogger.Log("config_schema_paths", "attempt", map[string]interface{}{
					"current_mode":    currentMode,
					"requested_paths": value,
				}, nil)
			}

			// Require privileged or emergency mode
			if currentMode != "privileged" && currentMode != "emergency" {
				if journalLogger != nil {
					_ = journalLogger.Log("config_schema_paths", "denied", map[string]interface{}{
						"reason":       "insufficient_privileges",
						"current_mode": currentMode,
						"required":     "privileged or emergency",
					}, nil)
				}

				printError("Permission denied: schema path changes require elevated privileges")
				return fmt.Errorf(
					"Changing schema paths requires privileged or emergency mode.\n"+
						"Current mode: %s\n"+
						"Upgrade with: chameleon config set mode=privileged",
					currentMode,
				)
			}

			// Verify path exists and is not a symlink
			pathsToCheck := strings.Split(value, ",")
			for _, p := range pathsToCheck {
				p = strings.TrimSpace(p)
				if _, err := os.Stat(p); err != nil {
					if journalLogger != nil {
						_ = journalLogger.Log("config_schema_paths", "failed", map[string]interface{}{
							"reason":       "path_not_found",
							"path":         p,
							"current_mode": currentMode,
						}, nil)
					}
					return fmt.Errorf("path not found: %s", p)
				}

				// Check for symlink (security risk)
				info, _ := os.Lstat(p)
				if info != nil && (info.Mode()&os.ModeSymlink) != 0 {
					if journalLogger != nil {
						_ = journalLogger.Log("config_schema_paths", "failed", map[string]interface{}{
							"reason":       "symlink_not_allowed",
							"path":         p,
							"current_mode": currentMode,
						}, nil)
					}
					return fmt.Errorf("symlinks are not allowed for schema paths (security risk): %s", p)
				}
			}

			// Extra confirmation for emergency mode (ultra-dangerous)
			if currentMode == "emergency" {
				printWarning("🚨 EMERGENCY MODE: This is extremely dangerous!")
				printWarning("You are about to change schema source paths in emergency mode.")
				printWarning("This can permanently break your database integrity.")
				fmt.Println()
				fmt.Print("Type 'I understand the risks' to continue: ")

				var emergencyConfirm string
				fmt.Scanln(&emergencyConfirm)

				if emergencyConfirm != "I understand the risks" {
					if journalLogger != nil {
						_ = journalLogger.Log("config_schema_paths", "cancelled", map[string]interface{}{
							"reason":       "emergency_confirmation_not_given",
							"current_mode": currentMode,
						}, nil)
					}
					printInfo("Cancelled: emergency confirmation not given")
					return nil
				}
			}

			// Warn user
			printWarning("⚠️  Changing schema source paths!")
			printInfo("This is a CRITICAL security change")
			printInfo("New paths: %s", value)
			fmt.Println()
			fmt.Print("Continue? [y/N]: ")

			var response string
			fmt.Scanln(&response)

			if response != "y" && response != "Y" {
				if journalLogger != nil {
					_ = journalLogger.Log("config_schema_paths", "cancelled", map[string]interface{}{
						"reason":       "user_confirmation_declined",
						"current_mode": currentMode,
					}, nil)
				}
				printInfo("Cancelled")
				return nil
			}

			// Load and update .chameleon.yml
			configLoader := factory.CreateConfigLoader()
			cfg, err := configLoader.Load()
			if err != nil {
				if journalLogger != nil {
					_ = journalLogger.LogError("config_schema_paths", err, map[string]interface{}{
						"action":       "load_config",
						"current_mode": currentMode,
					})
				}
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Update schema paths in config
			newPaths := []string{}
			for _, p := range pathsToCheck {
				newPaths = append(newPaths, strings.TrimSpace(p))
			}
			cfg.Schema.Paths = newPaths

			// Save config
			if err := configLoader.Save(cfg); err != nil {
				if journalLogger != nil {
					_ = journalLogger.LogError("config_schema_paths", err, map[string]interface{}{
						"action":       "save_config",
						"current_mode": currentMode,
					})
				}
				return fmt.Errorf("failed to save config: %w", err)
			}

			// Log successful schema path change (CRITICAL event to both journal and vault)
			if journalLogger != nil {
				_ = journalLogger.Log("config_schema_paths", "changed", map[string]interface{}{
					"new_paths": value,
					"mode":      currentMode,
				}, nil)
			}

			_ = v.AppendLog("SCHEMA_PATH", "", map[string]string{
				"action":    "schema_paths_changed",
				"new_paths": value,
				"mode":      currentMode,
			})

			printSuccess("Schema paths updated: %s", value)
			printWarning("This change is logged in integrity.log")
			printInfo("Run 'chameleon migrate' to apply changes")

			return nil
		default:
			return fmt.Errorf("unsupported key %q (supported: mode, schema-paths)", key)
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
  schema-paths

Examples:
  chameleon config get mode
  chameleon config get schema-paths`,
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

		case "schema-paths":
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			factory := admin.NewManagerFactory(workDir)
			configLoader := factory.CreateConfigLoader()
			cfg, err := configLoader.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Println(strings.Join(cfg.Schema.Paths, ","))
			return nil
		default:
			return fmt.Errorf("unsupported key %q (supported: mode, schema-paths)", key)
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
