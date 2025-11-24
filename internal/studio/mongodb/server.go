package mongodb

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
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

	engine := html.NewFileSystem(http.FS(TemplatesFS), ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

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
	// Serve static files
	staticFS, _ := fs.Sub(StaticFS, "static")
	s.app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(staticFS),
	}))

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
	port := s.findAvailablePort(s.port)
	if port != s.port {
		fmt.Printf("⚠️  Port %d is in use, using port %d instead\n", s.port, port)
		s.port = port
	}

	url := fmt.Sprintf("http://localhost:%d", s.port)

	fmt.Printf("🚀 FlashORM MongoDB Studio starting on %s\n", url)

	if openBrowser {
		go s.openBrowser(url)
	}

	return s.app.Listen(fmt.Sprintf(":%d", s.port))
}

func (s *Server) findAvailablePort(startPort int) int {
	port := startPort
	maxAttempts := 10

	for i := 0; i < maxAttempts; i++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
		port++
	}

	return startPort
}

func (s *Server) openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// UI Handlers
func (s *Server) handleIndex(c *fiber.Ctx) error {
	return c.Render("templates/index", fiber.Map{
		"Title":         "FlashORM MongoDB Studio",
		"ConnectionURL": s.connectionURL,
	})
}

func (s *Server) handleCollections(c *fiber.Ctx) error {
	return c.Render("templates/collections", fiber.Map{
		"Title": "Collections - FlashORM MongoDB Studio",
	})
}

func (s *Server) handleAggregation(c *fiber.Ctx) error {
	return c.Render("templates/aggregation", fiber.Map{
		"Title": "Aggregation - FlashORM MongoDB Studio",
	})
}

func (s *Server) handleIndexes(c *fiber.Ctx) error {
	return c.Render("templates/indexes", fiber.Map{
		"Title": "Indexes - FlashORM MongoDB Studio",
	})
}
