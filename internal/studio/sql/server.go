package sql

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Lumos-Labs-HQ/flash/internal/branch"
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
)

type Server struct {
	app     *fiber.App
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
					fmt.Printf("🔧 Studio using schema: %s (branch: %s)\n", currentBranch.Schema, currentBranch.Name)
				}
			}
		}
	}

	app := common.NewApp(TemplatesFS)

	server := &Server{
		app:     app,
		service: NewService(adapter, cfg),
		port:    port,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	common.SetupStaticFS(s.app, StaticFS)

	// UI routes
	s.app.Get("/", s.handleIndex)
	s.app.Get("/schema", s.handleSchema)
	s.app.Get("/sql", s.handleSQL)

	// API routes
	api := s.app.Group("/api")
	api.Get("/tables", s.handleGetTables)
	api.Get("/tables/:name", s.handleGetTableData)
	api.Get("/schema", s.handleGetSchema)
	api.Post("/tables/:name/save", s.handleSaveChanges)
	api.Post("/tables/:name/add", s.handleAddRow)
	api.Post("/tables/:name/delete", s.handleDeleteRows)
	api.Delete("/tables/:name/rows/:id", s.handleDeleteRow)
	api.Post("/sql", s.handleExecuteSQL)

	// Schema Editor API
	api.Post("/schema/preview", s.handlePreviewSchemaChange)
	api.Post("/schema/apply", s.handleApplySchemaChange)
	api.Get("/config/check", s.handleCheckConfig)
	api.Put("/tables/:name/rows/:id", s.handleUpdateRow)
	api.Post("/tables/:name/rows", s.handleInsertRow)

	// Branch API
	api.Get("/branches", s.handleGetBranches)
	api.Post("/branches/switch", s.handleSwitchBranch)

	// Editor hints API (cached on client-side)
	api.Get("/editor/hints", s.handleGetEditorHints)

	// Export/Import API
	api.Get("/export/:type", s.handleExport)
	api.Post("/import", s.handleImport)
}

func (s *Server) Start(openBrowser bool) error {
	return common.StartServer(s.app, &s.port, "Studio", openBrowser)
}

// UI Handlers
func (s *Server) handleIndex(c *fiber.Ctx) error {
	return c.Render("templates/index", fiber.Map{"Title": "FlashORM Studio"})
}

func (s *Server) handleSchema(c *fiber.Ctx) error {
	return c.Render("templates/schema", fiber.Map{"Title": "FlashORM Studio"})
}

func (s *Server) handleSQL(c *fiber.Ctx) error {
	return c.Render("templates/sql", fiber.Map{"Title": "SQL Editor - FlashORM Studio"})
}

// API Handlers
func (s *Server) handleGetTables(c *fiber.Ctx) error {
	tables, err := s.service.GetTables()
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, tables)
}

func (s *Server) handleGetTableData(c *fiber.Ctx) error {
	tableName := c.Params("name")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	// Parse filters from query parameter (JSON encoded)
	var filters []common.Filter
	if filtersJSON := c.Query("filters"); filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return common.JSONError(c, 400, "Invalid filters format")
		}
	}

	data, err := s.service.GetTableDataFiltered(tableName, page, limit, filters)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, data)
}

func (s *Server) handleGetSchema(c *fiber.Ctx) error {
	schema, err := s.service.GetSchemaVisualization()
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, schema)
}

func (s *Server) handleSaveChanges(c *fiber.Ctx) error {
	tableName := c.Params("name")

	var req common.SaveRequest
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.SaveChanges(tableName, req.Changes); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Changes saved successfully")
}

func (s *Server) handleAddRow(c *fiber.Ctx) error {
	tableName := c.Params("name")

	var req common.AddRowRequest
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.AddRow(tableName, req.Data); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Row added successfully")
}

func (s *Server) handleDeleteRow(c *fiber.Ctx) error {
	tableName := c.Params("name")
	rowID := c.Params("id")

	if err := s.service.DeleteRow(tableName, rowID); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Row deleted successfully")
}

func (s *Server) handleDeleteRows(c *fiber.Ctx) error {
	tableName := c.Params("name")

	var req struct {
		RowIDs []string `json:"row_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.DeleteRows(tableName, req.RowIDs); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, fmt.Sprintf("Deleted %d row(s) successfully", len(req.RowIDs)))
}

func (s *Server) handleExecuteSQL(c *fiber.Ctx) error {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	data, err := s.service.ExecuteSQL(req.Query)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, data)
}

func (s *Server) handleUpdateRow(c *fiber.Ctx) error {
	table := c.Params("name")
	id := c.Params("id")

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.UpdateRow(table, id, data); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONFiberMap(c, fiber.Map{"success": true})
}

func (s *Server) handleInsertRow(c *fiber.Ctx) error {
	table := c.Params("name")

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.InsertRow(table, data); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONFiberMap(c, fiber.Map{"success": true})
}

func (s *Server) handleGetEditorHints(c *fiber.Ctx) error {
	hints, err := s.service.GetEditorHints()
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, hints)
}
