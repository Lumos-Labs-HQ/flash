package db

import (
	"database/sql"
	"fmt"
	"os"

	"Rana718/Graft/internal/config"
	_ "github.com/lib/pq"
)

// Connection represents a database connection
type Connection struct {
	DB     *sql.DB
	Config *config.Config
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.Config) (*Connection, error) {
	dbURL := os.Getenv(cfg.Database.URLEnv)
	if dbURL == "" {
		return nil, fmt.Errorf("database URL not found in environment variable %s", cfg.Database.URLEnv)
	}

	db, err := sql.Open(cfg.Database.Provider, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		DB:     db,
		Config: cfg,
	}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	return c.DB.Close()
}

// CreateMigrationsTable creates the graft_migrations table if it doesn't exist
func (c *Connection) CreateMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS graft_migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64) NOT NULL
		);
	`
	
	_, err := c.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns a list of applied migrations
func (c *Connection) GetAppliedMigrations() ([]string, error) {
	query := "SELECT name FROM graft_migrations ORDER BY applied_at"
	
	rows, err := c.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan migration name: %w", err)
		}
		migrations = append(migrations, name)
	}

	return migrations, nil
}

// RecordMigration records a migration as applied
func (c *Connection) RecordMigration(name, checksum string) error {
	query := "INSERT INTO graft_migrations (name, checksum) VALUES ($1, $2)"
	
	_, err := c.DB.Exec(query, name, checksum)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// RemoveMigration removes a migration record
func (c *Connection) RemoveMigration(name string) error {
	query := "DELETE FROM graft_migrations WHERE name = $1"
	
	_, err := c.DB.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to remove migration: %w", err)
	}

	return nil
}

// GetTableNames returns all table names in the database
func (c *Connection) GetTableNames() ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name != 'graft_migrations'
		ORDER BY table_name
	`
	
	rows, err := c.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table names: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// DropAllTables drops all tables except graft_migrations
func (c *Connection) DropAllTables() error {
	tables, err := c.GetTableNames()
	if err != nil {
		return err
	}

	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)
		if _, err := c.DB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// ExecuteSQL executes a SQL statement
func (c *Connection) ExecuteSQL(query string) error {
	_, err := c.DB.Exec(query)
	return err
}
