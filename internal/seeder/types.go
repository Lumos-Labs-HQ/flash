package seeder

type SeedConfig struct {
	Count    int            // Default records per table
	Tables   map[string]int // Per-table counts
	Truncate bool           // Clear tables before seeding
	Force    bool           // Skip confirmations and continue on errors
	DryRun   bool           // Print sample data without inserting
	Exclude  []string       // Tables to skip
}

type TableInfo struct {
	Name         string
	Columns      []ColumnInfo
	PrimaryKey   string
	ForeignKeys  []ForeignKey
	Dependencies []string
}

type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	IsPK     bool
	IsFK     bool
	FKTable  string
	FKColumn string
}

type ForeignKey struct {
	Column    string
	RefTable  string
	RefColumn string
}

type GeneratedData struct {
	TableName   string
	Records     []map[string]interface{}
	InsertedIDs map[string][]interface{} // table -> list of IDs
}
