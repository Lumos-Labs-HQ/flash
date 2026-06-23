//go:build !dev
// +build !dev

package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lumos-Labs-HQ/flash/internal/plugin"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update installed plugins and flash CLI to the latest version",
	Long: `
Update all installed FlashORM plugins AND the flash CLI binary to the latest release.

By default this updates everything. Use --plugins-only to skip the binary update.

Examples:
  flash update              # Update plugins + flash CLI binary
  flash update --self-only  # Update only the flash CLI binary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		selfOnly, _ := cmd.Flags().GetBool("self-only")
		skipSelf, _ := cmd.Flags().GetBool("plugins-only")

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
		if len(installed) == 0 && skipSelf {
			color.Yellow("⚠️  No plugins installed.")
			fmt.Println()
			color.Cyan("💡 Install the core plugin: flash add-plug core")
			color.Cyan("💡 Install studio plugin:   flash add-plug studio")
			return nil
		}

		// Default: update both plugins and binary
		return manager.UpdateAllPlugins(!skipSelf, Version)
	},
}

func init() {
	updateCmd.Flags().Bool("plugins-only", false, "Only update plugins, skip flash CLI binary update")
	updateCmd.Flags().Bool("self-only", false, "Only update flash CLI binary, skip plugins")
}
