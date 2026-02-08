package sql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/branch"
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
)

type Service struct {
	adapter database.DatabaseAdapter
	cfg     *config.Config
	ctx     context.Context
}

func NewService(adapter database.DatabaseAdapter, cfg *config.Config) *Service {
	return &Service{adapter: adapter, cfg: cfg, ctx: context.Background()}
}

func (s *Service) ensureCorrectSchema() error {
	if s.cfg == nil {
		return nil
	}

	// Skip branch management if using direct DB URL (--db flag)
	if s.cfg.Database.URLEnv == "STUDIO_DB_URL" {
		return nil
	}

	// Skip if migrations path is not set or is default empty
	if s.cfg.MigrationsPath == "" || s.cfg.MigrationsPath == "db/migrations" {
		return nil
	}

	branchMgr := branch.NewMetadataManager(s.cfg.MigrationsPath)
	store, err := branchMgr.Load()
	if err != nil {
		return nil
	}

	currentBranch := store.GetBranch(store.Current)
	if currentBranch == nil {
		return nil
	}

	switch s.cfg.Database.Provider {
	case "postgresql", "postgres":
		query := fmt.Sprintf("SET search_path TO %s, public", currentBranch.Schema)
		_, err = s.adapter.ExecuteQuery(s.ctx, query)
		return err
	case "mysql", "sqlite", "sqlite3":
		type DatabaseSwitcher interface {
			SwitchDatabase(ctx context.Context, dbName string) error
		}
		if switcher, ok := s.adapter.(DatabaseSwitcher); ok {
			return switcher.SwitchDatabase(s.ctx, currentBranch.Schema)
		}
	}
	return nil
}

func (s *Service) GetTables() ([]common.TableInfo, error) {
	s.ensureCorrectSchema()
	tables, err := s.adapter.GetAllTableNames(s.ctx)
	if err != nil {
		return nil, err
	}

	result := make([]common.TableInfo, 0, len(tables))
	targetTables := make([]string, 0, len(tables))

	for _, table := range tables {
		if table != "_flash_migrations" {
			targetTables = append(targetTables, table)
		}
	}

	tableCounts, err := s.adapter.GetAllTableRowCounts(s.ctx, targetTables)
	if err != nil {
		tableCounts = make(map[string]int)
		for _, table := range targetTables {
			count, _ := s.adapter.GetTableRowCount(s.ctx, table)
			tableCounts[table] = count
		}
	}

	for _, table := range targetTables {
		result = append(result, common.TableInfo{Name: table, RowCount: tableCounts[table]})
	}

	return result, nil
}

func (s *Service) GetTableData(tableName string, page, limit int) (*common.TableData, error) {
	return s.GetTableDataFiltered(tableName, page, limit, nil)
}

func (s *Service) GetTableDataFiltered(tableName string, page, limit int, filters []common.Filter) (*common.TableData, error) {
	s.ensureCorrectSchema()
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		return nil, err
	}

	// Deduplicate columns (some adapters may return duplicates)
	seen := make(map[string]bool)
	columns := make([]common.ColumnInfo, 0, len(schema))
	columnTypes := make(map[string]string)
	for _, col := range schema {
		if seen[col.Name] {
			continue // Skip duplicate column
		}
		seen[col.Name] = true
		columns = append(columns, common.ColumnInfo{
			Name:             col.Name,
			Type:             col.Type,
			Nullable:         col.Nullable,
			PrimaryKey:       col.IsPrimary,
			Default:          col.Default,
			AutoIncrement:    col.IsAutoIncrement,
			ForeignKeyTable:  col.ForeignKeyTable,
			ForeignKeyColumn: col.ForeignKeyColumn,
		})
		columnTypes[col.Name] = col.Type
	}

	offset := (page - 1) * limit

	// Build WHERE clause from filters
	whereClause := s.buildWhereClause(filters, columnTypes)

	rows, err := s.getRowsFiltered(tableName, limit, offset, whereClause)
	if err != nil {
		return nil, err
	}

	total, _ := s.getFilteredRowCount(tableName, whereClause)

	return &common.TableData{
		Columns: columns,
		Rows:    rows,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

func (s *Service) SaveChanges(tableName string, changes []common.RowChange) error {
	s.ensureCorrectSchema()
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		return err
	}

	pkColumn := "id"
	for _, col := range schema {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	for _, change := range changes {
		if change.Action == "update" {
			query := fmt.Sprintf("UPDATE %s SET %s = '%s' WHERE %s = '%s'",
				common.QuoteIdentifier(tableName), common.QuoteIdentifier(change.Column),
				change.Value, common.QuoteIdentifier(pkColumn), change.RowID)

			if err := s.adapter.ExecuteMigration(s.ctx, query); err != nil {
				return fmt.Errorf("failed to update %s.%s: %w", tableName, change.Column, err)
			}
		}
	}
	return nil
}

func (s *Service) DeleteRows(tableName string, rowIDs []string) error {
	s.ensureCorrectSchema()
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		return err
	}

	pkColumn := "id"
	for _, col := range schema {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	for _, rowID := range rowIDs {
		query := fmt.Sprintf("DELETE FROM %s WHERE %s = '%s'",
			common.QuoteIdentifier(tableName), common.QuoteIdentifier(pkColumn), rowID)
		if err := s.adapter.ExecuteMigration(s.ctx, query); err != nil {
			return fmt.Errorf("failed to delete row %s: %w", rowID, err)
		}
	}
	return nil
}

func (s *Service) AddRow(tableName string, data map[string]any) error {
	s.ensureCorrectSchema()
	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}

	columns := []string{}
	values := []string{}

	for col, val := range data {
		columns = append(columns, common.QuoteIdentifier(col))
		if val == nil {
			values = append(values, "NULL")
		} else {
			strVal := fmt.Sprintf("%v", val)
			escapedVal := strings.ReplaceAll(strVal, "'", "''")
			values = append(values, fmt.Sprintf("'%s'", escapedVal))
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		common.QuoteIdentifier(tableName),
		strings.Join(columns, ", "),
		strings.Join(values, ", "))

	return s.adapter.ExecuteMigration(s.ctx, query)
}

func (s *Service) DeleteRow(tableName, rowID string) error {
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		escaped := strings.ReplaceAll(rowID, "'", "''")
		query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", common.QuoteIdentifier(tableName), escaped)
		return s.adapter.ExecuteMigration(s.ctx, query)
	}

	pkColumn := "id"
	for _, col := range schema {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	escaped := strings.ReplaceAll(rowID, "'", "''")
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = '%s'",
		common.QuoteIdentifier(tableName), common.QuoteIdentifier(pkColumn), escaped)
	return s.adapter.ExecuteMigration(s.ctx, query)
}


func (s *Service) getFilteredRowCount(tableName, whereClause string) (int, error) {
	if whereClause == "" {
		return s.adapter.GetTableRowCount(s.ctx, tableName)
	}

	query := fmt.Sprintf("SELECT COUNT(*) as count FROM %s WHERE %s",
		common.QuoteIdentifier(tableName), whereClause)

	result, err := s.adapter.ExecuteQuery(s.ctx, query)
	if err != nil {
		return 0, err
	}

	if len(result.Rows) > 0 {
		if count, ok := result.Rows[0]["count"]; ok {
			switch v := count.(type) {
			case int64:
				return int(v), nil
			case int:
				return v, nil
			case float64:
				return int(v), nil
			}
		}
	}

	return 0, nil
}

func (s *Service) buildWhereClause(filters []common.Filter, columnTypes map[string]string) string {
	if len(filters) == 0 {
		return ""
	}

	var conditions []string
	var currentGroup []string

	for i, filter := range filters {
		if filter.Column == "" {
			continue
		}

		condition := s.buildFilterCondition(filter, columnTypes)
		if condition == "" {
			continue
		}

		if i == 0 || filter.Logic == "where" {
			currentGroup = append(currentGroup, condition)
		} else if filter.Logic == "and" {
			currentGroup = append(currentGroup, condition)
		} else if filter.Logic == "or" {
			if len(currentGroup) > 0 {
				conditions = append(conditions, "("+strings.Join(currentGroup, " AND ")+")")
				currentGroup = []string{condition}
			} else {
				currentGroup = append(currentGroup, condition)
			}
		}
	}

	if len(currentGroup) > 0 {
		conditions = append(conditions, "("+strings.Join(currentGroup, " AND ")+")")
	}

	if len(conditions) == 0 {
		return ""
	}

	return strings.Join(conditions, " OR ")
}

func (s *Service) buildFilterCondition(filter common.Filter, columnTypes map[string]string) string {
	col := common.QuoteIdentifier(filter.Column)
	value := strings.ReplaceAll(filter.Value, "'", "''")

	colType := strings.ToLower(columnTypes[filter.Column])
	isNumeric := strings.Contains(colType, "int") || strings.Contains(colType, "serial") ||
		strings.Contains(colType, "decimal") || strings.Contains(colType, "numeric") ||
		strings.Contains(colType, "float") || strings.Contains(colType, "double") ||
		strings.Contains(colType, "real") || strings.Contains(colType, "money")

	switch filter.Operator {
	case "equals":
		if isNumeric {
			return fmt.Sprintf("%s = %s", col, value)
		}
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) = LOWER('%s')", col, value)
	case "not_equals":
		if isNumeric {
			return fmt.Sprintf("%s != %s", col, value)
		}
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) != LOWER('%s')", col, value)
	case "contains":
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) LIKE LOWER('%%%s%%')", col, value)
	case "not_contains":
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) NOT LIKE LOWER('%%%s%%')", col, value)
	case "starts_with":
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) LIKE LOWER('%s%%')", col, value)
	case "ends_with":
		return fmt.Sprintf("LOWER(CAST(%s AS TEXT)) LIKE LOWER('%%%s')", col, value)
	case "gt":
		if isNumeric {
			return fmt.Sprintf("%s > %s", col, value)
		}
		return fmt.Sprintf("%s > '%s'", col, value)
	case "lt":
		if isNumeric {
			return fmt.Sprintf("%s < %s", col, value)
		}
		return fmt.Sprintf("%s < '%s'", col, value)
	case "gte":
		if isNumeric {
			return fmt.Sprintf("%s >= %s", col, value)
		}
		return fmt.Sprintf("%s >= '%s'", col, value)
	case "lte":
		if isNumeric {
			return fmt.Sprintf("%s <= %s", col, value)
		}
		return fmt.Sprintf("%s <= '%s'", col, value)
	case "is_null":
		return fmt.Sprintf("%s IS NULL", col)
	case "is_not_null":
		return fmt.Sprintf("%s IS NOT NULL", col)
	case "is_empty":
		return fmt.Sprintf("(%s IS NULL OR CAST(%s AS TEXT) = '')", col, col)
	case "is_not_empty":
		return fmt.Sprintf("(%s IS NOT NULL AND CAST(%s AS TEXT) != '')", col, col)
	default:
		return ""
	}
}


func (s *Service) getRowsFiltered(tableName string, limit, offset int, whereClause string) ([]map[string]any, error) {
	var query string
	if whereClause != "" {
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT %d OFFSET %d",
			common.QuoteIdentifier(tableName), whereClause, limit, offset)
	} else {
		// Try to use paginated query first (only when no filter)
		type PaginatedFetcher interface {
			GetTableDataPaginated(ctx context.Context, tableName string, limit, offset int) ([]map[string]any, error)
		}

		if fetcher, ok := s.adapter.(PaginatedFetcher); ok {
			return fetcher.GetTableDataPaginated(s.ctx, tableName, limit, offset)
		}

		query = fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",
			common.QuoteIdentifier(tableName), limit, offset)
	}

	result, err := s.adapter.ExecuteQuery(s.ctx, query)
	if err != nil {
		data, err := s.adapter.GetTableData(s.ctx, tableName)
		if err != nil {
			return nil, err
		}

		start := offset
		end := offset + limit
		if start > len(data) {
			return []map[string]any{}, nil
		}
		if end > len(data) {
			end = len(data)
		}

		return data[start:end], nil
	}

	return result.Rows, nil
}

func (s *Service) GetSchemaVisualization() (map[string]any, error) {
	s.ensureCorrectSchema()

	// Use a channel to load tables concurrently with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	tables, err := s.adapter.GetCurrentSchema(ctx)
	if err != nil {
		return nil, err
	}

	enums, _ := s.adapter.GetCurrentEnums(ctx)

	nodes := make([]map[string]any, 0, len(tables))
	nodeIndex := make(map[string]string, len(tables))

	batchSize := 10
	for i := 0; i < len(tables); i += batchSize {
		end := i + batchSize
		if end > len(tables) {
			end = len(tables)
		}

		// Process batch
		for j := i; j < end; j++ {
			table := tables[j]
			nodeID := fmt.Sprintf("table-%d", j)
			nodeIndex[table.Name] = nodeID

			columns := make([]map[string]any, 0, len(table.Columns))
			columnMap := make(map[string]bool, len(table.Columns))

			for _, col := range table.Columns {
				if !columnMap[col.Name] {
					columnMap[col.Name] = true
					columns = append(columns, map[string]any{
						"name":             col.Name,
						"type":             col.Type,
						"isPrimary":        col.IsPrimary,
						"isForeign":        col.ForeignKeyTable != "",
						"nullable":         col.Nullable,
						"default":          col.Default,
						"foreignKeyTable":  col.ForeignKeyTable,
						"foreignKeyColumn": col.ForeignKeyColumn,
						"isUnique":         col.IsUnique,
						"isAutoIncrement":  col.IsAutoIncrement,
					})
				}
			}

			nodes = append(nodes, map[string]any{
				"id": nodeID,
				"data": map[string]any{
					"label":   table.Name,
					"columns": columns,
				},
				"position": map[string]int{
					"x": 100 + (j%4)*300,
					"y": 100 + (j/4)*250,
				},
			})
		}
	}

	edges := make([]map[string]any, 0)
	edgeMap := make(map[string]bool)

	for _, table := range tables {
		sourceID := nodeIndex[table.Name]
		for _, col := range table.Columns {
			if col.ForeignKeyTable != "" {
				if targetID, ok := nodeIndex[col.ForeignKeyTable]; ok {
					edgeID := fmt.Sprintf("%s-%s-%s", sourceID, targetID, col.Name)

					if !edgeMap[edgeID] {
						edgeMap[edgeID] = true

						// Use the actual FK target column if available, otherwise find PK
						targetColumn := col.ForeignKeyColumn
						if targetColumn == "" {
							for _, targetTable := range tables {
								if targetTable.Name == col.ForeignKeyTable {
									for _, targetCol := range targetTable.Columns {
										if targetCol.IsPrimary {
											targetColumn = targetCol.Name
											break
										}
									}
									break
								}
							}
						}

						edges = append(edges, map[string]any{
							"id":           edgeID,
							"source":       sourceID,
							"target":       targetID,
							"label":        col.Name,
							"sourceHandle": col.Name,
							"targetHandle": targetColumn,
						})
					}
				}
			}
		}
	}

	return map[string]any{"nodes": nodes, "edges": edges, "enums": enums}, nil
}

func (s *Service) ExecuteSQL(query string) (*common.TableData, error) {
	s.ensureCorrectSchema()
	query = strings.TrimSpace(query)

	queryUpper := strings.ToUpper(query)

	// Detect query type more comprehensively
	isSelectQuery := strings.HasPrefix(queryUpper, "SELECT") ||
		strings.HasPrefix(queryUpper, "SHOW") ||
		strings.HasPrefix(queryUpper, "DESCRIBE") ||
		strings.HasPrefix(queryUpper, "EXPLAIN") ||
		strings.HasPrefix(queryUpper, "WITH") ||
		strings.HasPrefix(queryUpper, "TABLE") ||
		strings.HasPrefix(queryUpper, "VALUES")

	// Handle SET statements - they may or may not return data depending on database
	isSetStatement := strings.HasPrefix(queryUpper, "SET")

	if isSelectQuery {
		result, err := s.adapter.ExecuteQuery(s.ctx, query)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}

		columns := make([]common.ColumnInfo, len(result.Columns))
		for i, col := range result.Columns {
			columns[i] = common.ColumnInfo{Name: col, Type: "TEXT"}
		}

		return &common.TableData{
			Columns: columns,
			Rows:    result.Rows,
			Total:   len(result.Rows),
			Page:    1,
			Limit:   len(result.Rows),
		}, nil
	}

	if isSetStatement {
		result, err := s.adapter.ExecuteQuery(s.ctx, query)
		if err == nil && result != nil {
			columns := make([]common.ColumnInfo, len(result.Columns))
			for i, col := range result.Columns {
				columns[i] = common.ColumnInfo{Name: col, Type: "TEXT"}
			}
			return &common.TableData{
				Columns: columns,
				Rows:    result.Rows,
				Total:   len(result.Rows),
				Page:    1,
				Limit:   len(result.Rows),
			}, nil
		}
	}

	if err := s.adapter.ExecuteMigration(s.ctx, query); err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return &common.TableData{
		Columns: []common.ColumnInfo{},
		Rows:    []map[string]any{},
		Total:   0,
		Page:    1,
		Limit:   0,
	}, nil
}

func (s *Service) UpdateRow(table string, id interface{}, data map[string]interface{}) error {
	s.ensureCorrectSchema()

	schema, err := s.adapter.GetTableColumns(s.ctx, table)
	if err != nil {
		return err
	}

	pkColumn := "id"
	for _, col := range schema {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	var setClauses []string
	for col, val := range data {
		if val == nil {
			setClauses = append(setClauses, fmt.Sprintf("%s = NULL", common.QuoteIdentifier(col)))
		} else {
			strVal := fmt.Sprintf("%v", val)
			escapedVal := strings.ReplaceAll(strVal, "'", "''")
			setClauses = append(setClauses, fmt.Sprintf("%s = '%s'", common.QuoteIdentifier(col), escapedVal))
		}
	}

	idStr := fmt.Sprintf("%v", id)
	escapedId := strings.ReplaceAll(idStr, "'", "''")

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%s'",
		common.QuoteIdentifier(table), strings.Join(setClauses, ", "),
		common.QuoteIdentifier(pkColumn), escapedId)

	return s.adapter.ExecuteMigration(s.ctx, query)
}

func (s *Service) InsertRow(table string, data map[string]interface{}) error {
	s.ensureCorrectSchema()

	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}

	var columns []string
	var values []string
	for col, val := range data {
		columns = append(columns, common.QuoteIdentifier(col))
		if val == nil {
			values = append(values, "NULL")
		} else {
			strVal := fmt.Sprintf("%v", val)
			escapedVal := strings.ReplaceAll(strVal, "'", "''")
			values = append(values, fmt.Sprintf("'%s'", escapedVal))
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		common.QuoteIdentifier(table), strings.Join(columns, ", "), strings.Join(values, ", "))

	return s.adapter.ExecuteMigration(s.ctx, query)
}

func (s *Service) GetBranches() ([]map[string]interface{}, string, error) {
	if s.cfg == nil {
		return nil, "", fmt.Errorf("no config loaded")
	}

	manager, err := branch.NewManager(s.cfg)
	if err != nil {
		return nil, "", err
	}
	defer manager.Close()

	branches, current, err := manager.ListBranches()
	if err != nil {
		return nil, "", err
	}

	result := make([]map[string]interface{}, len(branches))
	for i, b := range branches {
		result[i] = map[string]interface{}{
			"name":       b.Name,
			"parent":     b.Parent,
			"schema":     b.Schema,
			"created_at": b.CreatedAt,
			"is_default": b.IsDefault,
		}
	}

	return result, current, nil
}

func (s *Service) SwitchBranch(branchName string) error {
	if s.cfg == nil {
		return fmt.Errorf("no config loaded")
	}

	manager, err := branch.NewManager(s.cfg)
	if err != nil {
		return err
	}
	defer manager.Close()

	ctx := context.Background()
	if err := manager.SwitchBranch(ctx, branchName); err != nil {
		return err
	}

	branchSchema, err := manager.GetBranchSchema(branchName)
	if err != nil {
		return err
	}

	switch s.cfg.Database.Provider {
	case "postgresql", "postgres":
		query := fmt.Sprintf("SET search_path TO %s, public", branchSchema)
		if _, err := s.adapter.ExecuteQuery(ctx, query); err != nil {
			return fmt.Errorf("failed to set search_path: %w", err)
		}
	case "mysql", "sqlite", "sqlite3":
		type DatabaseSwitcher interface {
			SwitchDatabase(ctx context.Context, dbName string) error
		}
		if switcher, ok := s.adapter.(DatabaseSwitcher); ok {
			if err := switcher.SwitchDatabase(ctx, branchSchema); err != nil {
				return fmt.Errorf("failed to switch database: %w", err)
			}
		}
	}

	return nil
}

// GetEditorHints returns schema information optimized for editor autocomplete
// This data should be cached on the client side to avoid repeated database calls
func (s *Service) GetEditorHints() (map[string]any, error) {
	s.ensureCorrectSchema()

	tables, err := s.adapter.GetAllTableNames(s.ctx)
	if err != nil {
		return nil, err
	}

	// Build schema map: table -> columns
	schema := make(map[string][]map[string]string)

	for _, tableName := range tables {
		if tableName == "_flash_migrations" {
			continue
		}

		columns, err := s.adapter.GetTableColumns(s.ctx, tableName)
		if err != nil {
			// Skip tables we can't read columns from
			schema[tableName] = []map[string]string{}
			continue
		}

		cols := make([]map[string]string, 0, len(columns))
		seen := make(map[string]bool)
		for _, col := range columns {
			if seen[col.Name] {
				continue
			}
			seen[col.Name] = true
			cols = append(cols, map[string]string{
				"name": col.Name,
				"type": col.Type,
			})
		}
		schema[tableName] = cols
	}

	// Get database provider
	provider := "sql"
	if s.cfg != nil {
		provider = s.cfg.Database.Provider
	}

	return map[string]any{
		"provider": provider,
		"schema":   schema,
	}, nil
}

// sortTablesByDependency sorts tables in topological order based on foreign key dependencies
func (s *Service) sortTablesByDependency(ctx context.Context, tables []string) ([]string, error) {
	dependencies := make(map[string][]string)
	for _, t := range tables {
		dependencies[t] = []string{}
	}

	// Get foreign key info for each table
	for _, tableName := range tables {
		columns, err := s.adapter.GetTableColumns(ctx, tableName)
		if err != nil {
			continue
		}
		for _, col := range columns {
			if col.ForeignKeyTable != "" {
				dependencies[tableName] = append(dependencies[tableName], col.ForeignKeyTable)
			}
		}
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	for _, t := range tables {
		inDegree[t] = 0
	}

	// Count incoming edges (how many tables reference this table)
	for _, deps := range dependencies {
		for _, dep := range deps {
			if _, exists := inDegree[dep]; exists {
				inDegree[dep]++ // This is reversed - we want tables with no dependencies first
			}
		}
	}

	// Reset and calculate properly
	for _, t := range tables {
		inDegree[t] = len(dependencies[t])
	}

	// Queue tables with no dependencies
	var queue []string
	for _, t := range tables {
		if inDegree[t] == 0 {
			queue = append(queue, t)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// For each table that depends on current, reduce its in-degree
		for t, deps := range dependencies {
			for _, dep := range deps {
				if dep == current {
					inDegree[t]--
					if inDegree[t] == 0 {
						queue = append(queue, t)
					}
				}
			}
		}
	}

	// If we couldn't sort all tables (circular dependency), add remaining
	if len(sorted) < len(tables) {
		for _, t := range tables {
			found := false
			for _, s := range sorted {
				if s == t {
					found = true
					break
				}
			}
			if !found {
				sorted = append(sorted, t)
			}
		}
	}

	return sorted, nil
}

// getEnumTypes retrieves all custom ENUM types from PostgreSQL
func (s *Service) getEnumTypes(ctx context.Context) ([]common.ExportEnumType, error) {
	// This query works for PostgreSQL to get all enum types and their values
	query := `
		SELECT t.typname as enum_name,
		       array_agg(e.enumlabel ORDER BY e.enumsortorder) as enum_values
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public'
		GROUP BY t.typname
		ORDER BY t.typname
	`

	result, err := s.adapter.ExecuteQuery(ctx, query)
	if err != nil {
		// Not PostgreSQL or no enums - return empty
		return []common.ExportEnumType{}, nil
	}

	var enumTypes []common.ExportEnumType
	for _, row := range result.Rows {
		enumName, ok := row["enum_name"].(string)
		if !ok {
			continue
		}

		var values []string
		// Handle the array of enum values
		if enumValues, ok := row["enum_values"].([]any); ok {
			for _, v := range enumValues {
				if str, ok := v.(string); ok {
					values = append(values, str)
				}
			}
		} else if enumValuesStr, ok := row["enum_values"].(string); ok {
			// PostgreSQL may return as string like {val1,val2,val3}
			enumValuesStr = strings.Trim(enumValuesStr, "{}")
			if enumValuesStr != "" {
				values = strings.Split(enumValuesStr, ",")
			}
		}

		if len(values) > 0 {
			enumTypes = append(enumTypes, common.ExportEnumType{
				Name:   enumName,
				Values: values,
			})
		}
	}

	return enumTypes, nil
}

// ExportDatabase exports the database schema and/or data based on export type
func (s *Service) ExportDatabase(exportType common.ExportType) (*common.ExportData, error) {
	s.ensureCorrectSchema()

	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	// Get all tables
	tables, err := s.adapter.GetAllTableNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Sort tables by dependency order (tables without FK first)
	sortedTables, err := s.sortTablesByDependency(ctx, tables)
	if err != nil {
		// Fallback to original order if sorting fails
		sortedTables = tables
	}

	provider := "sql"
	if s.cfg != nil {
		provider = s.cfg.Database.Provider
	}

	exportData := &common.ExportData{
		Version:          "1.0",
		ExportedAt:       time.Now().UTC().Format(time.RFC3339),
		DatabaseProvider: provider,
		ExportType:       exportType,
		Tables:           make([]common.ExportTable, 0),
	}

	// Export ENUM types for schema exports (PostgreSQL)
	if exportType == common.ExportSchemaOnly || exportType == common.ExportComplete {
		if provider == "postgresql" {
			enumTypes, err := s.getEnumTypes(ctx)
			if err == nil && len(enumTypes) > 0 {
				exportData.EnumTypes = enumTypes
			}
		}
	}

	for _, tableName := range sortedTables {
		if tableName == "_flash_migrations" {
			continue
		}

		exportTable := common.ExportTable{
			Name: tableName,
		}

		// Export schema if needed
		if exportType == common.ExportSchemaOnly || exportType == common.ExportComplete {
			schema, err := s.getTableSchema(ctx, tableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
			}
			exportTable.Schema = schema
		}

		// Export data if needed
		if exportType == common.ExportDataOnly || exportType == common.ExportComplete {
			data, err := s.getAllTableData(ctx, tableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get data for table %s: %w", tableName, err)
			}
			exportTable.Data = data
		}

		exportData.Tables = append(exportData.Tables, exportTable)
	}

	return exportData, nil
}

// getTableSchema returns the schema for a table
func (s *Service) getTableSchema(ctx context.Context, tableName string) (*common.ExportTableSchema, error) {
	columns, err := s.adapter.GetTableColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}

	exportColumns := make([]common.ExportColumn, 0, len(columns))
	seen := make(map[string]bool)

	for _, col := range columns {
		if seen[col.Name] {
			continue
		}
		seen[col.Name] = true

		exportColumns = append(exportColumns, common.ExportColumn{
			Name:             col.Name,
			Type:             col.Type,
			Nullable:         col.Nullable,
			PrimaryKey:       col.IsPrimary,
			Default:          col.Default,
			AutoIncrement:    col.IsAutoIncrement,
			Unique:           col.IsUnique,
			ForeignKeyTable:  col.ForeignKeyTable,
			ForeignKeyColumn: col.ForeignKeyColumn,
		})
	}

	return &common.ExportTableSchema{
		Columns: exportColumns,
	}, nil
}

// getAllTableData returns all data from a table
func (s *Service) getAllTableData(ctx context.Context, tableName string) ([]map[string]any, error) {
	// Get total row count
	count, err := s.adapter.GetTableRowCount(ctx, tableName)
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return []map[string]any{}, nil
	}

	// Fetch all data in batches
	batchSize := 1000
	allData := make([]map[string]any, 0, count)

	for offset := 0; offset < count; offset += batchSize {
		query := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",
			common.QuoteIdentifier(tableName), batchSize, offset)

		result, err := s.adapter.ExecuteQuery(ctx, query)
		if err != nil {
			// Fallback to GetTableData
			data, err := s.adapter.GetTableData(ctx, tableName)
			if err != nil {
				return nil, err
			}
			return data, nil
		}

		allData = append(allData, result.Rows...)
	}

	return allData, nil
}

// sortImportTablesByDependency sorts import tables in topological order based on foreign key dependencies
func (s *Service) sortImportTablesByDependency(tables []common.ExportTable) []common.ExportTable {
	// Build dependency graph from schema info
	dependencies := make(map[string][]string)
	tableMap := make(map[string]common.ExportTable)

	for _, t := range tables {
		tableMap[t.Name] = t
		dependencies[t.Name] = []string{}

		if t.Schema != nil {
			for _, col := range t.Schema.Columns {
				if col.ForeignKeyTable != "" {
					dependencies[t.Name] = append(dependencies[t.Name], col.ForeignKeyTable)
				}
			}
		}
	}

	// Calculate in-degree (number of dependencies)
	inDegree := make(map[string]int)
	for _, t := range tables {
		inDegree[t.Name] = len(dependencies[t.Name])
	}

	// Queue tables with no dependencies
	var queue []string
	for _, t := range tables {
		if inDegree[t.Name] == 0 {
			queue = append(queue, t.Name)
		}
	}

	var sortedNames []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sortedNames = append(sortedNames, current)

		// For each table that depends on current, reduce its in-degree
		for name, deps := range dependencies {
			for _, dep := range deps {
				if dep == current {
					inDegree[name]--
					if inDegree[name] == 0 {
						queue = append(queue, name)
					}
				}
			}
		}
	}

	// Add any remaining tables (circular dependencies)
	for _, t := range tables {
		found := false
		for _, name := range sortedNames {
			if name == t.Name {
				found = true
				break
			}
		}
		if !found {
			sortedNames = append(sortedNames, t.Name)
		}
	}

	// Build sorted result
	sorted := make([]common.ExportTable, 0, len(tables))
	for _, name := range sortedNames {
		if t, ok := tableMap[name]; ok {
			sorted = append(sorted, t)
		}
	}

	return sorted
}

// getFKChecksState queries the current FK check state and returns a restore function.
// It only disables FK checks if they are currently enabled; if already disabled it's a no-op.
func (s *Service) disableFKChecksIfNeeded(ctx context.Context) (restore func()) {
	provider := ""
	if s.cfg != nil {
		provider = s.cfg.Database.Provider
	}

	switch provider {
	case "mysql":
		// Query current state
		res, err := s.adapter.ExecuteQuery(ctx, "SELECT @@FOREIGN_KEY_CHECKS AS fk")
		if err == nil && len(res.Rows) > 0 {
			val := fmt.Sprintf("%v", res.Rows[0]["fk"])
			if val == "0" {
				// Already disabled, nothing to do
				return func() {}
			}
		}
		_ = s.adapter.ExecuteMigration(ctx, "SET FOREIGN_KEY_CHECKS = 0")
		return func() {
			_ = s.adapter.ExecuteMigration(ctx, "SET FOREIGN_KEY_CHECKS = 1")
		}

	case "sqlite", "sqlite3":
		res, err := s.adapter.ExecuteQuery(ctx, "PRAGMA foreign_keys")
		if err == nil && len(res.Rows) > 0 {
			// PRAGMA returns "foreign_keys" column with 0 or 1
			for _, v := range res.Rows[0] {
				if fmt.Sprintf("%v", v) == "0" {
					// Already disabled
					return func() {}
				}
			}
		}
		_ = s.adapter.ExecuteMigration(ctx, "PRAGMA foreign_keys = OFF")
		return func() {
			_ = s.adapter.ExecuteMigration(ctx, "PRAGMA foreign_keys = ON")
		}

	default: // postgresql, postgres
		var original string
		res, err := s.adapter.ExecuteQuery(ctx, "SHOW session_replication_role")
		if err == nil && len(res.Rows) > 0 {
			for _, v := range res.Rows[0] {
				original = fmt.Sprintf("%v", v)
				break
			}
		}
		if original == "replica" {
			// Already disabled
			return func() {}
		}
		if original == "" {
			original = "origin"
		}
		_ = s.adapter.ExecuteMigration(ctx, "SET session_replication_role = 'replica'")
		return func() {
			_ = s.adapter.ExecuteMigration(ctx, fmt.Sprintf("SET session_replication_role = '%s'", original))
		}
	}
}

// createEnumType creates a PostgreSQL ENUM type
func (s *Service) createEnumType(ctx context.Context, enumType common.ExportEnumType) error {
	// Quote each enum value
	quotedValues := make([]string, len(enumType.Values))
	for i, v := range enumType.Values {
		quotedValues[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	}

	query := fmt.Sprintf("CREATE TYPE %s AS ENUM (%s)",
		common.QuoteIdentifier(enumType.Name),
		strings.Join(quotedValues, ", "))

	return s.adapter.ExecuteMigration(ctx, query)
}

// ImportDatabase imports data from an export file
func (s *Service) ImportDatabase(importData *common.ExportData) (*common.ImportResult, error) {
	s.ensureCorrectSchema()

	result := &common.ImportResult{
		EnumTypesCreated: make([]string, 0),
		TablesCreated:    make([]string, 0),
		TablesUpdated:    make([]string, 0),
		Errors:           make([]string, 0),
	}

	ctx, cancel := context.WithTimeout(s.ctx, 120*time.Second)
	defer cancel()

	// Phase 0: Create ENUM types first (before tables)
	if len(importData.EnumTypes) > 0 {
		for _, enumType := range importData.EnumTypes {
			if err := s.createEnumType(ctx, enumType); err != nil {
				// Check if enum already exists (not an error)
				if !strings.Contains(err.Error(), "already exists") {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to create enum %s: %v", enumType.Name, err))
				}
			} else {
				result.EnumTypesCreated = append(result.EnumTypesCreated, enumType.Name)
			}
		}
	}

	// Get existing tables
	existingTables, err := s.adapter.GetAllTableNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing tables: %w", err)
	}
	existingTableMap := make(map[string]bool)
	for _, t := range existingTables {
		existingTableMap[t] = true
	}

	// Sort tables by dependency order
	sortedTables := s.sortImportTablesByDependency(importData.Tables)

	// Collect FK constraints to add after all tables are created
	type fkConstraint struct {
		tableName string
		colName   string
		fkTable   string
		fkColumn  string
	}
	var pendingFKs []fkConstraint

	// Phase 1: Create tables WITHOUT foreign key constraints
	for _, table := range sortedTables {
		tableExists := existingTableMap[table.Name]

		if table.Schema != nil {
			if !tableExists {
				// Create the table without FK constraints
				if err := s.createTableFromSchemaNoFK(ctx, table.Name, table.Schema); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to create table %s: %v", table.Name, err))
					continue
				}
				result.TablesCreated = append(result.TablesCreated, table.Name)
				existingTableMap[table.Name] = true

				// Collect FK constraints to add later
				for _, col := range table.Schema.Columns {
					if col.ForeignKeyTable != "" && col.ForeignKeyColumn != "" {
						pendingFKs = append(pendingFKs, fkConstraint{
							tableName: table.Name,
							colName:   col.Name,
							fkTable:   col.ForeignKeyTable,
							fkColumn:  col.ForeignKeyColumn,
						})
					}
				}
			} else {
				// Update existing table - add missing columns
				added, err := s.updateTableSchema(ctx, table.Name, table.Schema)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to update schema for %s: %v", table.Name, err))
				} else {
					result.ColumnsAdded += added
					if added > 0 {
						result.TablesUpdated = append(result.TablesUpdated, table.Name)
					}
				}
			}
		}
	}

	// Phase 2: Disable FK checks (if enabled) and import data in dependency order
	restoreFK := s.disableFKChecksIfNeeded(ctx)
	for _, table := range sortedTables {
		if len(table.Data) > 0 && existingTableMap[table.Name] {
			inserted, updated, err := s.importTableData(ctx, table.Name, table.Data)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to import data for %s: %v", table.Name, err))
			} else {
				result.RowsInserted += inserted
				result.RowsUpdated += updated
			}
		}
	}
	restoreFK()

	// Phase 3: Add foreign key constraints (after all data is in place)
	for _, fk := range pendingFKs {
		if !existingTableMap[fk.fkTable] {
			// Referenced table doesn't exist, skip this FK
			continue
		}

		query := fmt.Sprintf("ALTER TABLE %s ADD FOREIGN KEY (%s) REFERENCES %s(%s)",
			common.QuoteIdentifier(fk.tableName),
			common.QuoteIdentifier(fk.colName),
			common.QuoteIdentifier(fk.fkTable),
			common.QuoteIdentifier(fk.fkColumn))

		if err := s.adapter.ExecuteMigration(ctx, query); err != nil {
			// FK constraint errors are non-fatal, just log them
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to add FK on %s.%s: %v", fk.tableName, fk.colName, err))
		}
	}

	return result, nil
}

// createTableFromSchemaNoFK creates a new table from the export schema WITHOUT foreign key constraints
func (s *Service) createTableFromSchemaNoFK(ctx context.Context, tableName string, schema *common.ExportTableSchema) error {
	var columnDefs []string

	for _, col := range schema.Columns {
		def := fmt.Sprintf("%s %s", common.QuoteIdentifier(col.Name), col.Type)

		if col.PrimaryKey {
			def += " PRIMARY KEY"
		}
		if col.AutoIncrement {
			// Handle auto-increment based on database type
			if s.cfg != nil && (s.cfg.Database.Provider == "mysql") {
				def += " AUTO_INCREMENT"
			}
			// For PostgreSQL, SERIAL type already handles auto-increment
		}
		if !col.Nullable && !col.PrimaryKey {
			def += " NOT NULL"
		}
		if col.Unique && !col.PrimaryKey {
			def += " UNIQUE"
		}
		if col.Default != "" {
			def += fmt.Sprintf(" DEFAULT %s", col.Default)
		}

		columnDefs = append(columnDefs, def)
	}

	query := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)",
		common.QuoteIdentifier(tableName),
		strings.Join(columnDefs, ",\n  "))

	return s.adapter.ExecuteMigration(ctx, query)
}

// updateTableSchema updates an existing table by adding missing columns
func (s *Service) updateTableSchema(ctx context.Context, tableName string, schema *common.ExportTableSchema) (int, error) {
	// Get existing columns
	existingCols, err := s.adapter.GetTableColumns(ctx, tableName)
	if err != nil {
		return 0, err
	}

	existingColMap := make(map[string]bool)
	for _, col := range existingCols {
		existingColMap[col.Name] = true
	}

	added := 0
	for _, col := range schema.Columns {
		if existingColMap[col.Name] {
			continue // Column already exists
		}

		// Add the missing column
		def := col.Type
		if !col.Nullable {
			// For adding columns, we need to allow NULL or provide a default
			if col.Default != "" {
				def += fmt.Sprintf(" DEFAULT %s", col.Default)
			}
			// Don't add NOT NULL when adding column without default to avoid errors
		}
		if col.Unique {
			def += " UNIQUE"
		}
		if col.Default != "" && col.Nullable {
			def += fmt.Sprintf(" DEFAULT %s", col.Default)
		}

		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			common.QuoteIdentifier(tableName),
			common.QuoteIdentifier(col.Name),
			def)

		if err := s.adapter.ExecuteMigration(ctx, query); err != nil {
			return added, fmt.Errorf("failed to add column %s: %w", col.Name, err)
		}
		added++
	}

	return added, nil
}

// importTableData imports data into an existing table using batch operations
func (s *Service) importTableData(ctx context.Context, tableName string, data []map[string]any) (int, int, error) {
	if len(data) == 0 {
		return 0, 0, nil
	}

	// Get primary key column once
	columns, err := s.adapter.GetTableColumns(ctx, tableName)
	if err != nil {
		return 0, 0, err
	}

	pkColumn := ""
	for _, col := range columns {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	// Batch-check which PKs already exist (single query instead of N queries)
	existingPKs := make(map[string]bool)
	if pkColumn != "" {
		const checkBatch = 500
		for i := 0; i < len(data); i += checkBatch {
			end := i + checkBatch
			if end > len(data) {
				end = len(data)
			}
			var pkValues []string
			for _, row := range data[i:end] {
				if pkValue, ok := row[pkColumn]; ok && pkValue != nil {
					strVal := fmt.Sprintf("%v", pkValue)
					escaped := strings.ReplaceAll(strVal, "'", "''")
					pkValues = append(pkValues, fmt.Sprintf("'%s'", escaped))
				}
			}
			if len(pkValues) == 0 {
				continue
			}
			query := fmt.Sprintf("SELECT %s FROM %s WHERE %s IN (%s)",
				common.QuoteIdentifier(pkColumn),
				common.QuoteIdentifier(tableName),
				common.QuoteIdentifier(pkColumn),
				strings.Join(pkValues, ","))
			result, err := s.adapter.ExecuteQuery(ctx, query)
			if err == nil {
				for _, row := range result.Rows {
					if v, ok := row[pkColumn]; ok {
						existingPKs[fmt.Sprintf("%v", v)] = true
					}
				}
			}
		}
	}

	// Split rows into new (batch insert) and existing (update)
	var newRows []map[string]any
	var updateRows []map[string]any
	for _, row := range data {
		if pkColumn != "" {
			if pkValue, ok := row[pkColumn]; ok && pkValue != nil {
				if existingPKs[fmt.Sprintf("%v", pkValue)] {
					updateRows = append(updateRows, row)
					continue
				}
			}
		}
		newRows = append(newRows, row)
	}

	inserted := 0
	updated := 0

	// Batch INSERT new rows (multi-row VALUES)
	if len(newRows) > 0 {
		// Collect stable column order from first row
		var colNames []string
		for col := range newRows[0] {
			colNames = append(colNames, col)
		}

		var quotedCols []string
		for _, col := range colNames {
			quotedCols = append(quotedCols, common.QuoteIdentifier(col))
		}
		colList := strings.Join(quotedCols, ", ")

		const insertBatch = 200
		for i := 0; i < len(newRows); i += insertBatch {
			end := i + insertBatch
			if end > len(newRows) {
				end = len(newRows)
			}
			batch := newRows[i:end]

			var valueGroups []string
			for _, row := range batch {
				var vals []string
				for _, col := range colNames {
					v, ok := row[col]
					if !ok || v == nil {
						vals = append(vals, "NULL")
					} else {
						strVal := fmt.Sprintf("%v", v)
						escaped := strings.ReplaceAll(strVal, "'", "''")
						vals = append(vals, fmt.Sprintf("'%s'", escaped))
					}
				}
				valueGroups = append(valueGroups, "("+strings.Join(vals, ", ")+")")
			}

			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
				common.QuoteIdentifier(tableName), colList,
				strings.Join(valueGroups, ", "))

			if err := s.adapter.ExecuteMigration(ctx, query); err != nil {
				// Fallback: insert one by one
				for _, row := range batch {
					var vals []string
					for _, col := range colNames {
						v, ok := row[col]
						if !ok || v == nil {
							vals = append(vals, "NULL")
						} else {
							strVal := fmt.Sprintf("%v", v)
							escaped := strings.ReplaceAll(strVal, "'", "''")
							vals = append(vals, fmt.Sprintf("'%s'", escaped))
						}
					}
					single := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
						common.QuoteIdentifier(tableName), colList,
						strings.Join(vals, ", "))
					if err := s.adapter.ExecuteMigration(ctx, single); err != nil {
						continue
					}
					inserted++
				}
			} else {
				inserted += len(batch)
			}
		}
	}

	// Update existing rows (individual updates, but without redundant schema lookups)
	for _, row := range updateRows {
		var setClauses []string
		for col, val := range row {
			if col == pkColumn {
				continue
			}
			if val == nil {
				setClauses = append(setClauses, fmt.Sprintf("%s = NULL", common.QuoteIdentifier(col)))
			} else {
				strVal := fmt.Sprintf("%v", val)
				escaped := strings.ReplaceAll(strVal, "'", "''")
				setClauses = append(setClauses, fmt.Sprintf("%s = '%s'", common.QuoteIdentifier(col), escaped))
			}
		}
		if len(setClauses) == 0 {
			continue
		}
		pkVal := fmt.Sprintf("%v", row[pkColumn])
		escapedPK := strings.ReplaceAll(pkVal, "'", "''")
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%s'",
			common.QuoteIdentifier(tableName),
			strings.Join(setClauses, ", "),
			common.QuoteIdentifier(pkColumn), escapedPK)
		if err := s.adapter.ExecuteMigration(ctx, query); err != nil {
			continue
		}
		updated++
	}

	return inserted, updated, nil
}
