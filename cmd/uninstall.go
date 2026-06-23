package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall FlashORM CLI and remove all data",
	Long:  "Removes the flash binary and the ~/.flash directory containing plugins and cache.",
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Print("This will remove the flash binary and ~/.flash directory. Continue? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Cancelled.")
				return
			}
		}

		home, err := os.UserHomeDir()
		if err != nil {
			color.Red("Failed to get home directory: %v", err)
			os.Exit(1)
		}

		// Remove ~/.flash directory
		flashDir := filepath.Join(home, ".flash")
		if _, err := os.Stat(flashDir); err == nil {
			if err := os.RemoveAll(flashDir); err != nil {
				color.Red("Failed to remove %s: %v", flashDir, err)
			} else {
				color.Green("✓ Removed %s", flashDir)
			}
		}

		// Remove the binary itself
		exe, err := os.Executable()
		if err == nil {
			exe, _ = filepath.EvalSymlinks(exe)
			if err := os.Remove(exe); err != nil {
				color.Yellow("⚠ Could not remove binary %s: %v", exe, err)
				color.Yellow("  Remove it manually: rm %s", exe)
			} else {
				color.Green("✓ Removed %s", exe)
			}
		}

		fmt.Println()
		color.Green("FlashORM has been uninstalled.")
	},
}
