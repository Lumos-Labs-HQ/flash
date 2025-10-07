package pull

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/database"
	"github.com/Rana718/Graft/internal/types"
	"github.com/Rana718/Graft/internal/utils"
)

type Options struct {
	Force      bool
	Backup     bool
	OutputPath string
}

type Service struct {
	config  *config.Config
	adapter database.DatabaseAdapter
	utils   *utils.InputUtils
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
		config:  cfg,
		adapter: adapter,
		utils:   &utils.InputUtils{},
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

	schema, err := s.adapter.PullCompleteSchema(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull database schema: %w", err)
	}

	if len(schema) == 0 {
		fmt.Println("📄 No tables found in database")
		return nil
	}

	if opts.Backup {
		if err := s.backupExistingSchema(schemaPath); err != nil {
			fmt.Printf("⚠️  Warning: Failed to backup existing schema: %v\n", err)
		} else {
			fmt.Println("💾 Backed up existing schema file")
		}
	}

	if !opts.Force {
		if _, err := os.Stat(schemaPath); err == nil {
			fmt.Printf("📁 Schema file already exists: %s\n", schemaPath)
			if !s.utils.AskConfirmation("Overwrite existing schema file?", false) {
				fmt.Println("❌ Operation cancelled")
				return nil
			}
		}
	}

	schemaContent := s.generateSchemaSQL(schema)

	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	fmt.Printf("✅ Successfully pulled database schema to %s\n", schemaPath)
	fmt.Printf("📊 Found %d tables with schema definitions\n", len(schema))

	return nil
}

func (s *Service) generateSchemaSQL(tables []types.SchemaTable) string {
	var builder strings.Builder

	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	for i, table := range tables {
		if i > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(fmt.Sprintf("-- %s table\n", strings.Title(table.Name)))
		builder.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table.Name))

		for j, column := range table.Columns {
			if j > 0 {
				builder.WriteString(",\n")
			}

			columnDef := s.formatColumnDefinition(column)
			builder.WriteString(fmt.Sprintf("    %s", columnDef))
		}

		builder.WriteString("\n);\n")
	}

	return builder.String()
}

func (s *Service) formatColumnDefinition(column types.SchemaColumn) string {
	var parts []string

	parts = append(parts, column.Name, column.Type)

	if column.IsPrimary {
		parts = append(parts, "PRIMARY KEY")
	} else {
		// Handle UNIQUE before NOT NULL for better formatting
		if column.IsUnique {
			parts = append(parts, "UNIQUE")
		}
		
		if !column.Nullable {
			parts = append(parts, "NOT NULL")
		}
	}

	if column.Default != "" {
		parts = append(parts, "DEFAULT", column.Default)
	}

	if column.ForeignKeyTable != "" && column.ForeignKeyColumn != "" {
		fkDef := fmt.Sprintf("REFERENCES %s(%s)", column.ForeignKeyTable, column.ForeignKeyColumn)
		if column.OnDeleteAction != "" {
			fkDef += fmt.Sprintf(" ON DELETE %s", strings.ToUpper(column.OnDeleteAction))
		}
		parts = append(parts, fkDef)
	}

	return strings.Join(parts, " ")
}

func (s *Service) backupExistingSchema(schemaPath string) error {
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return nil
	}

	backupPath := schemaPath + ".backup"
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read existing schema: %w", err)
	}

	return os.WriteFile(backupPath, content, 0644)
}
