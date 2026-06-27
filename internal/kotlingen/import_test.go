package kotlingen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func TestImportNullableOnly(t *testing.T) {
	// Schema where UUID only appears as nullable (from JOIN)
	g := &Generator{
		Config: &config.Config{
			Database: config.Database{Provider: "postgresql"},
			Gen:      config.Gen{Kotlin: config.KotlinGen{Enabled: true, Out: "/tmp/test"}},
		},
		schema: &parser.Schema{
			Tables: []*parser.Table{{
				Name: "logs",
				Columns: []*parser.Column{
					{Name: "id", Type: "SERIAL"},
					{Name: "message", Type: "TEXT"},
				},
			}},
		},
	}

	// Query with only nullable UUID column (from JOIN) — no UUID in params
	queries := []*parser.Query{{
		Name: "GetLogs",
		Cmd:  ":many",
		SQL:  "SELECT l.id, l.message, u.id AS user_uuid FROM logs l LEFT JOIN users u ON u.id = l.user_id",
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "SERIAL"},
			{Name: "message", Type: "TEXT"},
			{Name: "user_uuid", Type: "UUID", Nullable: true},
		},
	}}

	// Simulate the import check logic from incremental.go
	needsUUID := false
	needsLDT := false
	for _, q := range queries {
		for _, col := range q.Columns {
			kt := g.sqlTypeToKotlin(col.Type, false)
			if strings.Contains(kt, "UUID") {
				needsUUID = true
			}
			if strings.Contains(kt, "LocalDateTime") {
				needsLDT = true
			}
		}
	}

	if !needsUUID {
		t.Error("expected needsUUID=true for query with UUID column")
	}
	_ = needsLDT
}
