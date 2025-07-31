package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/migrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

// sqlcMigrateCmd represents the sqlc-migrate command
var sqlcMigrateCmd = &cobra.Command{
	Use:   "sqlc-migrate",
	Short: "Apply migrations and run SQLC generate",
	Long: `Apply all pending migrations and then run 'sqlc generate' to update Go types.
This command combines migration application with SQLC code generation for a 
seamless development workflow.

Requirements:
- SQLC must be installed and available in PATH
- sqlc_config_path must be set in graft.config.json
- Valid sqlc.yaml configuration file

This command will:
1. Apply all pending migrations
2. Run 'sqlc generate' to update Go types
3. Report any errors from either step`,
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

		// Check if SQLC config is set
		if cfg.SqlcConfigPath == "" {
			return fmt.Errorf("sqlc_config_path not set in configuration")
		}

		// Check if SQLC config file exists
		if _, err := os.Stat(cfg.SqlcConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("SQLC config file not found: %s", cfg.SqlcConfigPath)
		}

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

		// Apply migrations first
		log.Println("🚀 Applying migrations...")
		if err := m.Deploy(ctx); err != nil {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}

		// Run SQLC generate
		log.Println("🔧 Running SQLC generate...")
		if err := runSQLCGenerate(cfg.SqlcConfigPath); err != nil {
			return fmt.Errorf("failed to run SQLC generate: %w", err)
		}

		log.Println("🎉 Migrations applied and SQLC types generated successfully!")
		return nil
	},
}

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

	log.Println("✅ SQLC generate completed successfully")
	return nil
}

func init() {
	rootCmd.AddCommand(sqlcMigrateCmd)
}
