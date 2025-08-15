package migrator

import (
	"strings"

	"github.com/Rana718/Graft/internal/types"
)

// Checks if table changes contain meaningful modifications
// func hasRealTableChanges(modifiedTables []types.TableDiff) bool {
// 	for _, tableDiff := range modifiedTables {
// 		if len(tableDiff.NewColumns) > 0 || len(tableDiff.DroppedColumns) > 0 {
// 			return true
// 		}
// 		for _, modCol := range tableDiff.ModifiedColumns {
// 			if len(modCol.Changes) > 0 {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }

// Checks if a column is a primary key
func isPrimaryKeyColumn(col types.SchemaColumn) bool {
	return strings.Contains(strings.ToUpper(col.Type), "PRIMARY KEY") ||
		strings.Contains(strings.ToUpper(col.Type), "SERIAL")
}

// Checks if two types are equivalent
func isEquivalentType(currentType, targetType string) bool {
	current := normalizeTypeForComparison(currentType)
	target := normalizeTypeForComparison(targetType)
	return current == target
}

// Checks if two defaults are equivalent
func isEquivalentDefault(currentDefault, targetDefault string) bool {
	current := strings.TrimSpace(currentDefault)
	target := strings.TrimSpace(targetDefault)

	if current == "" && target == "" {
		return true
	}

	currentUpper := strings.ToUpper(current)
	targetUpper := strings.ToUpper(target)

	nowVariations := []string{"NOW()", "CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP()"}

	for _, variation := range nowVariations {
		if (currentUpper == variation || currentUpper == "'"+variation+"'") &&
			(targetUpper == variation || targetUpper == "'"+variation+"'") {
			return true
		}
	}

	if currentUpper == targetUpper {
		return true
	}

	if (current == "NULL" && target == "") || (current == "" && target == "NULL") {
		return true
	}

	return false
}

// Normalizes types for comparison
func normalizeTypeForComparison(dbType string) string {
	dbType = strings.ToUpper(strings.TrimSpace(dbType))

	typeMap := map[string]string{
		"INT":               "INTEGER",
		"INT4":              "INTEGER",
		"BIGINT":            "BIGINT",
		"INT8":              "BIGINT",
		"SMALLINT":          "SMALLINT",
		"INT2":              "SMALLINT",
		"SERIAL":            "INTEGER",
		"BIGSERIAL":         "BIGINT",
		"SMALLSERIAL":       "SMALLINT",
		"VARCHAR":           "VARCHAR",
		"CHARACTER VARYING": "VARCHAR",
		"CHAR":              "CHAR",
		"CHARACTER":         "CHAR",
		"TEXT":              "TEXT",
		"BOOLEAN":           "BOOLEAN",
		"BOOL":              "BOOLEAN",
		"REAL":              "REAL",
		"FLOAT4":            "REAL",
		"DOUBLE PRECISION":  "DOUBLE PRECISION",
		"FLOAT8":            "DOUBLE PRECISION",
		"NUMERIC":           "NUMERIC",
		"DECIMAL":           "NUMERIC",
		"DATE":              "DATE",
		"TIME":              "TIME",
		"TIMESTAMP":         "TIMESTAMP",
		"TIMESTAMPTZ":       "TIMESTAMPTZ",
		"JSON":              "JSON",
		"JSONB":             "JSONB",
		"UUID":              "UUID",
		"BYTEA":             "BYTEA",
		"INET":              "INET",
		"CIDR":              "CIDR",
		"MACADDR":           "MACADDR",
	}

	// Handle parameterized types
	if strings.Contains(dbType, "(") {
		baseType := strings.Split(dbType, "(")[0]
		if normalized, exists := typeMap[baseType]; exists {
			params := strings.Split(dbType, "(")[1]
			return normalized + "(" + params
		}
	}

	if normalized, exists := typeMap[dbType]; exists {
		return normalized
	}

	return dbType
}

// Extracts table name from various SQL statements
func extractTableName(sql string) string {
	sql = strings.TrimSpace(strings.ToUpper(sql))

	if strings.HasPrefix(sql, "CREATE TABLE") {
		parts := strings.Fields(sql)
		if len(parts) >= 3 {
			tableName := parts[2]
			tableName = strings.Trim(tableName, "\"'`")
			tableName = strings.Split(tableName, "(")[0]
			return strings.TrimSpace(tableName)
		}
	}

	if strings.HasPrefix(sql, "ALTER TABLE") {
		parts := strings.Fields(sql)
		if len(parts) >= 3 {
			tableName := parts[2]
			tableName = strings.Trim(tableName, "\"'`")
			return strings.TrimSpace(tableName)
		}
	}

	if strings.HasPrefix(sql, "DROP TABLE") {
		parts := strings.Fields(sql)
		if len(parts) >= 3 {
			tableName := parts[2]
			tableName = strings.Trim(tableName, "\"'`")
			return strings.TrimSpace(tableName)
		}
	}

	return ""
}

// Checks if a SQL statement is a DDL operation (Data Definition Language)
func isDDLStatement(sql string) bool {
	sql = strings.TrimSpace(strings.ToUpper(sql))
	ddlKeywords := []string{
		"CREATE TABLE", "ALTER TABLE", "DROP TABLE",
		"CREATE INDEX", "DROP INDEX",
		"CREATE SEQUENCE", "DROP SEQUENCE",
		"CREATE TYPE", "DROP TYPE",
		"CREATE FUNCTION", "DROP FUNCTION",
		"CREATE TRIGGER", "DROP TRIGGER",
	}

	for _, keyword := range ddlKeywords {
		if strings.HasPrefix(sql, keyword) {
			return true
		}
	}

	return false
}

// Checks if a column definition contains a NOT NULL constraint
func hasNotNullConstraint(columnDef string) bool {
	columnDef = strings.ToUpper(columnDef)
	return strings.Contains(columnDef, "NOT NULL")
}

// Checks if a column definition contains a DEFAULT value
func hasDefaultValue(columnDef string) bool {
	columnDef = strings.ToUpper(columnDef)
	return strings.Contains(columnDef, "DEFAULT")
}

// Extracts the column name from an ALTER TABLE ADD COLUMN statement
func extractColumnName(alterStatement string) string {
	alterStatement = strings.TrimSpace(strings.ToUpper(alterStatement))

	if strings.Contains(alterStatement, "ADD COLUMN") {
		parts := strings.Split(alterStatement, "ADD COLUMN")
		if len(parts) >= 2 {
			columnPart := strings.TrimSpace(parts[1])
			columnName := strings.Fields(columnPart)[0]
			columnName = strings.Trim(columnName, "\"'`")
			return strings.TrimSpace(columnName)
		}
	}

	if strings.Contains(alterStatement, "ADD ") && !strings.Contains(alterStatement, "ADD COLUMN") {
		parts := strings.Split(alterStatement, "ADD ")
		if len(parts) >= 2 {
			columnPart := strings.TrimSpace(parts[1])
			if !strings.HasPrefix(columnPart, "CONSTRAINT") {
				columnName := strings.Fields(columnPart)[0]
				columnName = strings.Trim(columnName, "\"'`")
				return strings.TrimSpace(columnName)
			}
		}
	}

	return ""
}

// Safely quotes table/column names for SQL
func quoteIdentifier(identifier string) string {
	// Remove existing quotes
	identifier = strings.Trim(identifier, "\"'`")

	// Add double quotes for PostgreSQL compatibility
	return "\"" + identifier + "\""
}

// Checks if a string is a SQL keyword that needs quoting
func isSQLKeyword(word string) bool {
	word = strings.ToUpper(word)
	keywords := []string{
		"SELECT", "FROM", "WHERE", "ORDER", "GROUP", "HAVING",
		"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"TABLE", "INDEX", "SEQUENCE", "VIEW", "FUNCTION", "TRIGGER",
		"USER", "ROLE", "GRANT", "REVOKE", "PRIMARY", "FOREIGN",
		"KEY", "CONSTRAINT", "UNIQUE", "NOT", "NULL", "DEFAULT",
		"CHECK", "REFERENCES", "ON", "CASCADE", "RESTRICT",
		"LIMIT", "OFFSET", "DISTINCT", "AS", "AND", "OR", "IN",
		"EXISTS", "BETWEEN", "LIKE", "IS", "CASE", "WHEN", "THEN",
		"ELSE", "END", "UNION", "INTERSECT", "EXCEPT", "JOIN",
		"INNER", "LEFT", "RIGHT", "FULL", "OUTER", "CROSS",
	}

	for _, keyword := range keywords {
		if word == keyword {
			return true
		}
	}

	return false
}
