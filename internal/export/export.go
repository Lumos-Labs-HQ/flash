package export

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func PerformExport(ctx context.Context, adapter database.DatabaseAdapter, exportPath, format string) (string, error) {
	tables, err := adapter.GetAllTableNames(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	if len(tables) == 0 {
		log.Println("No tables found in database")
		return "", nil
	}

	// Fetch full schema for type-aware export
	schemaTables, err := adapter.PullCompleteSchema(ctx)
	if err != nil {
		log.Printf("Warning: could not pull schema for export: %v", err)
		schemaTables = nil
	}
	schemaEnums, err := adapter.GetCurrentEnums(ctx)
	if err != nil {
		schemaEnums = nil
	}

	indexMap := make(map[string][]types.SchemaIndex)
	for _, t := range schemaTables {
		indexMap[t.Name] = t.Indexes
	}

	exportData := types.BackupData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Version:   "1.0",
		Tables:    make(map[string]interface{}, len(tables)),
		Comment:   "Database export",
	}

	// Fetch table data in parallel.
	type tableResult struct {
		name string
		data []map[string]interface{}
		err  error
	}

	var validTables []string
	for _, tableName := range tables {
		if tableName != "_flash_migrations" {
			validTables = append(validTables, tableName)
		}
	}

	results := make(chan tableResult, len(validTables))
	var wg sync.WaitGroup

	for _, tableName := range validTables {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			data, err := adapter.GetTableData(ctx, name)
			results <- tableResult{name, data, err}
		}(tableName)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		if result.err != nil {
			log.Printf("Warning: Failed to get data for table %s: %v", result.name, result.err)
		} else {
			exportData.Tables[result.name] = result.data
		}
	}

	// Build column type map from schema for type-aware export
	colTypeMap := make(map[string]map[string]string) // table → col → type
	if schemaTables != nil {
		for _, t := range schemaTables {
			colTypeMap[t.Name] = make(map[string]string, len(t.Columns))
			for _, c := range t.Columns {
				colTypeMap[t.Name][c.Name] = c.Type
			}
		}
	}

	switch format {
	case "csv":
		return exportToCSV(exportData, exportPath)
	case "sqlite":
		return exportToSQLite(ctx, adapter, exportData, exportPath, colTypeMap)
	default:
		return exportToJSON(exportData, exportPath, schemaTables, schemaEnums, indexMap)
	}
}

func exportToJSON(data types.BackupData, exportPath string, schemaTables []types.SchemaTable, schemaEnums []types.SchemaEnum, indexMap map[string][]types.SchemaIndex) (string, error) {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filePath := filepath.Join(exportPath, fmt.Sprintf("export_%s.json", timestamp))

	// Build schema-aware export payload
	payload := map[string]interface{}{
		"version":   "2.0",
		"timestamp": data.Timestamp,
		"comment":   data.Comment,
		"data":      data.Tables,
	}

	// Include full schema DDL if available
	if schemaTables != nil || schemaEnums != nil {
		schema := buildSchemaDDL(schemaTables, schemaEnums, indexMap)
		payload["schema"] = schema
	}

	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// buildSchemaDDL generates a JSON-serializable schema representation
func buildSchemaDDL(tables []types.SchemaTable, enums []types.SchemaEnum, indexMap map[string][]types.SchemaIndex) map[string]interface{} {
	schema := map[string]interface{}{}

	if enums != nil {
		enumList := make([]map[string]interface{}, len(enums))
		for i, e := range enums {
			enumList[i] = map[string]interface{}{
				"name":   e.Name,
				"values": e.Values,
			}
		}
		schema["enums"] = enumList
	}

	if tables != nil {
		tableList := make([]map[string]interface{}, len(tables))
		for i, t := range tables {
			if strings.HasPrefix(t.Name, "_flash_") {
				continue
			}
			cols := make([]map[string]interface{}, len(t.Columns))
			for j, c := range t.Columns {
				cols[j] = map[string]interface{}{
					"name":            c.Name,
					"type":            c.Type,
					"nullable":        c.Nullable,
					"default":         c.Default,
					"is_primary":      c.IsPrimary,
					"is_unique":       c.IsUnique,
					"check":           c.Check,
					"generated":       c.Generated,
					"foreign_key_table":  c.ForeignKeyTable,
					"foreign_key_column": c.ForeignKeyColumn,
					"on_delete":       c.OnDeleteAction,
				}
			}
			idxList := indexMap[t.Name]
			tableList[i] = map[string]interface{}{
				"name":    t.Name,
				"columns": cols,
				"indexes": idxList,
			}
		}
		schema["tables"] = tableList
	}

	return schema
}

func exportToCSV(data types.BackupData, exportPath string) (string, error) {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	dirPath := filepath.Join(exportPath, fmt.Sprintf("export_%s_csv", timestamp))

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create CSV directory: %w", err)
	}

	for tableName, tableData := range data.Tables {
		rows, ok := tableData.([]map[string]interface{})
		if !ok || len(rows) == 0 {
			continue
		}

		filePath := filepath.Join(dirPath, fmt.Sprintf("%s.csv", tableName))
		file, err := os.Create(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to create CSV file for %s: %w", tableName, err)
		}

		writer := csv.NewWriter(file)

		// Sort headers for deterministic CSV output.
		headers := make([]string, 0, len(rows[0]))
		for key := range rows[0] {
			headers = append(headers, key)
		}
		sort.Strings(headers)

		_ = writer.Write(headers)

		for _, row := range rows {
			values := make([]string, len(headers))
			for i, header := range headers {
				values[i] = fmt.Sprintf("%v", row[header])
			}
			_ = writer.Write(values)
		}

		writer.Flush()
		file.Close()
	}

	return dirPath, nil
}

func exportToSQLite(ctx context.Context, adapter database.DatabaseAdapter, data types.BackupData, exportPath string, colTypeMap map[string]map[string]string) (string, error) {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filePath := filepath.Join(exportPath, fmt.Sprintf("export_%s.db", timestamp))

	db, err := sql.Open("sqlite", filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create SQLite database: %w", err)
	}
	defer db.Close()

	for tableName, tableData := range data.Tables {
		rows, ok := tableData.([]map[string]interface{})
		if !ok || len(rows) == 0 {
			continue
		}

		// Sort columns for deterministic schema
		columns := make([]string, 0, len(rows[0]))
		for key := range rows[0] {
			columns = append(columns, key)
		}
		sort.Strings(columns)

		// Use real column types from schema when available
		createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName,
			buildColumnDefsWithTypes(columns, colTypeMap[tableName]))
		if _, err := db.Exec(createSQL); err != nil {
			return "", fmt.Errorf("failed to create table %s: %w", tableName, err)
		}

		for _, row := range rows {
			insertSQL := buildInsertSQL(tableName, columns)
			values := make([]interface{}, len(columns))
			for i, col := range columns {
				values[i] = row[col]
			}
			if _, err := db.Exec(insertSQL, values...); err != nil {
				log.Printf("Warning: Failed to insert row into %s: %v", tableName, err)
			}
		}
	}

	return filePath, nil
}

// buildColumnDefsWithTypes generates column definitions using real types from schema
func buildColumnDefsWithTypes(columns []string, colTypes map[string]string) string {
	defs := make([]string, len(columns))
	for i, col := range columns {
		sqlType := "TEXT"
		if mapped, ok := colTypes[col]; ok && mapped != "" {
			sqlType = mapTypeToSQLite(mapped)
		}
		defs[i] = fmt.Sprintf("%s %s", col, sqlType)
	}
	return strings.Join(defs, ", ")
}

// mapTypeToSQLite converts PostgreSQL/MySQL types to SQLite-compatible types
func mapTypeToSQLite(dbType string) string {
	upper := strings.ToUpper(dbType)
	switch {
	case strings.Contains(upper, "SERIAL"), strings.Contains(upper, "BIGINT"),
		strings.Contains(upper, "INT"), strings.Contains(upper, "INTEGER"):
		return "INTEGER"
	case strings.Contains(upper, "FLOAT"), strings.Contains(upper, "DOUBLE"),
		strings.Contains(upper, "REAL"), strings.Contains(upper, "NUMERIC"),
		strings.Contains(upper, "DECIMAL"):
		return "REAL"
	case strings.Contains(upper, "BOOL"):
		return "INTEGER"
	case strings.Contains(upper, "TIMESTAMP"), strings.Contains(upper, "DATE"),
		strings.Contains(upper, "TIME"):
		return "TEXT"
	case upper == "JSONB", upper == "JSON":
		return "TEXT"
	case upper == "INET", upper == "UUID":
		return "TEXT"
	case strings.Contains(upper, "[]"):
		return "TEXT"
	case strings.Contains(upper, "ENUM"), strings.Contains(upper, "ENUM"):
		return "TEXT"
	default:
		return "TEXT"
	}
}

func buildColumnDefs(columns []string) string {
	defs := make([]string, len(columns))
	for i, col := range columns {
		defs[i] = fmt.Sprintf("%s TEXT", col)
	}
	return strings.Join(defs, ", ")
}

func buildInsertSQL(table string, columns []string) string {
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), strings.Repeat("?, ", len(columns)-1)+"?")
}
