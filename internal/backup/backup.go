package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rana718/Graft/internal/database"
	"github.com/Rana718/Graft/internal/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackupManager
type BackupManager struct {
	db         *pgxpool.Pool
	backupPath string
}

// NewBackupManager creates backup manager
func NewBackupManager(db *pgxpool.Pool, backupPath string) *BackupManager {
	return &BackupManager{
		db:         db,
		backupPath: backupPath,
	}
}

// CreateBackup
func (bm *BackupManager) CreateBackup(ctx context.Context, comment string, getAppliedMigrations func(context.Context) (map[string]*time.Time, error)) (string, error) {
	applied, err := getAppliedMigrations(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get applied migrations: %w", err)
	}

	backup := types.BackupData{
		Timestamp: time.Now().Format("2006-01-02_15-04-05"),
		Version:   fmt.Sprintf("%d_migrations", len(applied)),
		Tables:    make(map[string]interface{}),
		Comment:   comment,
	}

	tables, err := bm.getAllTableNames(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	hasData := false
	for _, table := range tables {
		if table == "_graft_migrations" {
			continue
		}
		var count int
		if err := bm.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
			continue
		}
		if count > 0 {
			hasData = true
			break
		}
	}

	if !hasData && !strings.Contains(comment, "Manual backup") && !strings.Contains(comment, "Pre-reset") {
		return "", nil
	}

	for _, table := range tables {
		if table == "_graft_migrations" {
			continue
		}

		rows, err := bm.db.Query(ctx, fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			continue
		}

		var tableData []map[string]interface{}
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				continue
			}

			fieldDescriptions := rows.FieldDescriptions()
			rowData := make(map[string]interface{})
			for i, fd := range fieldDescriptions {
				columnName := string(fd.Name)
				rowData[columnName] = values[i]
			}
			tableData = append(tableData, rowData)
		}
		rows.Close()

		backup.Tables[table] = tableData
	}

	return bm.writeBackupFile(backup)
}

// writeBackupFile
func (bm *BackupManager) writeBackupFile(backup types.BackupData) (string, error) {
	if err := os.MkdirAll(bm.backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	filename := fmt.Sprintf("backup_%s.json", backup.Timestamp)
	backupPath := filepath.Join(bm.backupPath, filename)

	file, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(backup); err != nil {
		return "", fmt.Errorf("failed to write backup data: %w", err)
	}

	log.Printf("Database backup created: %s", backupPath)
	return backupPath, nil
}

// getAllTableNames
func (bm *BackupManager) getAllTableNames(ctx context.Context) ([]string, error) {
	query := `
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_type = 'BASE TABLE'
        ORDER BY table_name
    `

	rows, err := bm.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// GetAllTableNames
func (bm *BackupManager) GetAllTableNames(ctx context.Context) ([]string, error) {
	return bm.getAllTableNames(ctx)
}

// PerformBackup
func PerformBackup(ctx context.Context, db *pgxpool.Pool, backupPath, comment string) (string, error) {
	backupManager := NewBackupManager(db, backupPath)

	getAppliedMigrations := func(ctx context.Context) (map[string]*time.Time, error) {
		_, err := db.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS _graft_migrations (
				id VARCHAR(255) PRIMARY KEY,
				checksum VARCHAR(64) NOT NULL,
				finished_at TIMESTAMP WITH TIME ZONE,
				migration_name VARCHAR(255) NOT NULL,
				logs TEXT,
				rolled_back_at TIMESTAMP WITH TIME ZONE,
				started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				applied_steps_count INTEGER NOT NULL DEFAULT 0
			);`)
		if err != nil {
			return nil, fmt.Errorf("failed to create migrations table: %w", err)
		}

		applied := make(map[string]*time.Time)
		rows, err := db.Query(ctx,
			`SELECT id, finished_at FROM _graft_migrations WHERE finished_at IS NOT NULL AND rolled_back_at IS NULL`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			var finishedAt *time.Time
			if err := rows.Scan(&id, &finishedAt); err != nil {
				return nil, err
			}
			applied[id] = finishedAt
		}
		return applied, nil
	}

	return backupManager.CreateBackup(ctx, comment, getAppliedMigrations)
}

// PerformBackupWithAdapter performs backup using database adapter
func PerformBackupWithAdapter(ctx context.Context, adapter database.DatabaseAdapter, backupPath, comment string) (string, error) {
	// Get all table names
	tables, err := adapter.GetAllTableNames(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	if len(tables) == 0 {
		log.Println("No tables found in database, skipping backup")
		return "", nil
	}

	// Create backup data structure
	backupData := types.BackupData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Version:   "1.0",
		Tables:    make(map[string]interface{}),
		Comment:   comment,
	}

	// Get data from each table
	for _, tableName := range tables {
		// Skip internal migration table
		if tableName == "_graft_migrations" {
			continue
		}

		tableData, err := adapter.GetTableData(ctx, tableName)
		if err != nil {
			log.Printf("Warning: Failed to get data for table %s: %v", tableName, err)
			continue
		}

		backupData.Tables[tableName] = tableData
	}

	// Get applied migrations
	appliedMigrations, err := adapter.GetAppliedMigrations(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get applied migrations: %v", err)
	} else {
		backupData.Tables["_graft_migrations"] = appliedMigrations
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("backup_%s.json", timestamp)
	backupFilePath := filepath.Join(backupPath, filename)

	// Write backup to file
	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupFilePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	log.Printf("Backup created successfully: %s", backupFilePath)
	return backupFilePath, nil
}
