package migrator

import (
	"fmt"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func TestGenerateSQLFromDiff_AddColumnWithTypeEquivalentDiff_SQLite(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{
		adapter:  adapter,
		provider: "sqlite",
	}

	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{
			{
				Name: "users",
				OldTable: types.SchemaTable{
					Name: "users",
					Columns: []types.SchemaColumn{
						{Name: "id", Type: "INTEGER", IsPrimary: true, Nullable: true},
						{Name: "name", Type: "TEXT", Nullable: false},
					},
				},
				NewTable: types.SchemaTable{
					Name: "users",
					Columns: []types.SchemaColumn{
						{Name: "id", Type: "INTEGER", IsPrimary: true, Nullable: true},
						{Name: "name", Type: "VARCHAR(255)", Nullable: false},
						{Name: "is_active", Type: "BOOLEAN", Nullable: false, Default: "1"},
					},
				},
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:    "name",
						OldType: "TEXT",
						NewType: "VARCHAR(255)",
						OldColumn: types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false},
						NewColumn: types.SchemaColumn{Name: "name", Type: "VARCHAR(255)", Nullable: false},
					},
				},
				NewColumns: []types.SchemaColumn{
					{Name: "is_active", Type: "BOOLEAN", Nullable: false, Default: "1"},
				},
			},
		},
	}

	sql, _ := m.generateSQLFromDiff(diff, "test")
	fmt.Println("=== GENERATED SQL ===")
	fmt.Println(sql)
	fmt.Println("=== END SQL ===")
}
