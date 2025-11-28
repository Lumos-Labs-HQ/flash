package redis

import (
	"strconv"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
)

// API Handlers

func (s *Server) handleGetInfo(c *fiber.Ctx) error {
	info, err := s.service.GetInfo()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, info)
}

func (s *Server) handleGetDBSize(c *fiber.Ctx) error {
	size, err := s.service.GetDBSize()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONFiberMap(c, fiber.Map{"size": size})
}

func (s *Server) handleGetKeys(c *fiber.Ctx) error {
	pattern := c.Query("pattern", "*")
	cursor, _ := strconv.ParseUint(c.Query("cursor", "0"), 10, 64)
	count, _ := strconv.ParseInt(c.Query("count", "100"), 10, 64)

	result, err := s.service.GetKeys(pattern, cursor, count)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, result)
}

func (s *Server) handleGetKey(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "key is required")
	}

	keyInfo, err := s.service.GetKey(key)
	if err != nil {
		return common.JSONError(c, fiber.StatusNotFound, err.Error())
	}
	return common.JSON(c, keyInfo)
}

func (s *Server) handleSetKey(c *fiber.Ctx) error {
	var body struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
		Type  string      `json:"type"`
		TTL   int64       `json:"ttl"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Key == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "key is required")
	}

	if body.Type == "" {
		body.Type = "string"
	}

	if err := s.service.SetKey(body.Key, body.Value, body.Type, body.TTL); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONMessage(c, "key created successfully")
}

func (s *Server) handleUpdateKey(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "key is required")
	}

	var body struct {
		Value interface{} `json:"value"`
		Type  string      `json:"type"`
		TTL   int64       `json:"ttl"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Type == "" {
		existingKey, err := s.service.GetKey(key)
		if err != nil {
			return common.JSONError(c, fiber.StatusNotFound, err.Error())
		}
		body.Type = existingKey.Type
	}

	if err := s.service.SetKey(key, body.Value, body.Type, body.TTL); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONMessage(c, "key updated successfully")
}

func (s *Server) handleDeleteKey(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "key is required")
	}

	if err := s.service.DeleteKey(key); err != nil {
		return common.JSONError(c, fiber.StatusNotFound, err.Error())
	}

	return common.JSONMessage(c, "key deleted successfully")
}

func (s *Server) handleBulkDeleteKeys(c *fiber.Ctx) error {
	var body struct {
		Keys []string `json:"keys"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(body.Keys) == 0 {
		return common.JSONError(c, fiber.StatusBadRequest, "keys array is required")
	}

	deleted, err := s.service.BulkDeleteKeys(body.Keys)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{
		"success": true,
		"message": "keys deleted successfully",
		"deleted": deleted,
	})
}

func (s *Server) handleCLI(c *fiber.Ctx) error {
	var body struct {
		Command string `json:"command"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Command == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "command is required")
	}

	result := s.service.ExecuteCLI(body.Command)
	return common.JSON(c, result)
}

func (s *Server) handleGetDatabases(c *fiber.Ctx) error {
	databases, err := s.service.GetDatabases()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, databases)
}

func (s *Server) handleSelectDatabase(c *fiber.Ctx) error {
	dbStr := c.Params("db")
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid database index")
	}

	if err := s.service.SelectDatabase(db); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONMessage(c, "database selected successfully")
}

func (s *Server) handleFlushDB(c *fiber.Ctx) error {
	if err := s.service.FlushDB(); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONMessage(c, "database flushed successfully")
}
