package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	tmpl "github.com/Lumos-Labs-HQ/flash/template"
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().Bool("sqlite", false, "Initialize project for SQLite database")
	initCmd.Flags().Bool("postgresql", false, "Initialize project for PostgreSQL database")
	initCmd.Flags().Bool("postgres", false, "Initialize project for PostgreSQL database (alias)")
	initCmd.Flags().Bool("mysql", false, "Initialize project for MySQL database")
	initCmd.Flags().Bool("clickhouse", false, "Initialize project for ClickHouse database")
	initCmd.Flags().Bool("scylla", false, "Initialize project for ScyllaDB database")
	initCmd.Flags().Bool("cassandra", false, "Initialize project for Cassandra database (ScyllaDB-compatible)")
}

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new Flash project",
	Long:  "Initialize a new Flash project with the necessary directory structure and configuration files.",
	Args:  cobra.MaximumNArgs(1),
	Run:   runInit,
}

func runInit(cmd *cobra.Command, args []string) {
	projectName := ""
	if len(args) > 0 {
		projectName = args[0]
	}

	dbType := tmpl.PostgreSQL
	flagCount := 0

	if cmd.Flags().Changed("sqlite") {
		dbType = tmpl.SQLite
		flagCount++
	}
	if cmd.Flags().Changed("postgresql") || cmd.Flags().Changed("postgres") {
		dbType = tmpl.PostgreSQL
		flagCount++
	}
	if cmd.Flags().Changed("mysql") {
		dbType = tmpl.MySQL
		flagCount++
	}
	if cmd.Flags().Changed("clickhouse") {
		dbType = tmpl.ClickHouse
		flagCount++
	}
	if cmd.Flags().Changed("scylla") || cmd.Flags().Changed("cassandra") {
		dbType = tmpl.ScyllaDB
		flagCount++
	}

	if flagCount > 1 {
		fmt.Fprintln(os.Stderr, "please specify only one database type (--sqlite, --postgresql, --mysql, --clickhouse, --scylla, or --cassandra)")
		os.Exit(1)
	}

	projectTemplate := tmpl.NewProjectTemplateExt(dbType, isNodeProject(), isPythonProject(), isKotlinProject(), isJavaProject())

	// Auto-detect Java/Kotlin package from build files
	if projectTemplate.IsJavaProject {
		if pkg := tmpl.DetectJavaPackage("."); pkg != "" {
			projectTemplate.JavaPackage = pkg
		}
	}
	if projectTemplate.IsKotlinProject {
		if pkg := tmpl.DetectKotlinPackage("."); pkg != "" {
			projectTemplate.KotlinPackage = pkg
		}
	}

	initializeProject(projectName, projectTemplate)
}

func isNodeProject() bool {
	_, err := os.Stat("package.json")
	return err == nil
}

func isPythonProject() bool {
	for _, file := range []string{"requirements.txt", "pyproject.toml", "setup.py"} {
		if _, err := os.Stat(file); err == nil {
			return true
		}
	}
	return false
}

// isKotlinProject detects Kotlin projects by looking for Kotlin build files
// and source files. Gradle (Kotlin DSL or Groovy) and Maven with Kotlin plugin
// are all supported.
func isKotlinProject() bool {
	// Gradle Kotlin DSL
	for _, f := range []string{"build.gradle.kts", "settings.gradle.kts"} {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	// Groovy Gradle with kotlin plugin
	if _, err := os.Stat("build.gradle"); err == nil {
		data, _ := os.ReadFile("build.gradle")
		if strings.Contains(string(data), "kotlin") {
			return true
		}
	}
	// Maven with Kotlin plugin
	if _, err := os.Stat("pom.xml"); err == nil {
		data, _ := os.ReadFile("pom.xml")
		if strings.Contains(string(data), "kotlin") {
			return true
		}
	}
	// Any .kt source file in src/
	if matches, _ := filepath.Glob("src/**/*.kt"); len(matches) > 0 {
		return true
	}
	if matches, _ := filepath.Glob("src/*.kt"); len(matches) > 0 {
		return true
	}
	return false
}

// isJavaProject detects plain Java projects (Maven, Gradle without Kotlin,
// or bare .java source files). Kotlin is checked first so a Kotlin project
// is never mis-identified as Java.
func isJavaProject() bool {
	// Already handled by isKotlinProject — skip if Kotlin
	if isKotlinProject() {
		return false
	}
	// Maven
	if _, err := os.Stat("pom.xml"); err == nil {
		return true
	}
	// Gradle (Groovy DSL without kotlin keyword)
	if _, err := os.Stat("build.gradle"); err == nil {
		return true
	}
	// Bare Java source
	if matches, _ := filepath.Glob("src/**/*.java"); len(matches) > 0 {
		return true
	}
	if matches, _ := filepath.Glob("src/*.java"); len(matches) > 0 {
		return true
	}
	return false
}

func initializeProject(projectName string, projectTemplate *tmpl.ProjectTemplate) {
	dirs := projectTemplate.GetDirectoryStructure()
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	ext := ".sql"
	if projectTemplate.DatabaseType == tmpl.ScyllaDB {
		ext = ".cql"
	}

	files := map[string]string{
		"flash.toml":             projectTemplate.GetFlashORMConfig(),
		"db/schema/schema" + ext: projectTemplate.GetSchema(),
		"db/queries/users" + ext: projectTemplate.GetQueries(),
	}

	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		files[".env"] = projectTemplate.GetEnvTemplate()
	}

	if projectName != "" {
		if err := os.MkdirAll(projectName, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating project directory: %v\n", err)
			os.Exit(1)
		}
		for fileName := range files {
			files[filepath.Join(projectName, fileName)] = files[fileName]
			delete(files, fileName)
		}
	}

	for fileName, content := range files {
		dir := filepath.Dir(fileName)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
				os.Exit(1)
			}
		}
		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file %s: %v\n", fileName, err)
			os.Exit(1)
		}
	}

	fmt.Println("Flash project initialized successfully!")
	fmt.Println()
	fmt.Println("Created files and directories:")
	fmt.Println("  flash.toml")
	fmt.Printf("  db/schema/schema%s\n", ext)
	fmt.Printf("  db/queries/users%s\n", ext)
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		fmt.Println("  .env")
	}

	if projectName != "" {
		fmt.Printf("\nProject created in directory: %s\n", projectName)
		fmt.Printf("  cd %s\n", projectName)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Update .env with your database URL")
	fmt.Println("  2. Run 'flash apply' to create the database tables")
	fmt.Println("  3. Run 'flash generate' to generate the code")
	fmt.Println("  4. Run 'flash studio' to open the database studio")

	// Reset config cache so subsequent commands pick up the new config
	config.ResetConfigCache()
}
