package cmd

import (
	"fmt"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/db"
	"Rana718/Graft/internal/migration"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current migration status",
	Long: `Show the current migration status including applied and pending migrations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("graft is not initialized. Run 'graft init' first")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Connect to database
		conn, err := db.NewConnection(cfg)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer conn.Close()

		// Create migrations table if it doesn't exist
		if err := conn.CreateMigrationsTable(); err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}

		// Get migration manager
		migrationManager, err := migration.NewManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migration manager: %w", err)
		}

		// Get applied migrations
		appliedMigrations, err := conn.GetAppliedMigrations()
		if err != nil {
			return fmt.Errorf("failed to get applied migrations: %w", err)
		}

		// Get all local migrations
		localMigrations, err := migrationManager.GetLocalMigrations()
		if err != nil {
			return fmt.Errorf("failed to get local migrations: %w", err)
		}

		// Get pending migrations
		pendingMigrations, err := migrationManager.GetPendingMigrations(appliedMigrations)
		if err != nil {
			return fmt.Errorf("failed to get pending migrations: %w", err)
		}

		fmt.Printf("📊 Migration Status\n")
		fmt.Printf("==================\n\n")

		fmt.Printf("📁 Local migrations: %d\n", len(localMigrations))
		fmt.Printf("✅ Applied migrations: %d\n", len(appliedMigrations))
		fmt.Printf("⏳ Pending migrations: %d\n\n", len(pendingMigrations))

		if len(appliedMigrations) > 0 {
			fmt.Printf("✅ Applied Migrations:\n")
			for _, migration := range appliedMigrations {
				fmt.Printf("  - %s\n", migration)
			}
			fmt.Println()
		}

		if len(pendingMigrations) > 0 {
			fmt.Printf("⏳ Pending Migrations:\n")
			for _, migration := range pendingMigrations {
				fmt.Printf("  - %s\n", migration.Name)
			}
			fmt.Println()
			fmt.Printf("💡 Run 'graft apply' to apply pending migrations\n")
		} else {
			fmt.Printf("🎉 All migrations are up to date!\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
