//go:build !dev
// +build !dev

package cmd

import (
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/plugin"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update installed plugins and/or flash itself to the latest version",
	Long: `
Update all installed FlashORM plugins and optionally the flash CLI itself.

By default this updates all installed plugins. Use --self to also replace
the running flash binary with the latest release from GitHub.

Examples:
  flash update              # Update all installed plugins
  flash update --self       # Update plugins + flash CLI binary
  flash update --self-only  # Update only the flash CLI binary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		selfOnly, _ := cmd.Flags().GetBool("self-only")
		includeSelf, _ := cmd.Flags().GetBool("self")

		manager, err := plugin.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize plugin manager: %w", err)
		}

		if selfOnly {
			color.Cyan("⬆️  Updating flash CLI binary...")
			fmt.Println()
			return manager.UpdateFlashBinary(Version)
		}

		color.Cyan("⬆️  Checking for updates...")
		fmt.Println()

		// Show current installed plugins
		installed := manager.ListPlugins()
		if len(installed) == 0 && !includeSelf {
			color.Yellow("⚠️  No plugins installed.")
			fmt.Println()
			color.Cyan("💡 Install the core plugin: flash add-plug core")
			color.Cyan("💡 Install studio plugin:   flash add-plug studio")
			return nil
		}

		return manager.UpdateAllPlugins(includeSelf, Version)
	},
}

func init() {
	updateCmd.Flags().Bool("self", false, "Also update the flash CLI binary itself")
	updateCmd.Flags().Bool("self-only", false, "Only update flash CLI binary, skip plugins")
}
