package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/database"
	"github.com/spf13/cobra"
)

var rawCmd = &cobra.Command{
	Use:   "raw <sql-file>",
	Short: "Execute a raw SQL file against the database",
	Long: `Execute a raw SQL file directly against the database using the configured database adapter.
	
Examples:
  graft raw script.sql
  graft raw queries/update_users.sql`,
	Args: cobra.ExactArgs(1),
	RunE: runRaw,
}

func init() {
	rootCmd.AddCommand(rawCmd)
}

func runRaw(cmd *cobra.Command, args []string) error {
	sqlFile := args[0]

	if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
		return fmt.Errorf("SQL file not found: %s", sqlFile)
	}

	sqlContent, err := os.ReadFile(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	if len(sqlContent) == 0 {
		return fmt.Errorf("SQL file is empty: %s", sqlFile)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	adapter := database.NewAdapter(cfg.Database.Provider)

	dbURL, err := cfg.GetDatabaseURL()
	if err != nil {
		return fmt.Errorf("failed to get database URL: %w", err)
	}

	ctx := context.Background()
	if err := adapter.Connect(ctx, dbURL); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer adapter.Close()

	fmt.Printf("📄 Executing SQL file: %s\n", sqlFile)
	fmt.Printf("🎯 Database: %s\n", cfg.Database.Provider)
	fmt.Println()

	statements := splitSQLStatements(string(sqlContent))

	if len(statements) == 0 {
		return fmt.Errorf("no SQL statements found in file")
	}

	fmt.Printf("📝 Found %d SQL statement(s)\n", len(statements))
	fmt.Println()

	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		fmt.Printf("⚡ Executing statement %d...\n", i+1)

		if err := adapter.ExecuteMigration(ctx, statement); err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}

		fmt.Printf("✅ Statement %d executed successfully\n", i+1)
	}

	fmt.Println()
	fmt.Printf("🎉 All statements executed successfully!\n")
	return nil
}

func splitSQLStatements(content string) []string {
	var statements []string

	parts := strings.Split(content, ";")

	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" && !strings.HasPrefix(statement, "--") {
			statements = append(statements, statement)
		}
	}

	return statements
}
