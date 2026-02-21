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
	includeDir := "/usr/local/include"
	pkgConfigDir := "/usr/local/lib/pkgconfig"

	binaryExists := fileExists(binPath)
	libsCurrent, _ := filepath.Glob(filepath.Join(libDir, "libchameleon*"))
	libsLegacy, _ := filepath.Glob(filepath.Join(libDir, "libchameleon_core*"))
	libs := make([]string, 0, len(libsCurrent)+len(libsLegacy))
	seen := make(map[string]struct{})
	for _, lib := range append(libsCurrent, libsLegacy...) {
		if _, ok := seen[lib]; ok {
			continue
		}
		seen[lib] = struct{}{}
		libs = append(libs, lib)
	}
	headerPath := filepath.Join(includeDir, "chameleon.h")
	pkgConfigPath := filepath.Join(pkgConfigDir, "chameleon.pc")
	headerExists := fileExists(headerPath)
	pkgConfigExists := fileExists(pkgConfigPath)

	if !binaryExists && len(libs) == 0 && !headerExists && !pkgConfigExists {
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
	if headerExists {
		fmt.Println(" - Header:   ", headerPath)
	}
	if pkgConfigExists {
		fmt.Println(" - PkgConf:  ", pkgConfigPath)
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

	if headerExists {
		if err := os.Remove(headerPath); err != nil {
			fmt.Println("❌ Failed to remove header:", headerPath)
			fmt.Println("Try manually: sudo rm", headerPath)
			return
		}
	}

	if pkgConfigExists {
		if err := os.Remove(pkgConfigPath); err != nil {
			fmt.Println("❌ Failed to remove pkg-config file:", pkgConfigPath)
			fmt.Println("Try manually: sudo rm", pkgConfigPath)
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
