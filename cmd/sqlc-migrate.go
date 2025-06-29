package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/db"
	"Rana718/Graft/internal/migration"
	"github.com/spf13/cobra"
)

// sqlcMigrateCmd represents the sqlc-migrate command
var sqlcMigrateCmd = &cobra.Command{
	Use:   "sqlc-migrate",
	Short: "Apply all pending migrations and run SQLC generate",
	Long: `Apply all pending migrations to the database and automatically run SQLC generate.
This is a convenience command that combines 'graft apply' with SQLC code generation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("graft is not initialized. Run 'graft init' first")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.SQLCConfigPath == "" {
			return fmt.Errorf("SQLC config path not configured. Please set 'sqlc_config_path' in your graft.config.json")
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
			fmt.Println("✅ No pending migrations to apply")
			fmt.Println("🔄 Running SQLC generate...")
			if err := runSQLCGenerate(); err != nil {
				return fmt.Errorf("SQLC generate failed: %w", err)
			}
			fmt.Println("✅ SQLC generate completed")
			return nil
		}

		fmt.Printf("📋 Found %d pending migration(s):\n", len(pendingMigrations))
		for _, migration := range pendingMigrations {
			fmt.Printf("  - %s\n", migration.Name)
		}

		// Check for force flag
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print("\nDo you want to apply these migrations and run SQLC generate? (yes/no): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "yes" && response != "y" {
				fmt.Println("❌ Operation cancelled")
				return nil
			}
		}

		// Apply migrations
		for _, migration := range pendingMigrations {
			fmt.Printf("🔄 Applying migration: %s\n", migration.Name)

			// Validate migration
			if err := migrationManager.ValidateMigration(migration); err != nil {
				return fmt.Errorf("migration validation failed: %w", err)
			}

			// Execute migration
			if err := conn.ExecuteSQL(migration.Content); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
			}

			// Record migration
			if err := conn.RecordMigration(migration.Name, migration.Checksum); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
			}

			fmt.Printf("✅ Applied migration: %s\n", migration.Name)
		}

		fmt.Printf("\n🎉 Successfully applied %d migration(s)\n", len(pendingMigrations))

		// Run SQLC generate
		fmt.Println("🔄 Running SQLC generate...")
		if err := runSQLCGenerate(); err != nil {
			return fmt.Errorf("SQLC generate failed: %w", err)
		}
		fmt.Println("✅ SQLC generate completed")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sqlcMigrateCmd)
}
