package pull

import (
	"context"
	"fmt"
	"os"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/database"
	"github.com/Rana718/Graft/internal/schema"
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

	if len(dbTables) == 0 {
		fmt.Println("📄 No tables found in database")
		return nil
	}

	var existingSQL string
	if content, err := os.ReadFile(schemaPath); err == nil {
		existingSQL = string(content)
	}

	hasChanges, updatedSQL := s.comparator.CompareWithDatabase(existingSQL, dbTables)
	
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
	fmt.Printf("📊 Processed %d tables\n", len(dbTables))
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
