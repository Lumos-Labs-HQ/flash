package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/migrator"
	"github.com/Rana718/Graft/internal/types"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull database schema to update local schema file",
	Long: `Pull the current database schema and update the local schema file.
	
	This command introspects the current database and generates a schema.sql file
	that reflects the current state of the database. This is useful for:
	- Synchronizing your local schema with the database after manual changes
	- Creating an initial schema file from an existing database
	- Keeping your schema file up-to-date with the current database state

	The command will:
	1. Connect to the database
	2. Introspect all tables, columns, and constraints
	3. Generate a comprehensive schema.sql file
	4. Optionally create a backup of the existing schema file`,

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

		ctx := context.Background()

		force, _ := cmd.Flags().GetBool("force")
		backup, _ := cmd.Flags().GetBool("backup")
		// includeIndexes, _ := cmd.Flags().GetBool("indexes")
		outputPath, _ := cmd.Flags().GetString("output")

		// Use custom output path if provided
		schemaPath := cfg.SchemaPath
		if outputPath != "" {
			schemaPath = outputPath
		}

		fmt.Println("🔍 Introspecting database schema...")

		// Create migrator instance
		m, err := migrator.NewMigrator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migrator: %w", err)
		}
		defer m.Close()

		// Pull the current schema from database
		schema, err := pullDatabaseSchema(ctx, m)
		if err != nil {
			return fmt.Errorf("failed to pull database schema: %w", err)
		}

		if len(schema) == 0 {
			fmt.Println("📄 No tables found in database (excluding migration tables)")
			return nil
		}

		// Backup existing schema file if requested
		if backup {
			if err := backupExistingSchema(schemaPath); err != nil {
				fmt.Printf("⚠️  Warning: Failed to backup existing schema: %v\n", err)
			} else {
				fmt.Println("💾 Backed up existing schema file")
			}
		}

		// Check if file exists and ask for confirmation if not using force
		if !force {
			if _, err := os.Stat(schemaPath); err == nil {
				fmt.Printf("📁 Schema file already exists: %s\n", schemaPath)
				if !askUserConfirmation("Overwrite existing schema file?") {
					fmt.Println("❌ Operation cancelled")
					return nil
				}
			}
		}

		// Generate schema SQL content
		schemaContent := generateSchemaSQL(schema)

		// Write to schema file
		if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
			return fmt.Errorf("failed to write schema file: %w", err)
		}

		fmt.Printf("✅ Successfully pulled database schema to %s\n", schemaPath)
		fmt.Printf("📊 Found %d tables with schema definitions\n", len(schema))

		return nil
	},
}

func pullDatabaseSchema(ctx context.Context, m *migrator.Migrator) ([]types.SchemaTable, error) {
	// Use the existing migrator method to get current schema
	return m.PullSchema(ctx)
}

func generateSchemaSQL(tables []types.SchemaTable) string {
	var builder strings.Builder

	// Sort tables by name for consistent output
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	for i, table := range tables {
		if i > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

		// Keep original column order (don't sort columns to maintain database order)
		for j, column := range table.Columns {
			if j > 0 {
				builder.WriteString(",\n")
			}

			columnDef := formatColumnDefinitionClean(column)
			builder.WriteString(fmt.Sprintf("    %s", columnDef))
		}

		builder.WriteString("\n);\n")
	}

	return builder.String()
}

func formatColumnDefinitionClean(column types.SchemaColumn) string {
	var parts []string

	// Check if this is a primary key column to determine if it should be SERIAL
	isPrimaryKey := strings.Contains(strings.ToUpper(column.Type), "PRIMARY KEY")

	// Convert database types back to more standard SQL types
	colType := normalizeColumnType(column.Type, isPrimaryKey, column.Default)

	// Column name and type
	parts = append(parts, column.Name, colType)

	// Add constraints in proper order
	if isPrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	} else {
		// Only add NOT NULL if not primary key (primary keys are implicitly NOT NULL)
		if !column.Nullable {
			parts = append(parts, "NOT NULL")
		}
		if strings.Contains(strings.ToUpper(column.Type), "UNIQUE") {
			parts = append(parts, "UNIQUE")
		}
	}

	// Default value (only for non-SERIAL columns)
	if column.Default != "" && !strings.HasPrefix(strings.ToUpper(colType), "SERIAL") {
		// Clean up default values to match the original format
		defaultValue := cleanDefaultValue(column.Default)
		if defaultValue != "" {
			parts = append(parts, "DEFAULT", defaultValue)
		}
	}

	return strings.Join(parts, " ")
}

func normalizeColumnType(dbType string, isPrimaryKey bool, defaultValue string) string {
	// Clean up the type string
	cleanType := strings.ReplaceAll(dbType, " PRIMARY KEY", "")
	cleanType = strings.ReplaceAll(cleanType, " UNIQUE", "")
	cleanType = strings.TrimSpace(cleanType)

	// Convert PostgreSQL types back to more standard types
	upperType := strings.ToUpper(cleanType)

	switch {
	case upperType == "INTEGER" && isPrimaryKey && strings.Contains(defaultValue, "nextval("):
		return "SERIAL"
	case upperType == "INTEGER":
		return "INTEGER"
	case strings.HasPrefix(upperType, "CHARACTER VARYING"):
		// Extract length if present, otherwise default to 255
		if strings.Contains(upperType, "(") {
			return strings.ReplaceAll(cleanType, "character varying", "VARCHAR")
		}
		return "VARCHAR(255)"
	case upperType == "TIMESTAMP WITHOUT TIME ZONE":
		return "TIMESTAMP WITH TIME ZONE"
	case upperType == "TIMESTAMP WITH TIME ZONE":
		return "TIMESTAMP WITH TIME ZONE"
	default:
		return cleanType
	}
}

func cleanDefaultValue(defaultVal string) string {
	// Clean up common default value formats
	cleaned := strings.TrimSpace(defaultVal)

	// Handle timestamp defaults
	if strings.Contains(strings.ToUpper(cleaned), "NOW()") {
		return "NOW()"
	}

	// Handle sequence defaults (for SERIAL columns)
	if strings.Contains(cleaned, "nextval(") {
		return "" // Don't include sequence defaults for SERIAL columns
	}

	return cleaned
}

func backupExistingSchema(schemaPath string) error {
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		// No existing schema file to backup
		return nil
	}

	backupPath := schemaPath + ".backup"
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read existing schema: %w", err)
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

func askUserConfirmation(message string) bool {
	fmt.Printf("🤔 %s (y/N): ", message)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y"
}

func init() {
	rootCmd.AddCommand(pullCmd)

	pullCmd.Flags().BoolP("backup", "b", false, "Create backup of existing schema file before overwriting")
	pullCmd.Flags().BoolP("force", "f", false, "Skip confirmations and overwrite existing schema file")
	pullCmd.Flags().BoolP("indexes", "i", false, "Include indexes in the generated schema (disabled by default)")
	pullCmd.Flags().StringP("output", "o", "", "Custom output file path (overrides config schema_path)")
}
