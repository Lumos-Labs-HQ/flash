package backup

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/db"
)

// TableData represents data from a single table
type TableData struct {
	TableName string                   `json:"table_name"`
	Columns   []string                 `json:"columns"`
	Rows      []map[string]interface{} `json:"rows"`
}

// BackupData represents the complete database backup
type BackupData struct {
	Timestamp string      `json:"timestamp"`
	Tables    []TableData `json:"tables"`
}

// Manager handles backup operations
type Manager struct {
	Config      *config.Config
	ProjectRoot string
}

// NewManager creates a new backup manager
func NewManager(cfg *config.Config) (*Manager, error) {
	projectRoot, err := config.GetProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get project root: %w", err)
	}

	return &Manager{
		Config:      cfg,
		ProjectRoot: projectRoot,
	}, nil
}

// CreateBackup creates a backup of the current database
func (m *Manager) CreateBackup(conn *db.Connection) (string, error) {
	// Create backup directory with timestamp
	timestamp := time.Now().Format("2006-01-02_150405")
	backupDir := filepath.Join(m.ProjectRoot, m.Config.BackupPath, timestamp)
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Get all table names
	tables, err := conn.GetTableNames()
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	backupData := BackupData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Tables:    make([]TableData, 0, len(tables)),
	}

	// Backup each table
	for _, tableName := range tables {
		tableData, err := m.backupTable(conn.DB, tableName)
		if err != nil {
			return "", fmt.Errorf("failed to backup table %s: %w", tableName, err)
		}
		backupData.Tables = append(backupData.Tables, *tableData)
	}

	// Write backup to JSON file
	backupPath := filepath.Join(backupDir, "backup.json")
	backupJSON, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupPath, backupJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	fmt.Printf("✅ Database backup created: %s\n", backupPath)
	return backupPath, nil
}

// backupTable backs up a single table
func (m *Manager) backupTable(db *sql.DB, tableName string) (*TableData, error) {
	// Get column information
	columnsQuery := `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = $1 
		AND table_schema = 'public'
		ORDER BY ordinal_position
	`
	
	rows, err := db.Query(columnsQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, fmt.Errorf("failed to scan column name: %w", err)
		}
		columns = append(columns, columnName)
	}

	if len(columns) == 0 {
		return &TableData{
			TableName: tableName,
			Columns:   []string{},
			Rows:      []map[string]interface{}{},
		}, nil
	}

	// Get table data
	dataQuery := fmt.Sprintf("SELECT * FROM %s", tableName)
	dataRows, err := db.Query(dataQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query table data: %w", err)
	}
	defer dataRows.Close()

	var tableRows []map[string]interface{}
	
	for dataRows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := dataRows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				// Convert byte arrays to strings for JSON serialization
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			} else {
				row[col] = nil
			}
		}
		tableRows = append(tableRows, row)
	}

	return &TableData{
		TableName: tableName,
		Columns:   columns,
		Rows:      tableRows,
	}, nil
}

// ListBackups returns a list of available backups
func (m *Manager) ListBackups() ([]string, error) {
	backupDir := filepath.Join(m.ProjectRoot, m.Config.BackupPath)
	
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backupFile := filepath.Join(backupDir, entry.Name(), "backup.json")
			if _, err := os.Stat(backupFile); err == nil {
				backups = append(backups, entry.Name())
			}
		}
	}

	return backups, nil
}
