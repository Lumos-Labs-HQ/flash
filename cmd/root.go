//go:build !dev
// +build !dev

package cmd

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/plugin"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	Version = "2.3.0"
)

func showBanner() {
	greenColor := color.New(color.FgGreen, color.Bold)

	banner := []string{
		"╔══════════════════════════════════════════════════════════════╗",
		"║   	  ███████╗██╗      █████╗ ███████╗██╗  ██╗             ║",
		"║   	  ██╔════╝██║     ██╔══██╗██╔════╝██║  ██║              ║",
		"║   	  █████╗  ██║     ███████║███████╗███████║             ║",
		"║   	  ██╔══╝  ██║     ██╔══██║╚════██║██╔══██║              ║",
		"║   	  ██║     ███████╗██║  ██║███████║██║  ██║             ║",
		"║   	  ╚═╝     ╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝              ║",
		"║                                                             ║",
		"║         ⚡ Lightning-Fast Type-Safe ORM ⚡                   ║",
		"║                                                              ║",
		"║     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓            ║",
		"║     ▓                                                ▓       ║",
		"║     ▓      Go • TS • JS • Python • ORM              ▓        ║",
		"║     ▓                                                ▓       ║",
		"║     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓              ║",
		"╚══════════════════════════════════════════════════════════════╝",
	}

	for _, line := range banner {
		greenColor.Println(line)
	}

	fmt.Print("                        ")
	color.New(color.FgCyan, color.Bold).Print("Version: ")
	color.New(color.FgYellow, color.Bold).Printf("%s\n", Version)
}

var rootCmd = &cobra.Command{
	Use:   "flash",
	Short: "A type-safe ORM with code generation for Go, TypeScript, and JavaScript",
	Long: `
FlashORM is a powerful ORM and database toolkit that generates type-safe code 
from your SQL schemas and queries for multiple programming languages.

Supported Languages:
- Go (native type-safe structs and methods)
- TypeScript (with full type definitions)
- JavaScript (with JSDoc comments)
- Python (with async support)

Database Support:
- PostgreSQL (with advanced features)
- MySQL (full compatibility)
- SQLite (embedded databases)`,

	PersistentPreRunE: checkPluginRequirement,

	Run: func(cmd *cobra.Command, args []string) {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Printf("FlashORM CLI version %s\n", Version)
			os.Exit(0)
		}

		if len(args) == 0 {
			showBanner()
			fmt.Println()
			cmd.Help()
		}
	},
}

func Execute() error {
	// Check if the first argument is a plugin command
	if len(os.Args) > 1 {
		commandName := os.Args[1]

		// Skip if it's a built-in command
		builtInCommands := []string{"plugins", "add-plug", "rm-plug", "help", "completion", "--help", "-h", "--version", "-v"}
		isBuiltIn := false
		for _, cmd := range builtInCommands {
			if commandName == cmd {
				isBuiltIn = true
				break
			}
		}

		if !isBuiltIn {
			// Check if this command requires a plugin
			requiredPlugin, requiresPlugin := plugin.GetRequiredPlugin(commandName)
			if requiresPlugin {
				manager, err := plugin.NewManager()
				if err != nil {
					return fmt.Errorf("failed to initialize plugin manager: %w", err)
				}

				// Check if the exact plugin is installed
				if manager.IsPluginInstalled(requiredPlugin) {
					return manager.ExecutePlugin(requiredPlugin, os.Args[1:])
				}

				if manager.IsPluginInstalled("all") {
					return manager.ExecutePlugin("all", os.Args[1:])
				}

				color.Red("❌ Command '%s' requires plugin '%s'", commandName, requiredPlugin)
				fmt.Println()
				color.Cyan("📦 Install it using:")
				color.Cyan("   flash add-plug %s    # Install only what you need", requiredPlugin)
				color.Cyan("   flash add-plug all   # Install everything")
				return fmt.Errorf("missing required plugin: %s", requiredPlugin)
			}
		}
	}

	return rootCmd.Execute()
}

// checkPluginRequirement checks if a command requires a plugin and handles it
func checkPluginRequirement(cmd *cobra.Command, args []string) error {
	commandName := cmd.Name()
	if commandName == "flash" || commandName == "plugins" || commandName == "add-plug" ||
		commandName == "rm-plug" || commandName == "help" || commandName == "version" {
		return nil
	}

	requiredPlugin, requiresPlugin := plugin.GetRequiredPlugin(commandName)
	if !requiresPlugin {
		return nil
	}

	manager, err := plugin.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize plugin manager: %w", err)
	}

	if manager.IsPluginInstalled(requiredPlugin) {
		return manager.ExecutePlugin(requiredPlugin, os.Args[1:])
	}

	if manager.IsPluginInstalled("all") {
		return manager.ExecutePlugin("all", os.Args[1:])
	}

	color.Red("❌ Command '%s' requires plugin '%s'", commandName, requiredPlugin)
	fmt.Println()
	color.Cyan("📦 Install it using:")
	color.Cyan("   flash add-plug %s    # Install only what you need", requiredPlugin)
	color.Cyan("   flash add-plug all   # Install everything")
	return fmt.Errorf("missing required plugin: %s", requiredPlugin)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./flash.config.json)")
	rootCmd.PersistentFlags().BoolP("force", "f", false, "Skip confirmations")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.Flags().BoolP("version", "v", false, "Show CLI version")
}

func initConfig() {
	if err := godotenv.Load(); err != nil {
		godotenv.Load(".env")
		godotenv.Load(".env.local")
	}

	config.ConfigFile = cfgFile
}
