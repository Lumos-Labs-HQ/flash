package redis

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	mux           *http.ServeMux
	tmpl          *template.Template
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

	mux := http.NewServeMux()
	tmpl := common.ParseTemplates(TemplatesFS)

	server := &Server{
		mux:           mux,
		tmpl:          tmpl,
		service:       NewService(client),
		port:          port,
		connectionURL: connectionURL,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	common.SetupStaticFS(s.mux, StaticFS)

	// UI Routes
	s.mux.HandleFunc("GET /{$}", s.handleIndex)

	// API Routes - Server info
	s.mux.HandleFunc("GET /api/info", s.handleGetInfo)
	s.mux.HandleFunc("GET /api/info/extended", s.handleGetExtendedInfo)
	s.mux.HandleFunc("GET /api/dbsize", s.handleGetDBSize)

	// Key operations
	s.mux.HandleFunc("GET /api/keys", s.handleGetKeys)
	s.mux.HandleFunc("POST /api/keys", s.handleSetKey)
	s.mux.HandleFunc("POST /api/keys/bulk-delete", s.handleBulkDeleteKeys)
	s.mux.HandleFunc("GET /api/key", s.handleGetKey)
	s.mux.HandleFunc("PUT /api/key", s.handleUpdateKey)
	s.mux.HandleFunc("DELETE /api/key", s.handleDeleteKey)
	s.mux.HandleFunc("POST /api/flush", s.handleFlushDB)

	// CLI
	s.mux.HandleFunc("POST /api/cli", s.handleCLI)

	// Database selection
	s.mux.HandleFunc("GET /api/databases", s.handleGetDatabases)
	s.mux.HandleFunc("POST /api/databases/{db}", s.handleSelectDatabase)

	// Export/Import
	s.mux.HandleFunc("GET /api/export", s.handleExportKeys)
	s.mux.HandleFunc("POST /api/import", s.handleImportKeys)

	// Memory Analysis
	s.mux.HandleFunc("GET /api/memory/stats", s.handleGetMemoryStats)
	s.mux.HandleFunc("GET /api/memory/overview", s.handleGetMemoryOverview)
	s.mux.HandleFunc("GET /api/memory/key", s.handleGetKeyMemory)

	// Slow Log
	s.mux.HandleFunc("GET /api/slowlog", s.handleGetSlowLog)
	s.mux.HandleFunc("DELETE /api/slowlog", s.handleResetSlowLog)
	s.mux.HandleFunc("GET /api/slowlog/len", s.handleGetSlowLogLen)

	// Lua Scripting
	s.mux.HandleFunc("POST /api/script/eval", s.handleExecuteScript)
	s.mux.HandleFunc("POST /api/script/load", s.handleLoadScript)
	s.mux.HandleFunc("POST /api/script/evalsha", s.handleExecuteScriptBySHA)
	s.mux.HandleFunc("DELETE /api/scripts", s.handleFlushScripts)

	// Bulk TTL
	s.mux.HandleFunc("POST /api/bulk-ttl", s.handleBulkSetTTL)

	// Config Management
	s.mux.HandleFunc("GET /api/config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /api/config", s.handleSetConfig)
	s.mux.HandleFunc("POST /api/config/rewrite", s.handleRewriteConfig)
	s.mux.HandleFunc("POST /api/config/resetstat", s.handleResetConfigStats)

	// Cluster/Replication
	s.mux.HandleFunc("GET /api/replication", s.handleGetReplicationInfo)
	s.mux.HandleFunc("GET /api/cluster", s.handleGetClusterInfo)

	// ACL Management
	s.mux.HandleFunc("GET /api/acl/users", s.handleGetACLUsers)
	s.mux.HandleFunc("GET /api/acl/users/{username}", s.handleGetACLUser)
	s.mux.HandleFunc("POST /api/acl/users", s.handleCreateACLUser)
	s.mux.HandleFunc("DELETE /api/acl/users/{username}", s.handleDeleteACLUser)
	s.mux.HandleFunc("GET /api/acl/log", s.handleGetACLLog)
	s.mux.HandleFunc("DELETE /api/acl/log", s.handleResetACLLog)

	// Pub/Sub
	s.mux.HandleFunc("POST /api/pubsub/publish", s.handlePublish)
	s.mux.HandleFunc("GET /api/pubsub/channels", s.handleGetChannels)
}

func (s *Server) Start(openBrowser bool) error {
	return common.StartServer(s.mux, &s.port, "Redis Studio", openBrowser)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	databases := make([]int, 16)
	for i := 0; i < 16; i++ {
		databases[i] = i
	}
	s.tmpl.ExecuteTemplate(w, "index.html", common.Map{
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
