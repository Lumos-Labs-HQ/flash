package sql

import (
	"net/http"
	"os"

	"github.com/Lumos-Labs-HQ/flash/internal/studio/common"
)

func (s *Server) handlePreviewSchemaChange(w http.ResponseWriter, r *http.Request) {
	var change SchemaChange
	if err := common.ParseJSON(r, &change); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	preview, err := s.service.PreviewSchemaChange(&change)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONRaw(w, preview)
}

func (s *Server) handleApplySchemaChange(w http.ResponseWriter, r *http.Request) {
	var change SchemaChange
	if err := common.ParseJSON(r, &change); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	configPath := ""
	if _, err := os.Stat("./flash.config.json"); err == nil {
		configPath = "./flash.config.json"
	}

	if err := s.service.ApplySchemaChange(&change, configPath); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	message := "Schema change applied successfully"
	if configPath == "" {
		message += " (migration files not created - no config found)"
	}

	common.JSONMap(w, common.Map{"success": true, "message": message})
}

func (s *Server) handleCheckConfig(w http.ResponseWriter, r *http.Request) {
	exists := false
	if _, err := os.Stat("./flash.config.json"); err == nil {
		exists = true
	}
	common.JSONMap(w, common.Map{"exists": exists})
}

func (s *Server) handleGetBranches(w http.ResponseWriter, r *http.Request) {
	branches, current, err := s.service.GetBranches()
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.JSONMap(w, common.Map{"branches": branches, "current": current})
}

func (s *Server) handleSwitchBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Branch string `json:"branch"`
	}
	if err := common.ParseJSON(r, &req); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := s.service.SwitchBranch(req.Branch); err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.JSONMap(w, common.Map{
		"success": true,
		"message": "Branch switched. Please refresh the page to see changes.",
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	exportTypeStr := r.PathValue("type")

	var exportType common.ExportType
	switch exportTypeStr {
	case "schema_only":
		exportType = common.ExportSchemaOnly
	case "data_only":
		exportType = common.ExportDataOnly
	case "complete":
		exportType = common.ExportComplete
	default:
		common.JSONError(w, http.StatusBadRequest, "Invalid export type. Use: schema_only, data_only, or complete")
		return
	}

	data, err := s.service.ExportDatabase(exportType)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.JSON(w, data)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	var importData common.ExportData
	if err := common.ParseJSON(r, &importData); err != nil {
		common.JSONError(w, http.StatusBadRequest, "Invalid import data format")
		return
	}

	if importData.Version == "" || len(importData.Tables) == 0 {
		common.JSONError(w, http.StatusBadRequest, "Invalid import data: missing version or tables")
		return
	}

	result, err := s.service.ImportDatabase(&importData)
	if err != nil {
		common.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.JSONMap(w, common.Map{
		"success": true,
		"message": "Import completed",
		"result":  result,
	})
}
