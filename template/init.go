package template

import "fmt"

type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
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

	var genSection string

	if pt.IsNodeProject {
		genSection = `  "gen": {
    "js": {
      "enabled": true
    }
  }`
	} else if pt.IsPythonProject {
		genSection = `  "gen": {
    "python": {
      "enabled": true
    }
  }`
	} else {
		genSection = `  "gen": {
    "go": {
      "enabled": true
    }
  }`
	}

	configParts := []string{
		`  "version": "2"`,
		`  "schema_dir": "db/schema"`,
		`  "queries": "db/queries/"`,
		`  "migrations_path": "db/migrations"`,
		`  "export_path": "db/export"`,
		fmt.Sprintf(`  "database": {
    "provider": "%s",
    "url_env": "DATABASE_URL"
  }`, cfg.provider),
	}

	if genSection != "" {
		configParts = append(configParts, genSection)
	}

	config := "{\n"
	for i, part := range configParts {
		config += part
		if i < len(configParts)-1 {
			config += ",\n"
		} else {
			config += "\n"
		}
	}
	config += "}"

	return config
}

func (pt *ProjectTemplate) GetSchema() string {
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
	}

	if dt, exists := types[dbType]; exists {
		return dt
	}
	return PostgreSQL
}
