package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// ConfigFile is the path to the config file, set by the cmd package from --config flag.
var ConfigFile string

var (
	configCacheMu    sync.Mutex
	configCache      *Config
	configCacheErr   error
	configCachePath  string
	configCacheMtime int64
)

type Config struct {
	Version        string   `toml:"version"`
	SchemaPath     string   `toml:"schema_path"` // Deprecated: use SchemaDir instead
	SchemaDir      string   `toml:"schema_dir"`  // New: folder containing .sql schema files
	Queries        string   `toml:"queries"`
	MigrationsPath string   `toml:"migrations_path"`
	ExportPath     string   `toml:"export_path"`
	Database       Database `toml:"database"`
	Gen            Gen      `toml:"gen"`
}

type Database struct {
	Provider string `toml:"provider"`
	URLEnv   string `toml:"url_env"`
}

type Gen struct {
	Go     GoGen     `toml:"go"`
	JS     JSGen     `toml:"js"`
	Python PythonGen `toml:"python"`
}

type GoGen struct {
	Enabled bool   `toml:"enabled"`
	Driver  string `toml:"driver"` // "database/sql" (default) or "pgx"
	Out     string `toml:"out"`
}

type JSGen struct {
	Enabled bool   `toml:"enabled"`
	Out     string `toml:"out"`
	Driver  string `toml:"driver"` // "pg" (default) or "postgres"
}

type PythonGen struct {
	Enabled bool   `toml:"enabled"`
	Out     string `toml:"out"`
	Async   bool   `toml:"async"`  // true = async (default), false = sync
	Driver  string `toml:"driver"` // database-specific driver
}

// rawPythonGen uses a pointer so we can detect whether "async" was explicitly set.
type rawPythonGen struct {
	Enabled bool   `toml:"enabled"`
	Out     string `toml:"out"`
	Async   *bool  `toml:"async"` // nil = not present in TOML
	Driver  string `toml:"driver"`
}

type rawGen struct {
	Go     GoGen        `toml:"go"`
	JS     JSGen        `toml:"js"`
	Python rawPythonGen `toml:"python"`
}

type rawConfig struct {
	Version        string   `toml:"version"`
	SchemaPath     string   `toml:"schema_path"`
	SchemaDir      string   `toml:"schema_dir"`
	Queries        string   `toml:"queries"`
	MigrationsPath string   `toml:"migrations_path"`
	ExportPath     string   `toml:"export_path"`
	Database       Database `toml:"database"`
	Gen            rawGen   `toml:"gen"`
}

// Load reads and returns the config, with in-memory caching keyed by file path
// and modification time. This avoids redundant disk I/O when Load() is called
// multiple times within a single command invocation.
func Load() (*Config, error) {
	path := ConfigFile
	if path == "" {
		path = "flash.toml"
	}

	// Resolve to absolute path for reliable cache keying
	if absPath, err := filepath.Abs(path); err == nil {
		path = absPath
	}

	// Check file mod time for cache invalidation
	var mtime int64
	if fi, err := os.Stat(path); err == nil {
		mtime = fi.ModTime().UnixNano()
	}

	configCacheMu.Lock()
	defer configCacheMu.Unlock()

	if configCache != nil && configCachePath == path && configCacheMtime == mtime && configCacheErr == nil {
		// Return a shallow copy so callers can't mutate the cache
		cfg := *configCache
		return &cfg, nil
	}

	cfg, err := loadUncached()
	configCache = cfg
	configCacheErr = err
	configCachePath = path
	configCacheMtime = mtime
	if err != nil {
		return nil, err
	}
	// Return a copy so callers can't mutate the cache
	copyCfg := *cfg
	return &copyCfg, nil
}

// ResetConfigCache clears the in-memory config cache. It is intended for tests
// that need to reload config from a different file or working directory.
func ResetConfigCache() {
	configCacheMu.Lock()
	defer configCacheMu.Unlock()
	configCache = nil
	configCacheErr = nil
	configCachePath = ""
	configCacheMtime = 0
}

// loadUncached performs the actual config loading without caching.
func loadUncached() (*Config, error) {
	var cfg Config

	path := ConfigFile
	if path == "" {
		path = "flash.toml"
	}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	pythonAsyncSet := false
	if data != nil {
		var raw rawConfig
		if err := toml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
		// Copy raw values into the main config struct
		cfg.Version = raw.Version
		cfg.SchemaPath = raw.SchemaPath
		cfg.SchemaDir = raw.SchemaDir
		cfg.Queries = raw.Queries
		cfg.MigrationsPath = raw.MigrationsPath
		cfg.ExportPath = raw.ExportPath
		cfg.Database = raw.Database
		cfg.Gen.Go = raw.Gen.Go
		cfg.Gen.JS = raw.Gen.JS
		cfg.Gen.Python.Enabled = raw.Gen.Python.Enabled
		cfg.Gen.Python.Out = raw.Gen.Python.Out
		cfg.Gen.Python.Driver = raw.Gen.Python.Driver
		if raw.Gen.Python.Async != nil {
			cfg.Gen.Python.Async = *raw.Gen.Python.Async
			pythonAsyncSet = true
		}
	}

	// Set defaults
	if cfg.Version == "" {
		cfg.Version = "2"
	}
	// Support both old schema_path and new schema_dir
	if cfg.SchemaDir == "" {
		if cfg.SchemaPath != "" {
			// Legacy: if schema_path is a file, use its directory
			// If it looks like a directory (no .sql extension), use it directly
			if strings.HasSuffix(cfg.SchemaPath, ".sql") {
				cfg.SchemaDir = filepath.Dir(cfg.SchemaPath)
			} else {
				cfg.SchemaDir = cfg.SchemaPath
			}
		} else {
			cfg.SchemaDir = "db/schema"
		}
	}
	// Keep SchemaPath for backward compatibility
	if cfg.SchemaPath == "" {
		cfg.SchemaPath = filepath.Join(cfg.SchemaDir, "schema.sql")
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
		cfg.Gen.Python.Out = "flash_gen"
	}
	if cfg.Gen.Go.Out == "" {
		cfg.Gen.Go.Out = "flash_gen"
	}
	if cfg.Gen.Python.Enabled && !pythonAsyncSet {
		cfg.Gen.Python.Async = true
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
		c.GetSchemaDir(),
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
	supportedProviders := []string{"postgresql", "postgres", "mysql", "sqlite", "sqlite3", "clickhouse", "scylla", "scylladb", "cassandra"}
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
	case "clickhouse":
		return "clickhouse"
	case "scylla", "scylladb", "cassandra":
		return "scylla"
	default:
		return "postgresql"
	}
}

func (c *Config) GetSchemaDir() string {
	if c.SchemaDir != "" {
		return c.SchemaDir
	}
	return filepath.Dir(c.SchemaPath)
}

// GetSchemaFiles returns all .sql files in the schema directory
func (c *Config) GetSchemaFiles() ([]string, error) {
	schemaDir := c.GetSchemaDir()

	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		// If directory doesn't exist, check if schema_path is a file
		if os.IsNotExist(err) && c.SchemaPath != "" {
			if _, err := os.Stat(c.SchemaPath); err == nil {
				return []string{c.SchemaPath}, nil
			}
		}
		return nil, fmt.Errorf("failed to read schema directory %s: %w", schemaDir, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, filepath.Join(schemaDir, entry.Name()))
		}
	}

	// Sort files for consistent ordering
	// Files are typically named like: 001_users.sql, 002_posts.sql or users.sql, posts.sql
	sort.Strings(files)
	return files, nil
}

func (c *Config) IsNodeProject() bool {
	_, err := os.Stat("package.json")
	return err == nil
}
