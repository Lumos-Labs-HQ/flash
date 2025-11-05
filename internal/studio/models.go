package studio

type TableInfo struct {
	Name     string `json:"name"`
	RowCount int    `json:"row_count"`
}

type ColumnInfo struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Nullable         bool   `json:"nullable"`
	PrimaryKey       bool   `json:"primary_key"`
	Default          string `json:"default,omitempty"`
	AutoIncrement    bool   `json:"auto_increment,omitempty"` // NEW: Indicates auto-increment columns
	ForeignKeyTable  string `json:"foreign_key_table,omitempty"`
	ForeignKeyColumn string `json:"foreign_key_column,omitempty"`
}

type TableData struct {
	Columns []ColumnInfo     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
}

type RowChange struct {
	RowID  string `json:"row_id"`
	Column string `json:"column"`
	Value  any    `json:"value"`
	Action string `json:"action"` // "update", "insert", "delete"
}

type SaveRequest struct {
	Changes []RowChange `json:"changes"`
}

type AddRowRequest struct {
	Data map[string]any `json:"data"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
