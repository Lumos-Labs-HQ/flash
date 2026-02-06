package sql

import (
	"os"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) handlePreviewSchemaChange(c *fiber.Ctx) error {
	var change SchemaChange
	if err := c.BodyParser(&change); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	preview, err := s.service.PreviewSchemaChange(&change)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return c.JSON(preview)
}

func (s *Server) handleApplySchemaChange(c *fiber.Ctx) error {
	var change SchemaChange
	if err := c.BodyParser(&change); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	configPath := ""
	if _, err := os.Stat("./flash.config.json"); err == nil {
		configPath = "./flash.config.json"
	}

	if err := s.service.ApplySchemaChange(&change, configPath); err != nil {
		return common.JSONError(c, 500, err.Error())
	}

	message := "Schema change applied successfully"
	if configPath == "" {
		message += " (migration files not created - no config found)"
	}

	return common.JSONFiberMap(c, fiber.Map{"success": true, "message": message})
}

func (s *Server) handleCheckConfig(c *fiber.Ctx) error {
	exists := false
	if _, err := os.Stat("./flash.config.json"); err == nil {
		exists = true
	}
	return common.JSONFiberMap(c, fiber.Map{"exists": exists})
}

func (s *Server) handleGetBranches(c *fiber.Ctx) error {
	branches, current, err := s.service.GetBranches()
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}
	return common.JSONFiberMap(c, fiber.Map{"branches": branches, "current": current})
}

func (s *Server) handleSwitchBranch(c *fiber.Ctx) error {
	var req struct {
		Branch string `json:"branch"`
	}
	if err := c.BodyParser(&req); err != nil {
		return common.JSONError(c, 400, "Invalid request")
	}

	if err := s.service.SwitchBranch(req.Branch); err != nil {
		return common.JSONError(c, 500, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{
		"success": true,
		"message": "Branch switched. Please refresh the page to see changes.",
	})
}

func (s *Server) handleExport(c *fiber.Ctx) error {
	exportTypeStr := c.Params("type")

	var exportType common.ExportType
	switch exportTypeStr {
	case "schema_only":
		exportType = common.ExportSchemaOnly
	case "data_only":
		exportType = common.ExportDataOnly
	case "complete":
		exportType = common.ExportComplete
	default:
		return common.JSONError(c, 400, "Invalid export type. Use: schema_only, data_only, or complete")
	}

	data, err := s.service.ExportDatabase(exportType)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}

	return common.JSON(c, data)
}

func (s *Server) handleImport(c *fiber.Ctx) error {
	var importData common.ExportData
	if err := c.BodyParser(&importData); err != nil {
		return common.JSONError(c, 400, "Invalid import data format")
	}

	if importData.Version == "" || len(importData.Tables) == 0 {
		return common.JSONError(c, 400, "Invalid import data: missing version or tables")
	}

	result, err := s.service.ImportDatabase(&importData)
	if err != nil {
		return common.JSONError(c, 500, err.Error())
	}

	return common.JSONFiberMap(c, fiber.Map{
		"success": true,
		"message": "Import completed",
		"result":  result,
	})
}
