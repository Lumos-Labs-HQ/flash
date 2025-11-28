package migrator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Lumos-Labs-HQ/flash/internal/branch"
	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

type BranchAwareMigrator struct {
	*Migrator
	branchName   string
	branchSchema string
}

func NewBranchAwareMigrator(cfg *config.Config) (*BranchAwareMigrator, error) {
	migrator, err := NewMigrator(cfg)
	if err != nil {
		return nil, err
	}

	// Load current branch
	branchManager := branch.NewMetadataManager(cfg.MigrationsPath)
	store, err := branchManager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load branch metadata: %w", err)
	}

	currentBranch := store.GetBranch(store.Current)
	if currentBranch == nil {
		return nil, fmt.Errorf("current branch not found")
	}

	bam := &BranchAwareMigrator{
		Migrator:     migrator,
		branchName:   currentBranch.Name,
		branchSchema: currentBranch.Schema,
	}

	ctx := context.Background()
	switch cfg.Database.Provider {
	case "postgresql", "postgres":
		if currentBranch.Schema != "public" && currentBranch.Schema != "" {
			query := fmt.Sprintf("SET search_path TO %s, public", currentBranch.Schema)
			if _, err := migrator.adapter.ExecuteQuery(ctx, query); err != nil {
				fmt.Printf("Warning: Could not set search_path to %s: %v\n", currentBranch.Schema, err)
			} else {
				fmt.Printf("🔧 Using schema: %s\n", currentBranch.Schema)
			}
		}
	case "mysql":
		if currentBranch.Schema != "" && currentBranch.Schema != "public" {
			type DatabaseSwitcher interface {
				SwitchDatabase(ctx context.Context, dbName string) error
			}
			if switcher, ok := migrator.adapter.(DatabaseSwitcher); ok {
				if err := switcher.SwitchDatabase(ctx, currentBranch.Schema); err != nil {
					fmt.Printf("Warning: Could not switch to database %s: %v\n", currentBranch.Schema, err)
				} else {
					fmt.Printf("🔧 Using database: %s\n", currentBranch.Schema)
				}
			}
		}
	case "sqlite", "sqlite3":
		if currentBranch.Schema != "" && currentBranch.Schema != "public" {
			type DatabaseSwitcher interface {
				SwitchDatabase(ctx context.Context, dbName string) error
			}
			if switcher, ok := migrator.adapter.(DatabaseSwitcher); ok {
				if err := switcher.SwitchDatabase(ctx, currentBranch.Schema); err != nil {
					fmt.Printf("Warning: Could not switch to file %s: %v\n", currentBranch.Schema, err)
				} else {
					fmt.Printf("🔧 Using file: %s\n", currentBranch.Schema)
				}
			}
		}
	}

	return bam, nil
}

func GetCurrentBranchInfo(cfg *config.Config) (string, string, error) {
	branchManager := branch.NewMetadataManager(cfg.MigrationsPath)
	store, err := branchManager.Load()
	if err != nil {
		return "", "", err
	}

	currentBranch := store.GetBranch(store.Current)
	if currentBranch == nil {
		return "", "", fmt.Errorf("current branch not found")
	}

	return currentBranch.Name, currentBranch.Schema, nil
}

func GetBranchSchemaPath(cfg *config.Config, branchName string) (string, error) {
	branchManager := branch.NewMetadataManager(cfg.MigrationsPath)
	store, err := branchManager.Load()
	if err != nil {
		return "", err
	}

	branch := store.GetBranch(branchName)
	if branch == nil {
		return "", fmt.Errorf("branch '%s' not found", branchName)
	}

	return filepath.Join(cfg.MigrationsPath, ".flash", fmt.Sprintf("schema_%s.sql", branchName)), nil
}
