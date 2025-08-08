package cmd

import (
	"context"
	"fmt"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/migrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the database",
	Long: `Reset the database by dropping all tables and data.
	This is a destructive operation that will:

	1. Prompt for confirmation (unless --force is used)
	2. Offer to create a backup before reset
	3. Drop all tables in the database
	4. Optionally remove migration files

	⚠️  WARNING: This will permanently delete all data in your database!

	Use --force to skip all confirmation prompts.`,
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

		return m.Reset(ctx)
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
