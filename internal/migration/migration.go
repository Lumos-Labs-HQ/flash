package migration

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"Rana718/Graft/internal/config"
)

// Migration represents a database migration
type Migration struct {
	Name     string
	Path     string
	Content  string
	Checksum string
}

// Manager handles migration operations
type Manager struct {
	Config      *config.Config
	ProjectRoot string
}

// NewManager creates a new migration manager
func NewManager(cfg *config.Config) (*Manager, error) {
	projectRoot, err := config.GetProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get project root: %w", err)
	}

	return &Manager{
		Config:      cfg,
		ProjectRoot: projectRoot,
	}, nil
}

// CreateMigration creates a new migration file
func (m *Manager) CreateMigration(name string) (*Migration, error) {
	// Generate timestamp-based filename
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, strings.ReplaceAll(name, " ", "_"))
	
	migrationPath := filepath.Join(m.ProjectRoot, m.Config.MigrationsPath, filename)
	
	// Create migration file with template
	template := fmt.Sprintf(`-- Migration: %s
-- Created at: %s

-- Add your SQL statements here
-- Example:
-- CREATE TABLE users (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     email VARCHAR(255) UNIQUE NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );
`, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(migrationPath, []byte(template), 0644); err != nil {
		return nil, fmt.Errorf("failed to create migration file: %w", err)
	}

	migration := &Migration{
		Name:     filename,
		Path:     migrationPath,
		Content:  template,
		Checksum: m.calculateChecksum(template),
	}

	fmt.Printf("✅ Migration created: %s\n", migrationPath)
	fmt.Printf("📝 Edit the file to add your SQL statements\n")

	return migration, nil
}

// GetLocalMigrations returns all migration files from the migrations directory
func (m *Manager) GetLocalMigrations() ([]*Migration, error) {
	migrationsDir := filepath.Join(m.ProjectRoot, m.Config.MigrationsPath)
	
	var migrations []*Migration
	
	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}
		
		migration := &Migration{
			Name:     d.Name(),
			Path:     path,
			Content:  string(content),
			Checksum: m.calculateChecksum(string(content)),
		}
		
		migrations = append(migrations, migration)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}
	
	// Sort migrations by name (which includes timestamp)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})
	
	return migrations, nil
}

// GetPendingMigrations returns migrations that haven't been applied to the database
func (m *Manager) GetPendingMigrations(appliedMigrations []string) ([]*Migration, error) {
	localMigrations, err := m.GetLocalMigrations()
	if err != nil {
		return nil, err
	}
	
	appliedMap := make(map[string]bool)
	for _, applied := range appliedMigrations {
		appliedMap[applied] = true
	}
	
	var pending []*Migration
	for _, migration := range localMigrations {
		if !appliedMap[migration.Name] {
			pending = append(pending, migration)
		}
	}
	
	return pending, nil
}

// calculateChecksum calculates SHA256 checksum of migration content
func (m *Manager) calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// ValidateMigration validates a migration file
func (m *Manager) ValidateMigration(migration *Migration) error {
	if migration.Content == "" {
		return fmt.Errorf("migration %s is empty", migration.Name)
	}
	
	// Basic SQL validation - check for dangerous operations
	content := strings.ToLower(migration.Content)
	if strings.Contains(content, "drop database") {
		return fmt.Errorf("migration %s contains dangerous DROP DATABASE statement", migration.Name)
	}
	
	return nil
}
