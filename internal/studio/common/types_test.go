package common

import (
	"testing"
)

// ── Type struct zero-value sanity ─────────────────────────────────────────────

func TestResponse_ZeroValue(t *testing.T) {
	var r Response
	if r.Success || r.Message != "" || r.Data != nil {
		t.Errorf("zero Response = %+v", r)
	}
}

func TestTableInfo_Fields(t *testing.T) {
	ti := TableInfo{Name: "users", RowCount: 42}
	if ti.Name != "users" || ti.RowCount != 42 {
		t.Errorf("TableInfo = %+v", ti)
	}
}

func TestColumnInfo_Fields(t *testing.T) {
	ci := ColumnInfo{
		Name:       "id",
		Type:       "INTEGER",
		PrimaryKey: true,
		Nullable:   false,
	}
	if ci.Name != "id" || !ci.PrimaryKey || ci.Nullable {
		t.Errorf("ColumnInfo = %+v", ci)
	}
}

func TestTableData_Fields(t *testing.T) {
	td := TableData{
		Columns: []ColumnInfo{{Name: "id"}},
		Rows:    []map[string]any{{"id": 1}},
		Total:   1,
		Page:    1,
		Limit:   10,
	}
	if len(td.Columns) != 1 || len(td.Rows) != 1 || td.Total != 1 {
		t.Errorf("TableData = %+v", td)
	}
}

func TestExportData_Fields(t *testing.T) {
	ed := ExportData{
		Version:          "1.0",
		DatabaseProvider: "postgresql",
		ExportType:       ExportComplete,
		Tables:           []ExportTable{{Name: "users"}},
	}
	if ed.Version != "1.0" || ed.ExportType != ExportComplete || len(ed.Tables) != 1 {
		t.Errorf("ExportData = %+v", ed)
	}
}

func TestExportTypeConstants(t *testing.T) {
	if ExportSchemaOnly != "schema_only" {
		t.Errorf("ExportSchemaOnly = %q", ExportSchemaOnly)
	}
	if ExportDataOnly != "data_only" {
		t.Errorf("ExportDataOnly = %q", ExportDataOnly)
	}
	if ExportComplete != "complete" {
		t.Errorf("ExportComplete = %q", ExportComplete)
	}
}

func TestFilter_Fields(t *testing.T) {
	f := Filter{Logic: "AND", Column: "email", Operator: "=", Value: "a@b.com"}
	if f.Logic != "AND" || f.Column != "email" {
		t.Errorf("Filter = %+v", f)
	}
}

func TestImportResult_Fields(t *testing.T) {
	ir := ImportResult{
		TablesCreated: []string{"users"},
		RowsInserted:  10,
	}
	if len(ir.TablesCreated) != 1 || ir.RowsInserted != 10 {
		t.Errorf("ImportResult = %+v", ir)
	}
}
