package schema

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Rana718/Graft/internal/types"
)

type SchemaComparator struct{}

func NewSchemaComparator() *SchemaComparator {
	return &SchemaComparator{}
}

// CompareSchemas compares existing schema file with database schema
func (sc *SchemaComparator) CompareSchemas(existingSQL string, dbTables []types.SchemaTable) (bool, string) {
	existingTables := sc.parseExistingSchema(existingSQL)
	dbTablesNormalized := sc.normalizeDBTables(dbTables)
	
	if sc.areTablesEqual(existingTables, dbTablesNormalized) {
		return false, "" // No changes needed
	}
	
	// Generate new schema
	newSchema := sc.generateSchema(dbTablesNormalized)
	return true, newSchema
}

// parseExistingSchema extracts table definitions from existing SQL
func (sc *SchemaComparator) parseExistingSchema(sql string) map[string]*NormalizedTable {
	tables := make(map[string]*NormalizedTable)
	
	// Remove comments and normalize whitespace
	cleanSQL := sc.cleanSQL(sql)
	
	// Extract CREATE TABLE statements
	createTableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\((.*?)\);`)
	matches := createTableRegex.FindAllStringSubmatch(cleanSQL, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := strings.ToLower(match[1])
			columnsDef := match[2]
			
			table := &NormalizedTable{
				Name:    tableName,
				Columns: sc.parseColumns(columnsDef),
			}
			
			tables[tableName] = table
		}
	}
	
	return tables
}

// normalizeDBTables converts database tables to normalized format
func (sc *SchemaComparator) normalizeDBTables(dbTables []types.SchemaTable) map[string]*NormalizedTable {
	tables := make(map[string]*NormalizedTable)
	
	for _, dbTable := range dbTables {
		table := &NormalizedTable{
			Name:    strings.ToLower(dbTable.Name),
			Columns: make(map[string]*NormalizedColumn),
		}
		
		for _, dbCol := range dbTable.Columns {
			col := &NormalizedColumn{
				Name:     strings.ToLower(dbCol.Name),
				Type:     sc.normalizeType(dbCol.Type),
				Nullable: dbCol.Nullable,
				Primary:  dbCol.IsPrimary,
				Unique:   dbCol.IsUnique,
				Default:  sc.normalizeDefault(dbCol.Default),
				ForeignKey: sc.normalizeForeignKey(dbCol.ForeignKeyTable, dbCol.ForeignKeyColumn, dbCol.OnDeleteAction),
			}
			
			table.Columns[col.Name] = col
		}
		
		tables[table.Name] = table
	}
	
	return tables
}

// parseColumns extracts column definitions from CREATE TABLE statement
func (sc *SchemaComparator) parseColumns(columnsDef string) map[string]*NormalizedColumn {
	columns := make(map[string]*NormalizedColumn)
	
	// Split by comma, but be careful with nested parentheses
	columnParts := sc.splitColumns(columnsDef)
	
	for _, part := range columnParts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(strings.ToUpper(part), "FOREIGN KEY") {
			continue // Skip foreign key constraints and empty parts
		}
		
		col := sc.parseColumnDefinition(part)
		if col != nil {
			columns[col.Name] = col
		}
	}
	
	return columns
}

// parseColumnDefinition parses a single column definition
func (sc *SchemaComparator) parseColumnDefinition(def string) *NormalizedColumn {
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return nil
	}
	
	col := &NormalizedColumn{
		Name:     strings.ToLower(parts[0]),
		Type:     sc.normalizeType(parts[1]),
		Nullable: true, // Default to nullable
	}
	
	defUpper := strings.ToUpper(def)
	
	// Check constraints
	col.Primary = strings.Contains(defUpper, "PRIMARY KEY")
	col.Unique = strings.Contains(defUpper, "UNIQUE")
	col.Nullable = !strings.Contains(defUpper, "NOT NULL")
	
	// Extract default value
	if defaultMatch := regexp.MustCompile(`(?i)DEFAULT\s+([^,\s]+(?:\([^)]*\))?)`).FindStringSubmatch(def); len(defaultMatch) > 1 {
		col.Default = sc.normalizeDefault(defaultMatch[1])
	}
	
	// Extract foreign key reference
	if refMatch := regexp.MustCompile(`(?i)REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)(?:\s+ON\s+DELETE\s+(\w+(?:\s+\w+)?))?`).FindStringSubmatch(def); len(refMatch) > 2 {
		col.ForeignKey = sc.normalizeForeignKey(refMatch[1], refMatch[2], refMatch[3])
	}
	
	return col
}

// areTablesEqual compares two sets of normalized tables
func (sc *SchemaComparator) areTablesEqual(existing, db map[string]*NormalizedTable) bool {
	if len(existing) != len(db) {
		return false
	}
	
	for tableName, dbTable := range db {
		existingTable, exists := existing[tableName]
		if !exists {
			return false
		}
		
		if !sc.areColumnsEqual(existingTable.Columns, dbTable.Columns) {
			return false
		}
	}
	
	return true
}

// areColumnsEqual compares two sets of normalized columns
func (sc *SchemaComparator) areColumnsEqual(existing, db map[string]*NormalizedColumn) bool {
	if len(existing) != len(db) {
		return false
	}
	
	for colName, dbCol := range db {
		existingCol, exists := existing[colName]
		if !exists {
			return false
		}
		
		if !sc.areColumnPropertiesEqual(existingCol, dbCol) {
			return false
		}
	}
	
	return true
}

// areColumnPropertiesEqual compares individual column properties
func (sc *SchemaComparator) areColumnPropertiesEqual(existing, db *NormalizedColumn) bool {
	return existing.Type == db.Type &&
		existing.Nullable == db.Nullable &&
		existing.Primary == db.Primary &&
		existing.Unique == db.Unique &&
		existing.Default == db.Default &&
		existing.ForeignKey == db.ForeignKey
}

// Helper functions for normalization
func (sc *SchemaComparator) normalizeType(dataType string) string {
	// Normalize common type variations
	typeUpper := strings.ToUpper(strings.TrimSpace(dataType))
	
	// Handle common variations
	switch {
	case typeUpper == "INTEGER" || typeUpper == "INT":
		return "INT"
	case strings.HasPrefix(typeUpper, "VARCHAR"):
		return typeUpper
	case typeUpper == "TIMESTAMP WITHOUT TIME ZONE":
		return "TIMESTAMP"
	case typeUpper == "TIMESTAMP WITH TIME ZONE":
		return "TIMESTAMP WITH TIME ZONE"
	default:
		return typeUpper
	}
}

func (sc *SchemaComparator) normalizeDefault(defaultVal string) string {
	if defaultVal == "" {
		return ""
	}
	
	defaultUpper := strings.ToUpper(strings.TrimSpace(defaultVal))
	
	// Normalize common default values
	switch {
	case strings.Contains(defaultUpper, "NOW()") || strings.Contains(defaultUpper, "CURRENT_TIMESTAMP"):
		return "NOW()"
	case strings.Contains(defaultUpper, "NEXTVAL"):
		return "" // Skip sequence defaults for SERIAL columns
	default:
		return strings.Trim(defaultVal, "'\"")
	}
}

func (sc *SchemaComparator) normalizeForeignKey(table, column, onDelete string) string {
	if table == "" || column == "" {
		return ""
	}
	
	fk := strings.ToLower(table) + "." + strings.ToLower(column)
	if onDelete != "" {
		fk += ":" + strings.ToUpper(strings.TrimSpace(onDelete))
	}
	
	return fk
}

func (sc *SchemaComparator) cleanSQL(sql string) string {
	// Remove single-line comments
	sql = regexp.MustCompile(`--.*`).ReplaceAllString(sql, "")
	
	// Remove multi-line comments
	sql = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(sql, "")
	
	// Normalize whitespace
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")
	
	return strings.TrimSpace(sql)
}

func (sc *SchemaComparator) splitColumns(columnsDef string) []string {
	var parts []string
	var current strings.Builder
	parenLevel := 0
	
	for _, char := range columnsDef {
		switch char {
		case '(':
			parenLevel++
			current.WriteRune(char)
		case ')':
			parenLevel--
			current.WriteRune(char)
		case ',':
			if parenLevel == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}
	
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	
	return parts
}

func (sc *SchemaComparator) generateSchema(tables map[string]*NormalizedTable) string {
	var builder strings.Builder
	
	// Sort tables by name for consistent output
	var tableNames []string
	for name := range tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)
	
	for i, tableName := range tableNames {
		if i > 0 {
			builder.WriteString("\n")
		}
		
		table := tables[tableName]
		builder.WriteString(sc.generateTableSQL(table))
	}
	
	return builder.String()
}

func (sc *SchemaComparator) generateTableSQL(table *NormalizedTable) string {
	var builder strings.Builder
	
	builder.WriteString("-- ")
	builder.WriteString(strings.Title(table.Name))
	builder.WriteString(" table\n")
	builder.WriteString("CREATE TABLE IF NOT EXISTS ")
	builder.WriteString(table.Name)
	builder.WriteString(" (\n")
	
	// Sort columns by name for consistent output
	var columnNames []string
	for name := range table.Columns {
		columnNames = append(columnNames, name)
	}
	sort.Strings(columnNames)
	
	for i, colName := range columnNames {
		if i > 0 {
			builder.WriteString(",\n")
		}
		
		col := table.Columns[colName]
		builder.WriteString("    ")
		builder.WriteString(sc.generateColumnSQL(col))
	}
	
	builder.WriteString("\n);\n")
	return builder.String()
}

func (sc *SchemaComparator) generateColumnSQL(col *NormalizedColumn) string {
	var parts []string
	
	parts = append(parts, col.Name, col.Type)
	
	if col.Primary {
		parts = append(parts, "PRIMARY KEY")
	} else {
		if col.Unique {
			parts = append(parts, "UNIQUE")
		}
		if !col.Nullable {
			parts = append(parts, "NOT NULL")
		}
	}
	
	if col.Default != "" {
		parts = append(parts, "DEFAULT", col.Default)
	}
	
	if col.ForeignKey != "" {
		fkParts := strings.Split(col.ForeignKey, ":")
		tableDotColumn := strings.Split(fkParts[0], ".")
		if len(tableDotColumn) == 2 {
			fkSQL := "REFERENCES " + tableDotColumn[0] + "(" + tableDotColumn[1] + ")"
			if len(fkParts) > 1 {
				fkSQL += " ON DELETE " + fkParts[1]
			}
			parts = append(parts, fkSQL)
		}
	}
	
	return strings.Join(parts, " ")
}

// NormalizedTable represents a table in normalized form for comparison
type NormalizedTable struct {
	Name    string
	Columns map[string]*NormalizedColumn
}

// NormalizedColumn represents a column in normalized form for comparison
type NormalizedColumn struct {
	Name       string
	Type       string
	Nullable   bool
	Primary    bool
	Unique     bool
	Default    string
	ForeignKey string
}
