package cmd

import (
	"fmt"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/db"
	"Rana718/Graft/internal/migration"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Push local migrations to the database without running them",
	Long: `Push local migrations to the database by recording them in the migrations table
without actually executing the SQL. This is useful for marking migrations as applied
when they have been run manually or through other means.`,
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

		// Get pending migrations
		pendingMigrations, err := migrationManager.GetPendingMigrations(appliedMigrations)
		if err != nil {
			return fmt.Errorf("failed to get pending migrations: %w", err)
		}

		if len(pendingMigrations) == 0 {
			fmt.Println("✅ No pending migrations to deploy")
			return nil
		}

		fmt.Printf("📋 Deploying %d migration(s) (recording without execution):\n", len(pendingMigrations))

		// Record migrations without executing them
		for _, migration := range pendingMigrations {
			fmt.Printf("📝 Recording migration: %s\n", migration.Name)

			// Validate migration
			if err := migrationManager.ValidateMigration(migration); err != nil {
				return fmt.Errorf("migration validation failed: %w", err)
			}

			// Record migration without executing
			if err := conn.RecordMigration(migration.Name, migration.Checksum); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
			}

			fmt.Printf("✅ Recorded migration: %s\n", migration.Name)
		}

		fmt.Printf("\n🎉 Successfully deployed %d migration(s)\n", len(pendingMigrations))
		fmt.Printf("⚠️  Note: Migrations were recorded but not executed. Make sure to run the SQL manually if needed.\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
