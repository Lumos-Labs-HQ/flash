package mongodb

import (
	"context"
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
)

type Server struct {
	app           *fiber.App
	service       *Service
	port          int
	connectionURL string
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

	app := common.NewApp(TemplatesFS)

	server := &Server{
		app:           app,
		service:       NewService(adapter),
		port:          port,
		connectionURL: dbURL,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	common.SetupStaticFS(s.app, StaticFS)

	// UI Routes
	s.app.Get("/", s.handleIndex)
	s.app.Get("/collections", s.handleCollections)
	s.app.Get("/aggregation", s.handleAggregation)
	s.app.Get("/indexes", s.handleIndexes)

	// API Routes
	api := s.app.Group("/api")

	// Databases
	api.Get("/databases", s.handleGetDatabases)
	api.Post("/databases", s.handleCreateDatabase)
	api.Post("/databases/:name/select", s.handleSelectDatabase)
	api.Delete("/databases/:name", s.handleDropDatabase)

	// Collections
	api.Get("/collections", s.handleGetCollections)
	api.Get("/collections/:name", s.handleGetCollectionData)
	api.Post("/collections", s.handleCreateCollection)
	api.Delete("/collections/:name", s.handleDropCollection)

	// Documents
	api.Get("/collections/:name/documents", s.handleGetDocuments)
	api.Post("/collections/:name/documents", s.handleInsertDocument)
	api.Put("/collections/:name/documents/:id", s.handleUpdateDocument)
	api.Delete("/collections/:name/documents/:id", s.handleDeleteDocument)
	api.Post("/collections/:name/documents/bulk-delete", s.handleBulkDeleteDocuments)

	// Aggregation
	api.Post("/collections/:name/aggregate", s.handleAggregate)

	// Indexes
	api.Get("/collections/:name/indexes", s.handleGetIndexes)
	api.Post("/collections/:name/indexes", s.handleCreateIndex)
	api.Delete("/collections/:name/indexes/:indexName", s.handleDropIndex)

	// Query
	api.Post("/collections/:name/query", s.handleQuery)

	// Stats
	api.Get("/stats", s.handleGetStats)
	api.Get("/collections/:name/stats", s.handleGetCollectionStats)
}

func (s *Server) Start(openBrowser bool) error {
	return common.StartServer(s.app, &s.port, "MongoDB Studio", openBrowser)
}

// UI Handlers
func (s *Server) handleIndex(c *fiber.Ctx) error {
	return c.Render("templates/index", fiber.Map{
		"Title":         "FlashORM MongoDB Studio",
		"ConnectionURL": s.connectionURL,
	})
}

func (s *Server) handleCollections(c *fiber.Ctx) error {
	return c.Render("templates/collections", fiber.Map{"Title": "Collections - FlashORM MongoDB Studio"})
}

func (s *Server) handleAggregation(c *fiber.Ctx) error {
	return c.Render("templates/aggregation", fiber.Map{"Title": "Aggregation - FlashORM MongoDB Studio"})
}

func (s *Server) handleIndexes(c *fiber.Ctx) error {
	return c.Render("templates/indexes", fiber.Map{"Title": "Indexes - FlashORM MongoDB Studio"})
}
