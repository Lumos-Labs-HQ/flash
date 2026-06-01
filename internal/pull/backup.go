package pull

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (s *Service) createDirBackup(schemaDir string) error {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	backupDir := schemaDir + ".backup"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		srcPath := filepath.Join(schemaDir, entry.Name())
		dstPath := filepath.Join(backupDir, entry.Name())

		content, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dstPath, content, 0644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getTableIndexes(ctx context.Context, tables []types.SchemaTable) (map[string][]types.SchemaIndex, error) {
	result := make(map[string][]types.SchemaIndex)

	type IndexFetcher interface {
		GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error)
	}

	fetcher, ok := s.adapter.(IndexFetcher)
	if !ok {
		for _, table := range tables {
			if len(table.Indexes) > 0 {
				result[table.Name] = table.Indexes
			}
		}
		return result, nil
	}

	for _, table := range tables {
		indexes, err := fetcher.GetTableIndexes(ctx, table.Name)
		if err != nil {
			continue
		}
		result[table.Name] = indexes
	}

	return result, nil
}
