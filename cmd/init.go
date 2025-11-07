package cmd

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/graft/template"
	"github.com/spf13/cobra"
)

var (
	sqliteFlag     bool
	postgresqlFlag bool
	mysqlFlag      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Graft project",
	Long:  `Initialize a new Graft project with database migrations and code generation configuration.`,
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
	rootCmd.AddCommand(initCmd)

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
		"graft.config.json": tmpl.GetGraftConfig(),
	}

	if _, err := os.Stat("db/schema/schema.sql"); os.IsNotExist(err) {
		files["db/schema/schema.sql"] = tmpl.GetSchema()
	}

	if _, err := os.Stat("db/queries/users.sql"); os.IsNotExist(err) {
		files["db/queries/users.sql"] = tmpl.GetQueries()
	}

	if os.Getenv("DATABASE_URL") == "" {
		files[".env"] = tmpl.GetEnvTemplate()
	}

	for filePath, content := range files {
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
	}

	projectType := "Go"
	if isNodeProject {
		projectType = "Node.js"
	} else if isPythonProject {
		projectType = "Python"
	}

	fmt.Printf("✅ Successfully initialized Graft project for %s with %s database support\n", projectType, dbType)
	fmt.Println()
	fmt.Println("📁 Project structure created:")
	for _, dir := range directories {
		fmt.Printf("   %s/\n", dir)
	}
	fmt.Println()
	fmt.Println("📝 Configuration file created:")
	fmt.Println("   graft.config.json")
	
	if isNodeProject {
		fmt.Println()
		fmt.Println("🟢 Node.js project detected!")
		fmt.Println("   JavaScript code generation is enabled")
		fmt.Println("   Run 'graft gen' to generate type-safe JS code")
	}
	
	if isPythonProject {
		fmt.Println()
		fmt.Println("🐍 Python project detected!")
		fmt.Println("   Python code generation is enabled")
		fmt.Println("   Run 'graft gen' to generate type-safe Python code")
	}
	
	if os.Getenv("DATABASE_URL") != "" {
		fmt.Println()
		fmt.Println("ℹ️  Using existing DATABASE_URL from environment")
	}
	
	if _, err := os.Stat("db/schema/schema.sql"); err == nil {
		fmt.Println("ℹ️  Skipped db/schema/schema.sql (already exists)")
	}
	
	if _, err := os.Stat("db/queries/users.sql"); err == nil {
		fmt.Println("ℹ️  Skipped db/queries/users.sql (already exists)")
	}
	
	fmt.Println()
	fmt.Printf("🚀 Next steps:\n")
	fmt.Printf("   graft migrate \"create users\"  # Create migrations\n")
	fmt.Printf("   graft apply                    # Apply migrations\n")
	fmt.Printf("   graft gen                      # Generate code\n")

	return nil
}
