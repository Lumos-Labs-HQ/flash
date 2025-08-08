package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/migrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup [comment]",
	Short: "Create a database backup",
	Long: `Create a manual backup of the database.
	The backup includes all table data and migration history in JSON format.

	The backup will be saved in the backup directory specified in your config
	with a timestamp-based filename.

	Examples:
	  graft backup "before major update"
	  graft backup "pre-production backup"
	  graft backup  # Creates backup with default comment`,
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

		var comment string
		if len(args) > 0 {
			comment = strings.Join(args, " ")
		} else {
			comment = "Manual backup"
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

		return m.Backup(ctx, comment)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
