package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "graft",
	Short: "A Prisma-like CLI tool for database migrations and schema management",
	Long: `Graft is a Go-based CLI tool that provides database migration capabilities
similar to Prisma, with support for schema comparison, backup management,
and optional SQLC integration.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is graft.config.json or graft.config.yaml)")
	rootCmd.PersistentFlags().Bool("force", false, "skip confirmations")
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find project root directory
		projectRoot, err := findProjectRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
			os.Exit(1)
		}

		// Search config in project root with names "graft.config" (without extension).
		viper.AddConfigPath(projectRoot)
		viper.SetConfigName("graft.config")
		viper.SetConfigType("json") // Default to JSON, but will try YAML too
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, check if graft is initialized
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, will be handled by individual commands
		} else {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}
}

// findProjectRoot finds the project root by looking for go.mod, package.json, or .git
func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return findProjectRootRecursive(currentDir)
}

func findProjectRootRecursive(dir string) (string, error) {
	// Check for common project indicators
	indicators := []string{"go.mod", "package.json", ".git", "graft.config.json", "graft.config.yaml"}
	
	for _, indicator := range indicators {
		if _, err := os.Stat(dir + "/" + indicator); err == nil {
			return dir, nil
		}
	}

	// Check parent directory
	parent := dir + "/.."
	if parent == dir {
		// Reached root directory
		return dir, nil
	}

	return findProjectRootRecursive(parent)
}
