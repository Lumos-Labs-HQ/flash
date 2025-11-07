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

	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackupManager struct {
	db         *pgxpool.Pool
	backupPath string
}

func NewBackupManager(db *pgxpool.Pool, backupPath string) *BackupManager {
	return &BackupManager{db: db, backupPath: backupPath}
}

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

	if !bm.shouldCreateBackup(ctx, tables, comment) {
		return "", nil
	}

	for _, table := range tables {
		if table == "_flash_migrations" {
			continue
		}
		bm.backupTable(ctx, table, &backup)
	}

	return bm.writeBackupFile(backup)
}

func (bm *BackupManager) shouldCreateBackup(ctx context.Context, tables []string, comment string) bool {
	if strings.Contains(comment, "Manual backup") || strings.Contains(comment, "Pre-reset") {
		return true
	}

	for _, table := range tables {
		if table != "_flash_migrations" {
			var count int
			if err := bm.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err == nil && count > 0 {
				return true
			}
		}
	}
	return false
}

func (bm *BackupManager) backupTable(ctx context.Context, table string, backup *types.BackupData) {
	rows, err := bm.db.Query(ctx, fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return
	}
	defer rows.Close()

	var tableData []map[string]interface{}
	fieldDescriptions := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}

		rowData := make(map[string]interface{}, len(fieldDescriptions))
		for i, fd := range fieldDescriptions {
			rowData[string(fd.Name)] = values[i]
		}
		tableData = append(tableData, rowData)
	}

	backup.Tables[table] = tableData
}

func (bm *BackupManager) writeBackupFile(backup types.BackupData) (string, error) {
	if err := os.MkdirAll(bm.backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupPath := filepath.Join(bm.backupPath, fmt.Sprintf("backup_%s.json", backup.Timestamp))

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

func (bm *BackupManager) getAllTableNames(ctx context.Context) ([]string, error) {
	rows, err := bm.db.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
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

func (bm *BackupManager) GetAllTableNames(ctx context.Context) ([]string, error) {
	return bm.getAllTableNames(ctx)
}

// func PerformBackup(ctx context.Context, db *pgxpool.Pool, backupPath, comment string) (string, error) {
// 	backupManager := NewBackupManager(db, backupPath)
// 	return backupManager.CreateBackup(ctx, comment, createMigrationGetter(db))
// }

// func createMigrationGetter(db *pgxpool.Pool) func(context.Context) (map[string]*time.Time, error) {
// 	return func(ctx context.Context) (map[string]*time.Time, error) {
// 		if _, err := db.Exec(ctx, `
// 			CREATE TABLE IF NOT EXISTS _flash_migrations (
// 				id VARCHAR(255) PRIMARY KEY,
// 				checksum VARCHAR(64) NOT NULL,
// 				finished_at TIMESTAMP WITH TIME ZONE,
// 				migration_name VARCHAR(255) NOT NULL,
// 				logs TEXT,
// 				rolled_back_at TIMESTAMP WITH TIME ZONE,
// 				started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
// 				applied_steps_count INTEGER NOT NULL DEFAULT 0
// 			)`); err != nil {
// 			return nil, fmt.Errorf("failed to create migrations table: %w", err)
// 		}

// 		applied := make(map[string]*time.Time)
// 		rows, err := db.Query(ctx, `
// 			SELECT id, finished_at
// 			FROM _flash_migrations
// 			WHERE finished_at IS NOT NULL AND rolled_back_at IS NULL`)
// 		if err != nil {
// 			return nil, err
// 		}
// 		defer rows.Close()

// 		for rows.Next() {
// 			var id string
// 			var finishedAt *time.Time
// 			if err := rows.Scan(&id, &finishedAt); err != nil {
// 				return nil, err
// 			}
// 			applied[id] = finishedAt
// 		}
// 		return applied, nil
// 	}
// }

func PerformBackupWithAdapter(ctx context.Context, adapter database.DatabaseAdapter, backupPath, comment string) (string, error) {
	tables, err := adapter.GetAllTableNames(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	if len(tables) == 0 {
		log.Println("No tables found in database, skipping backup")
		return "", nil
	}

	backupData := types.BackupData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Version:   "1.0",
		Tables:    make(map[string]interface{}, len(tables)),
		Comment:   comment,
	}

	for _, tableName := range tables {
		if tableName == "_flash_migrations" {
			continue
		}

		if tableData, err := adapter.GetTableData(ctx, tableName); err != nil {
			log.Printf("Warning: Failed to get data for table %s: %v", tableName, err)
		} else {
			backupData.Tables[tableName] = tableData
		}
	}

	return writeBackupToFile(backupData, backupPath)
}

func writeBackupToFile(backupData types.BackupData, backupPath string) (string, error) {
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFilePath := filepath.Join(backupPath, fmt.Sprintf("backup_%s.json", timestamp))

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
