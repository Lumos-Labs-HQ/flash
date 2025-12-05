//go:build plugins
// +build plugins

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/template"
	"github.com/spf13/cobra"
)

var (
	sqliteFlag     bool
	postgresqlFlag bool
	mysqlFlag      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new FlashORM project",
	Long:  `Initialize a new FlashORM project with database migrations and code generation configuration.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbType := template.PostgreSQL
		flagCount := 0

		if sqliteFlag {
			dbType = template.SQLite
			flagCount++
		}
		if postgresqlFlag {
			dbType = template.PostgreSQL
			flagCount++
		}
		if mysqlFlag {
			dbType = template.MySQL
			flagCount++
		}

		if flagCount > 1 {
			return fmt.Errorf("please specify only one database type (--sqlite, --postgresql, or --mysql)")
		}

		return initializeProject(dbType)
	},
}

func init() {
	// Command is registered by plugin executors, not the base CLI

	initCmd.Flags().BoolVar(&sqliteFlag, "sqlite", false, "Initialize project for SQLite database")
	initCmd.Flags().BoolVar(&postgresqlFlag, "postgresql", false, "Initialize project for PostgreSQL database")
	initCmd.Flags().BoolVar(&mysqlFlag, "mysql", false, "Initialize project for MySQL database")
}

func initializeProject(dbType template.DatabaseType) error {
	// Detect project type
	isNodeProject := false
	isPythonProject := false

	if _, err := os.Stat("package.json"); err == nil {
		isNodeProject = true
	}

	if _, err := os.Stat("requirements.txt"); err == nil {
		isPythonProject = true
	} else if _, err := os.Stat("pyproject.toml"); err == nil {
		isPythonProject = true
	}

	tmpl := template.NewProjectTemplate(dbType, isNodeProject, isPythonProject)

	directories := tmpl.GetDirectoryStructure()
	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	files := map[string]string{
		"flash.config.json": tmpl.GetFlashORMConfig(),
	}

	// Check if any .sql files exist in db/schema directory
	schemaExists := false
	if entries, err := os.ReadDir("db/schema"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				schemaExists = true
				break
			}
		}
	}
	if !schemaExists {
		files["db/schema/users.sql"] = tmpl.GetSchema()
	}

	if _, err := os.Stat("db/queries/users.sql"); os.IsNotExist(err) {
		files["db/queries/users.sql"] = tmpl.GetQueries()
	}

	for filePath, content := range files {
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
	}

	// Handle .env file separately to preserve existing variables
	envCreated := false
	if err := handleEnvFile(tmpl.GetEnvTemplate()); err != nil {
		return fmt.Errorf("failed to handle .env file: %w", err)
	} else {
		envCreated = true
	}
	_ = envCreated

	projectType := "Go"
	if isNodeProject {
		projectType = "Node.js"
	} else if isPythonProject {
		projectType = "Python"
	}

	fmt.Printf("✅ Successfully initialized FlashORM project for %s with %s database support\n", projectType, dbType)
	fmt.Println()
	fmt.Println("📁 Project structure created:")
	for _, dir := range directories {
		fmt.Printf("   %s/\n", dir)
	}
	fmt.Println()
	fmt.Println("📝 Configuration file created:")
	fmt.Println("   flash.config.json")

	if isNodeProject {
		fmt.Println()
		fmt.Println("🟢 Node.js project detected!")
		fmt.Println("   JavaScript code generation is enabled")
		fmt.Println("   Run 'flash gen' to generate type-safe JS code")
	}

	if isPythonProject {
		fmt.Println()
		fmt.Println("🐍 Python project detected!")
		fmt.Println("   Python code generation is enabled")
		fmt.Println("   Run 'Flash gen' to generate type-safe Python code")
	}

	if os.Getenv("DATABASE_URL") != "" {
		fmt.Println()
		fmt.Println("ℹ️  Using existing DATABASE_URL from environment")
	}

	if schemaExists {
		fmt.Println("ℹ️  Skipped schema files (db/schema already has .sql files)")
	}

	if _, err := os.Stat("db/queries/users.sql"); err == nil {
		fmt.Println("ℹ️  Skipped db/queries/users.sql (already exists)")
	}

	fmt.Println()
	fmt.Printf("🚀 Next steps:\n")
	fmt.Printf("   flash migrate \"create users\"  # Create migrations\n")
	fmt.Printf("   flash apply                    # Apply migrations\n")
	fmt.Printf("   flash gen                      # Generate code\n")

	return nil
}

func handleEnvFile(defaultEnvContent string) error {
	envPath := ".env"

	// Check if .env file exists
	existingContent, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(envPath, []byte(defaultEnvContent), 0644)
		}
		return err
	}

	existingStr := string(existingContent)
	if strings.Contains(existingStr, "DATABASE_URL") {
		return nil
	}

	// Append DATABASE_URL to existing .env
	if len(existingStr) > 0 && !strings.HasSuffix(existingStr, "\n") {
		existingStr += "\n"
	}

	existingStr += "\n# Added by FlashORM\n" + defaultEnvContent

	return os.WriteFile(envPath, []byte(existingStr), 0644)
}
