package redis

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	app           *fiber.App
	service       *Service
	port          int
	connectionURL string
}

func NewServer(connectionURL string, port int) *Server {
	opts, err := redis.ParseURL(connectionURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse Redis URL: %v", err))
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	engine := html.NewFileSystem(http.FS(TemplatesFS), ".html")

	app := fiber.New(fiber.Config{Views: engine})

	server := &Server{
		app:           app,
		service:       NewService(client),
		port:          port,
		connectionURL: connectionURL,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	staticFS, _ := fs.Sub(StaticFS, "static")
	s.app.Use("/static", filesystem.New(filesystem.Config{
		Root: http.FS(staticFS),
	}))

	// UI Routes
	s.app.Get("/", s.handleIndex)

	// API Routes
	api := s.app.Group("/api")

	// Server info
	api.Get("/info", s.handleGetInfo)
	api.Get("/info/extended", s.handleGetExtendedInfo)
	api.Get("/dbsize", s.handleGetDBSize)

	// Key operations
	api.Get("/keys", s.handleGetKeys)
	api.Post("/keys", s.handleSetKey)
	api.Post("/keys/bulk-delete", s.handleBulkDeleteKeys)
	api.Get("/key", s.handleGetKey)
	api.Put("/key", s.handleUpdateKey)
	api.Delete("/key", s.handleDeleteKey)
	api.Post("/flush", s.handleFlushDB)

	// CLI
	api.Post("/cli", s.handleCLI)

	// Database selection
	api.Get("/databases", s.handleGetDatabases)
	api.Post("/databases/:db", s.handleSelectDatabase)

	// Export/Import
	api.Get("/export", s.handleExportKeys)
	api.Post("/import", s.handleImportKeys)

	// Memory Analysis
	api.Get("/memory/stats", s.handleGetMemoryStats)
	api.Get("/memory/overview", s.handleGetMemoryOverview)
	api.Get("/memory/key", s.handleGetKeyMemory)

	// Slow Log
	api.Get("/slowlog", s.handleGetSlowLog)
	api.Delete("/slowlog", s.handleResetSlowLog)
	api.Get("/slowlog/len", s.handleGetSlowLogLen)

	// Lua Scripting
	api.Post("/script/eval", s.handleExecuteScript)
	api.Post("/script/load", s.handleLoadScript)
	api.Post("/script/evalsha", s.handleExecuteScriptBySHA)
	api.Delete("/scripts", s.handleFlushScripts)

	// Bulk TTL
	api.Post("/bulk-ttl", s.handleBulkSetTTL)

	// Config Management
	api.Get("/config", s.handleGetConfig)
	api.Put("/config", s.handleSetConfig)
	api.Post("/config/rewrite", s.handleRewriteConfig)
	api.Post("/config/resetstat", s.handleResetConfigStats)

	// Cluster/Replication
	api.Get("/replication", s.handleGetReplicationInfo)
	api.Get("/cluster", s.handleGetClusterInfo)

	// ACL Management
	api.Get("/acl/users", s.handleGetACLUsers)
	api.Get("/acl/users/:username", s.handleGetACLUser)
	api.Post("/acl/users", s.handleCreateACLUser)
	api.Delete("/acl/users/:username", s.handleDeleteACLUser)
	api.Get("/acl/log", s.handleGetACLLog)
	api.Delete("/acl/log", s.handleResetACLLog)

	// Pub/Sub
	api.Post("/pubsub/publish", s.handlePublish)
	api.Get("/pubsub/channels", s.handleGetChannels)
}

func (s *Server) Start(openBrowser bool) error {
	port := common.FindAvailablePort(s.port)
	if port != s.port {
		fmt.Printf("⚠️  Port %d is in use, using port %d instead\n", s.port, port)
		s.port = port
	}

	url := fmt.Sprintf("http://localhost:%d", s.port)
	fmt.Printf("🚀 FlashORM Redis Studio starting on %s\n", url)

	if openBrowser {
		go common.OpenBrowser(url)
	}

	return s.app.Listen(fmt.Sprintf(":%d", s.port))
}

func (s *Server) handleIndex(c *fiber.Ctx) error {
	databases := make([]int, 16)
	for i := 0; i < 16; i++ {
		databases[i] = i
	}
	return c.Render("templates/index", fiber.Map{
		"Title":     "FlashORM Redis Studio",
		"Host":      maskRedisURL(s.connectionURL),
		"Databases": databases,
	})
}

func maskRedisURL(url string) string {
	if len(url) < 20 {
		return "redis://***"
	}
	return url[:10] + "***" + url[len(url)-5:]
}
