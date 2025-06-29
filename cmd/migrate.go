package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/migration"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate [migration_name]",
	Short: "Create a new migration",
	Long: `Create a new migration file with the specified name.
If no name is provided, you will be prompted to enter one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("graft is not initialized. Run 'graft init' first")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var migrationName string
		if len(args) > 0 {
			migrationName = strings.Join(args, " ")
		} else {
			fmt.Print("Enter migration name: ")
			reader := bufio.NewReader(os.Stdin)
			migrationName, err = reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read migration name: %w", err)
			}
			migrationName = strings.TrimSpace(migrationName)
		}

		if migrationName == "" {
			return fmt.Errorf("migration name cannot be empty")
		}

		migrationManager, err := migration.NewManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migration manager: %w", err)
		}

		_, err = migrationManager.CreateMigration(migrationName)
		if err != nil {
			return fmt.Errorf("failed to create migration: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
