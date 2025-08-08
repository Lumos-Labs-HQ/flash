package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize graft in the current project",
	Long: `Initialize graft in the current project by creating:
	- graft.config.json - Configuration file
	- db/schema/ - Schema directory with example schema
	- db/queries/ - SQL queries directory for SQLC
	- sqlc.yml - SQLC configuration file
	- .env - Environment variables template

	Note: Migration and backup directories are created automatically when needed.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return initializeProject()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initializeProject() error {
	fmt.Println("🚀 Initializing Graft in current project...")

	config := map[string]interface{}{
		"schema_path":      "db/schema/schema.sql",
		"migrations_path":  "db/migrations",
		"sqlc_config_path": "sqlc.yml",
		"backup_path":      "db/backup",
		"database": map[string]string{
			"provider": "postgresql",
			"url_env":  "DATABASE_URL",
		},
	}

	configFile, err := os.Create("graft.config.json")
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("✅ Created graft.config.json")

	directories := []string{
		"db/schema",
		"db/queries",
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("✅ Created directory: %s/\n", dir)
	}

	sqlcConfig := `version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/schema/"
    gen:
      go:
        package: "graft"
        out: "graft_gen/"
        sql_package: "pgx/v5"
`

	if err := os.WriteFile("sqlc.yml", []byte(sqlcConfig), 0644); err != nil {
		return fmt.Errorf("failed to create SQLC config: %w", err)
	}
	fmt.Println("✅ Created sqlc.yml")

	schemaPath := filepath.Join("db", "schema", "schema.sql")
	schemaContent := `-- Example schema file
-- Add your database schema here

-- Example table:
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

`

	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}
	fmt.Printf("✅ Created example schema: %s\n", schemaPath)

	queriesPath := filepath.Join("db", "queries", "users.sql")
	queriesContent := `-- Example queries for SQLC
-- Add your SQL queries here

-- Example queries:
-- name: GetUser :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1;


-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING id, name, email, created_at, updated_at;
`

	if err := os.WriteFile(queriesPath, []byte(queriesContent), 0644); err != nil {
		return fmt.Errorf("failed to create queries file: %w", err)
	}
	fmt.Printf("✅ Created example queries: %s\n", queriesPath)

	envContent := `# Database connection URL
		# Replace with your actual database credentials
		DATABASE_URL=postgres://username:password@localhost:5432/database_name
	`

	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
			return fmt.Errorf("failed to create .env: %w", err)
		}
		fmt.Println("✅ Created .env")
	}

	fmt.Println("\n🎉 Graft initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Set your DATABASE_URL environment variable")
	fmt.Println("2. Edit db/schema/schema.sql with your database schema")
	fmt.Println("3. Edit db/queries/users.sql with your SQL queries")
	fmt.Println("4. Run 'graft migrate \"initial migration\"' to create your first migration")
	fmt.Println("5. Run 'graft gen' to generate Go types using SQLC")

	return nil
}
