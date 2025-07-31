package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "graft",
	Short: "A database migration CLI tool",
	Long: `Graft is a Go-based CLI tool that provides database migration capabilities 
similar to Prisma, with support for schema comparison, backup management, 
and optional SQLC integration.

Features:
- Project-aware configuration management
- Database-agnostic design (currently supports PostgreSQL)
- Migration tracking and validation
- Automatic backup system
- SQLC integration`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./graft.config.json)")
	rootCmd.PersistentFlags().BoolP("force", "f", false, "Skip confirmations")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load .env file if it exists (silently ignore if not found)
	if err := godotenv.Load(); err != nil {
		// Try loading from common locations
		godotenv.Load(".env")
		godotenv.Load(".env.local")
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config file in current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName("graft.config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
