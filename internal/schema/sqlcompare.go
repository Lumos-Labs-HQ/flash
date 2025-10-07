package schema

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Rana718/Graft/internal/types"
)

type SQLComparator struct{}

func NewSQLComparator() *SQLComparator {
	return &SQLComparator{}
}

// CompareWithDatabase compares existing SQL file with database tables
func (sc *SQLComparator) CompareWithDatabase(existingSQL string, dbTables []types.SchemaTable) (bool, string) {
	// Parse existing SQL into normalized structure
	existingTables := sc.parseSQL(existingSQL)
	
	// Convert database tables to normalized structure
	dbTablesNorm := sc.normalizeDBTables(dbTables)
	
	// Compare structures
	if sc.areEqual(existingTables, dbTablesNorm) {
		return false, "" // No changes needed
	}
	
	// Generate updated SQL preserving original formatting where possible
	updatedSQL := sc.generateUpdatedSQL(existingSQL, existingTables, dbTablesNorm)
	return true, updatedSQL
}

// parseSQL extracts table structures from SQL preserving original order
func (sc *SQLComparator) parseSQL(sql string) map[string]*TableStructure {
	tables := make(map[string]*TableStructure)
	
	// Find all CREATE TABLE statements
	createTableRegex := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\((.*?)\);`)
	matches := createTableRegex.FindAllStringSubmatch(sql, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := strings.ToLower(strings.TrimSpace(match[1]))
			columnsDef := match[2]
			
			columns, columnOrder := sc.parseColumnsWithOrder(columnsDef)
			
			table := &TableStructure{
				Name:        tableName,
				Columns:     columns,
				ColumnOrder: columnOrder, // This preserves the original SQL file order
			}
			
			tables[tableName] = table
		}
	}
	
	return tables
}

// parseColumnsWithOrder extracts column definitions preserving order
func (sc *SQLComparator) parseColumnsWithOrder(columnsDef string) (map[string]*ColumnStructure, []string) {
	columns := make(map[string]*ColumnStructure)
	var columnOrder []string
	
	// Split by comma, handling nested parentheses
	parts := sc.smartSplit(columnsDef, ',')
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(strings.ToUpper(part), "FOREIGN KEY") {
			continue
		}
		
		col := sc.parseColumn(part)
		if col != nil {
			columns[col.Name] = col
			columnOrder = append(columnOrder, col.Name)
		}
	}
	
	return columns, columnOrder
}

// parseColumns extracts column definitions
func (sc *SQLComparator) parseColumns(columnsDef string) map[string]*ColumnStructure {
	columns := make(map[string]*ColumnStructure)
	
	// Split by comma, handling nested parentheses
	parts := sc.smartSplit(columnsDef, ',')
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(strings.ToUpper(part), "FOREIGN KEY") {
			continue
		}
		
		col := sc.parseColumn(part)
		if col != nil {
			columns[col.Name] = col
		}
	}
	
	return columns
}

// parseColumn parses individual column definition
func (sc *SQLComparator) parseColumn(def string) *ColumnStructure {
	// Extract column name (first word)
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return nil
	}
	
	col := &ColumnStructure{
		Name: strings.ToLower(strings.TrimSpace(parts[0])),
	}
	
	defUpper := strings.ToUpper(def)
	
	// Extract and normalize properties
	col.Properties = sc.extractProperties(defUpper)
	
	return col
}

// extractProperties extracts all column properties in normalized form
func (sc *SQLComparator) extractProperties(def string) map[string]string {
	props := make(map[string]string)
	
	// Extract data type
	if typeMatch := regexp.MustCompile(`^\s*\w+\s+([A-Z]+(?:\([^)]*\))?)`).FindStringSubmatch(def); len(typeMatch) > 1 {
		props["TYPE"] = sc.normalizeType(typeMatch[1])
	}
	
	// Extract constraints
	if strings.Contains(def, "PRIMARY KEY") {
		props["PRIMARY"] = "true"
	}
	
	if strings.Contains(def, "UNIQUE") {
		props["UNIQUE"] = "true"
	}
	
	if strings.Contains(def, "NOT NULL") {
		props["NOT_NULL"] = "true"
	} else if !strings.Contains(def, "PRIMARY KEY") {
		props["NULLABLE"] = "true"
	}
	
	// Extract default value
	if defaultMatch := regexp.MustCompile(`DEFAULT\s+([^,\s]+(?:\([^)]*\))?)`).FindStringSubmatch(def); len(defaultMatch) > 1 {
		props["DEFAULT"] = sc.normalizeDefault(defaultMatch[1])
	}
	
	// Extract foreign key
	if refMatch := regexp.MustCompile(`REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)(?:\s+ON\s+DELETE\s+(\w+(?:\s+\w+)?))?`).FindStringSubmatch(def); len(refMatch) > 2 {
		fkRef := strings.ToLower(refMatch[1]) + "." + strings.ToLower(refMatch[2])
		if len(refMatch) > 3 && refMatch[3] != "" {
			fkRef += ":" + strings.ToUpper(strings.TrimSpace(refMatch[3]))
		}
		props["FOREIGN_KEY"] = fkRef
	}
	
	return props
}

// normalizeDBTables converts database tables to comparable structure
func (sc *SQLComparator) normalizeDBTables(dbTables []types.SchemaTable) map[string]*TableStructure {
	tables := make(map[string]*TableStructure)
	
	for _, dbTable := range dbTables {
		table := &TableStructure{
			Name:        strings.ToLower(dbTable.Name),
			Columns:     make(map[string]*ColumnStructure),
			ColumnOrder: make([]string, 0, len(dbTable.Columns)),
		}
		
		// Preserve the original column order from database
		for _, dbCol := range dbTable.Columns {
			colName := strings.ToLower(dbCol.Name)
			table.ColumnOrder = append(table.ColumnOrder, colName)
			
			col := &ColumnStructure{
				Name:       colName,
				Properties: make(map[string]string),
			}
			
			// Set properties
			col.Properties["TYPE"] = sc.normalizeType(dbCol.Type)
			
			if dbCol.IsPrimary {
				col.Properties["PRIMARY"] = "true"
			}
			
			if dbCol.IsUnique {
				col.Properties["UNIQUE"] = "true"
			}
			
			if !dbCol.Nullable {
				col.Properties["NOT_NULL"] = "true"
			} else if !dbCol.IsPrimary {
				col.Properties["NULLABLE"] = "true"
			}
			
			if dbCol.Default != "" {
				col.Properties["DEFAULT"] = sc.normalizeDefault(dbCol.Default)
			}
			
			if dbCol.ForeignKeyTable != "" && dbCol.ForeignKeyColumn != "" {
				fkRef := strings.ToLower(dbCol.ForeignKeyTable) + "." + strings.ToLower(dbCol.ForeignKeyColumn)
				if dbCol.OnDeleteAction != "" {
					fkRef += ":" + strings.ToUpper(strings.TrimSpace(dbCol.OnDeleteAction))
				}
				col.Properties["FOREIGN_KEY"] = fkRef
			}
			
			table.Columns[colName] = col
		}
		
		tables[table.Name] = table
	}
	
	return tables
}

// areEqual compares two table structures
func (sc *SQLComparator) areEqual(existing, db map[string]*TableStructure) bool {
	if len(existing) != len(db) {
		return false
	}
	
	for tableName, dbTable := range db {
		existingTable, exists := existing[tableName]
		if !exists {
			return false
		}
		
		if !sc.areTablesEqual(existingTable, dbTable) {
			return false
		}
	}
	
	return true
}

// areTablesEqual compares individual tables
func (sc *SQLComparator) areTablesEqual(existing, db *TableStructure) bool {
	if len(existing.Columns) != len(db.Columns) {
		return false
	}
	
	for colName, dbCol := range db.Columns {
		existingCol, exists := existing.Columns[colName]
		if !exists {
			return false
		}
		
		if !sc.areColumnsEqual(existingCol, dbCol) {
			return false
		}
	}
	
	return true
}

// areColumnsEqual compares column properties
func (sc *SQLComparator) areColumnsEqual(existing, db *ColumnStructure) bool {
	// Compare all properties
	for key, dbValue := range db.Properties {
		existingValue, exists := existing.Properties[key]
		if !exists && dbValue != "" {
			return false
		}
		if exists && existingValue != dbValue {
			return false
		}
	}
	
	// Check for extra properties in existing that shouldn't be there
	for key, existingValue := range existing.Properties {
		dbValue, exists := db.Properties[key]
		if !exists && existingValue != "" {
			return false
		}
		if exists && existingValue != dbValue {
			return false
		}
	}
	
	return true
}

// generateUpdatedSQL creates updated SQL preserving original formatting and order
func (sc *SQLComparator) generateUpdatedSQL(originalSQL string, existing, db map[string]*TableStructure) string {
	if originalSQL == "" {
		// No existing file, generate new one with database order
		return sc.generateCleanSQL(db)
	}
	
	// Preserve original SQL structure and only update what's different
	return sc.updateExistingSQL(originalSQL, existing, db)
}

// updateExistingSQL updates only the different parts while preserving original structure
func (sc *SQLComparator) updateExistingSQL(originalSQL string, existing, db map[string]*TableStructure) string {
	result := originalSQL
	
	// Find and replace each table definition
	createTableRegex := regexp.MustCompile(`(?is)(CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\((.*?)\);)`)
	
	result = createTableRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := createTableRegex.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		
		tableName := strings.ToLower(strings.TrimSpace(submatches[2]))
		
		// Check if this table exists in database
		dbTable, dbExists := db[tableName]
		existingTable, existingExists := existing[tableName]
		
		if !dbExists {
			// Table doesn't exist in database, remove it
			return ""
		}
		
		if !existingExists {
			// New table, generate fresh
			return sc.generateTableSQL(dbTable)
		}
		
		// Update existing table definition
		return sc.updateTableDefinition(match, existingTable, dbTable)
	})
	
	// Add any new tables that don't exist in original SQL
	for tableName, dbTable := range db {
		if _, exists := existing[tableName]; !exists {
			if result != "" && !strings.HasSuffix(result, "\n") {
				result += "\n"
			}
			result += "\n" + sc.generateTableSQL(dbTable)
		}
	}
	
	return result
}

// updateTableDefinition updates a table definition preserving original column order
func (sc *SQLComparator) updateTableDefinition(originalTableSQL string, existing, db *TableStructure) string {
	// Extract the table header and footer
	createTableRegex := regexp.MustCompile(`(?is)(CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?\w+\s*\()(.*?)(\);)`)
	matches := createTableRegex.FindStringSubmatch(originalTableSQL)
	
	if len(matches) < 4 {
		// Fallback to generating new table
		return sc.generateTableSQL(db)
	}
	
	header := matches[1]
	columnsSection := matches[2]
	footer := matches[3]
	
	// Update columns section preserving original order
	updatedColumns := sc.updateColumnsSection(columnsSection, existing, db)
	
	return header + updatedColumns + footer
}

// updateColumnsSection updates column definitions preserving original order and formatting
func (sc *SQLComparator) updateColumnsSection(originalColumns string, existing, db *TableStructure) string {
	// Split original columns while preserving formatting
	parts := sc.smartSplitPreservingWhitespace(originalColumns, ',')
	
	var updatedParts []string
	processedColumns := make(map[string]bool)
	
	// Process existing columns in original order
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" || strings.HasPrefix(strings.ToUpper(trimmedPart), "FOREIGN KEY") {
			updatedParts = append(updatedParts, part)
			continue
		}
		
		// Extract column name
		colName := sc.extractColumnName(trimmedPart)
		if colName == "" {
			updatedParts = append(updatedParts, part)
			continue
		}
		
		colNameLower := strings.ToLower(colName)
		processedColumns[colNameLower] = true
		
		// Check if column exists in database
		dbCol, dbExists := db.Columns[colNameLower]
		existingCol, existingExists := existing.Columns[colNameLower]
		
		if !dbExists {
			// Column doesn't exist in database, skip it
			continue
		}
		
		if !existingExists || !sc.areColumnsEqual(existingCol, dbCol) {
			// Column is new or different, update it
			// Preserve original indentation
			indentation := sc.extractIndentation(part)
			updatedParts = append(updatedParts, indentation+sc.generateColumnSQL(dbCol))
		} else {
			// Column is the same, keep original
			updatedParts = append(updatedParts, part)
		}
	}
	
	// Add any new columns that weren't in the original
	for colName, dbCol := range db.Columns {
		if !processedColumns[colName] {
			// Add new column with standard indentation
			updatedParts = append(updatedParts, "    "+sc.generateColumnSQL(dbCol))
		}
	}
	
	return strings.Join(updatedParts, ",")
}

// Helper functions for preserving formatting
func (sc *SQLComparator) smartSplitPreservingWhitespace(text string, delimiter rune) []string {
	var parts []string
	var current strings.Builder
	parenLevel := 0
	
	for _, char := range text {
		switch char {
		case '(':
			parenLevel++
			current.WriteRune(char)
		case ')':
			parenLevel--
			current.WriteRune(char)
		case delimiter:
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

func (sc *SQLComparator) extractColumnName(columnDef string) string {
	// Extract first word as column name
	fields := strings.Fields(strings.TrimSpace(columnDef))
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func (sc *SQLComparator) extractIndentation(text string) string {
	// Extract leading whitespace
	for i, char := range text {
		if char != ' ' && char != '\t' && char != '\n' && char != '\r' {
			return text[:i]
		}
	}
	return ""
}

func (sc *SQLComparator) generateTableSQL(table *TableStructure) string {
	var result strings.Builder
	
	result.WriteString("-- ")
	result.WriteString(strings.Title(table.Name))
	result.WriteString(" table\n")
	result.WriteString("CREATE TABLE IF NOT EXISTS ")
	result.WriteString(table.Name)
	result.WriteString(" (\n")
	
	// Use database column order for new tables
	columnNames := table.ColumnOrder
	if len(columnNames) == 0 {
		for name := range table.Columns {
			columnNames = append(columnNames, name)
		}
	}
	
	for j, colName := range columnNames {
		if j > 0 {
			result.WriteString(",\n")
		}
		
		col := table.Columns[colName]
		if col != nil {
			result.WriteString("    ")
			result.WriteString(sc.generateColumnSQL(col))
		}
	}
	
	result.WriteString("\n);")
	return result.String()
}

// generateCleanSQL generates clean SQL from database structure preserving order
func (sc *SQLComparator) generateCleanSQL(tables map[string]*TableStructure) string {
	var result strings.Builder
	
	// Sort table names for consistent output
	var tableNames []string
	for name := range tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)
	
	for i, tableName := range tableNames {
		if i > 0 {
			result.WriteString("\n")
		}
		
		table := tables[tableName]
		result.WriteString("-- ")
		result.WriteString(strings.Title(table.Name))
		result.WriteString(" table\n")
		result.WriteString("CREATE TABLE IF NOT EXISTS ")
		result.WriteString(table.Name)
		result.WriteString(" (\n")
		
		// Preserve original column order from database (don't sort)
		columnNames := table.ColumnOrder
		if len(columnNames) == 0 {
			// Fallback to map keys if order not preserved
			for name := range table.Columns {
				columnNames = append(columnNames, name)
			}
		}
		
		for j, colName := range columnNames {
			if j > 0 {
				result.WriteString(",\n")
			}
			
			col := table.Columns[colName]
			if col != nil {
				result.WriteString("    ")
				result.WriteString(sc.generateColumnSQL(col))
			}
		}
		
		result.WriteString("\n);\n")
	}
	
	return result.String()
}

// generateColumnSQL generates SQL for a column
func (sc *SQLComparator) generateColumnSQL(col *ColumnStructure) string {
	var parts []string
	
	parts = append(parts, col.Name)
	
	if dataType, exists := col.Properties["TYPE"]; exists {
		parts = append(parts, dataType)
	}
	
	if col.Properties["PRIMARY"] == "true" {
		parts = append(parts, "PRIMARY KEY")
	} else {
		if col.Properties["UNIQUE"] == "true" {
			parts = append(parts, "UNIQUE")
		}
		if col.Properties["NOT_NULL"] == "true" {
			parts = append(parts, "NOT NULL")
		}
	}
	
	if defaultVal, exists := col.Properties["DEFAULT"]; exists && defaultVal != "" {
		parts = append(parts, "DEFAULT", defaultVal)
	}
	
	if fkRef, exists := col.Properties["FOREIGN_KEY"]; exists && fkRef != "" {
		fkParts := strings.Split(fkRef, ":")
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

// Helper functions
func (sc *SQLComparator) normalizeType(dataType string) string {
	typeUpper := strings.ToUpper(strings.TrimSpace(dataType))
	
	switch {
	case typeUpper == "INTEGER":
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

func (sc *SQLComparator) normalizeDefault(defaultVal string) string {
	if defaultVal == "" {
		return ""
	}
	
	defaultUpper := strings.ToUpper(strings.TrimSpace(defaultVal))
	
	switch {
	case strings.Contains(defaultUpper, "NOW()") || strings.Contains(defaultUpper, "CURRENT_TIMESTAMP"):
		return "NOW()"
	case strings.Contains(defaultUpper, "NEXTVAL"):
		return ""
	default:
		return strings.Trim(defaultVal, "'\"")
	}
}

func (sc *SQLComparator) smartSplit(text string, delimiter rune) []string {
	var parts []string
	var current strings.Builder
	parenLevel := 0
	
	for _, char := range text {
		switch char {
		case '(':
			parenLevel++
			current.WriteRune(char)
		case ')':
			parenLevel--
			current.WriteRune(char)
		case delimiter:
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

// Data structures
type TableStructure struct {
	Name        string
	Columns     map[string]*ColumnStructure
	ColumnOrder []string // Preserve original column order
}

type ColumnStructure struct {
	Name       string
	Properties map[string]string // TYPE, PRIMARY, UNIQUE, NOT_NULL, DEFAULT, FOREIGN_KEY, etc.
}
