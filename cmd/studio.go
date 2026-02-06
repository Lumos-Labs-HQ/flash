//go:build plugin_studio || plugin_all || dev
// +build plugin_studio plugin_all dev

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/studio"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/mongodb"
	"github.com/Lumos-Labs-HQ/flash/internal/studio/redis"
	"github.com/spf13/cobra"
)

var studioCmd = &cobra.Command{
	Use:   "studio",
	Short: "Open FlashORM Studio - Visual database editor",
	Long: `
Launch FlashORM Studio, a web-based interface for viewing and editing your database.
Similar to Prisma Studio, it provides an intuitive UI for managing your data.

The studio will start a local web server and open in your default browser.

Examples:
  flash studio
  flash studio --db "postgres://user:pass@localhost:5432/mydb"
  flash studio --db "mongodb://localhost:27017/mydb"
  flash studio --redis "redis://localhost:6379"
  flash studio --port 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("db")
		redisURL, _ := cmd.Flags().GetString("redis")
		port, _ := cmd.Flags().GetInt("port")
		browser, _ := cmd.Flags().GetBool("browser")

		if redisURL != "" {
			fmt.Printf("🔴 Starting Redis Studio: %s\n", maskDBURL(redisURL))
			redisServer := redis.NewServer(redisURL, port)
			return redisServer.Start(browser)
		}

		var cfg *config.Config
		var err error

		if dbURL != "" {
			fmt.Printf("📊 Using database: %s\n", maskDBURL(dbURL))

			provider := detectProvider(dbURL)

			cfg = &config.Config{
				Database: config.Database{
					Provider: provider,
					URLEnv:   "STUDIO_DB_URL",
				},
			}

			os.Setenv("STUDIO_DB_URL", dbURL)
		} else {
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}
		}

		if cfg.Database.Provider == "mongodb" || cfg.Database.Provider == "mongo" {
			fmt.Println("🍃 Starting MongoDB Studio...")
			mongoServer := mongodb.NewServer(cfg, port)
			return mongoServer.Start(browser)
		} else {
			fmt.Println("🗄️  Starting SQL Studio...")
			server := studio.New(cfg, port)
			return server.Start(browser)
		}
	},
}

func init() {
	studioCmd.Flags().IntP("port", "p", 5555, "Port to run studio on")
	studioCmd.Flags().BoolP("browser", "b", true, "Open browser automatically")
	studioCmd.Flags().String("db", "", "Database URL (overrides config/env)")
	studioCmd.Flags().String("redis", "", "Redis URL for Redis Studio (e.g., redis://localhost:6379)")
}

func maskDBURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	if idx := len(url) / 2; idx > 0 {
		return url[:10] + "***" + url[len(url)-10:]
	}
	return "***"
}

func detectProvider(dbURL string) string {
	// Check for MongoDB first
	if strings.HasPrefix(dbURL, "mongodb://") || strings.HasPrefix(dbURL, "mongodb+srv://") {
		return "mongodb"
	}

	// Check other databases
	switch {
	case len(dbURL) >= 10 && (dbURL[:10] == "postgres://" || dbURL[:10] == "postgresql"):
		return "postgresql"
	case len(dbURL) >= 8 && dbURL[:8] == "mysql://":
		return "mysql"
	case len(dbURL) >= 9 && dbURL[:9] == "sqlite://":
		return "sqlite"
	default:
		if strings.Contains(dbURL, "mongodb") {
			return "mongodb"
		} else if strings.Contains(dbURL, "postgres") {
			return "postgresql"
		}
		return "postgresql"
	}
}
