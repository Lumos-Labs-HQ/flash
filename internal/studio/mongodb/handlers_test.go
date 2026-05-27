package mongodb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
)

// Test the handler pattern: handlers use common.JSONError / common.JSON.
// We verify the HTTP contract without a real MongoDB connection by calling
// the helpers directly through httptest.

func TestHandlerPattern_MissingParam_Returns400(t *testing.T) {
	// Simulate what handleSelectDatabase does when dbName is empty.
	w := httptest.NewRecorder()
	common.JSONError(w, http.StatusBadRequest, "database name is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "database name is required") {
		t.Errorf("body = %q, missing error message", w.Body.String())
	}
}

func TestHandlerPattern_Success_Returns200(t *testing.T) {
	w := httptest.NewRecorder()
	common.JSONMessage(w, "Database switched successfully")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Database switched successfully") {
		t.Errorf("body = %q", w.Body.String())
	}
}

// SchemaChange-like struct used in MongoDB handlers
func TestParseJSON_ValidBody(t *testing.T) {
	body := strings.NewReader(`{"name":"testdb"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/json")

	var req struct{ Name string `json:"name"` }
	if err := common.ParseJSON(r, &req); err != nil {
		t.Fatalf("ParseJSON error: %v", err)
	}
	if req.Name != "testdb" {
		t.Errorf("Name = %q, want testdb", req.Name)
	}
}
