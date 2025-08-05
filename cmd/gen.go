package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"Rana718/Graft/internal/config"

	"github.com/spf13/cobra"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate SQLC types",
	Long: `Generate Go types from SQL queries using SQLC.
This command runs 'sqlc generate' to update Go types based on your SQL schemas and queries.

Requirements:
- SQLC must be installed and available in PATH
- sqlc_config_path must be set in graft.config.json
- Valid sqlc.yaml configuration file

This command will:
1. Run 'sqlc generate' to update Go types
2. Report any errors from the generation process`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check if SQLC config is set
		if cfg.SqlcConfigPath == "" {
			return fmt.Errorf("sqlc_config_path not set in configuration")
		}

		// Check if SQLC config file exists
		if _, err := os.Stat(cfg.SqlcConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("SQLC config file not found: %s", cfg.SqlcConfigPath)
		}

		// Run SQLC generate
		if err := runSQLCGenerate(cfg.SqlcConfigPath); err != nil {
			return fmt.Errorf("failed to run SQLC generate: %w", err)
		}

		fmt.Println("🎉 SQLC types generated successfully!")
		return nil
	},
}

// runSQLCGenerate runs sqlc generate with the specified config
func runSQLCGenerate(configPath string) error {
	// Check if sqlc is available
	if _, err := exec.LookPath("sqlc"); err != nil {
		return fmt.Errorf("sqlc not found in PATH. Please install SQLC: https://docs.sqlc.dev/en/latest/overview/install.html")
	}

	// Run sqlc generate with the specified config
	cmd := exec.Command("sqlc", "generate", "-f", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlc generate failed: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(genCmd)
}
