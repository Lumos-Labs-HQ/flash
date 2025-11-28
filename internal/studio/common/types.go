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
