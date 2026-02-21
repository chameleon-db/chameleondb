package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall ChameleonDB from this system",
	Long: `Safely removes ChameleonDB binaries and libraries.

This command will NOT remove:
 - Project files
 - Databases
 - User configuration`,
	Run: runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) {
	binPath := "/usr/local/bin/chameleon"
	libDir := "/usr/local/lib"

	binaryExists := fileExists(binPath)
	libs, _ := filepath.Glob(filepath.Join(libDir, "libchameleon_core*"))

	if !binaryExists && len(libs) == 0 {
		fmt.Println("ChameleonDB is not installed.")
		return
	}

	fmt.Println("🦎 ChameleonDB Uninstaller")
	fmt.Println("──────────────────────────")
	fmt.Println()

	fmt.Println("Detected components:")
	if binaryExists {
		fmt.Println(" - Binary:   ", binPath)
	}
	for _, lib := range libs {
		fmt.Println(" - Library:  ", lib)
	}

	fmt.Println()
	fmt.Println("This will NOT remove:")
	fmt.Println(" - Project files")
	fmt.Println(" - Databases")
	fmt.Println(" - User configuration (~/.chameleon)")
	fmt.Println()

	if !confirm("Proceed with uninstall? [y/N]: ") {
		fmt.Println("Uninstall cancelled.")
		return
	}

	// Remove binary
	if err := os.Remove(binPath); err != nil {
		fmt.Println("❌ Cannot remove binary (permission denied)")
		fmt.Println()
		fmt.Println("ChameleonDB was installed system-wide.")
		fmt.Println("Please re-run uninstall with elevated privileges:")
		fmt.Println()
		fmt.Println("  sudo chameleon uninstall")
		return
	}

	// Remove libraries
	for _, lib := range libs {
		if err := os.Remove(lib); err != nil {
			fmt.Println("❌ Failed to remove library:", lib)
			fmt.Println("Try manually: sudo rm", lib)
			return
		}
	}

	fmt.Println()
	fmt.Println("✔ ChameleonDB uninstalled successfully")
	fmt.Println()
	fmt.Println("If you want to come back, we’ll be waiting 🦎")
	fmt.Println("👉 curl -sSL https://chameleondb.dev/install | sh")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
