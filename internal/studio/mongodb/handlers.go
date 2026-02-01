package mongodb

import (
	"encoding/json"
	"strconv"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

// Database Handlers
func (s *Server) handleGetDatabases(c *fiber.Ctx) error {
	databases, err := s.service.GetDatabases()
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, databases)
}

func (s *Server) handleSelectDatabase(c *fiber.Ctx) error {
	dbName := c.Params("name")
	if dbName == "" {
		return common.JSONError(c, 400, "database name is required")
	}

	if err := s.service.SwitchDatabase(dbName); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Database switched successfully")
}

func (s *Server) handleDropDatabase(c *fiber.Ctx) error {
	dbName := c.Params("name")
	if dbName == "" {
		return common.JSONError(c, 400, "database name is required")
	}

	if err := s.service.DropDatabase(dbName); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Database dropped successfully")
}

func (s *Server) handleCreateDatabase(c *fiber.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "invalid request body")
	}
	if req.Name == "" {
		return common.JSONError(c, 400, "database name is required")
	}

	if err := s.service.CreateDatabase(req.Name); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Database created successfully")
}

// Collection Handlers
func (s *Server) handleGetCollections(c *fiber.Ctx) error {
	dbName := c.Query("database")
	if dbName == "" {
		return common.JSONError(c, 400, "database parameter is required")
	}

	collections, err := s.service.GetCollections(dbName)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, collections)
}

func (s *Server) handleGetCollectionData(c *fiber.Ctx) error {
	dbName := c.Query("database")
	if dbName == "" {
		return common.JSONError(c, 400, "database parameter is required")
	}

	if err := s.service.SwitchDatabase(dbName); err != nil {
		return common.JSONError(c, 500, err.Error())
	}

	name := c.Params("name")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	result, err := s.service.GetDocuments(dbName, name, page, limit)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, result)
}

func (s *Server) handleCreateCollection(c *fiber.Ctx) error {
	dbName := c.Query("database")
	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	var req struct {
		Name    string                 `json:"name"`
		Options map[string]interface{} `json:"options"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.CreateCollection(req.Name, req.Options); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Collection created successfully")
}

func (s *Server) handleDropCollection(c *fiber.Ctx) error {
	name := c.Params("name")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	if err := s.service.DropCollection(name); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Collection dropped successfully")
}

// Document Handlers
func (s *Server) handleGetDocuments(c *fiber.Ctx) error {
	dbName := c.Query("database")
	if dbName == "" {
		return common.JSONError(c, 400, "database parameter is required")
	}

	name := c.Params("name")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	filterStr := c.Query("filter", "")

	var filter bson.M
	if filterStr != "" {
		if err := json.Unmarshal([]byte(filterStr), &filter); err != nil {
			return common.JSONError(c, 400, "Invalid filter JSON: "+err.Error())
		}
	}

	result, err := s.service.GetDocumentsWithFilter(dbName, name, page, limit, filter)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, result)
}

func (s *Server) handleInsertDocument(c *fiber.Ctx) error {
	name := c.Params("name")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	var document map[string]interface{}
	if err := c.BodyParser(&document); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	id, err := s.service.InsertDocument(name, document)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{"success": true, "message": "Document inserted successfully", "id": id})
}

func (s *Server) handleUpdateDocument(c *fiber.Ctx) error {
	name := c.Params("name")
	id := c.Params("id")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	var document map[string]interface{}
	if err := c.BodyParser(&document); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.UpdateDocument(name, id, document); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Document updated successfully")
}

func (s *Server) handleDeleteDocument(c *fiber.Ctx) error {
	name := c.Params("name")
	id := c.Params("id")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	if err := s.service.DeleteDocument(name, id); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Document deleted successfully")
}

func (s *Server) handleBulkDeleteDocuments(c *fiber.Ctx) error {
	name := c.Params("name")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.BulkDeleteDocuments(name, req.IDs); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Documents deleted successfully")
}

// Aggregation Handler
func (s *Server) handleAggregate(c *fiber.Ctx) error {
	name := c.Params("name")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	var rawPipeline []interface{}
	if err := c.BodyParser(&rawPipeline); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	pipeline := make([]bson.M, len(rawPipeline))
	for i, stage := range rawPipeline {
		if stageMap, ok := stage.(map[string]interface{}); ok {
			pipeline[i] = bson.M(stageMap)
		}
	}

	result, err := s.service.Aggregate(name, pipeline)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, result)
}

// Index Handlers
func (s *Server) handleGetIndexes(c *fiber.Ctx) error {
	name := c.Params("name")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	indexes, err := s.service.GetIndexes(name)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, indexes)
}

func (s *Server) handleCreateIndex(c *fiber.Ctx) error {
	name := c.Params("name")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	var req struct {
		Keys   map[string]interface{} `json:"keys"`
		Unique bool                   `json:"unique"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.CreateIndex(name, req.Keys, req.Unique); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Index created successfully")
}

func (s *Server) handleDropIndex(c *fiber.Ctx) error {
	name := c.Params("name")
	indexName := c.Params("indexName")
	dbName := c.Query("database")

	if dbName != "" {
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return common.JSONError(c, 500, "Failed to switch database: "+err.Error())
		}
	}

	if err := s.service.DropIndex(name, indexName); err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONMessage(c, "Index dropped successfully")
}

// Query Handler
func (s *Server) handleQuery(c *fiber.Ctx) error {
	name := c.Params("name")

	var req struct {
		Filter string `json:"filter"`
		Limit  int    `json:"limit"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	var filter bson.M
	if err := json.Unmarshal([]byte(req.Filter), &filter); err != nil {
		return common.JSONError(c, 400, "Invalid filter format")
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	result, err := s.service.Query(name, filter, req.Limit)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, result)
}

// Stats Handlers
func (s *Server) handleGetStats(c *fiber.Ctx) error {
	stats, err := s.service.GetStats()
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, stats)
}

func (s *Server) handleGetCollectionStats(c *fiber.Ctx) error {
	name := c.Params("name")

	stats, err := s.service.GetCollectionStats(name)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSON(c, stats)
}
