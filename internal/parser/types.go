package parser

type Schema struct {
	Tables   []*Table
	Enums    []*Enum
	Keyspace string // CQL keyspace name (ScyllaDB/Cassandra)
	UDTs     []*UDT // CQL user-defined types
}

type UDT struct {
	Name   string
	Fields []*UDTField
}

type UDTField struct {
	Name string
	Type string
}

type Enum struct {
	Name   string
	Values []string
}

type Table struct {
	Name    string
	Columns []*Column
}

type Column struct {
	Name     string
	Type     string
	Nullable bool
}

type Query struct {
	Name         string
	SQL          string
	Cmd          string
	Comment      string
	Params       []*Param
	Columns      []*QueryColumn
	SourceFile   string
	RequiredCols []string // from -- @required: col1, col2 or * for all
}

type Param struct {
	Name     string
	Type     string
	ParamNum int  // the actual $N number in SQL (1-based)
	Nullable bool // true if the corresponding schema column is nullable
}

type QueryColumn struct {
	Name         string
	Type         string
	Table        string
	Nullable     bool
	IsComputed   bool   // true if Name came from an expression, not a bare column ref
	OriginalExpr string // the raw expression (e.g. "preferences->'key'", "RANK() OVER (...)")
}
