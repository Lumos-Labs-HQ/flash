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
	Version = "1.6.0"
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
		"║        🚀 Database Migration CLI Tool 🚀                     ║",
		"║                                                              ║",
		"║     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓             ║",
		"║     ▓                                                ▓       ║",
		"║     ▓  Migration • Schema • Backup • SQLC           ▓        ║",
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
	Use:   "graft",
	Short: "A database migration CLI tool",
	Long: `
Graft is a Go-based CLI tool that provides database migration capabilities 
similar to Prisma, with support for schema comparison, backup management, 
and optional SQLC integration.

Features:
- Project-aware configuration management
- Database-agnostic design (currently supports PostgreSQL)
- Migration tracking and validation
- Automatic backup system
- SQLC integration`,

	Run: func(cmd *cobra.Command, args []string) {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Printf("Graft CLI version %s\n", Version)
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./graft.config.json)")
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
		viper.SetConfigName("graft.config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
