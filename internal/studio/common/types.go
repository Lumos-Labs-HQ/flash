package common

// TableInfo represents basic table information
type TableInfo struct {
	Name     string `json:"name"`
	RowCount int    `json:"row_count"`
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Nullable         bool   `json:"nullable"`
	PrimaryKey       bool   `json:"primary_key"`
	Default          string `json:"default,omitempty"`
	AutoIncrement    bool   `json:"auto_increment,omitempty"`
	ForeignKeyTable  string `json:"foreign_key_table,omitempty"`
	ForeignKeyColumn string `json:"foreign_key_column,omitempty"`
}

// TableData represents paginated table data
type TableData struct {
	Columns []ColumnInfo     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
}

// RowChange represents a single row modification
type RowChange struct {
	RowID  string `json:"row_id"`
	Column string `json:"column"`
	Value  any    `json:"value"`
	Action string `json:"action"`
}

// SaveRequest represents a batch save request
type SaveRequest struct {
	Changes []RowChange `json:"changes"`
}

// AddRowRequest represents a new row insertion request
type AddRowRequest struct {
	Data map[string]any `json:"data"`
}

// Response is a standard API response
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// BranchInfo represents branch metadata
type BranchInfo struct {
	Name      string `json:"name"`
	Parent    string `json:"parent"`
	Schema    string `json:"schema"`
	CreatedAt string `json:"created_at"`
	IsDefault bool   `json:"is_default"`
}

// Filter represents a single filter condition for server-side filtering
type Filter struct {
	Logic    string `json:"logic"`    
	Column   string `json:"column"`   
	Operator string `json:"operator"` 
	Value    string `json:"value"`    
}

// ExportType defines the type of export
type ExportType string

const (
	ExportSchemaOnly ExportType = "schema_only"
	ExportDataOnly   ExportType = "data_only"
	ExportComplete   ExportType = "complete"
)

// ExportEnumType represents a PostgreSQL ENUM type
type ExportEnumType struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// ExportColumn represents a column in the export schema
type ExportColumn struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Nullable         bool   `json:"nullable"`
	PrimaryKey       bool   `json:"primary_key"`
	Default          string `json:"default,omitempty"`
	AutoIncrement    bool   `json:"auto_increment,omitempty"`
	Unique           bool   `json:"unique,omitempty"`
	ForeignKeyTable  string `json:"foreign_key_table,omitempty"`
	ForeignKeyColumn string `json:"foreign_key_column,omitempty"`
}

// ExportIndex represents an index in the export schema
type ExportIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// ExportTableSchema represents the schema of a table
type ExportTableSchema struct {
	Columns []ExportColumn `json:"columns"`
	Indexes []ExportIndex  `json:"indexes,omitempty"`
}

// ExportTable represents a single table in the export
type ExportTable struct {
	Name   string             `json:"name"`
	Schema *ExportTableSchema `json:"schema,omitempty"`
	Data   []map[string]any   `json:"data,omitempty"`
}

// ExportData represents the complete export structure
type ExportData struct {
	Version          string           `json:"version"`
	ExportedAt       string           `json:"exported_at"`
	DatabaseProvider string           `json:"database_provider"`
	ExportType       ExportType       `json:"export_type"`
	EnumTypes        []ExportEnumType `json:"enum_types,omitempty"`
	Tables           []ExportTable    `json:"tables"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	EnumTypesCreated []string `json:"enum_types_created,omitempty"`
	TablesCreated    []string `json:"tables_created"`
	TablesUpdated    []string `json:"tables_updated"`
	ColumnsAdded     int      `json:"columns_added"`
	RowsInserted     int      `json:"rows_inserted"`
	RowsUpdated      int      `json:"rows_updated"`
	Errors           []string `json:"errors,omitempty"`
}
