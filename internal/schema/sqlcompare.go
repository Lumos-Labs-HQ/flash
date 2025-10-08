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
	existingTables := sc.parseSQL(existingSQL)
	
	dbTablesNorm := sc.normalizeDBTables(dbTables)
	
	if sc.areEqual(existingTables, dbTablesNorm) {
		return false, "" 
	}
	
	updatedSQL := sc.generateUpdatedSQL(existingSQL, existingTables, dbTablesNorm)
	return true, updatedSQL
}

// parseSQL extracts table structures from SQL, completely ignoring commented sections
func (sc *SQLComparator) parseSQL(sql string) map[string]*TableStructure {
	tables := make(map[string]*TableStructure)
	
	cleanSQL := sc.removeAllComments(sql)
	
	createTableRegex := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\((.*?)\);`)
	matches := createTableRegex.FindAllStringSubmatch(cleanSQL, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := strings.ToLower(strings.TrimSpace(match[1]))
			columnsDef := match[2]
			
			columns, columnOrder := sc.parseColumnsWithOrder(columnsDef)
			
			table := &TableStructure{
				Name:        tableName,
				Columns:     columns,
				ColumnOrder: columnOrder,
			}
			
			tables[tableName] = table
		}
	}
	
	return tables
}

// removeAllComments removes all commented lines from SQL
func (sc *SQLComparator) removeAllComments(sql string) string {
	lines := strings.Split(sql, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		if strings.HasPrefix(trimmedLine, "--") {
			continue
		}
		
		cleanLines = append(cleanLines, line)
	}
	
	return strings.Join(cleanLines, "\n")
}

// parseColumnsWithOrder extracts column definitions preserving order
func (sc *SQLComparator) parseColumnsWithOrder(columnsDef string) (map[string]*ColumnStructure, []string) {
	columns := make(map[string]*ColumnStructure)
	var columnOrder []string
	
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

// parseColumn parses individual column definition
func (sc *SQLComparator) parseColumn(def string) *ColumnStructure {
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return nil
	}
	
	col := &ColumnStructure{
		Name: strings.ToLower(strings.TrimSpace(parts[0])),
	}
	
	defUpper := strings.ToUpper(def)
	
	col.Properties = sc.extractProperties(defUpper)
	
	return col
}

// extractProperties extracts all column properties in normalized form
func (sc *SQLComparator) extractProperties(def string) map[string]string {
	props := make(map[string]string)
	
	if typeMatch := regexp.MustCompile(`^\s*\w+\s+([A-Z]+(?:\([^)]*\))?)`).FindStringSubmatch(def); len(typeMatch) > 1 {
		props["TYPE"] = sc.normalizeType(typeMatch[1])
	}
	
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
	
	if defaultMatch := regexp.MustCompile(`DEFAULT\s+([^,\s]+(?:\([^)]*\))?)`).FindStringSubmatch(def); len(defaultMatch) > 1 {
		props["DEFAULT"] = sc.normalizeDefault(defaultMatch[1])
	}
	
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
		
		for _, dbCol := range dbTable.Columns {
			colName := strings.ToLower(dbCol.Name)
			table.ColumnOrder = append(table.ColumnOrder, colName)
			
			col := &ColumnStructure{
				Name:       colName,
				Properties: make(map[string]string),
			}
			
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
	for key, dbValue := range db.Properties {
		existingValue, exists := existing.Properties[key]
		if !exists && dbValue != "" {
			return false
		}
		if exists && existingValue != dbValue {
			return false
		}
	}
	
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
		return sc.generateCleanSQL(db)
	}
	
	return sc.updateExistingSQL(originalSQL, existing, db)
}

// only non-commented CREATE TABLE statements
func (sc *SQLComparator) updateExistingSQL(originalSQL string, existing, db map[string]*TableStructure) string {
	result := originalSQL
	
	createTableRegex := regexp.MustCompile(`(?m)^(\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(((?:[^)]|\([^)]*\))*)\);)`)
	
	result = createTableRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := createTableRegex.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		
		tableName := strings.ToLower(strings.TrimSpace(submatches[2]))
		
		dbTable, dbExists := db[tableName]
		existingTable, existingExists := existing[tableName]
		
		if !dbExists {
			return ""
		}
		
		if !existingExists || !sc.areTablesEqual(existingTable, dbTable) {
			return sc.generateTableSQL(dbTable)
		}
		
		return match
	})
	
	for tableName, dbTable := range db {
		if _, exists := existing[tableName]; !exists {
			if result != "" && !strings.HasSuffix(result, "\n") {
				result += "\n"
			}
			result += "\n" + sc.generateTableSQL(dbTable) + "\n"
		}
	}
	
	return result
}

// generateCleanSQL generates clean SQL from database structure preserving order
func (sc *SQLComparator) generateCleanSQL(tables map[string]*TableStructure) string {
	var result strings.Builder
	
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
		result.WriteString(sc.generateTableSQL(table))
		result.WriteString("\n")
	}
	
	return result.String()
}

func (sc *SQLComparator) generateTableSQL(table *TableStructure) string {
	var result strings.Builder
	
	// result.WriteString("-- ")
	// result.WriteString(strings.Title(table.Name))
	// result.WriteString(" table\n")
	result.WriteString("CREATE TABLE IF NOT EXISTS ")
	result.WriteString(table.Name)
	result.WriteString(" (\n")
	
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

type TableStructure struct {
	Name        string
	Columns     map[string]*ColumnStructure
	ColumnOrder []string 
}

type ColumnStructure struct {
	Name       string
	Properties map[string]string 
}
