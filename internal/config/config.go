package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the graft configuration
type Config struct {
	SchemaPath     string   `json:"schema_path" yaml:"schema_path"`
	MigrationsPath string   `json:"migrations_path" yaml:"migrations_path"`
	SQLCConfigPath string   `json:"sqlc_config_path" yaml:"sqlc_config_path"`
	BackupPath     string   `json:"backup_path" yaml:"backup_path"`
	Database       Database `json:"database" yaml:"database"`
}

// Database represents database configuration
type Database struct {
	Provider string `json:"provider" yaml:"provider"`
	URLEnv   string `json:"url_env" yaml:"url_env"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		SchemaPath:     "db/schema.sql",
		MigrationsPath: "migrations",
		BackupPath:     "db_backup",
		Database: Database{
			Provider: "postgresql",
			URLEnv:   "DATABASE_URL",
		},
	}
}

// LoadConfig loads configuration from viper
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Try to unmarshal from viper
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// IsInitialized checks if graft is initialized in the current project
func IsInitialized() bool {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return false
	}

	// Check for config files
	configFiles := []string{
		filepath.Join(projectRoot, "graft.config.json"),
		filepath.Join(projectRoot, "graft.config.yaml"),
	}

	for _, configFile := range configFiles {
		if _, err := os.Stat(configFile); err == nil {
			return true
		}
	}

	return false
}

// InitializeProject creates a default configuration file
func InitializeProject() error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	configPath := filepath.Join(projectRoot, "graft.config.json")
	
	// Check if already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("graft is already initialized")
	}

	config := DefaultConfig()
	
	// Create config file
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Create directories
	dirs := []string{
		filepath.Join(projectRoot, config.MigrationsPath),
		filepath.Join(projectRoot, config.BackupPath),
		filepath.Dir(filepath.Join(projectRoot, config.SchemaPath)),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	fmt.Printf("✅ Graft initialized successfully!\n")
	fmt.Printf("📁 Config file created: %s\n", configPath)
	fmt.Printf("📁 Migrations directory: %s\n", filepath.Join(projectRoot, config.MigrationsPath))
	fmt.Printf("📁 Backup directory: %s\n", filepath.Join(projectRoot, config.BackupPath))

	return nil
}

// GetProjectRoot returns the project root directory
func GetProjectRoot() (string, error) {
	return findProjectRoot()
}

// findProjectRoot finds the project root by looking for go.mod, package.json, or .git
func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return findProjectRootRecursive(currentDir)
}

func findProjectRootRecursive(dir string) (string, error) {
	// Check for common project indicators
	indicators := []string{"go.mod", "package.json", ".git", "graft.config.json", "graft.config.yaml"}
	
	for _, indicator := range indicators {
		if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
			return dir, nil
		}
	}

	// Check parent directory
	parent := filepath.Dir(dir)
	if parent == dir {
		// Reached root directory
		return dir, nil
	}

	return findProjectRootRecursive(parent)
}
