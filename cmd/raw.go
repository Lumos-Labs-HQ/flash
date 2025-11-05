package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Lumos-Labs-HQ/graft/internal/config"
	"github.com/Lumos-Labs-HQ/graft/internal/database"
	"github.com/spf13/cobra"
)

var rawCmd = &cobra.Command{
	Use:   "raw <sql-file>",
	Short: "Execute a raw SQL file against the database",
	Long: `
Execute a raw SQL file directly against the database using the configured database adapter.
	
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

	query := strings.TrimSpace(string(sqlContent))

	// Check if it's a SELECT query or other query that returns data
	queryUpper := strings.ToUpper(query)
	isSelectQuery := strings.HasPrefix(queryUpper, "SELECT") ||
		strings.HasPrefix(queryUpper, "SHOW") ||
		strings.HasPrefix(queryUpper, "DESCRIBE") ||
		strings.HasPrefix(queryUpper, "EXPLAIN") ||
		strings.HasPrefix(queryUpper, "WITH")

	if isSelectQuery {
		// Execute as query and display results
		fmt.Println("⚡ Executing query...")
		result, err := adapter.ExecuteQuery(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}

		if len(result.Rows) == 0 {
			fmt.Println("✅ Query executed successfully")
			fmt.Println("📊 No rows returned")
			return nil
		}

		// Display results in a formatted table
		fmt.Printf("✅ Query executed successfully\n")
		fmt.Printf("📊 %d row(s) returned\n\n", len(result.Rows))

		displayResultsTable(result.Columns, result.Rows)
	} else {
		// Execute as migration for non-SELECT queries
		statements := splitSQLStatements(query)

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
	}

	return nil
}

// displayResultsTable displays query results in a formatted table
func displayResultsTable(columns []string, rows []map[string]interface{}) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make(map[string]int)
	for _, col := range columns {
		colWidths[col] = len(col)
	}

	for _, row := range rows {
		for _, col := range columns {
			val := formatValue(row[col])
			if len(val) > colWidths[col] {
				colWidths[col] = len(val)
			}
		}
	}

	// Print header
	fmt.Print("┌")
	for i, col := range columns {
		fmt.Print(strings.Repeat("─", colWidths[col]+2))
		if i < len(columns)-1 {
			fmt.Print("┬")
		}
	}
	fmt.Println("┐")

	fmt.Print("│")
	for _, col := range columns {
		fmt.Printf(" %-*s │", colWidths[col], col)
	}
	fmt.Println()

	fmt.Print("├")
	for i, col := range columns {
		fmt.Print(strings.Repeat("─", colWidths[col]+2))
		if i < len(columns)-1 {
			fmt.Print("┼")
		}
	}
	fmt.Println("┤")

	// Print rows
	for _, row := range rows {
		fmt.Print("│")
		for _, col := range columns {
			val := formatValue(row[col])
			fmt.Printf(" %-*s │", colWidths[col], val)
		}
		fmt.Println()
	}

	// Print footer
	fmt.Print("└")
	for i, col := range columns {
		fmt.Print(strings.Repeat("─", colWidths[col]+2))
		if i < len(columns)-1 {
			fmt.Print("┴")
		}
	}
	fmt.Println("┘")
}

// formatValue formats a value for display
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", val)
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
