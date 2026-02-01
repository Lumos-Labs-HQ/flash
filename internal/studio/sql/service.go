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
