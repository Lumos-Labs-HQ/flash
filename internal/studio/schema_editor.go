package studio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

type SchemaChange struct {
	Type       string           `json:"type"` // add_column, drop_column, modify_column, add_table, drop_table, create_table, create_enum, alter_enum, drop_enum
	Table      string           `json:"table"`
	Column     *ColumnChange    `json:"column,omitempty"`
	Columns    []ColumnChange   `json:"columns,omitempty"`
	TableDef   *TableDefinition `json:"table_def,omitempty"`
	EnumName   string           `json:"enum_name,omitempty"`
	EnumValues []string         `json:"enum_values,omitempty"`
	SQL        string           `json:"sql"`
}

type ColumnChange struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Nullable      bool              `json:"nullable"`
	Default       string            `json:"default,omitempty"`
	OldName       string            `json:"old_name,omitempty"`
	Unique        bool              `json:"unique,omitempty"`
	AutoIncrement bool              `json:"auto_increment,omitempty"`
	ForeignKey    *ForeignKeyChange `json:"foreign_key,omitempty"`
}

type ForeignKeyChange struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type TableDefinition struct {
	Name    string         `json:"name"`
	Columns []ColumnChange `json:"columns"`
}

type SchemaPreview struct {
	SQL         string   `json:"sql"`
	Description string   `json:"description"`
	Changes     []string `json:"changes"`
}

// PreviewSchemaChange generates SQL preview for a schema change
func (s *Service) PreviewSchemaChange(change *SchemaChange) (*SchemaPreview, error) {
	sql := s.generateSQL(change)

	preview := &SchemaPreview{
		SQL:         sql,
		Description: s.getChangeDescription(change),
		Changes: []string{
			"Apply to database immediately",
			"Create migration file",
			"Update db/schema/schema.sql",
		},
	}

	return preview, nil
}

// ApplySchemaChange applies the change to database and syncs files
func (s *Service) ApplySchemaChange(change *SchemaChange, configPath string) error {
	// 1. Check if column already exists (for add_column)
	if change.Type == "add_column" {
		exists, err := s.adapter.CheckColumnExists(s.ctx, change.Table, change.Column.Name)
		if err == nil && exists {
			return fmt.Errorf("column '%s' already exists in table '%s'", change.Column.Name, change.Table)
		}
	}

	// 2. Apply to database
	sql := s.generateSQL(change)
	_, err := s.adapter.ExecuteQuery(s.ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to apply schema change: %w", err)
	}

	// 3. Generate migration file (skip if no config path)
	if configPath != "" {
		if err := s.generateMigrationFile(change, sql, configPath); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to generate migration: %v\n", err)
		}

		// 4. Update schema.sql file (skip if no config path)
		if err := s.syncSchemaFile(configPath); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to sync schema file: %v\n", err)
		}
	}

	return nil
}

func (s *Service) generateSQL(change *SchemaChange) string {
	switch change.Type {
	case "add_column":
		return s.generateAddColumn(change)
	case "drop_column":
		return s.generateDropColumn(change)
	case "modify_column":
		return s.generateModifyColumn(change)
	case "add_table":
		return s.generateAddTable(change)
	case "drop_table":
		return fmt.Sprintf("DROP TABLE %s;", change.Table)
	case "create_table":
		return s.generateCreateTable(change)
	case "create_enum":
		return s.generateCreateEnum(change)
	case "alter_enum":
		return s.generateAlterEnum(change)
	case "drop_enum":
		return fmt.Sprintf("DROP TYPE %s;", change.EnumName)
	default:
		return change.SQL
	}
}

func (s *Service) generateAddColumn(change *SchemaChange) string {
	col := change.Column

	// For auto-increment, use appropriate syntax
	colType := col.Type
	if col.AutoIncrement {
		// PostgreSQL auto-increment
		if strings.ToUpper(col.Type) == "INTEGER" {
			colType = "SERIAL"
		} else if strings.ToUpper(col.Type) == "BIGINT" {
			colType = "BIGSERIAL"
		} else if strings.ToUpper(col.Type) == "SMALLINT" {
			colType = "SMALLSERIAL"
		}
	}

	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		change.Table, col.Name, colType)

	if !col.Nullable {
		sql += " NOT NULL"
	}

	if col.Unique {
		sql += " UNIQUE"
	}

	if col.Default != "" && !col.AutoIncrement {
		sql += fmt.Sprintf(" DEFAULT %s", col.Default)
	}

	sql += ";"

	// Add foreign key constraint if specified
	if col.ForeignKey != nil {
		constraintName := fmt.Sprintf("fk_%s_%s", change.Table, col.Name)
		sql += fmt.Sprintf("\nALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s);",
			change.Table, constraintName, col.Name, col.ForeignKey.Table, col.ForeignKey.Column)
	}

	return sql
}

func (s *Service) generateDropColumn(change *SchemaChange) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
		change.Table, change.Column.Name)
}

func (s *Service) generateModifyColumn(change *SchemaChange) string {
	col := change.Column
	// PostgreSQL syntax
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
		change.Table, col.Name, col.Type)
}

func (s *Service) generateAddTable(change *SchemaChange) string {
	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", change.TableDef.Name))

	for i, col := range change.TableDef.Columns {
		sql.WriteString(fmt.Sprintf("  %s %s", col.Name, col.Type))

		if !col.Nullable {
			sql.WriteString(" NOT NULL")
		}

		if col.Default != "" {
			sql.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
		}

		if i < len(change.TableDef.Columns)-1 {
			sql.WriteString(",\n")
		}
	}

	sql.WriteString("\n);")
	return sql.String()
}

func (s *Service) getChangeDescription(change *SchemaChange) string {
	switch change.Type {
	case "add_column":
		return fmt.Sprintf("Add column '%s' to table '%s'",
			change.Column.Name, change.Table)
	case "drop_column":
		return fmt.Sprintf("Drop column '%s' from table '%s'",
			change.Column.Name, change.Table)
	case "modify_column":
		return fmt.Sprintf("Modify column '%s' in table '%s'",
			change.Column.Name, change.Table)
	case "add_table":
		return fmt.Sprintf("Create table '%s'", change.TableDef.Name)
	case "drop_table":
		return fmt.Sprintf("Drop table '%s'", change.Table)
	default:
		return "Custom schema change"
	}
}

func (s *Service) generateMigrationFile(change *SchemaChange, sql, configPath string) error {
	// Get migrations directory from config
	migrationsPath := "db/migrations"
	if configPath != "" {
		dir := filepath.Dir(configPath)
		migrationsPath = filepath.Join(dir, "db/migrations")
	}

	// Create migrations directory if not exists
	if err := os.MkdirAll(migrationsPath, 0755); err != nil {
		return err
	}

	// Generate migration filename
	timestamp := time.Now().Format("20060102150405")
	description := strings.ReplaceAll(s.getChangeDescription(change), " ", "_")
	description = strings.ToLower(description)
	filename := fmt.Sprintf("%s_%s.sql", timestamp, description)

	// Create migration content
	content := fmt.Sprintf(`-- Migration: %s
-- Created: %s
-- Generated by Studio

%s
`, s.getChangeDescription(change), time.Now().Format(time.RFC3339), sql)

	// Write migration file
	path := filepath.Join(migrationsPath, filename)
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *Service) syncSchemaFile(configPath string) error {
	// Get schema path from config
	schemaPath := "db/schema/schema.sql"
	if configPath != "" {
		dir := filepath.Dir(configPath)
		schemaPath = filepath.Join(dir, "db/schema/schema.sql")
	}

	// Pull current schema from database
	tables, err := s.adapter.PullCompleteSchema(s.ctx)
	if err != nil {
		return err
	}

	// Generate schema SQL
	sql := s.generateSchemaSQL(tables)

	// Write to file
	return os.WriteFile(schemaPath, []byte(sql), 0644)
}

func (s *Service) generateSchemaSQL(tables []types.SchemaTable) string {
	var sql strings.Builder

	for i, table := range tables {
		if table.Name == "_flash_migrations" {
			continue
		}

		sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

		for j, col := range table.Columns {
			sql.WriteString(fmt.Sprintf("  %s %s", col.Name, col.Type))

			if !col.Nullable {
				sql.WriteString(" NOT NULL")
			}

			if col.Default != "" {
				sql.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
			}

			if j < len(table.Columns)-1 {
				sql.WriteString(",\n")
			}
		}

		sql.WriteString("\n);")

		if i < len(tables)-1 {
			sql.WriteString("\n\n")
		}
	}

	return sql.String()
}

func (s *Service) generateCreateTable(change *SchemaChange) string {
	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", change.Table))

	for i, col := range change.Columns {
		// Handle auto-increment
		colType := col.Type
		if col.AutoIncrement {
			if strings.ToUpper(col.Type) == "INTEGER" {
				colType = "SERIAL"
			} else if strings.ToUpper(col.Type) == "BIGINT" {
				colType = "BIGSERIAL"
			} else if strings.ToUpper(col.Type) == "SMALLINT" {
				colType = "SMALLSERIAL"
			}
		}

		sql.WriteString(fmt.Sprintf("  %s %s", col.Name, colType))

		if !col.Nullable {
			sql.WriteString(" NOT NULL")
		}

		if col.Unique {
			sql.WriteString(" UNIQUE")
		}

		if col.Default != "" && !col.AutoIncrement {
			sql.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
		}

		// Check if this is primary key column
		if i == 0 || col.Name == "id" {
			sql.WriteString(" PRIMARY KEY")
		}

		if i < len(change.Columns)-1 {
			sql.WriteString(",\n")
		}
	}

	sql.WriteString("\n);")
	return sql.String()
}

func (s *Service) generateCreateEnum(change *SchemaChange) string {
	values := make([]string, len(change.EnumValues))
	for i, v := range change.EnumValues {
		values[i] = fmt.Sprintf("'%s'", v)
	}
	return fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", change.EnumName, strings.Join(values, ", "))
}

func (s *Service) generateAlterEnum(change *SchemaChange) string {
	// PostgreSQL doesn't support ALTER TYPE directly, need to drop and recreate
	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("-- Recreating enum %s\n", change.EnumName))
	sql.WriteString(fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE;\n", change.EnumName))

	values := make([]string, len(change.EnumValues))
	for i, v := range change.EnumValues {
		values[i] = fmt.Sprintf("'%s'", v)
	}
	sql.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", change.EnumName, strings.Join(values, ", ")))

	return sql.String()
}
