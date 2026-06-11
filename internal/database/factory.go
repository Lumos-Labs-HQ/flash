package database

import (
	"github.com/Lumos-Labs-HQ/flash/internal/database/clickhouse"
	"github.com/Lumos-Labs-HQ/flash/internal/database/mongodb"
	"github.com/Lumos-Labs-HQ/flash/internal/database/mysql"
	"github.com/Lumos-Labs-HQ/flash/internal/database/postgres"
	"github.com/Lumos-Labs-HQ/flash/internal/database/scylla"
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
	case "mongodb", "mongo":
		return mongodb.New()
	case "clickhouse":
		return clickhouse.New()
	case "scylla", "scylladb", "cassandra":
		return scylla.New()
	default:
		return postgres.New()
	}
}
