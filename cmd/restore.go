package cmd

import (
	"context"
	"fmt"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/migrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore database from backup",
	Long: `Restore the database from a previously created backup file.
This is a destructive operation that will overwrite all existing data.

The backup file should be a JSON file created by the 'graft backup' command.

⚠️  WARNING: This will overwrite all existing data in your database!

Examples:
  graft restore db_backup/backup_2024-01-15_10-30-00.json
  graft restore --force backup.json  # Skip confirmation prompts`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		if err := cfg.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		backupFile := args[0]

		// Connect to database
		dbURL, err := cfg.GetDatabaseURL()
		if err != nil {
			return err
		}

		ctx := context.Background()
		config, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			return fmt.Errorf("failed to parse database URL: %w", err)
		}

		config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

		db, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to create connection pool: %w", err)
		}
		defer db.Close()

		if err := db.Ping(ctx); err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")
		m := migrator.NewMigrator(db, cfg.MigrationsPath, cfg.BackupPath, force)

		return m.Restore(ctx, backupFile)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
