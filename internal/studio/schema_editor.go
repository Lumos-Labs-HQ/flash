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
	Type       string           `json:"type"` 
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
	IsPrimary     bool              `json:"is_primary,omitempty"`
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
	if change.Type == "add_column" {
		exists, err := s.adapter.CheckColumnExists(s.ctx, change.Table, change.Column.Name)
		if err == nil && exists {
			return fmt.Errorf("column '%s' already exists in table '%s'", change.Column.Name, change.Table)
		}
	}

	sql := s.generateSQL(change)
	_, err := s.adapter.ExecuteQuery(s.ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to apply schema change: %w", err)
	}

	if configPath != "" {
		if err := s.generateMigrationFile(change, sql, configPath); err != nil {
			fmt.Printf("Warning: failed to generate migration: %v\n", err)
		}

		if err := s.syncSchemaFile(configPath); err != nil {
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

// sanitizeDefaultValue ensures default value is safe and valid for PostgreSQL
func sanitizeDefaultValue(defaultVal string, colType string) string {
	if defaultVal == "" {
		return ""
	}

	defaultVal = strings.TrimSpace(defaultVal)

	// Handle common cases
	upper := strings.ToUpper(defaultVal)

	if upper == "NULL" {
		return "NULL"
	}

	validFunctions := []string{
		"NOW()", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME",
		"GEN_RANDOM_UUID()", "UUID_GENERATE_V4()",
		"TRUE", "FALSE",
	}

	for _, fn := range validFunctions {
		if upper == fn || upper == strings.TrimSuffix(fn, "()") {
			return fn
		}
	}

	if strings.Contains(strings.ToUpper(colType), "BOOL") {
		if upper == "TRUE" || upper == "T" || upper == "1" {
			return "TRUE"
		}
		if upper == "FALSE" || upper == "F" || upper == "0" {
			return "FALSE"
		}
	}

	if strings.Contains(strings.ToUpper(colType), "INT") ||
		strings.Contains(strings.ToUpper(colType), "SERIAL") ||
		strings.Contains(strings.ToUpper(colType), "DECIMAL") ||
		strings.Contains(strings.ToUpper(colType), "NUMERIC") ||
		strings.Contains(strings.ToUpper(colType), "FLOAT") ||
		strings.Contains(strings.ToUpper(colType), "DOUBLE") ||
		strings.Contains(strings.ToUpper(colType), "REAL") {
		if _, err := fmt.Sscanf(defaultVal, "%f", new(float64)); err == nil {
			return defaultVal
		}
	}

	if strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'") {
		return defaultVal
	}

	if strings.Contains(defaultVal, "(") && strings.Contains(defaultVal, ")") {
		return defaultVal
	}

	// Escape single quotes
	escaped := strings.ReplaceAll(defaultVal, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
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
		sanitized := sanitizeDefaultValue(col.Default, colType)
		if sanitized != "" {
			sql += fmt.Sprintf(" DEFAULT %s", sanitized)
		}
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
	var statements []string

	// Handle type change
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
	statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", change.Table, col.Name, colType))

	if col.Nullable {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", change.Table, col.Name))
	} else {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", change.Table, col.Name))
	}

	if col.Default != "" && !col.AutoIncrement {
		sanitized := sanitizeDefaultValue(col.Default, colType)
		if sanitized != "" {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", change.Table, col.Name, sanitized))
		}
	} else if !col.AutoIncrement {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", change.Table, col.Name))
	}

	return strings.Join(statements, ";\n") + ";"
}

func (s *Service) generateAddTable(change *SchemaChange) string {
	var sql strings.Builder
	var foreignKeys []string

	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", change.TableDef.Name))

	for i, col := range change.TableDef.Columns {
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

		isPrimary := col.IsPrimary

		if isPrimary {
			sql.WriteString(" PRIMARY KEY")
		}

		if !col.Nullable && !isPrimary {
			sql.WriteString(" NOT NULL")
		}

		if col.Unique && !isPrimary {
			sql.WriteString(" UNIQUE")
		}

		if col.Default != "" && !col.AutoIncrement {
			sanitized := sanitizeDefaultValue(col.Default, colType)
			if sanitized != "" {
				sql.WriteString(fmt.Sprintf(" DEFAULT %s", sanitized))
			}
		}

		if col.ForeignKey != nil {
			constraintName := fmt.Sprintf("fk_%s_%s", change.TableDef.Name, col.Name)
			fkSQL := fmt.Sprintf("  CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				constraintName, col.Name, col.ForeignKey.Table, col.ForeignKey.Column)
			foreignKeys = append(foreignKeys, fkSQL)
		}

		if i < len(change.TableDef.Columns)-1 || len(foreignKeys) > 0 {
			sql.WriteString(",\n")
		}
	}

	for i, fk := range foreignKeys {
		sql.WriteString(fk)
		if i < len(foreignKeys)-1 {
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
	migrationsPath := "db/migrations"
	if configPath != "" {
		dir := filepath.Dir(configPath)
		migrationsPath = filepath.Join(dir, "db/migrations")
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
	schemaPath := "db/schema/schema.sql"
	if configPath != "" {
		dir := filepath.Dir(configPath)
		schemaPath = filepath.Join(dir, "db/schema/schema.sql")
	}

	tables, err := s.adapter.PullCompleteSchema(s.ctx)
	if err != nil {
		return err
	}

	enums, _ := s.adapter.GetCurrentEnums(s.ctx)

	sql := s.generateSchemaSQL(tables, enums)
	return os.WriteFile(schemaPath, []byte(sql), 0644)
}

func (s *Service) generateSchemaSQL(tables []types.SchemaTable, enums []types.SchemaEnum) string {
	var sql strings.Builder

	if len(enums) > 0 {
		for _, enum := range enums {
			sql.WriteString(fmt.Sprintf("CREATE TYPE \"%s\" AS ENUM (", enum.Name))
			for i, val := range enum.Values {
				sql.WriteString(fmt.Sprintf("'%s'", val))
				if i < len(enum.Values)-1 {
					sql.WriteString(", ")
				}
			}
			sql.WriteString(");\n")
		}
		sql.WriteString("\n")
	}

	for i, table := range tables {
		if table.Name == "_flash_migrations" {
			continue
		}

		sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (\n", table.Name))

		var foreignKeys []string

		for _, col := range table.Columns {
			if col.ForeignKeyTable != "" && col.ForeignKeyColumn != "" {
				fkDef := fmt.Sprintf("  FOREIGN KEY (\"%s\") REFERENCES \"%s\"(\"%s\")",
					col.Name, col.ForeignKeyTable, col.ForeignKeyColumn)

				if col.OnDeleteAction != "" {
					fkDef += fmt.Sprintf(" ON DELETE %s", col.OnDeleteAction)
				}

				foreignKeys = append(foreignKeys, fkDef)
			}
		}

		for j, col := range table.Columns {
			sql.WriteString(fmt.Sprintf("  \"%s\" %s", col.Name, col.Type))

			// PRIMARY KEY
			if col.IsPrimary {
				sql.WriteString(" PRIMARY KEY")
			}

			// UNIQUE (skip if primary key)
			if col.IsUnique && !col.IsPrimary {
				sql.WriteString(" UNIQUE")
			}

			// NOT NULL (skip if primary key)
			if !col.Nullable && !col.IsPrimary {
				sql.WriteString(" NOT NULL")
			}

			// DEFAULT
			if col.Default != "" && !strings.Contains(col.Default, "nextval") {
				sql.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
			}

			// Add comma if not last column or if there are foreign keys
			if j < len(table.Columns)-1 || len(foreignKeys) > 0 {
				sql.WriteString(",")
			}
			sql.WriteString("\n")
		}

		for j, fkDef := range foreignKeys {
			sql.WriteString(fkDef)
			if j < len(foreignKeys)-1 {
				sql.WriteString(",")
			}
			sql.WriteString("\n")
		}

		sql.WriteString(");")

		if i < len(tables)-1 {
			sql.WriteString("\n\n")
		}
	}

	return sql.String()
}

func (s *Service) generateCreateTable(change *SchemaChange) string {
	var sql strings.Builder
	var foreignKeys []string

	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", change.Table))

	for i, col := range change.Columns {
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

		// Check if marked as primary key
		isPrimary := col.IsPrimary

		if isPrimary {
			sql.WriteString(" PRIMARY KEY")
		}

		if !col.Nullable && !isPrimary {
			sql.WriteString(" NOT NULL")
		}

		if col.Unique && !isPrimary {
			sql.WriteString(" UNIQUE")
		}

		if col.Default != "" && !col.AutoIncrement {
			sanitized := sanitizeDefaultValue(col.Default, colType)
			if sanitized != "" {
				sql.WriteString(fmt.Sprintf(" DEFAULT %s", sanitized))
			}
		}

		// Collect foreign key constraints
		if col.ForeignKey != nil {
			constraintName := fmt.Sprintf("fk_%s_%s", change.Table, col.Name)
			fkSQL := fmt.Sprintf("  CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				constraintName, col.Name, col.ForeignKey.Table, col.ForeignKey.Column)
			foreignKeys = append(foreignKeys, fkSQL)
		}

		if i < len(change.Columns)-1 || len(foreignKeys) > 0 {
			sql.WriteString(",\n")
		}
	}

	// Add foreign key constraints
	for i, fk := range foreignKeys {
		sql.WriteString(fk)
		if i < len(foreignKeys)-1 {
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
