package database

import (
	"github.com/Lumos-Labs-HQ/flash/internal/database/mysql"
	"github.com/Lumos-Labs-HQ/flash/internal/database/postgres"
	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
)

func NewAdapter(provider string) DatabaseAdapter {
	switch provider {
	case "postgresql", "postgres":
		return postgres.New()
	case "mysql":
		return mysql.New()
	case "sqlite", "sqlite3":
		return sqlite.New()
	default:
		return postgres.New()
	}
}
