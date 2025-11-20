package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Lumos-Labs-HQ/flash/internal/plugin"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "List installed plugins",
	Long: `
List all installed FlashORM plugins with their versions and available commands.

Examples:
  flash plugins           # List installed plugins
  flash plugins --online  # List all available plugins from GitHub`,
	RunE: func(cmd *cobra.Command, args []string) error {
		online, _ := cmd.Flags().GetBool("online")

		manager, err := plugin.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize plugin manager: %w", err)
		}

		// Online mode - show available plugins from GitHub
		if online {
			return showOnlinePlugins(manager)
		}

		// Local mode - show installed plugins
		plugins := manager.ListPlugins()

		if len(plugins) == 0 {
			color.Yellow("📦 No plugins installed")
			fmt.Println()
			color.Cyan("💡 Install plugins using: flash add-plug <plugin-name>")
			fmt.Println()
			color.White("Available plugins:")
			for _, name := range plugin.GetAllPlugins() {
				fmt.Printf("  • %s - %s\n", color.GreenString(name), plugin.GetPluginDescription(name))
			}
			fmt.Println()
			color.Cyan("💡 Check online plugins: flash plugins --online")
			return nil
		}

		color.Green("📦 Installed Plugins (%d)", len(plugins))
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tINSTALLED\tSIZE\tCOMMANDS")
		fmt.Fprintln(w, "----\t-------\t---------\t----\t--------")

		for _, p := range plugins {
			installDate := p.InstallDate.Format("2006-01-02")
			size := formatSize(p.Size)
			commands := fmt.Sprintf("%d commands", len(p.Commands))

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				color.CyanString(p.Name),
				color.GreenString(p.Version),
				installDate,
				size,
				commands,
			)
		}
		w.Flush()

		fmt.Println()
		color.Cyan("💡 Add more plugins: flash add-plug <plugin-name>")
		color.Cyan("💡 Remove plugins: flash rm-plug <plugin-name>")
		color.Cyan("💡 Check online plugins: flash plugins --online")

		return nil
	},
}

func init() {
	pluginsCmd.Flags().BoolP("online", "o", false, "Show all available plugins from GitHub repository")
}

// showOnlinePlugins displays all available plugins from GitHub with their status
func showOnlinePlugins(manager *plugin.Manager) error {
	color.Cyan("🌐 Fetching available plugins from GitHub...")
	fmt.Println()

	availablePlugins, err := manager.FetchAvailablePlugins()
	if err != nil {
		color.Red("❌ Failed to fetch plugins from GitHub: %v", err)
		fmt.Println()
		color.Yellow("Showing local plugin information instead:")
		fmt.Println()

		// Fallback to local metadata
		showLocalPluginMetadata(manager)
		return nil
	}

	if len(availablePlugins) == 0 {
		color.Yellow("⚠️  No plugin binaries found in the latest GitHub release")
		fmt.Println()
		color.White("This might mean:")
		fmt.Println("  • Plugins haven't been built and uploaded to releases yet")
		fmt.Println("  • The release workflow needs to be configured")
		fmt.Println("  • Plugin binaries are named differently than expected")
		fmt.Println()
		color.Cyan("💡 Build and upload plugins using: make build-plugins")
		fmt.Println()

		// Show what plugins are defined locally
		color.Yellow("📝 Locally defined plugins (not yet in releases):")
		fmt.Println()
		showLocalPluginMetadata(manager)
		return nil
	}

	color.Green("📦 Available Plugins (%d)", len(availablePlugins))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tVERSION (Latest)\tDESCRIPTION")
	fmt.Fprintln(w, "----\t------\t----------------\t-----------")

	for _, ap := range availablePlugins {
		status := color.RedString("Not Installed")
		if manager.IsPluginInstalled(ap.Name) {
			localInfo, _ := manager.GetPluginInfo(ap.Name)
			if localInfo.Version == ap.Version {
				status = color.GreenString("✓ Installed")
			} else {
				status = color.YellowString("⚠ Update Available")
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			color.CyanString(ap.Name),
			status,
			ap.Version,
			ap.Description,
		)
	}
	w.Flush()

	fmt.Println()
	color.White("📝 Plugin Details:")
	for _, ap := range availablePlugins {
		fmt.Printf("\n  %s\n", color.CyanString(ap.Name))
		fmt.Printf("    Description: %s\n", ap.Description)
		fmt.Printf("    Commands: %s\n", color.GreenString(fmt.Sprintf("%d commands", len(ap.Commands))))
		fmt.Printf("    Latest Version: %s\n", ap.Version)

		if manager.IsPluginInstalled(ap.Name) {
			localInfo, _ := manager.GetPluginInfo(ap.Name)
			fmt.Printf("    Installed Version: %s\n", localInfo.Version)
			if localInfo.Version != ap.Version {
				color.Yellow("    → Update available: flash add-plug %s", ap.Name)
			}
		} else {
			color.Cyan("    → Install: flash add-plug %s", ap.Name)
		}
	}

	fmt.Println()

	return nil
}

// showLocalPluginMetadata shows plugin metadata from local registry
func showLocalPluginMetadata(manager *plugin.Manager) {
	allPlugins := plugin.GetAllPlugins()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tDESCRIPTION")
	fmt.Fprintln(w, "----\t------\t-----------")

	for _, name := range allPlugins {
		status := color.RedString("Not Installed")
		if manager.IsPluginInstalled(name) {
			status = color.GreenString("✓ Installed")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			color.CyanString(name),
			status,
			plugin.GetPluginDescription(name),
		)
	}
	w.Flush()

	fmt.Println()
	color.Cyan("💡 Install a plugin: flash add-plug <plugin-name>")
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
