package mongodb

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"go.mongodb.org/mongo-driver/bson"
)

// Database Handlers
func (s *Server) handleGetDatabases(w http.ResponseWriter, r *http.Request) {
	databases, err := s.service.GetDatabases()
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, databases)
}

func (s *Server) handleSelectDatabase(w http.ResponseWriter, r *http.Request) {
	dbName := r.PathValue("name")
	if dbName == "" {
		common.JSONError(w, http.StatusBadRequest, "database name is required")
		return
	}

	if err := s.service.SwitchDatabase(dbName); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Database switched successfully")
}

func (s *Server) handleDropDatabase(w http.ResponseWriter, r *http.Request) {
	dbName := r.PathValue("name")
	if dbName == "" {
		common.JSONError(w, http.StatusBadRequest, "database name is required")
		return
	}

	if err := s.service.DropDatabase(dbName); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Database dropped successfully")
}

func (s *Server) handleCreateDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		common.JSONError(w, http.StatusBadRequest, "database name is required")
		return
	}

	if err := s.service.CreateDatabase(req.Name); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Database created successfully")
}

// Collection Handlers
func (s *Server) handleGetCollections(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database")
	if dbName == "" {
		common.JSONError(w, http.StatusBadRequest, "database parameter is required")
		return
	}

	collections, err := s.service.GetCollections(dbName)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, collections)
}

func (s *Server) handleGetCollectionData(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database")
	if dbName == "" {
		common.JSONError(w, http.StatusBadRequest, "database parameter is required")
		return
	}

	if err := s.service.SwitchDatabase(dbName); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	name := r.PathValue("name")
	page, _ := strconv.Atoi(common.Query(r, "page", "1"))
	limit, _ := strconv.Atoi(common.Query(r, "limit", "50"))

	result, err := s.service.GetDocuments(dbName, name, page, limit)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, result)
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database")
	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	var req struct {
		Name    string                 `json:"name"`
		Options map[string]interface{} `json:"options"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.CreateCollection(req.Name, req.Options); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Collection created successfully")
}

func (s *Server) handleDropCollection(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	if err := s.service.DropCollection(name); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Collection dropped successfully")
}

// Document Handlers
func (s *Server) handleGetDocuments(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database")
	if dbName == "" {
		common.JSONError(w, http.StatusBadRequest, "database parameter is required")
		return
	}

	name := r.PathValue("name")
	page, _ := strconv.Atoi(common.Query(r, "page", "1"))
	limit, _ := strconv.Atoi(common.Query(r, "limit", "50"))
	filterStr := common.Query(r, "filter", "")

	var filter bson.M
	if filterStr != "" {
		if err := json.Unmarshal([]byte(filterStr), &filter); err != nil {
			common.JSONError(w, http.StatusBadRequest, "Invalid filter JSON: "+err.Error())
			return
		}
	}

	result, err := s.service.GetDocumentsWithFilter(dbName, name, page, limit, filter)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, result)
}

func (s *Server) handleInsertDocument(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	var document map[string]interface{}
	if err := common.ParseJSON(r, &document); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	id, err := s.service.InsertDocument(name, document)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.JSONMap(w, common.Map{"success": true, "message": "Document inserted successfully", "id": id})
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	id := r.PathValue("id")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	var document map[string]interface{}
	if err := common.ParseJSON(r, &document); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.UpdateDocument(name, id, document); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Document updated successfully")
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	id := r.PathValue("id")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	if err := s.service.DeleteDocument(name, id); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Document deleted successfully")
}

func (s *Server) handleBulkDeleteDocuments(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.BulkDeleteDocuments(name, req.IDs); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Documents deleted successfully")
}

// Aggregation Handler
func (s *Server) handleAggregate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	var rawPipeline []interface{}
	if err := common.ParseJSON(r, &rawPipeline); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	pipeline := make([]bson.M, len(rawPipeline))
	for i, stage := range rawPipeline {
		if stageMap, ok := stage.(map[string]interface{}); ok {
			pipeline[i] = bson.M(stageMap)
		}
	}

	result, err := s.service.Aggregate(name, pipeline)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, result)
}

// Index Handlers
func (s *Server) handleGetIndexes(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	indexes, err := s.service.GetIndexes(name)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, indexes)
}

func (s *Server) handleCreateIndex(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	var req struct {
		Keys   map[string]interface{} `json:"keys"`
		Unique bool                   `json:"unique"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.CreateIndex(name, req.Keys, req.Unique); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Index created successfully")
}

func (s *Server) handleDropIndex(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	indexName := r.PathValue("indexName")
	dbName := r.URL.Query().Get("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			common.JSONError(w, http.StatusInternalServerError, "Failed to switch database: "+err.Error())
			return
		}
	}

	if err := s.service.DropIndex(name, indexName); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMessage(w, "Index dropped successfully")
}

// Query Handler
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req struct {
		Filter string `json:"filter"`
		Limit  int    `json:"limit"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	var filter bson.M
	if err := json.Unmarshal([]byte(req.Filter), &filter); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid filter format")
		return
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	result, err := s.service.Query(name, filter, req.Limit)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, result)
}

// Stats Handlers
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.service.GetStats()
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, stats)
}

func (s *Server) handleGetCollectionStats(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	stats, err := s.service.GetCollectionStats(name)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSON(w, stats)
}
