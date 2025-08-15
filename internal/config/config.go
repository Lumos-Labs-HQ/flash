package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	SchemaPath     string   `json:"schema_path" mapstructure:"schema_path"`
	MigrationsPath string   `json:"migrations_path" mapstructure:"migrations_path"`
	SqlcConfigPath string   `json:"sqlc_config_path" mapstructure:"sqlc_config_path"`
	BackupPath     string   `json:"backup_path" mapstructure:"backup_path"`
	Database       Database `json:"database" mapstructure:"database"`
}

type Database struct {
	Provider string `json:"provider" mapstructure:"provider"`
	URLEnv   string `json:"url_env" mapstructure:"url_env"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.SchemaPath == "" {
		cfg.SchemaPath = "db/schema.sql"
	}
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "migrations"
	}
	if cfg.BackupPath == "" {
		cfg.BackupPath = "db_backup"
	}
	if cfg.Database.Provider == "" {
		cfg.Database.Provider = "postgresql"
	}
	if cfg.Database.URLEnv == "" {
		cfg.Database.URLEnv = "DATABASE_URL"
	}

	return &cfg, nil
}

func (c *Config) GetDatabaseURL() (string, error) {
	dbURL := os.Getenv(c.Database.URLEnv)
	if dbURL == "" {
		return "", fmt.Errorf("database URL not found in environment variable %s", c.Database.URLEnv)
	}
	return dbURL, nil
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.MigrationsPath,
		filepath.Dir(c.SchemaPath),
	}

	for _, dir := range dirs {
		if dir == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (c *Config) Validate() error {
	supportedProviders := []string{"postgresql", "postgres", "mysql", "sqlite", "sqlite3"}
	supported := false
	for _, provider := range supportedProviders {
		if c.Database.Provider == provider {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("unsupported database provider: %s. Supported providers: %v", c.Database.Provider, supportedProviders)
	}

	if c.MigrationsPath == "" {
		return fmt.Errorf("migrations_path cannot be empty")
	}

	if c.BackupPath == "" {
		return fmt.Errorf("backup_path cannot be empty")
	}

	return nil
}
