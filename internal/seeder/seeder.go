package seeder

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

type Seeder struct {
	Config *config.Config
	DB     *sql.DB
}

type Seed struct {
	Name        string
	Path        string
	Content     string
	Environment string
	Description string
}

func New(cfg *config.Config, db *sql.DB) *Seeder {
	return &Seeder{
		Config: cfg,
		DB:     db,
	}
}

func (s *Seeder) EnsureSeedsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS flash_seeds (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			executed_at TIMESTAMP DEFAULT NOW()
		);
	`
	
	provider := s.Config.Database.Provider
	if provider == "sqlite" || provider == "sqlite3" {
		query = `
			CREATE TABLE IF NOT EXISTS flash_seeds (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT UNIQUE NOT NULL,
				executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`
	} else if provider == "mysql" {
		query = `
			CREATE TABLE IF NOT EXISTS flash_seeds (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) UNIQUE NOT NULL,
				executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`
	}
	
	_, err := s.DB.Exec(query)
	return err
}

func (s *Seeder) ReadSeedFiles() ([]*Seed, error) {
	seedsPath := s.Config.SeedsPath
	if seedsPath == "" {
		seedsPath = "db/seeds"
	}

	if _, err := os.Stat(seedsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("seeds directory not found: %s", seedsPath)
	}

	var seeds []*Seed
	err := filepath.Walk(seedsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		seed := &Seed{
			Name:    filepath.Base(path),
			Path:    path,
			Content: string(content),
		}

		seed.parseMetadata()
		seeds = append(seeds, seed)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].Name < seeds[j].Name
	})

	return seeds, nil
}

func (seed *Seed) parseMetadata() {
	lines := strings.Split(seed.Content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-- Environment:") {
			seed.Environment = strings.TrimSpace(strings.TrimPrefix(line, "-- Environment:"))
		}
		if strings.HasPrefix(line, "-- Description:") {
			seed.Description = strings.TrimSpace(strings.TrimPrefix(line, "-- Description:"))
		}
	}
}

func (s *Seeder) GetExecutedSeeds() (map[string]bool, error) {
	executed := make(map[string]bool)
	
	rows, err := s.DB.Query("SELECT name FROM flash_seeds")
	if err != nil {
		return executed, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return executed, err
		}
		executed[name] = true
	}

	return executed, rows.Err()
}

func (s *Seeder) ExecuteSeed(seed *Seed, force bool) error {
	executed, err := s.GetExecutedSeeds()
	if err != nil {
		return err
	}

	if executed[seed.Name] && !force {
		return fmt.Errorf("seed already executed: %s (use --force to re-run)", seed.Name)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(seed.Content)
	if err != nil {
		return fmt.Errorf("failed to execute seed %s: %w", seed.Name, err)
	}

	if force && executed[seed.Name] {
		_, err = tx.Exec("UPDATE flash_seeds SET executed_at = $1 WHERE name = $2", time.Now(), seed.Name)
	} else {
		_, err = tx.Exec("INSERT INTO flash_seeds (name) VALUES ($1)", seed.Name)
	}
	
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Seeder) RunAll(environment string, force bool) error {
	if err := s.EnsureSeedsTable(); err != nil {
		return fmt.Errorf("failed to create seeds table: %w", err)
	}

	seeds, err := s.ReadSeedFiles()
	if err != nil {
		return err
	}

	if len(seeds) == 0 {
		return fmt.Errorf("no seed files found in %s", s.Config.SeedsPath)
	}

	executed := 0
	for _, seed := range seeds {
		if environment != "" && seed.Environment != "" {
			envs := strings.Split(seed.Environment, ",")
			found := false
			for _, env := range envs {
				if strings.TrimSpace(env) == environment {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		fmt.Printf("🌱 Running seed: %s\n", seed.Name)
		if seed.Description != "" {
			fmt.Printf("   %s\n", seed.Description)
		}

		if err := s.ExecuteSeed(seed, force); err != nil {
			return err
		}
		executed++
	}

	if executed == 0 {
		fmt.Println("⚠️  No seeds to execute")
	} else {
		fmt.Printf("✅ Successfully executed %d seed(s)\n", executed)
	}

	return nil
}
