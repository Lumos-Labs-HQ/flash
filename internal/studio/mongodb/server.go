package mongodb

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
)

type Server struct {
	mux           *http.ServeMux
	tmpl          *template.Template
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

	mux := http.NewServeMux()
	tmpl := common.ParseTemplates(TemplatesFS)

	server := &Server{
		mux:           mux,
		tmpl:          tmpl,
		service:       NewService(adapter),
		port:          port,
		connectionURL: dbURL,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	common.SetupStaticFS(s.mux, StaticFS)

	// UI Routes
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /collections", s.handleCollections)
	s.mux.HandleFunc("GET /aggregation", s.handleAggregation)
	s.mux.HandleFunc("GET /indexes", s.handleIndexes)

	// API Routes - Databases
	s.mux.HandleFunc("GET /api/databases", s.handleGetDatabases)
	s.mux.HandleFunc("POST /api/databases", s.handleCreateDatabase)
	s.mux.HandleFunc("POST /api/databases/{name}/select", s.handleSelectDatabase)
	s.mux.HandleFunc("DELETE /api/databases/{name}", s.handleDropDatabase)

	// API Routes - Collections
	s.mux.HandleFunc("GET /api/collections", s.handleGetCollections)
	s.mux.HandleFunc("GET /api/collections/{name}", s.handleGetCollectionData)
	s.mux.HandleFunc("POST /api/collections", s.handleCreateCollection)
	s.mux.HandleFunc("DELETE /api/collections/{name}", s.handleDropCollection)

	// API Routes - Documents
	s.mux.HandleFunc("GET /api/collections/{name}/documents", s.handleGetDocuments)
	s.mux.HandleFunc("POST /api/collections/{name}/documents", s.handleInsertDocument)
	s.mux.HandleFunc("PUT /api/collections/{name}/documents/{id}", s.handleUpdateDocument)
	s.mux.HandleFunc("DELETE /api/collections/{name}/documents/{id}", s.handleDeleteDocument)
	s.mux.HandleFunc("POST /api/collections/{name}/documents/bulk-delete", s.handleBulkDeleteDocuments)

	// API Routes - Aggregation
	s.mux.HandleFunc("POST /api/collections/{name}/aggregate", s.handleAggregate)

	// API Routes - Indexes
	s.mux.HandleFunc("GET /api/collections/{name}/indexes", s.handleGetIndexes)
	s.mux.HandleFunc("POST /api/collections/{name}/indexes", s.handleCreateIndex)
	s.mux.HandleFunc("DELETE /api/collections/{name}/indexes/{indexName}", s.handleDropIndex)

	// API Routes - Query
	s.mux.HandleFunc("POST /api/collections/{name}/query", s.handleQuery)

	// API Routes - Stats
	s.mux.HandleFunc("GET /api/stats", s.handleGetStats)
	s.mux.HandleFunc("GET /api/collections/{name}/stats", s.handleGetCollectionStats)
}

func (s *Server) Start(openBrowser bool) error {
	return common.StartServer(s.mux, &s.port, "MongoDB Studio", openBrowser)
}

// UI Handlers
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "index.html", common.Map{
		"Title":         "FlashORM MongoDB Studio",
		"ConnectionURL": s.connectionURL,
	})
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "collections.html", common.Map{"Title": "Collections - FlashORM MongoDB Studio"})
}

func (s *Server) handleAggregation(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "aggregation.html", common.Map{"Title": "Aggregation - FlashORM MongoDB Studio"})
}

func (s *Server) handleIndexes(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "indexes.html", common.Map{"Title": "Indexes - FlashORM MongoDB Studio"})
}
