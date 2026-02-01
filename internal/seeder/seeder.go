package seeder

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/fatih/color"
)

// validIdentifier validates SQL identifiers (table/column names) to prevent SQL injection
var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Seeder struct {
	config      *config.Config
	adapter     database.DatabaseAdapter
	generator   *DataGenerator
	graph       *DependencyGraph
	insertedIDs map[string][]interface{}
	seedConfig  SeedConfig
}

func NewSeeder(cfg *config.Config) (*Seeder, error) {
	adapter := database.NewAdapter(cfg.Database.Provider)

	dbURL, err := cfg.GetDatabaseURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get database URL: %w", err)
	}

	if err := adapter.Connect(context.Background(), dbURL); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	generator, err := NewDataGenerator()
	if err != nil {
		adapter.Close()
		return nil, fmt.Errorf("failed to create data generator: %w", err)
	}

	return &Seeder{
		config:      cfg,
		adapter:     adapter,
		generator:   generator,
		graph:       NewDependencyGraph(),
		insertedIDs: make(map[string][]interface{}),
	}, nil
}

// isValidIdentifier checks if a string is a valid SQL identifier
func isValidIdentifier(name string) bool {
	return validIdentifier.MatchString(name)
}

func (s *Seeder) Close() error {
	return s.adapter.Close()
}

func (s *Seeder) Seed(ctx context.Context, seedConfig SeedConfig) error {
	s.seedConfig = seedConfig
	color.Cyan("🌱 Starting database seeding...")

	// Parse schema
	tables, err := s.parseSchema()
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	if len(tables) == 0 {
		color.Yellow("⚠️  No tables found in schema")
		return nil
	}

	// Validate all table and column names
	for tableName, table := range tables {
		if !isValidIdentifier(tableName) {
			return fmt.Errorf("invalid table name: %s", tableName)
		}
		for _, col := range table.Columns {
			if !isValidIdentifier(col.Name) {
				return fmt.Errorf("invalid column name in table %s: %s", tableName, col.Name)
			}
		}
	}

	// Build dependency graph
	for _, table := range tables {
		s.graph.AddTable(table)
	}

	order, err := s.graph.BuildInsertionOrder()
	if err != nil {
		return fmt.Errorf("failed to build insertion order: %w", err)
	}

	// Validate FK references exist when relations are enabled
	if seedConfig.Relations {
		if err := s.validateForeignKeys(tables, order); err != nil {
			return fmt.Errorf("foreign key validation failed: %w", err)
		}
	}

	color.Green("📊 Found %d tables", len(tables))
	color.Cyan("📋 Insertion order: %s", strings.Join(order, " → "))
	fmt.Println()

	// Truncate if requested (outside transaction)
	if seedConfig.Truncate {
		if err := s.truncateTables(ctx, order); err != nil {
			if !seedConfig.Force {
				return fmt.Errorf("failed to truncate tables: %w (use --force to continue)", err)
			}
			color.Yellow("⚠️  Truncate failed but continuing with --force: %v", err)
		}
	}

	// Start transaction for seeding (unless disabled)
	inTransaction := false
	if !seedConfig.NoTransaction {
		if err := s.beginTransaction(ctx); err != nil {
			color.Yellow("⚠️  Could not start transaction: %v (continuing without transaction)", err)
		} else {
			inTransaction = true
			color.Cyan("🔒 Transaction started")
		}
	}

	// Seed tables in order using batch inserts
	var seedErr error
	for _, tableName := range order {
		table := tables[tableName]
		count := seedConfig.Count
		if tableCount, exists := seedConfig.Tables[tableName]; exists {
			count = tableCount
		}

		if err := s.seedTable(ctx, table, count, seedConfig.Relations); err != nil {
			if !seedConfig.Force {
				seedErr = fmt.Errorf("failed to seed table %s: %w", tableName, err)
				break
			}
			color.Yellow("⚠️  Failed to seed %s but continuing with --force: %v", tableName, err)
		}
	}

	// Handle transaction commit/rollback
	if inTransaction {
		if seedErr != nil {
			color.Yellow("🔄 Rolling back transaction due to error...")
			if rbErr := s.rollbackTransaction(ctx); rbErr != nil {
				return fmt.Errorf("seed failed and rollback failed: %v (original: %w)", rbErr, seedErr)
			}
			color.Yellow("✅ Transaction rolled back")
			return seedErr
		}

		if err := s.commitTransaction(ctx); err != nil {
			s.rollbackTransaction(ctx)
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		color.Cyan("🔓 Transaction committed")
	} else if seedErr != nil {
		return seedErr
	}

	color.Green("\n✅ Database seeding completed successfully!")
	return nil
}

// beginTransaction starts a new database transaction
func (s *Seeder) beginTransaction(ctx context.Context) error {
	var query string
	switch s.config.Database.Provider {
	case "mysql":
		query = "START TRANSACTION"
	default:
		query = "BEGIN"
	}
	_, err := s.adapter.ExecuteQuery(ctx, query)
	return err
}

// commitTransaction commits the current transaction
func (s *Seeder) commitTransaction(ctx context.Context) error {
	_, err := s.adapter.ExecuteQuery(ctx, "COMMIT")
	return err
}

// rollbackTransaction rolls back the current transaction
func (s *Seeder) rollbackTransaction(ctx context.Context) error {
	_, err := s.adapter.ExecuteQuery(ctx, "ROLLBACK")
	return err
}

// validateForeignKeys checks that all FK references can be satisfied
func (s *Seeder) validateForeignKeys(tables map[string]*TableInfo, order []string) error {
	orderIndex := make(map[string]int)
	for i, name := range order {
		orderIndex[name] = i
	}

	for _, table := range tables {
		for _, col := range table.Columns {
			if col.IsFK && !col.Nullable {
				refIdx, exists := orderIndex[col.FKTable]
				if !exists {
					return fmt.Errorf("table %s has NOT NULL FK column %s referencing non-existent table %s",
						table.Name, col.Name, col.FKTable)
				}
				thisIdx := orderIndex[table.Name]
				if refIdx >= thisIdx {
					return fmt.Errorf("table %s has NOT NULL FK column %s but referenced table %s is seeded later (circular dependency?)",
						table.Name, col.Name, col.FKTable)
				}
			}
		}
	}
	return nil
}

func (s *Seeder) parseSchema() (map[string]*TableInfo, error) {
	schemaFiles, err := s.config.GetSchemaFiles()
	if err != nil {
		return nil, err
	}

	tables := make(map[string]*TableInfo)
	createTableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["']?(\w+)["']?\s*\(([\s\S]*?)\);`)

	for _, file := range schemaFiles {
		content, err := s.readFile(file)
		if err != nil {
			continue
		}

		matches := createTableRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			tableName := match[1]
			tableBody := match[2]

			table := s.parseTableDefinition(tableName, tableBody)
			tables[tableName] = table
		}
	}

	return tables, nil
}

func (s *Seeder) parseTableDefinition(tableName, body string) *TableInfo {
	table := &TableInfo{
		Name:         tableName,
		Columns:      []ColumnInfo{},
		ForeignKeys:  []ForeignKey{},
		Dependencies: []string{},
	}

	lines := strings.Split(body, ",")
	fkRegex := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(["']?(\w+)["']?\)\s*REFERENCES\s+["']?(\w+)["']?\s*\(["']?(\w+)["']?\)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lineUpper := strings.ToUpper(line)

		// Check for foreign key constraint
		if fkMatch := fkRegex.FindStringSubmatch(line); fkMatch != nil {
			fk := ForeignKey{
				Column:    fkMatch[1],
				RefTable:  fkMatch[2],
				RefColumn: fkMatch[3],
			}
			table.ForeignKeys = append(table.ForeignKeys, fk)
			if fkMatch[2] != tableName {
				table.Dependencies = append(table.Dependencies, fkMatch[2])
			}
			continue
		}

		// Skip constraint definitions
		if strings.HasPrefix(lineUpper, "PRIMARY") ||
			strings.HasPrefix(lineUpper, "UNIQUE") ||
			strings.HasPrefix(lineUpper, "CHECK") ||
			strings.HasPrefix(lineUpper, "CONSTRAINT") ||
			strings.HasPrefix(lineUpper, "INDEX") ||
			strings.HasPrefix(lineUpper, "KEY") {
			continue
		}

		// Parse column definition
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		colName := strings.Trim(parts[0], `"'`)
		colType := parts[1]

		col := ColumnInfo{
			Name:     colName,
			Type:     colType,
			Nullable: !strings.Contains(lineUpper, "NOT NULL"),
			IsPK:     strings.Contains(lineUpper, "PRIMARY KEY") || strings.Contains(strings.ToUpper(colType), "SERIAL"),
		}

		// Check for inline REFERENCES
		if strings.Contains(lineUpper, "REFERENCES") {
			refRegex := regexp.MustCompile(`(?i)REFERENCES\s+["']?(\w+)["']?\s*\(["']?(\w+)["']?\)`)
			if refMatch := refRegex.FindStringSubmatch(line); refMatch != nil {
				col.IsFK = true
				col.FKTable = refMatch[1]
				col.FKColumn = refMatch[2]
				if refMatch[1] != tableName {
					table.Dependencies = append(table.Dependencies, refMatch[1])
				}
			}
		}

		if col.IsPK {
			table.PrimaryKey = colName
		}

		table.Columns = append(table.Columns, col)
	}

	// Mark columns that are foreign keys
	for _, fk := range table.ForeignKeys {
		for i := range table.Columns {
			if table.Columns[i].Name == fk.Column {
				table.Columns[i].IsFK = true
				table.Columns[i].FKTable = fk.RefTable
				table.Columns[i].FKColumn = fk.RefColumn
				break
			}
		}
	}

	return table
}

func (s *Seeder) seedTable(ctx context.Context, table *TableInfo, count int, withRelations bool) error {
	color.Cyan("  📝 Seeding %s (%d records)...", table.Name, count)

	batchSize := s.seedConfig.Batch
	if batchSize <= 0 {
		batchSize = 100 
	}

	var batch []map[string]interface{}

	for i := 0; i < count; i++ {
		record := make(map[string]interface{})

		for _, col := range table.Columns {
			// Skip auto-increment primary keys
			if col.IsPK {
				typeUpper := strings.ToUpper(col.Type)
				if strings.Contains(typeUpper, "SERIAL") ||
					strings.Contains(typeUpper, "AUTO_INCREMENT") ||
					strings.Contains(typeUpper, "AUTOINCREMENT") ||
					(strings.Contains(typeUpper, "INTEGER") && s.config.Database.Provider == "sqlite") {
					continue
				}
			}

			// Handle foreign keys
			if col.IsFK && withRelations {
				if ids, exists := s.insertedIDs[col.FKTable]; exists && len(ids) > 0 {
					record[col.Name] = ids[s.generator.rand.Intn(len(ids))]
				} else if !col.Nullable {
					// NOT NULL FK but no referenced data - this should have been caught in validation
					return fmt.Errorf("cannot set NULL for NOT NULL FK column %s (no data in referenced table %s)",
						col.Name, col.FKTable)
				} else {
					record[col.Name] = nil
				}
			} else {
				record[col.Name] = s.generator.GenerateForColumn(col.Name, col.Type, col.Nullable)
			}
		}

		batch = append(batch, record)

		// Insert batch when full or at end
		if len(batch) >= batchSize || i == count-1 {
			ids, err := s.insertBatch(ctx, table.Name, batch, table.PrimaryKey)
			if err != nil {
				return fmt.Errorf("failed to insert batch: %w", err)
			}
			s.insertedIDs[table.Name] = append(s.insertedIDs[table.Name], ids...)
			batch = batch[:0] // reset batch
		}
	}

	color.Green("  ✅ %s seeded successfully", table.Name)
	return nil
}

// insertBatch inserts multiple records in a single statement (when possible)
func (s *Seeder) insertBatch(ctx context.Context, tableName string, records []map[string]interface{}, pkColumn string) ([]interface{}, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Validate table name
	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// For small batches or databases that don't support multi-row inserts well, insert one by one
	if len(records) == 1 || s.config.Database.Provider == "sqlite" || s.config.Database.Provider == "sqlite3" {
		var ids []interface{}
		for _, record := range records {
			id, err := s.insertRecord(ctx, tableName, record, pkColumn)
			if err != nil {
				return ids, err
			}
			if id != nil {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}

	// Build multi-row INSERT for PostgreSQL/MySQL
	// Get column order from first record
	var columns []string
	for col := range records[0] {
		if !isValidIdentifier(col) {
			return nil, fmt.Errorf("invalid column name: %s", col)
		}
		columns = append(columns, col)
	}

	var allValueStrs []string
	for _, record := range records {
		var valueStrs []string
		for _, col := range columns {
			val := record[col]
			valueStrs = append(valueStrs, s.formatValue(val))
		}
		allValueStrs = append(allValueStrs, "("+strings.Join(valueStrs, ", ")+")")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(allValueStrs, ", "),
	)

	// Add RETURNING for PostgreSQL
	if (s.config.Database.Provider == "postgresql" || s.config.Database.Provider == "postgres") && pkColumn != "" {
		if !isValidIdentifier(pkColumn) {
			return nil, fmt.Errorf("invalid primary key column: %s", pkColumn)
		}
		query += fmt.Sprintf(" RETURNING %s", pkColumn)
	}

	result, err := s.adapter.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Extract IDs from result
	var ids []interface{}
	if result != nil && len(result.Rows) > 0 && pkColumn != "" {
		for _, row := range result.Rows {
			if val, ok := row[pkColumn]; ok {
				ids = append(ids, val)
			}
		}
	}

	return ids, nil
}

// formatValue formats a value for SQL insertion
func (s *Seeder) formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case string:
		// Escape single quotes and backslashes
		escaped := strings.ReplaceAll(v, "'", "''")
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
		return fmt.Sprintf("'%s'", escaped)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case time.Time:
		return fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05"))
	default:
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}

func (s *Seeder) insertRecord(ctx context.Context, tableName string, record map[string]interface{}, pkColumn string) (interface{}, error) {
	// Validate table name
	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	var columns []string
	var valueStrs []string

	for col, val := range record {
		// Validate column name
		if !isValidIdentifier(col) {
			return nil, fmt.Errorf("invalid column name: %s", col)
		}
		columns = append(columns, col)
		valueStrs = append(valueStrs, s.formatValue(val))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(valueStrs, ", "),
	)

	// Add RETURNING for PostgreSQL
	if s.config.Database.Provider == "postgresql" || s.config.Database.Provider == "postgres" {
		if pkColumn != "" {
			if !isValidIdentifier(pkColumn) {
				return nil, fmt.Errorf("invalid primary key column: %s", pkColumn)
			}
			query += fmt.Sprintf(" RETURNING %s", pkColumn)
		}
	}

	result, err := s.adapter.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Extract inserted ID
	if result != nil && len(result.Rows) > 0 {
		if pkColumn != "" {
			if val, ok := result.Rows[0][pkColumn]; ok {
				return val, nil
			}
		}
	}

	// For SQLite, query the last inserted ID
	if s.config.Database.Provider == "sqlite" || s.config.Database.Provider == "sqlite3" {
		idResult, err := s.adapter.ExecuteQuery(ctx, "SELECT last_insert_rowid()")
		if err == nil && idResult != nil && len(idResult.Rows) > 0 {
			for _, v := range idResult.Rows[0] {
				return v, nil
			}
		}
	}

	return nil, nil
}

func (s *Seeder) truncateTables(ctx context.Context, order []string) error {
	color.Yellow("🗑️  Truncating tables...")

	var errors []string

	// Reverse order for truncation (to respect FK constraints)
	for i := len(order) - 1; i >= 0; i-- {
		tableName := order[i]

		// Validate table name
		if !isValidIdentifier(tableName) {
			errors = append(errors, fmt.Sprintf("invalid table name: %s", tableName))
			continue
		}

		var query string
		var err error

		switch s.config.Database.Provider {
		case "postgresql", "postgres":
			query = fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)
			_, err = s.adapter.ExecuteQuery(ctx, query)
		case "mysql":
			query = fmt.Sprintf("TRUNCATE TABLE %s", tableName)
			_, err = s.adapter.ExecuteQuery(ctx, query)
		case "sqlite", "sqlite3":
			// Delete all rows
			query = fmt.Sprintf("DELETE FROM %s", tableName)
			_, err = s.adapter.ExecuteQuery(ctx, query)
			if err == nil {
				resetQuery := fmt.Sprintf("DELETE FROM sqlite_sequence WHERE name='%s'", tableName)
				s.adapter.ExecuteQuery(ctx, resetQuery) 
			}
		default:
			query = fmt.Sprintf("DELETE FROM %s", tableName)
			_, err = s.adapter.ExecuteQuery(ctx, query)
		}

		if err != nil {
			errMsg := fmt.Sprintf("failed to truncate %s: %v", tableName, err)
			errors = append(errors, errMsg)
			color.Yellow("  ⚠️  %s", errMsg)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("truncate errors: %s", strings.Join(errors, "; "))
	}

	color.Green("✅ Tables truncated")
	fmt.Println()
	return nil
}

func (s *Seeder) readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
