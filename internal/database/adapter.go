package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

type QueryResult struct {
	Columns []string
	Rows    []map[string]interface{}
}

type DatabaseAdapter interface {
	Connect(ctx context.Context, url string) error
	Close() error
	Ping(ctx context.Context) error

	// Migration table management
	CreateMigrationsTable(ctx context.Context) error
	EnsureMigrationTableCompatibility(ctx context.Context) error
	CleanupBrokenMigrationRecords(ctx context.Context) error

	// Migration operations
	GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error)
	RecordMigration(ctx context.Context, migrationID, name, checksum string) error
	ExecuteMigration(ctx context.Context, migrationSQL string) error
	ExecuteQuery(ctx context.Context, query string) (*QueryResult, error)

	// Schema operations
	GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error)
	GetCurrentEnums(ctx context.Context) ([]types.SchemaEnum, error)
	GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error)
	GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error)
	GetAllTableNames(ctx context.Context) ([]string, error)
	PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error)

	// Conflict detection
	CheckTableExists(ctx context.Context, tableName string) (bool, error)
	CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error)
	CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error)
	CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error)
	CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error)

	// Backup operations
	GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
	GetTableRowCount(ctx context.Context, tableName string) (int, error)
	GetAllTableRowCounts(ctx context.Context, tableNames []string) (map[string]int, error)
	DropTable(ctx context.Context, tableName string) error
	DropEnum(ctx context.Context, enumName string) error

	// SQL generation
	GenerateCreateTableSQL(table types.SchemaTable) string
	GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string
	GenerateDropColumnSQL(tableName, columnName string) string
	GenerateAddIndexSQL(index types.SchemaIndex) string
	GenerateDropIndexSQL(indexName string) string

	// Data type mapping
	MapColumnType(dbType string) string
	FormatColumnType(column types.SchemaColumn) string
}

type DatabaseConnection interface {
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Begin(ctx context.Context) (*sql.Tx, error)
}

func NewAdapter(provider string) DatabaseAdapter {
	switch provider {
	case "postgresql", "postgres":
		return NewPostgresAdapter()
	case "mysql":
		return NewMySQLAdapter()
	case "sqlite", "sqlite3":
		return NewSQLiteAdapter()
	default:
		return NewPostgresAdapter()
	}
}
