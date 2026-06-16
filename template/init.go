package template

import "fmt"

type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	ClickHouse DatabaseType = "clickhouse"
	ScyllaDB   DatabaseType = "scylla"
)

type ProjectTemplate struct {
	DatabaseType    DatabaseType
	IsNodeProject   bool
	IsPythonProject bool
}

type dbConfig struct {
	provider         string
	engine           string
	primaryKey       string
	autoIncrement    string
	textType         string
	timestampType    string
	timestampDefault string
	queryParam       string
	returnType       string
	envExample       string
}

var dbConfigs = map[DatabaseType]dbConfig{
	SQLite: {
		provider:         "sqlite",
		engine:           "sqlite",
		primaryKey:       "INTEGER PRIMARY KEY AUTOINCREMENT",
		autoIncrement:    "AUTOINCREMENT",
		textType:         "TEXT",
		timestampType:    "DATETIME",
		timestampDefault: "CURRENT_TIMESTAMP",
		queryParam:       "?",
		returnType:       ":one",
		envExample:       "sqlite://./data.sqlite",
	},
	MySQL: {
		provider:         "mysql",
		engine:           "mysql",
		primaryKey:       "INT AUTO_INCREMENT PRIMARY KEY",
		autoIncrement:    "AUTO_INCREMENT",
		textType:         "VARCHAR(255)",
		timestampType:    "TIMESTAMP",
		timestampDefault: "CURRENT_TIMESTAMP",
		queryParam:       "?",
		returnType:       ":execresult",
		envExample:       "mysql://username:password@localhost:3306/database_name",
	},
	PostgreSQL: {
		provider:         "postgresql",
		engine:           "postgresql",
		primaryKey:       "SERIAL PRIMARY KEY",
		autoIncrement:    "SERIAL",
		textType:         "VARCHAR(255)",
		timestampType:    "TIMESTAMP WITH TIME ZONE",
		timestampDefault: "NOW()",
		queryParam:       "$1",
		returnType:       ":one",
		envExample:       "postgres://username:password@localhost:5432/database_name",
	},
	ClickHouse: {
		provider:         "clickhouse",
		engine:           "clickhouse",
		primaryKey:       "UInt64",
		autoIncrement:    "",
		textType:         "String",
		timestampType:    "DateTime",
		timestampDefault: "now()",
		queryParam:       "?",
		returnType:       ":exec",
		envExample:       "clickhouse://username:password@localhost:9000/database_name",
	},
	ScyllaDB: {
		provider:         "scylla",
		engine:           "scylla",
		primaryKey:       "uuid PRIMARY KEY",
		autoIncrement:    "",
		textType:         "text",
		timestampType:    "timestamp",
		timestampDefault: "toTimestamp(now())",
		queryParam:       "?",
		returnType:       ":one",
		envExample:       "scylla://host:9042/keyspace_name",
	},
}

func NewProjectTemplate(dbType DatabaseType, isNodeProject bool, isPythonProject bool) *ProjectTemplate {
	return &ProjectTemplate{
		DatabaseType:    dbType,
		IsNodeProject:   isNodeProject,
		IsPythonProject: isPythonProject,
	}
}

func (pt *ProjectTemplate) GetFlashORMConfig() string {
	cfg := dbConfigs[pt.DatabaseType]

	config := pt.getDriverHeaderComment() + "\n"
	config += "version = \"2\"\n"
	config += "schema_dir = \"db/schema\"\n"
	config += "queries = \"db/queries/\"\n"
	config += "migrations_path = \"db/migrations\"\n"
	config += "export_path = \"db/export\"\n\n"

	config += "[database]\n"
	config += fmt.Sprintf("provider = \"%s\"\n", cfg.provider)
	config += "url_env = \"DATABASE_URL\"\n"

	genSection := pt.getGenSection()
	if genSection != "" {
		config += "\n" + genSection
	}

	return config
}

func (pt *ProjectTemplate) GetSchema() string {
	switch pt.DatabaseType {
	case ScyllaDB:
		return scyllaSchema
	default:
		return pt.getRelationalSchema()
	}
}

const scyllaSchema = `-- === KEYSPACE ===
CREATE KEYSPACE myapp
WITH replication = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
};

-- === TABLES ===

CREATE TABLE myapp.users (
    id          uuid PRIMARY KEY,
    username    text,
    email       text,
    full_name   text,
    is_active   boolean,
    tags        set<text>,
    metadata    map<text, text>,
    created_at  timestamp,
    updated_at  timestamp
);

CREATE INDEX myapp.users_email_idx ON myapp.users (email);
`

func (pt *ProjectTemplate) getRelationalSchema() string {
	cfg := dbConfigs[pt.DatabaseType]
	updateClause := ""
	if pt.DatabaseType == MySQL {
		updateClause = " ON UPDATE CURRENT_TIMESTAMP"
	}

	return fmt.Sprintf(`CREATE TABLE users (
    id %s,
    name %s NOT NULL,
    email %s UNIQUE NOT NULL,
    created_at %s NOT NULL DEFAULT %s,
    updated_at %s NOT NULL DEFAULT %s%s
);
`, cfg.primaryKey, cfg.textType, cfg.textType, cfg.timestampType,
		cfg.timestampDefault, cfg.timestampType, cfg.timestampDefault, updateClause)
}

func (pt *ProjectTemplate) GetQueries() string {
	if pt.DatabaseType == ScyllaDB {
		return scyllaQueries
	}
	cfg := dbConfigs[pt.DatabaseType]
	param2 := cfg.queryParam
	if pt.DatabaseType == PostgreSQL {
		param2 = "$2"
	}

	return fmt.Sprintf(`-- name: GetUser :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE id = %s LIMIT 1;

-- name: CreateUser %s
INSERT INTO users (name, email)
VALUES (%s, %s)%s;
`, cfg.queryParam, cfg.returnType, cfg.queryParam, param2, pt.getReturningClause())
}

const scyllaQueries = `-- name: CreateUser :exec
INSERT INTO myapp.users (id, username, email, full_name, is_active, tags, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByID :one
SELECT * FROM myapp.users WHERE id = ?;

-- name: GetUserByEmail :many
SELECT id, username, email, created_at FROM myapp.users WHERE email = ? ALLOW FILTERING;

-- name: UpdateUserProfile :exec
UPDATE myapp.users SET full_name = ?, updated_at = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM myapp.users WHERE id = ?;
`

func (pt *ProjectTemplate) getReturningClause() string {
	if pt.DatabaseType == MySQL {
		return ""
	}
	return "\nRETURNING id, name, email, created_at, updated_at"
}

func (pt *ProjectTemplate) GetEnvTemplate() string {
	cfg := dbConfigs[pt.DatabaseType]
	return fmt.Sprintf("DATABASE_URL=%s\n", cfg.envExample)
}

func (pt *ProjectTemplate) GetDirectoryStructure() []string {
	return []string{"db/schema", "db/queries"}
}

func ValidateDatabaseType(dbType string) DatabaseType {
	types := map[string]DatabaseType{
		"sqlite":     SQLite,
		"mysql":      MySQL,
		"postgresql": PostgreSQL,
		"postgres":   PostgreSQL,
		"clickhouse": ClickHouse,
		"scylla":     ScyllaDB,
		"scylladb":   ScyllaDB,
		"cassandra":  ScyllaDB,
	}

	if dt, exists := types[dbType]; exists {
		return dt
	}
	return PostgreSQL
}

func (pt *ProjectTemplate) getGenSection() string {
	if pt.IsNodeProject {
		return `[gen.js]
enabled = true
out = "flash_gen"`
	}
	if pt.IsPythonProject {
		return `[gen.python]
enabled = true
out = "flash_gen"
async = true`
	}
	return `[gen.go]
enabled = true`
}

func (pt *ProjectTemplate) getDriverHeaderComment() string {
	switch pt.DatabaseType {
	case PostgreSQL:
		return `# FlashORM — PostgreSQL Drivers
#   Go:     "pgx" (native) | "database/sql" (lib/pq)
#   JS:     "pg" (node-postgres) | "postgres" (porsager/postgres)
#   Python: "psycopg3" | "asyncpg"
# Add driver = "<name>" inside the [gen.*] block below.`
	case MySQL:
		return `# FlashORM — MySQL Drivers
#   Go:     "database/sql" (go-sql-driver/mysql)
#   JS:     "mysql2" | "serverless-mysql"
#   Python: "pymysql" (sync) | "asyncmy" (async)
# Add driver = "<name>" inside the [gen.*] block below.`
	case SQLite:
		return `# FlashORM — SQLite Drivers
#   Go:     "database/sql" (mattn/go-sqlite3, modernc.org/sqlite)
#   JS:     "better-sqlite3" | "bun:sqlite"
#   Python: "sqlite3" (sync) | "aiosqlite" (async)
# Add driver = "<name>" inside the [gen.*] block below.`
	case ClickHouse:
		return `# FlashORM — ClickHouse Drivers
#   Go:     "clickhouse-go/v2"
#   JS:     "@clickhouse/client"
#   Python: "clickhouse-driver" (sync) | "asynch" (async)
# Add driver = "<name>" inside the [gen.*] block below.`
	case ScyllaDB:
		return `# FlashORM — ScyllaDB/Cassandra Drivers
#   Go:     "apache/cassandra-gocql-driver/v2"
#   JS:     "cassandra-driver"
#   Python: "scylla-driver" (sync) | "cassandra-driver" (async)
# Add driver = "<name>" inside the [gen.*] block below.`
	default:
		return `# FlashORM — See docs for available drivers per database.`
	}
}
