package cmd

import (
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/plugin"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var removePluginCmd = &cobra.Command{
	Use:     "rm-plug [plugin-name]",
	Aliases: []string{"remove-plug"},
	Short:   "Remove a FlashORM plugin",
	Long: `
Remove an installed FlashORM plugin.

Examples:
  flash rm-plug migration    # Remove migration plugin
  flash rm-plug studio       # Remove studio plugin`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		// Initialize plugin manager
		manager, err := plugin.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize plugin manager: %w", err)
		}

		// Check if plugin is installed
		if !manager.IsPluginInstalled(pluginName) {
			color.Yellow("⚠️  Plugin '%s' is not installed", pluginName)
			fmt.Println()

			// Show installed plugins
			plugins := manager.ListPlugins()
			if len(plugins) > 0 {
				color.White("Installed plugins:")
				for _, p := range plugins {
					fmt.Printf("  • %s (v%s)\n", color.CyanString(p.Name), p.Version)
				}
			} else {
				color.White("No plugins are currently installed")
			}

			return nil
		}

		// Get plugin info before removal
		info, _ := manager.GetPluginInfo(pluginName)

		// Confirm removal unless --force flag is set
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			color.Yellow("⚠️  This will remove plugin '%s' (v%s)", pluginName, info.Version)
			color.Yellow("   Commands that will become unavailable: %v", info.Commands)
			fmt.Println()

			var confirm string
			fmt.Print("Continue? (y/N): ")
			fmt.Scanln(&confirm)

			if confirm != "y" && confirm != "Y" {
				color.Cyan("❌ Cancelled")
				return nil
			}
		}

		// Remove plugin
		if err := manager.RemovePlugin(pluginName); err != nil {
			return fmt.Errorf("failed to remove plugin: %w", err)
		}

		return nil
	},
}

func init() {
	removePluginCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}
