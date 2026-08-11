package mongodb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
)

type Server struct {
	*common.BaseServer
	service *Service
}

func NewServer(cfg *config.Config, port int, host, authToken string) *Server {
	adapter := database.NewAdapter(cfg.Database.Provider)

	dbURL, err := cfg.GetDatabaseURL()
	if err != nil {
		panic(fmt.Sprintf("Failed to get database URL: %v", err))
	}

	if err := adapter.Connect(context.Background(), dbURL); err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	mux := http.NewServeMux()

	server := &Server{
		BaseServer: &common.BaseServer{
			Mux:           mux,
			Tmpl:          common.ParseTemplates(TemplatesFS),
			Port:          port,
			Host:          host,
			AuthToken:     authToken,
			ConnectionURL: dbURL,
			Name:          "MongoDB Studio",
		},
		service: NewService(adapter),
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	common.SetupStaticFS(s.Mux, StaticFS)

	// UI Routes
	s.Mux.HandleFunc("GET /{$}", s.handleIndex)
	s.Mux.HandleFunc("GET /collections", s.handleCollections)
	s.Mux.HandleFunc("GET /aggregation", s.handleAggregation)
	s.Mux.HandleFunc("GET /indexes", s.handleIndexes)

	// API Routes - Databases
	s.Mux.HandleFunc("GET /api/databases", s.handleGetDatabases)
	s.Mux.HandleFunc("POST /api/databases", s.handleCreateDatabase)
	s.Mux.HandleFunc("POST /api/databases/{name}/select", s.handleSelectDatabase)
	s.Mux.HandleFunc("DELETE /api/databases/{name}", s.handleDropDatabase)

	// API Routes - Collections
	s.Mux.HandleFunc("GET /api/collections", s.handleGetCollections)
	s.Mux.HandleFunc("GET /api/collections/{name}", s.handleGetCollectionData)
	s.Mux.HandleFunc("POST /api/collections", s.handleCreateCollection)
	s.Mux.HandleFunc("DELETE /api/collections/{name}", s.handleDropCollection)

	// API Routes - Documents
	s.Mux.HandleFunc("GET /api/collections/{name}/documents", s.handleGetDocuments)
	s.Mux.HandleFunc("POST /api/collections/{name}/documents", s.handleInsertDocument)
	s.Mux.HandleFunc("PUT /api/collections/{name}/documents/{id}", s.handleUpdateDocument)
	s.Mux.HandleFunc("DELETE /api/collections/{name}/documents/{id}", s.handleDeleteDocument)
	s.Mux.HandleFunc("POST /api/collections/{name}/documents/bulk-delete", s.handleBulkDeleteDocuments)

	// API Routes - Aggregation
	s.Mux.HandleFunc("POST /api/collections/{name}/aggregate", s.handleAggregate)

	// API Routes - Schema
	s.Mux.HandleFunc("GET /api/collections/{name}/schema", s.handleGetSchema)

	// API Routes - Indexes
	s.Mux.HandleFunc("GET /api/collections/{name}/indexes", s.handleGetIndexes)
	s.Mux.HandleFunc("POST /api/collections/{name}/indexes", s.handleCreateIndex)
	s.Mux.HandleFunc("DELETE /api/collections/{name}/indexes/{indexName}", s.handleDropIndex)

	// API Routes - Query
	s.Mux.HandleFunc("POST /api/collections/{name}/query", s.handleQuery)

	// API Routes - Stats
	s.Mux.HandleFunc("GET /api/stats", s.handleGetStats)
	s.Mux.HandleFunc("GET /api/collections/{name}/stats", s.handleGetCollectionStats)
}

// UI Handlers
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "index.html", common.Map{
		"Title":         "FlashORM MongoDB Studio",
		"ConnectionURL": s.ConnectionURL,
	})
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "collections.html", common.Map{"Title": "Collections - FlashORM MongoDB Studio"})
}

func (s *Server) handleAggregation(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "aggregation.html", common.Map{"Title": "Aggregation - FlashORM MongoDB Studio"})
}

func (s *Server) handleIndexes(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "indexes.html", common.Map{"Title": "Indexes - FlashORM MongoDB Studio"})
}
