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

type Seeder struct {
	config      *config.Config
	adapter     database.DatabaseAdapter
	generator   *DataGenerator
	graph       *DependencyGraph
	insertedIDs map[string][]interface{}
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

	return &Seeder{
		config:      cfg,
		adapter:     adapter,
		generator:   NewDataGenerator(),
		graph:       NewDependencyGraph(),
		insertedIDs: make(map[string][]interface{}),
	}, nil
}

func (s *Seeder) Close() error {
	return s.adapter.Close()
}

func (s *Seeder) Seed(ctx context.Context, seedConfig SeedConfig) error {
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

	// Build dependency graph
	for _, table := range tables {
		s.graph.AddTable(table)
	}

	order, err := s.graph.BuildInsertionOrder()
	if err != nil {
		return fmt.Errorf("failed to build insertion order: %w", err)
	}

	color.Green("📊 Found %d tables", len(tables))
	color.Cyan("📋 Insertion order: %s", strings.Join(order, " → "))
	fmt.Println()

	// Truncate if requested
	if seedConfig.Truncate {
		if err := s.truncateTables(ctx, order); err != nil {
			return fmt.Errorf("failed to truncate tables: %w", err)
		}
	}

	// Seed tables in order
	for _, tableName := range order {
		table := tables[tableName]
		count := seedConfig.Count
		if tableCount, exists := seedConfig.Tables[tableName]; exists {
			count = tableCount
		}

		if err := s.seedTable(ctx, table, count, seedConfig.Relations); err != nil {
			return fmt.Errorf("failed to seed table %s: %w", tableName, err)
		}
	}

	color.Green("\n✅ Database seeding completed successfully!")
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

	for i := 0; i < count; i++ {
		record := make(map[string]interface{})

		for _, col := range table.Columns {
			// Skip auto-increment primary keys
			if col.IsPK {
				// Skip if it's a serial type or autoincrement
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
				} else {
					record[col.Name] = nil
				}
			} else {
				record[col.Name] = s.generator.GenerateForColumn(col.Name, col.Type, col.Nullable)
			}
		}

		// Insert record
		id, err := s.insertRecord(ctx, table.Name, record, table.PrimaryKey)
		if err != nil {
			return fmt.Errorf("failed to insert record: %w", err)
		}

		if id != nil {
			s.insertedIDs[table.Name] = append(s.insertedIDs[table.Name], id)
		}
	}

	color.Green("  ✅ %s seeded successfully", table.Name)
	return nil
}

func (s *Seeder) insertRecord(ctx context.Context, tableName string, record map[string]interface{}, pkColumn string) (interface{}, error) {
	var columns []string
	var valueStrs []string

	for col, val := range record {
		columns = append(columns, col)
		
		// Format value for SQL
		if val == nil {
			valueStrs = append(valueStrs, "NULL")
		} else {
			switch v := val.(type) {
			case string:
				// Escape single quotes
				escaped := strings.ReplaceAll(v, "'", "''")
				valueStrs = append(valueStrs, fmt.Sprintf("'%s'", escaped))
			case int, int32, int64, float32, float64:
				valueStrs = append(valueStrs, fmt.Sprintf("%v", v))
			case bool:
				if v {
					valueStrs = append(valueStrs, "1")
				} else {
					valueStrs = append(valueStrs, "0")
				}
			case time.Time:
				valueStrs = append(valueStrs, fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05")))
			default:
				valueStrs = append(valueStrs, fmt.Sprintf("'%v'", v))
			}
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(valueStrs, ", "),
	)

	// Debug: print query
	// fmt.Println("Query:", query)

	// Add RETURNING for PostgreSQL
	if s.config.Database.Provider == "postgresql" || s.config.Database.Provider == "postgres" {
		if pkColumn != "" {
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
			// Debug: print what we got
			// fmt.Printf("DEBUG last_insert_rowid result: %+v\n", idResult.Rows[0])
			for _, v := range idResult.Rows[0] {
				return v, nil
			}
		}
	}

	return nil, nil
}

func (s *Seeder) truncateTables(ctx context.Context, order []string) error {
	color.Yellow("🗑️  Truncating tables...")

	// Reverse order for truncation
	for i := len(order) - 1; i >= 0; i-- {
		tableName := order[i]

		var query string
		switch s.config.Database.Provider {
		case "postgresql", "postgres":
			query = fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)
			if _, err := s.adapter.ExecuteQuery(ctx, query); err != nil {
				color.Yellow("  ⚠️  Failed to truncate %s: %v", tableName, err)
			}
		case "mysql":
			query = fmt.Sprintf("TRUNCATE TABLE %s", tableName)
			if _, err := s.adapter.ExecuteQuery(ctx, query); err != nil {
				color.Yellow("  ⚠️  Failed to truncate %s: %v", tableName, err)
			}
		case "sqlite", "sqlite3":
			// Delete all rows
			query = fmt.Sprintf("DELETE FROM %s", tableName)
			if _, err := s.adapter.ExecuteQuery(ctx, query); err != nil {
				color.Yellow("  ⚠️  Failed to delete from %s: %v", tableName, err)
				continue
			}
			// Reset autoincrement counter
			resetQuery := fmt.Sprintf("DELETE FROM sqlite_sequence WHERE name='%s'", tableName)
			if _, err := s.adapter.ExecuteQuery(ctx, resetQuery); err != nil {
				// Ignore error if sqlite_sequence doesn't exist
			}
		default:
			query = fmt.Sprintf("DELETE FROM %s", tableName)
			if _, err := s.adapter.ExecuteQuery(ctx, query); err != nil {
				color.Yellow("  ⚠️  Failed to truncate %s: %v", tableName, err)
			}
		}
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
