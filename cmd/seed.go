package cmd

import (
	"database/sql"
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/seeder"
	"github.com/spf13/cobra"
	
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var (
	seedFile  string
	seedEnv   string
	seedForce bool
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run database seeds",
	Long:  `Execute SQL seed files to populate your database with initial or test data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		dbURL, err := cfg.GetDatabaseURL()
		if err != nil {
			return err
		}

		db, err := openDB(cfg.Database.Provider, dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		s := seeder.New(cfg, db)

		if seedFile != "" {
			return runSingleSeed(s, seedFile)
		}

		return s.RunAll(seedEnv, seedForce)
	},
}

func openDB(provider, url string) (*sql.DB, error) {
	var driverName string
	switch provider {
	case "postgresql", "postgres":
		driverName = "pgx"
	case "mysql":
		driverName = "mysql"
	case "sqlite", "sqlite3":
		driverName = "sqlite3"
	default:
		driverName = "pgx"
	}
	
	db, err := sql.Open(driverName, url)
	if err != nil {
		return nil, err
	}
	
	if err := db.Ping(); err != nil {
		return nil, err
	}
	
	return db, nil
}

func runSingleSeed(s *seeder.Seeder, filename string) error {
	seeds, err := s.ReadSeedFiles()
	if err != nil {
		return err
	}

	for _, seed := range seeds {
		if seed.Name == filename {
			if err := s.EnsureSeedsTable(); err != nil {
				return err
			}
			fmt.Printf("🌱 Running seed: %s\n", seed.Name)
			return s.ExecuteSeed(seed, seedForce)
		}
	}

	return fmt.Errorf("seed file not found: %s", filename)
}

func init() {
	rootCmd.AddCommand(seedCmd)
	seedCmd.Flags().StringVar(&seedFile, "file", "", "Run specific seed file")
	seedCmd.Flags().StringVar(&seedEnv, "env", "", "Run seeds for specific environment (dev, test, prod)")
	seedCmd.Flags().BoolVar(&seedForce, "force", false, "Re-run already executed seeds")
}
