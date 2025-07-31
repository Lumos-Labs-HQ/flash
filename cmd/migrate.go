package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/migrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate [name]",
	Short: "Create a new migration",
	Long: `Create a new migration file with the specified name.
If no name is provided, you will be prompted to enter one.

Examples:
  graft migrate "create users table"
  graft migrate "add email index"
  graft migrate  # Interactive mode`,
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

		var migrationName string
		if len(args) > 0 {
			migrationName = strings.Join(args, " ")
		} else {
			// Interactive mode
			fmt.Print("Enter migration name: ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			migrationName = strings.TrimSpace(input)
		}

		if migrationName == "" {
			return fmt.Errorf("migration name cannot be empty")
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

		if err := m.GenerateMigration(migrationName, cfg.SchemaPath); err != nil {
			return err
		}

		// Run SQLC generate if config is available
		if cfg.SqlcConfigPath != "" {
			if _, err := os.Stat(cfg.SqlcConfigPath); err == nil {
				fmt.Println("🔧 Running SQLC generate...")
				if err := runSQLCGenerate(cfg.SqlcConfigPath); err != nil {
					fmt.Printf("⚠️  Warning: SQLC generate failed: %v\n", err)
					fmt.Println("You can run 'graft sqlc-migrate' later to generate types")
				} else {
					fmt.Println("✅ SQLC types generated successfully")
				}
			}
		}

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
	rootCmd.AddCommand(migrateCmd)
}
