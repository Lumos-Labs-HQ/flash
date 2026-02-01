package redis

import (
	"strconv"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
)

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

func (s *Server) handleExportKeys(c *fiber.Ctx) error {
	pattern := c.Query("pattern", "*")

	keys, err := s.service.ExportKeys(pattern)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSON(c, fiber.Map{
		"keys":  keys,
		"count": len(keys),
	})
}

func (s *Server) handleImportKeys(c *fiber.Ctx) error {
	var body struct {
		Keys      []ExportedKey `json:"keys"`
		Overwrite bool          `json:"overwrite"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	imported, skipped, err := s.service.ImportKeys(body.Keys, body.Overwrite)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{
		"success":  true,
		"imported": imported,
		"skipped":  skipped,
	})
}

func (s *Server) handleGetMemoryStats(c *fiber.Ctx) error {
	pattern := c.Query("pattern", "*")
	limit, _ := strconv.Atoi(c.Query("limit", "100"))

	memoryInfos, typeStats, err := s.service.GetMemoryStats(pattern, limit)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSON(c, fiber.Map{
		"keys":       memoryInfos,
		"type_stats": typeStats,
	})
}

func (s *Server) handleGetMemoryOverview(c *fiber.Ctx) error {
	overview, err := s.service.GetMemoryOverview()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, overview)
}

func (s *Server) handleGetKeyMemory(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "key is required")
	}

	info, err := s.service.GetKeyMemory(key)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, info)
}

func (s *Server) handleGetSlowLog(c *fiber.Ctx) error {
	count, _ := strconv.Atoi(c.Query("count", "50"))

	entries, err := s.service.GetSlowLog(count)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, entries)
}

func (s *Server) handleResetSlowLog(c *fiber.Ctx) error {
	if err := s.service.ResetSlowLog(); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONMessage(c, "slow log reset successfully")
}

func (s *Server) handleGetSlowLogLen(c *fiber.Ctx) error {
	length, err := s.service.GetSlowLogLen()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONFiberMap(c, fiber.Map{"length": length})
}

func (s *Server) handleExecuteScript(c *fiber.Ctx) error {
	var body struct {
		Script string        `json:"script"`
		Keys   []string      `json:"keys"`
		Args   []interface{} `json:"args"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Script == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "script is required")
	}

	if body.Keys == nil {
		body.Keys = []string{}
	}
	if body.Args == nil {
		body.Args = []interface{}{}
	}

	result := s.service.ExecuteScript(body.Script, body.Keys, body.Args)
	return common.JSON(c, result)
}

func (s *Server) handleLoadScript(c *fiber.Ctx) error {
	var body struct {
		Script string `json:"script"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	sha, err := s.service.LoadScript(body.Script)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{"sha": sha})
}

func (s *Server) handleExecuteScriptBySHA(c *fiber.Ctx) error {
	var body struct {
		SHA  string        `json:"sha"`
		Keys []string      `json:"keys"`
		Args []interface{} `json:"args"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.SHA == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "sha is required")
	}

	if body.Keys == nil {
		body.Keys = []string{}
	}
	if body.Args == nil {
		body.Args = []interface{}{}
	}

	result := s.service.ExecuteScriptBySHA(body.SHA, body.Keys, body.Args)
	return common.JSON(c, result)
}

func (s *Server) handleFlushScripts(c *fiber.Ctx) error {
	if err := s.service.FlushScripts(); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONMessage(c, "scripts flushed successfully")
}

func (s *Server) handleBulkSetTTL(c *fiber.Ctx) error {
	var body struct {
		Pattern string `json:"pattern"`
		TTL     int64  `json:"ttl"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Pattern == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "pattern is required")
	}

	updated, err := s.service.BulkSetTTL(body.Pattern, body.TTL)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{
		"success": true,
		"updated": updated,
	})
}

func (s *Server) handleGetConfig(c *fiber.Ctx) error {
	pattern := c.Query("pattern", "*")

	config, err := s.service.GetConfig(pattern)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, config)
}

func (s *Server) handleSetConfig(c *fiber.Ctx) error {
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Key == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "key is required")
	}

	if err := s.service.SetConfig(body.Key, body.Value); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONMessage(c, "configuration updated successfully")
}

func (s *Server) handleRewriteConfig(c *fiber.Ctx) error {
	if err := s.service.RewriteConfig(); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONMessage(c, "configuration rewritten successfully")
}

func (s *Server) handleResetConfigStats(c *fiber.Ctx) error {
	if err := s.service.ResetConfigStats(); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONMessage(c, "statistics reset successfully")
}

func (s *Server) handleGetReplicationInfo(c *fiber.Ctx) error {
	info, err := s.service.GetReplicationInfo()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, info)
}

func (s *Server) handleGetClusterInfo(c *fiber.Ctx) error {
	info, err := s.service.GetClusterInfo()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, info)
}

func (s *Server) handleGetACLUsers(c *fiber.Ctx) error {
	users, err := s.service.GetACLUsers()
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, users)
}

func (s *Server) handleGetACLUser(c *fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "username is required")
	}

	user, err := s.service.GetACLUser(username)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, user)
}

func (s *Server) handleCreateACLUser(c *fiber.Ctx) error {
	var body struct {
		Username string   `json:"username"`
		Rules    []string `json:"rules"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Username == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "username is required")
	}

	if err := s.service.CreateACLUser(body.Username, body.Rules); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONMessage(c, "user created successfully")
}

func (s *Server) handleDeleteACLUser(c *fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "username is required")
	}

	if err := s.service.DeleteACLUser(username); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONMessage(c, "user deleted successfully")
}

func (s *Server) handleGetACLLog(c *fiber.Ctx) error {
	count, _ := strconv.Atoi(c.Query("count", "10"))

	logs, err := s.service.GetACLLog(count)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSON(c, logs)
}

func (s *Server) handleResetACLLog(c *fiber.Ctx) error {
	if err := s.service.ResetACLLog(); err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return common.JSONMessage(c, "ACL log reset successfully")
}

func (s *Server) handlePublish(c *fiber.Ctx) error {
	var body struct {
		Channel string `json:"channel"`
		Message string `json:"message"`
	}

	if err := c.BodyParser(&body); err != nil {
		return common.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Channel == "" {
		return common.JSONError(c, fiber.StatusBadRequest, "channel is required")
	}

	receivers, err := s.service.Publish(body.Channel, body.Message)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{
		"success":   true,
		"receivers": receivers,
	})
}

func (s *Server) handleGetChannels(c *fiber.Ctx) error {
	pattern := c.Query("pattern", "*")

	channels, err := s.service.GetPubSubChannels(pattern)
	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Get subscriber counts
	var numSub map[string]int64
	if len(channels) > 0 {
		numSub, _ = s.service.GetPubSubNumSub(channels)
	}

	result := make([]fiber.Map, 0, len(channels))
	for _, ch := range channels {
		count := int64(0)
		if numSub != nil {
			count = numSub[ch]
		}
		result = append(result, fiber.Map{
			"channel":     ch,
			"subscribers": count,
		})
	}

	numPat, _ := s.service.GetPubSubNumPat()

	return common.JSON(c, fiber.Map{
		"channels":             result,
		"pattern_subscribers":  numPat,
	})
}

func (s *Server) handleGetExtendedInfo(c *fiber.Ctx) error {
	section := c.Query("section", "all")

	var info string
	var err error

	if section == "all" {
		info, err = s.service.client.Info(s.service.ctx).Result()
	} else {
		info, err = s.service.client.Info(s.service.ctx, section).Result()
	}

	if err != nil {
		return common.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Parse into sections
	result := make(map[string]map[string]string)
	currentSection := "default"

	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			currentSection = strings.TrimPrefix(line, "# ")
			result[currentSection] = make(map[string]string)
		} else if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if result[currentSection] == nil {
					result[currentSection] = make(map[string]string)
				}
				result[currentSection][parts[0]] = parts[1]
			}
		}
	}

	return common.JSON(c, result)
}
