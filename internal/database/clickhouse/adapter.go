package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	chdriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Lumos-Labs-HQ/flash/internal/database/common"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

type Adapter struct {
	db *sql.DB
}

func New() *Adapter {
	return &Adapter{}
}

// typeMap normalises ClickHouse type names to canonical forms used by Flash.
var typeMap = map[string]string{
	"int8": "INT8", "int16": "INT16", "int32": "INT32", "int64": "INT64",
	"uint8": "UINT8", "uint16": "UINT16", "uint32": "UINT32", "uint64": "UINT64",
	"float32": "FLOAT32", "float64": "FLOAT64",
	"string": "STRING", "fixedstring": "FIXEDSTRING",
	"date": "DATE", "date32": "DATE32",
	"datetime": "DATETIME", "datetime64": "DATETIME64",
	"uuid": "UUID",
	"boolean": "BOOLEAN", "bool": "BOOLEAN",
	"decimal": "DECIMAL",
	"json": "JSON",
}

func (a *Adapter) Connect(ctx context.Context, url string) error {
	opts, err := chdriver.ParseDSN(url)
	if err != nil {
		return fmt.Errorf("failed to parse ClickHouse DSN: %w", err)
	}
	db := chdriver.OpenDB(opts)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	a.db = db
	return nil
}

func (a *Adapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	return a.db.PingContext(ctx)
}

// ── Migrations table ────────────────────────────────────────────────────────

func (a *Adapter) CreateMigrationsTable(ctx context.Context) error {
	// ReplacingMergeTree deduplicates rows by primary key on merge — good for an idempotent migrations log.
	_, err := a.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS _flash_migrations (
			id            String,
			migration_name String,
			checksum      String,
			started_at    DateTime DEFAULT now(),
			finished_at   Nullable(DateTime),
			applied_steps_count UInt32 DEFAULT 0
		) ENGINE = ReplacingMergeTree()
		ORDER BY id
	`)
	return err
}

func (a *Adapter) EnsureMigrationTableCompatibility(_ context.Context) error { return nil }

func (a *Adapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := a.db.ExecContext(ctx,
		`ALTER TABLE _flash_migrations DELETE WHERE finished_at IS NULL AND started_at < now() - INTERVAL 1 HOUR`)
	return err
}

func (a *Adapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, finished_at FROM _flash_migrations WHERE finished_at IS NOT NULL`)
	if err != nil {
		// Table doesn't exist yet — treat as no migrations applied
		if strings.Contains(err.Error(), "doesn't exist") || strings.Contains(err.Error(), "UNKNOWN_TABLE") {
			return make(map[string]*time.Time), nil
		}
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]*time.Time)
	for rows.Next() {
		var id string
		var finishedAt *time.Time
		if err := rows.Scan(&id, &finishedAt); err != nil {
			continue
		}
		applied[id] = finishedAt
	}
	return applied, rows.Err()
}

func (a *Adapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	now := time.Now()
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO _flash_migrations (id, migration_name, checksum, started_at, finished_at, applied_steps_count) VALUES (?, ?, ?, ?, ?, 1)`,
		migrationID, name, checksum, now, now)
	return err
}

func (a *Adapter) RemoveMigrationRecord(ctx context.Context, migrationID string) error {
	_, err := a.db.ExecContext(ctx,
		`ALTER TABLE _flash_migrations DELETE WHERE id = ?`, migrationID)
	return err
}

// ExecuteAndRecordMigration runs DDL statements then records the migration.
// ClickHouse has no transactions, so we run statements individually.
func (a *Adapter) ExecuteAndRecordMigration(ctx context.Context, migrationID, name, checksum, migrationSQL string) error {
	if migrationSQL != "" {
		if err := a.ExecuteMigration(ctx, migrationSQL); err != nil {
			return err
		}
	}
	return a.RecordMigration(ctx, migrationID, name, checksum)
}

// ExecuteMigration runs each SQL statement individually (no transactions in ClickHouse).
func (a *Adapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	for _, stmt := range common.ParseSQLStatements(migrationSQL) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := a.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("clickhouse: failed to execute %q: %w", stmt, err)
		}
	}
	return nil
}

// ── Query helpers ────────────────────────────────────────────────────────────

func (a *Adapter) ExecuteQuery(ctx context.Context, query string) (*common.QueryResult, error) {
	return a.ExecuteQueryWithArgs(ctx, query)
}

func (a *Adapter) ExecuteQueryWithArgs(ctx context.Context, query string, args ...interface{}) (*common.QueryResult, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		results = append(results, row)
	}
	return &common.QueryResult{Columns: cols, Rows: results}, rows.Err()
}

func (a *Adapter) ExecuteDMLWithArgs(ctx context.Context, query string, args ...interface{}) error {
	_, err := a.db.ExecContext(ctx, query, args...)
	return err
}

func (a *Adapter) QuoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func (a *Adapter) ProviderName() string { return "clickhouse" }

// ── Type mapping ─────────────────────────────────────────────────────────────

func (a *Adapter) MapColumnType(dbType string) string {
	// Strip Nullable(...) wrapper for comparison
	core := stripNullable(dbType)
	if mapped, ok := typeMap[strings.ToLower(core)]; ok {
		return mapped
	}
	return strings.ToUpper(core)
}

func stripNullable(t string) string {
	t = strings.TrimSpace(t)
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "nullable(") && strings.HasSuffix(t, ")") {
		return t[9 : len(t)-1]
	}
	return t
}

// ── Schema introspection ─────────────────────────────────────────────────────

func (a *Adapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	db := a.currentDatabase(ctx)
	rows, err := a.db.QueryContext(ctx,
		`SELECT name FROM system.tables WHERE database = ? AND name NOT LIKE '_flash_%' ORDER BY name`, db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			continue
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

func (a *Adapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return a.PullCompleteSchema(ctx)
}

func (a *Adapter) GetCurrentEnums(_ context.Context) ([]types.SchemaEnum, error) {
	// ClickHouse doesn't have user-defined enum types at the DB level
	return nil, nil
}

func (a *Adapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	db := a.currentDatabase(ctx)
	rows, err := a.db.QueryContext(ctx,
		`SELECT name, type, default_expression FROM system.columns WHERE database = ? AND table = ? ORDER BY position`,
		db, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []types.SchemaColumn
	for rows.Next() {
		var name, chType, defaultExpr string
		if err := rows.Scan(&name, &chType, &defaultExpr); err != nil {
			continue
		}
		cols = append(cols, types.SchemaColumn{
			Name:     name,
			Type:     chType,
			Nullable: strings.HasPrefix(strings.ToLower(chType), "nullable("),
			Default:  defaultExpr,
		})
	}
	return cols, rows.Err()
}

func (a *Adapter) GetTableIndexes(_ context.Context, _ string) ([]types.SchemaIndex, error) {
	// ClickHouse doesn't have traditional indexes
	return nil, nil
}

func (a *Adapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	db := a.currentDatabase(ctx)
	rows, err := a.db.QueryContext(ctx,
		`SELECT table, name, type, default_expression
		 FROM system.columns
		 WHERE database = ? AND table NOT LIKE '_flash_%'
		 ORDER BY table, position`,
		db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableMap := make(map[string]*types.SchemaTable)
	var order []string

	for rows.Next() {
		var tableName, colName, chType, defaultExpr string
		if err := rows.Scan(&tableName, &colName, &chType, &defaultExpr); err != nil {
			continue
		}
		if _, ok := tableMap[tableName]; !ok {
			tableMap[tableName] = &types.SchemaTable{Name: tableName}
			order = append(order, tableName)
		}
		tableMap[tableName].Columns = append(tableMap[tableName].Columns, types.SchemaColumn{
			Name:     colName,
			Type:     chType,
			Nullable: strings.HasPrefix(strings.ToLower(chType), "nullable("),
			Default:  defaultExpr,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tables := make([]types.SchemaTable, 0, len(order))
	for _, name := range order {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}

// ── SQL generation ───────────────────────────────────────────────────────────

// GenerateCreateTableSQL generates a ClickHouse CREATE TABLE using MergeTree engine.
// ClickHouse requires an ORDER BY clause; we use the primary key columns or the first column.
func (a *Adapter) GenerateCreateTableSQL(table types.SchemaTable) string {
	var lines []string
	var pkCols []string

	lines = append(lines, fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (", table.Name))

	for i, col := range table.Columns {
		comma := ","
		if i == len(table.Columns)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("    `%s` %s%s", col.Name, a.FormatColumnType(col), comma))
		if col.IsPrimary {
			pkCols = append(pkCols, "`"+col.Name+"`")
		}
	}
	lines = append(lines, ")")
	lines = append(lines, "ENGINE = MergeTree()")

	if len(pkCols) > 0 {
		lines = append(lines, fmt.Sprintf("ORDER BY (%s);", strings.Join(pkCols, ", ")))
	} else if len(table.Columns) > 0 {
		lines = append(lines, fmt.Sprintf("ORDER BY (`%s`);", table.Columns[0].Name))
	} else {
		lines = append(lines, "ORDER BY tuple();")
	}

	return strings.Join(lines, "\n")
}

func (a *Adapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN IF NOT EXISTS `%s` %s;",
		tableName, column.Name, a.FormatColumnType(column))
}

func (a *Adapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN IF EXISTS `%s`;", tableName, columnName)
}

func (a *Adapter) GenerateAlterColumnSQL(tableName string, column types.SchemaColumn, oldType string) string {
	if column.Type == oldType {
		return ""
	}
	return fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` %s;",
		tableName, column.Name, a.FormatColumnType(column))
}

// ClickHouse uses data skipping indexes, not traditional SQL indexes.
// We generate a basic minmax index as a reasonable default.
func (a *Adapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	cols := strings.Join(index.Columns, ", ")
	indexType := "minmax"
	if index.Unique {
		// ClickHouse doesn't enforce uniqueness via indexes; bloom_filter is the closest for equality checks
		indexType = "bloom_filter"
	}
	return fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (%s) TYPE %s GRANULARITY 1;",
		index.Table, index.Name, cols, indexType)
}

func (a *Adapter) GenerateDropIndexSQL(index types.SchemaIndex) string {
	return fmt.Sprintf("ALTER TABLE `%s` DROP INDEX IF EXISTS `%s`;", index.Table, index.Name)
}

func (a *Adapter) FormatColumnType(column types.SchemaColumn) string {
	t := column.Type
	if column.Nullable && !strings.HasPrefix(strings.ToLower(t), "nullable(") {
		t = fmt.Sprintf("Nullable(%s)", t)
	}
	if column.Default != "" {
		t += fmt.Sprintf(" DEFAULT %s", column.Default)
	}
	return t
}

// ── Conflict detection ───────────────────────────────────────────────────────

func (a *Adapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	db := a.currentDatabase(ctx)
	var count uint64
	err := a.db.QueryRowContext(ctx,
		`SELECT count() FROM system.tables WHERE database = ? AND name = ?`, db, tableName).Scan(&count)
	return count > 0, err
}

func (a *Adapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	db := a.currentDatabase(ctx)
	var count uint64
	err := a.db.QueryRowContext(ctx,
		`SELECT count() FROM system.columns WHERE database = ? AND table = ? AND name = ?`,
		db, tableName, columnName).Scan(&count)
	return count > 0, err
}

// ClickHouse doesn't have NOT NULL constraints in the traditional sense.
// A non-Nullable column is implicitly NOT NULL.
func (a *Adapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
	db := a.currentDatabase(ctx)
	var chType string
	err := a.db.QueryRowContext(ctx,
		`SELECT type FROM system.columns WHERE database = ? AND table = ? AND name = ?`,
		db, tableName, columnName).Scan(&chType)
	if err != nil {
		return false, err
	}
	return !strings.HasPrefix(strings.ToLower(chType), "nullable("), nil
}

// ClickHouse has no FK or UNIQUE constraints.
func (a *Adapter) CheckForeignKeyConstraint(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (a *Adapter) CheckUniqueConstraint(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// ── Data operations ──────────────────────────────────────────────────────────

func (a *Adapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM `%s`", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var result []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (a *Adapter) GetTableRowCount(ctx context.Context, tableName string) (int, error) {
	var count uint64
	err := a.db.QueryRowContext(ctx, fmt.Sprintf("SELECT count() FROM `%s`", tableName)).Scan(&count)
	return int(count), err
}

func (a *Adapter) GetAllTableRowCounts(ctx context.Context, tableNames []string) (map[string]int, error) {
	result := make(map[string]int, len(tableNames))
	for _, t := range tableNames {
		count, err := a.GetTableRowCount(ctx, t)
		if err != nil {
			result[t] = 0
			continue
		}
		result[t] = count
	}
	return result, nil
}

func (a *Adapter) DropTable(ctx context.Context, tableName string) error {
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
	return err
}

func (a *Adapter) DropEnum(_ context.Context, _ string) error {
	// ClickHouse has no user-defined enum types at the DB level
	return nil
}

// ── Branch operations ─────────────────────────────────────────────────────────
// ClickHouse maps "branches" to separate databases.

func (a *Adapter) CreateBranchSchema(ctx context.Context, branchName string) error {
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", branchName))
	return err
}

func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error {
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", branchName))
	return err
}

func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceDB, targetDB string) error {
	if err := a.DropBranchSchema(ctx, targetDB); err != nil {
		return err
	}
	if err := a.CreateBranchSchema(ctx, targetDB); err != nil {
		return err
	}

	rows, err := a.db.QueryContext(ctx,
		`SELECT name FROM system.tables WHERE database = ? AND name NOT LIKE '_flash_%'`, sourceDB)
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			continue
		}
		tables = append(tables, t)
	}

	for _, t := range tables {
		// CREATE TABLE ... AS ... copies structure + data in ClickHouse
		q := fmt.Sprintf("CREATE TABLE `%s`.`%s` AS `%s`.`%s`", targetDB, t, sourceDB, t)
		if _, err := a.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed to clone table %s: %w", t, err)
		}
	}
	return nil
}

func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchDB string) ([]types.SchemaTable, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT table, name, type FROM system.columns WHERE database = ? ORDER BY table, position`, branchDB)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableMap := make(map[string]*types.SchemaTable)
	var order []string
	for rows.Next() {
		var tableName, colName, chType string
		if err := rows.Scan(&tableName, &colName, &chType); err != nil {
			continue
		}
		if _, ok := tableMap[tableName]; !ok {
			tableMap[tableName] = &types.SchemaTable{Name: tableName}
			order = append(order, tableName)
		}
		tableMap[tableName].Columns = append(tableMap[tableName].Columns, types.SchemaColumn{
			Name:     colName,
			Type:     chType,
			Nullable: strings.HasPrefix(strings.ToLower(chType), "nullable("),
		})
	}

	tables := make([]types.SchemaTable, 0, len(order))
	for _, n := range order {
		tables = append(tables, *tableMap[n])
	}
	return tables, rows.Err()
}

func (a *Adapter) SetActiveSchema(_ context.Context, _ string) error {
	// ClickHouse doesn't have a per-connection schema/search_path
	return nil
}

func (a *Adapter) GetTableNamesInSchema(ctx context.Context, dbName string) ([]string, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT name FROM system.tables WHERE database = ? ORDER BY name`, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			continue
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (a *Adapter) currentDatabase(ctx context.Context) string {
	var db string
	_ = a.db.QueryRowContext(ctx, "SELECT currentDatabase()").Scan(&db)
	if db == "" {
		db = "default"
	}
	return db
}
