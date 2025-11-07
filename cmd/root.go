package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	Version = "2.0.0"
)

func showBanner() {
	greenColor := color.New(color.FgGreen, color.Bold)

	banner := []string{
		"╔══════════════════════════════════════════════════════════════╗",
		"║   ██████╗ ██████╗  █████╗ ███████╗████████╗                  ║",
		"║  ██╔════╝ ██╔══██╗██╔══██╗██╔════╝╚══██╔══╝                  ║",
		"║  ██║  ███╗██████╔╝███████║█████╗     ██║                     ║",
		"║  ██║   ██║██╔══██╗██╔══██║██╔══╝     ██║                     ║",
		"║  ╚██████╔╝██║  ██║██║  ██║██║        ██║                     ║",
		"║   ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝        ╚═╝                     ║",
		"║                                                              ║",
		"║      🚀 Type-Safe ORM & Migration Tool 🚀                    ║",
		"║                                                              ║",
		"║     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓             ║",
		"║     ▓                                                ▓       ║",
		"║     ▓  Go • TypeScript • JavaScript • Migration    ▓        ║",
		"║     ▓                                                ▓       ║",
		"║     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓             ║",
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

Features:
- 🎯 Type-safe code generation for Go, TypeScript, and JavaScript
- 📦 Schema-first approach with SQL migrations
- 🔄 Automatic migration tracking and versioning
- 🎨 Built-in database studio for visual data management
- 🚀 Query-based code generation (write SQL, get type-safe functions)
- 💾 Backup and restore capabilities
- 🔌 Support for PostgreSQL, MySQL, and SQLite
- 🛠️ CLI tools for complete database workflow

Supported Languages:
- Go (native type-safe structs and methods)
- TypeScript (with full type definitions)
- JavaScript (with JSDoc comments)

Database Support:
- PostgreSQL (with advanced features)
- MySQL (full compatibility)
- SQLite (embedded databases)`,

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
	return rootCmd.Execute()
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

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName("flash.config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
