package types

import (
	"time"
)

type SchemaEnum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type SchemaTable struct {
	Name    string
	Columns []SchemaColumn
	Indexes []SchemaIndex
}

type SchemaColumn struct {
	Name             string
	Type             string
	Nullable         bool
	Default          string
	IsPrimary        bool
	IsUnique         bool
	IsAutoIncrement  bool // NEW: Indicates if column is auto-increment (SERIAL, AUTO_INCREMENT, etc.)
	ForeignKeyTable  string
	ForeignKeyColumn string
	OnDeleteAction   string
}

type SchemaIndex struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
}

type SchemaDiff struct {
	NewTables      []SchemaTable
	DroppedTables  []string
	ModifiedTables []TableDiff
	NewIndexes     []SchemaIndex
	DroppedIndexes []SchemaIndex // Changed from []string to include table name for MySQL DROP INDEX
	NewEnums       []SchemaEnum
	DroppedEnums   []string
}

type TableDiff struct {
	Name            string
	NewColumns      []SchemaColumn
	DroppedColumns  []SchemaColumn // Changed from []string to preserve column info for DOWN migration
	ModifiedColumns []ColumnDiff
}

type ColumnDiff struct {
	Name    string
	OldType string
	NewType string
	Changes []string
}

type MigrationConflict struct {
	Type        string
	TableName   string
	ColumnName  string
	Description string
	Solutions   []string
	Severity    string
}

type Migration struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Applied   bool       `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
	FilePath  string     `json:"file_path"`
	Checksum  string     `json:"checksum"`
	CreatedAt time.Time  `json:"created_at"`
}

type MigrationSQL struct {
	Up   string
	Down string
}

type MigrationFile struct {
	ID       string
	Name     string
	Up       string
	Down     string
	Checksum string
	FilePath string
}

type BackupData struct {
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Tables    map[string]interface{} `json:"tables"`
	Comment   string                 `json:"comment"`
}

type MigrationStatusItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

type MigrationStatus struct {
	TotalMigrations   int `json:"total_migrations"`
	AppliedMigrations int `json:"applied_migrations"`
	PendingMigrations int `json:"pending_migrations"`
}
