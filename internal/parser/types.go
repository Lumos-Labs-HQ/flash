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
