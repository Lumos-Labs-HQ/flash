package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize graft in the current project",
	Long: `Initialize graft in the current project by creating:
- graft.config.json - Configuration file
- migrations/ - Migration files directory  
- db_backup/ - Backup directory
- db/ - Schema directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return initializeProject()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initializeProject() error {
	fmt.Println("🚀 Initializing Graft in current project...")

	// Create default configuration
	config := map[string]interface{}{
		"schema_path":     "db/schema.sql",
		"migrations_path": "migrations",
		"sqlc_config_path": "",
		"backup_path":     "db_backup",
		"database": map[string]string{
			"provider": "postgresql",
			"url_env":  "DATABASE_URL",
		},
	}

	// Create config file
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

	// Create directories
	directories := []string{
		"migrations",
		"db_backup", 
		"db",
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("✅ Created directory: %s/\n", dir)
	}

	// Create example schema file
	schemaPath := filepath.Join("db", "schema.sql")
	schemaContent := `-- Example schema file
-- Add your database schema here

-- Example table:
-- CREATE TABLE users (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     email VARCHAR(255) UNIQUE NOT NULL,
--     created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
-- );

-- Example index:
-- CREATE INDEX idx_users_email ON users(email);
`

	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}
	fmt.Printf("✅ Created example schema: %s\n", schemaPath)

	// Create .env example
	envContent := `# Database connection URL
# Replace with your actual database credentials
DATABASE_URL=postgres://username:password@localhost:5432/database_name

# Example for local development:
# DATABASE_URL=postgres://postgres:password@localhost:5432/myapp_dev
`

	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if err := os.WriteFile(".env.example", []byte(envContent), 0644); err != nil {
			return fmt.Errorf("failed to create .env.example: %w", err)
		}
		fmt.Println("✅ Created .env.example")
	}

	fmt.Println("\n🎉 Graft initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Set your DATABASE_URL environment variable")
	fmt.Println("2. Edit db/schema.sql with your database schema")
	fmt.Println("3. Run 'graft migrate \"initial migration\"' to create your first migration")
	fmt.Println("4. Run 'graft apply' to apply migrations")

	return nil
}
