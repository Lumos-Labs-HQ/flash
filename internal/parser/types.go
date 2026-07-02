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
	RequiredCols []string    // from -- @required: col1, col2 or * for all
	JsonTypes    []*JsonType // from -- @json annotations
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
	IsComputed   bool      // true if Name came from an expression, not a bare column ref
	OriginalExpr string    // the raw expression (e.g. "preferences->'key'", "RANK() OVER (...)")
	JsonDef      *JsonType // non-nil if this column has a @json type override
}

// JsonType defines a typed JSON structure for a JSONB column.
// Parsed from -- @json annotations or imported from .json files.
type JsonType struct {
	Column string       // column name this type maps to (from "as <name>")
	Name   string       // generated class/struct name (PascalCase of column)
	Fields []*JsonField // fields of the JSON object
}

// JsonField is a single field within a JSON type definition.
type JsonField struct {
	Name     string // JSON key name
	Type     string // type: "string", "int", "float", "boolean", "string[]", "int[]", or nested object name
	Nullable bool   // all JSON fields are nullable by default
}
