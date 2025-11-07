package pull

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/schema"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

type Options struct {
	Backup     bool
	OutputPath string
}

type Service struct {
	config     *config.Config
	adapter    database.DatabaseAdapter
	comparator *schema.SQLComparator
}

func NewService(cfg *config.Config) (*Service, error) {
	adapter := database.NewAdapter(cfg.Database.Provider)

	dbURL, err := cfg.GetDatabaseURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get database URL: %w", err)
	}

	if err := adapter.Connect(context.Background(), dbURL); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Service{
		config:     cfg,
		adapter:    adapter,
		comparator: schema.NewSQLComparator(),
	}, nil
}

func (s *Service) Close() {
	if s.adapter != nil {
		s.adapter.Close()
	}
}

func (s *Service) PullSchema(ctx context.Context, opts Options) error {
	schemaPath := s.config.SchemaPath
	if opts.OutputPath != "" {
		schemaPath = opts.OutputPath
	}

	fmt.Println("🔍 Introspecting database schema...")

	dbTables, err := s.adapter.PullCompleteSchema(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull database schema: %w", err)
	}

	// Also pull enums for PostgreSQL
	dbEnums, err := s.adapter.GetCurrentEnums(ctx)
	if err != nil {
		// If adapter doesn't support enums, continue with empty list
		dbEnums = []types.SchemaEnum{}
	}

	if len(dbTables) == 0 && len(dbEnums) == 0 {
		fmt.Println("📄 No tables or enums found in database")
		return nil
	}

	var existingSQL string
	if content, err := os.ReadFile(schemaPath); err == nil {
		existingSQL = string(content)
	}

	hasChanges, updatedSQL := s.comparator.CompareWithDatabase(existingSQL, dbTables)

	// Add enums at the beginning of the SQL file
	if len(dbEnums) > 0 {
		enumSQL := s.generateEnumSQL(dbEnums)
		if enumSQL != "" {
			// Remove existing enums from updatedSQL and prepend new ones
			updatedSQL = s.removeExistingEnums(updatedSQL)
			updatedSQL = enumSQL + "\n\n" + updatedSQL
			hasChanges = true
		}
	}

	if !hasChanges {
		fmt.Println("✅ Schema is up to date - no structural changes detected")
		return nil
	}

	if opts.Backup && existingSQL != "" {
		if err := s.createBackup(schemaPath); err != nil {
			fmt.Printf("⚠️  Warning: Failed to create backup: %v\n", err)
		} else {
			fmt.Println("💾 Created backup of existing schema")
		}
	}

	if err := os.WriteFile(schemaPath, []byte(updatedSQL), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	fmt.Printf("✅ Schema updated: %s\n", schemaPath)
	if len(dbEnums) > 0 {
		fmt.Printf("📊 Processed %d enums and %d tables\n", len(dbEnums), len(dbTables))
	} else {
		fmt.Printf("📊 Processed %d tables\n", len(dbTables))
	}
	fmt.Println("🎯 Only actual structural differences were updated")

	return nil
}

func (s *Service) createBackup(schemaPath string) error {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	backupPath := schemaPath + ".backup"
	return os.WriteFile(backupPath, content, 0644)
}

func (s *Service) generateEnumSQL(enums []types.SchemaEnum) string {
	if len(enums) == 0 {
		return ""
	}

	var parts []string
	for _, enum := range enums {
		values := make([]string, len(enum.Values))
		for i, v := range enum.Values {
			values[i] = fmt.Sprintf("'%s'", v)
		}
		parts = append(parts, fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", enum.Name, strings.Join(values, ", ")))
	}
	return strings.Join(parts, "\n")
}

func (s *Service) removeExistingEnums(sql string) string {
	// Remove existing CREATE TYPE statements
	enumRegex := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+\w+\s+AS\s+ENUM\s*\([^)]+\)\s*;[\s\n]*`)
	return enumRegex.ReplaceAllString(sql, "")
}
