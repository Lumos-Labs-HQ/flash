package sql

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/Lumos-Labs-HQ/flash/internal/branch"
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
)

type Server struct {
	mux     *http.ServeMux
	tmpl    *template.Template
	service *Service
	port    int
}

func NewServer(cfg *config.Config, port int) *Server {
	adapter := database.NewAdapter(cfg.Database.Provider)

	dbURL, err := cfg.GetDatabaseURL()
	if err != nil {
		panic(fmt.Sprintf("Failed to get database URL: %v", err))
	}

	if err := adapter.Connect(context.Background(), dbURL); err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	if cfg.Database.URLEnv != "STUDIO_DB_URL" && cfg.MigrationsPath != "" {
		ctx := context.Background()
		branchMgr := branch.NewMetadataManager(cfg.MigrationsPath)
		if store, err := branchMgr.Load(); err == nil {
			if currentBranch := store.GetBranch(store.Current); currentBranch != nil {
				if cfg.Database.Provider == "postgresql" || cfg.Database.Provider == "postgres" {
					query := fmt.Sprintf("SET search_path TO %s, public", currentBranch.Schema)
					adapter.ExecuteQuery(ctx, query)
					fmt.Printf("Studio using schema: %s (branch: %s)\n", currentBranch.Schema, currentBranch.Name)
				}
			}
		}
	}

	mux := http.NewServeMux()
	tmpl := common.ParseTemplates(TemplatesFS)

	server := &Server{
		mux:     mux,
		tmpl:    tmpl,
		service: NewService(adapter, cfg),
		port:    port,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	common.SetupStaticFS(s.mux, StaticFS)

	// UI routes
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /schema", s.handleSchema)
	s.mux.HandleFunc("GET /sql", s.handleSQL)

	// API routes
	s.mux.HandleFunc("GET /api/tables", s.handleGetTables)
	s.mux.HandleFunc("GET /api/tables/{name}", s.handleGetTableData)
	s.mux.HandleFunc("GET /api/schema", s.handleGetSchema)
	s.mux.HandleFunc("POST /api/tables/{name}/save", s.handleSaveChanges)
	s.mux.HandleFunc("POST /api/tables/{name}/add", s.handleAddRow)
	s.mux.HandleFunc("POST /api/tables/{name}/delete", s.handleDeleteRows)
	s.mux.HandleFunc("DELETE /api/tables/{name}/rows/{id}", s.handleDeleteRow)
	s.mux.HandleFunc("POST /api/sql", s.handleExecuteSQL)

	// Schema Editor API
	s.mux.HandleFunc("POST /api/schema/preview", s.handlePreviewSchemaChange)
	s.mux.HandleFunc("POST /api/schema/apply", s.handleApplySchemaChange)
	s.mux.HandleFunc("GET /api/config/check", s.handleCheckConfig)
	s.mux.HandleFunc("PUT /api/tables/{name}/rows/{id}", s.handleUpdateRow)
	s.mux.HandleFunc("POST /api/tables/{name}/rows", s.handleInsertRow)

	// Branch API
	s.mux.HandleFunc("GET /api/branches", s.handleGetBranches)
	s.mux.HandleFunc("POST /api/branches/switch", s.handleSwitchBranch)

	// Editor hints API (cached on client-side)
	s.mux.HandleFunc("GET /api/editor/hints", s.handleGetEditorHints)

	// Export/Import API
	s.mux.HandleFunc("GET /api/export/{type}", s.handleExport)
	s.mux.HandleFunc("POST /api/import", s.handleImport)
}

func (s *Server) Start(openBrowser bool) error {
	return common.StartServer(s.mux, &s.port, "Studio", openBrowser)
}

// UI Handlers
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "index.html", common.Map{"Title": "FlashORM Studio"})
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "schema.html", common.Map{"Title": "FlashORM Studio"})
}

func (s *Server) handleSQL(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "sql.html", common.Map{"Title": "SQL Editor - FlashORM Studio"})
}

// API Handlers
func (s *Server) handleGetTables(w http.ResponseWriter, r *http.Request) {
	tables, err := s.service.GetTables()
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, tables)
}

func (s *Server) handleGetTableData(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	page, _ := strconv.Atoi(common.Query(r, "page", "1"))
	limit, _ := strconv.Atoi(common.Query(r, "limit", "50"))

	// Parse filters from query parameter (JSON encoded)
	var filters []common.Filter
	if filtersJSON := r.URL.Query().Get("filters"); filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			common.JSONError(w, http.StatusBadRequest, "Invalid filters format")
			return
		}
	}

	data, err := s.service.GetTableDataFiltered(tableName, page, limit, filters)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, data)
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	schema, err := s.service.GetSchemaVisualization()
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, schema)
}

func (s *Server) handleSaveChanges(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")

	var req common.SaveRequest
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.SaveChanges(tableName, req.Changes); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Changes saved successfully")
}

func (s *Server) handleAddRow(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")

	var req common.AddRowRequest
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.AddRow(tableName, req.Data); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Row added successfully")
}

func (s *Server) handleDeleteRow(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	rowID := r.PathValue("id")

	if err := s.service.DeleteRow(tableName, rowID); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Row deleted successfully")
}

func (s *Server) handleDeleteRows(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")

	var req struct {
		RowIDs []string `json:"row_ids"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.DeleteRows(tableName, req.RowIDs); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, fmt.Sprintf("Deleted %d row(s) successfully", len(req.RowIDs)))
}

func (s *Server) handleExecuteSQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	data, err := s.service.ExecuteSQL(req.Query)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, data)
}

func (s *Server) handleUpdateRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("name")
	id := r.PathValue("id")

	var data map[string]interface{}
	if err := common.ParseJSON(r, &data); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.UpdateRow(table, id, data); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMap(w, common.Map{"success": true})
}

func (s *Server) handleInsertRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("name")

	var data map[string]interface{}
	if err := common.ParseJSON(r, &data); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.InsertRow(table, data); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMap(w, common.Map{"success": true})
}

func (s *Server) handleGetEditorHints(w http.ResponseWriter, r *http.Request) {
	hints, err := s.service.GetEditorHints()
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, hints)
}
