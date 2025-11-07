package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Version        string   `json:"version" mapstructure:"version"`
	SchemaPath     string   `json:"schema_path" mapstructure:"schema_path"`
	Queries        string   `json:"queries" mapstructure:"queries"`
	MigrationsPath string   `json:"migrations_path" mapstructure:"migrations_path"`
	ExportPath     string   `json:"export_path" mapstructure:"export_path"`
	Database       Database `json:"database" mapstructure:"database"`
	Gen            Gen      `json:"gen" mapstructure:"gen"`
}

type Database struct {
	Provider string `json:"provider" mapstructure:"provider"`
	URLEnv   string `json:"url_env" mapstructure:"url_env"`
}

type Gen struct {
	Go GoGen `json:"go,omitempty" mapstructure:"go"`
	JS JSGen `json:"js,omitempty" mapstructure:"js"`
    Python PythonGen `json:"python,omitempty" mapstructure:"python"`
}

type GoGen struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`
}

type JSGen struct {
	Enabled bool   `json:"enabled,omitempty" mapstructure:"enabled"`
	Out     string `json:"out,omitempty" mapstructure:"out"`
}
type PythonGen struct {
	Enabled bool   `json:"enabled,omitempty" mapstructure:"enabled"`
	Out     string `json:"out,omitempty" mapstructure:"out"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	if cfg.Version == "" {
		cfg.Version = "2"
	}
	if cfg.SchemaPath == "" {
		cfg.SchemaPath = "db/schema/schema.sql"
	}
	if cfg.Queries == "" {
		cfg.Queries = "db/queries/"
	}
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "db/migrations"
	}
	if cfg.ExportPath == "" {
		cfg.ExportPath = "db/export"
	}
	if cfg.Database.Provider == "" {
		cfg.Database.Provider = "postgresql"
	}
	if cfg.Database.URLEnv == "" {
		cfg.Database.URLEnv = "DATABASE_URL"
	}
	if cfg.Gen.JS.Out == "" && cfg.Gen.JS.Enabled {
		cfg.Gen.JS.Out = "flash_gen"
	}
	if cfg.Gen.Python.Out == "" && cfg.Gen.Python.Enabled {
		cfg.Gen.Python.Out = "graft_gen"
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

	if c.ExportPath == "" {
		return fmt.Errorf("export_path cannot be empty")
	}

	return nil
}

func (c *Config) GetSqlcEngine() string {
	switch c.Database.Provider {
	case "postgresql", "postgres":
		return "postgresql"
	case "mysql":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return "postgresql"
	}
}

func (c *Config) GetSchemaDir() string {
	return filepath.Dir(c.SchemaPath)
}

func (c *Config) IsNodeProject() bool {
	_, err := os.Stat("package.json")
	return err == nil
}
