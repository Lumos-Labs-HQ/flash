package mongodb

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

// API Handlers

// handleGetDatabases lists all databases
func (s *Server) handleGetDatabases(c *fiber.Ctx) error {
	databases, err := s.service.GetDatabases()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    databases,
	})
}

// handleSelectDatabase switches to a different database
func (s *Server) handleSelectDatabase(c *fiber.Ctx) error {
	dbName := c.Params("name")
	if dbName == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "database name is required",
		})
	}

	if err := s.service.SwitchDatabase(dbName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Database switched successfully",
	})
}

func (s *Server) handleGetCollections(c *fiber.Ctx) error {
	// Check if database parameter is provided
	dbName := c.Query("database")
	if dbName == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "database parameter is required",
		})
	}

	collections, err := s.service.GetCollections(dbName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    collections,
	})
}

func (s *Server) handleGetCollectionData(c *fiber.Ctx) error {
	// Check if database parameter is provided
	dbName := c.Query("database")
	fmt.Printf("[DEBUG Handler] Received database param: '%s' (len=%d)\n", dbName, len(dbName))
	if dbName != "" {
		// Switch to the requested database
		fmt.Printf("[DEBUG Handler] Calling SwitchDatabase with: '%s'\n", dbName)
		if err := s.service.SwitchDatabase(dbName); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		}
	}

	name := c.Params("name")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	if dbName == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "database parameter is required",
		})
	}

	result, err := s.service.GetDocuments(dbName, name, page, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

func (s *Server) handleCreateCollection(c *fiber.Ctx) error {
	var req struct {
		Name    string                 `json:"name"`
		Options map[string]interface{} `json:"options"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	if err := s.service.CreateCollection(req.Name, req.Options); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Collection created successfully",
	})
}

func (s *Server) handleDropCollection(c *fiber.Ctx) error {
	name := c.Params("name")

	if err := s.service.DropCollection(name); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Collection dropped successfully",
	})
}

func (s *Server) handleGetDocuments(c *fiber.Ctx) error {
	dbName := c.Query("database")
	if dbName == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "database parameter is required",
		})
	}

	name := c.Params("name")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	result, err := s.service.GetDocuments(dbName, name, page, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

func (s *Server) handleInsertDocument(c *fiber.Ctx) error {
	name := c.Params("name")

	var document map[string]interface{}
	if err := c.BodyParser(&document); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	id, err := s.service.InsertDocument(name, document)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Document inserted successfully",
		"id":      id,
	})
}

func (s *Server) handleUpdateDocument(c *fiber.Ctx) error {
	name := c.Params("name")
	id := c.Params("id")

	var document map[string]interface{}
	if err := c.BodyParser(&document); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	if err := s.service.UpdateDocument(name, id, document); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Document updated successfully",
	})
}

func (s *Server) handleDeleteDocument(c *fiber.Ctx) error {
	name := c.Params("name")
	id := c.Params("id")

	if err := s.service.DeleteDocument(name, id); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Document deleted successfully",
	})
}

func (s *Server) handleBulkDeleteDocuments(c *fiber.Ctx) error {
	name := c.Params("name")

	var req struct {
		IDs []string `json:"ids"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	if err := s.service.BulkDeleteDocuments(name, req.IDs); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Documents deleted successfully",
	})
}

func (s *Server) handleAggregate(c *fiber.Ctx) error {
	name := c.Params("name")

	// Parse as JSON first to handle dynamic structure
	var rawPipeline []interface{}
	if err := c.BodyParser(&rawPipeline); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	// Convert to []bson.M
	pipeline := make([]bson.M, len(rawPipeline))
	for i, stage := range rawPipeline {
		if stageMap, ok := stage.(map[string]interface{}); ok {
			pipeline[i] = bson.M(stageMap)
		}
	}

	result, err := s.service.Aggregate(name, pipeline)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

func (s *Server) handleGetIndexes(c *fiber.Ctx) error {
	name := c.Params("name")

	indexes, err := s.service.GetIndexes(name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    indexes,
	})
}

func (s *Server) handleCreateIndex(c *fiber.Ctx) error {
	name := c.Params("name")

	var req struct {
		Keys   map[string]interface{} `json:"keys"`
		Unique bool                   `json:"unique"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	if err := s.service.CreateIndex(name, req.Keys, req.Unique); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Index created successfully",
	})
}

func (s *Server) handleDropIndex(c *fiber.Ctx) error {
	name := c.Params("name")
	indexName := c.Params("indexName")

	if err := s.service.DropIndex(name, indexName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Index dropped successfully",
	})
}

func (s *Server) handleQuery(c *fiber.Ctx) error {
	name := c.Params("name")

	var req struct {
		Filter string `json:"filter"`
		Limit  int    `json:"limit"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request",
		})
	}

	// Parse filter as BSON
	var filter bson.M
	if err := json.Unmarshal([]byte(req.Filter), &filter); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid filter format",
		})
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	result, err := s.service.Query(name, filter, req.Limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

func (s *Server) handleGetStats(c *fiber.Ctx) error {
	stats, err := s.service.GetStats()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

func (s *Server) handleGetCollectionStats(c *fiber.Ctx) error {
	name := c.Params("name")

	stats, err := s.service.GetCollectionStats(name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}
