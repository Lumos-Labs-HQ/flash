package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "flash.config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// No config file — all defaults should apply.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SchemaDir != "db/schema" {
		t.Errorf("SchemaDir = %q, want db/schema", cfg.SchemaDir)
	}
	if cfg.MigrationsPath != "db/migrations" {
		t.Errorf("MigrationsPath = %q, want db/migrations", cfg.MigrationsPath)
	}
	if cfg.Database.Provider != "postgresql" {
		t.Errorf("Provider = %q, want postgresql", cfg.Database.Provider)
	}
	if cfg.Database.URLEnv != "DATABASE_URL" {
		t.Errorf("URLEnv = %q, want DATABASE_URL", cfg.Database.URLEnv)
	}
	if cfg.Version != "2" {
		t.Errorf("Version = %q, want 2", cfg.Version)
	}
}

func TestLoad_ExplicitValues(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	writeConfig(t, dir, `{
		"version": "2",
		"schema_dir": "custom/schema",
		"queries": "custom/queries/",
		"migrations_path": "custom/migrations",
		"database": {"provider": "mysql", "url_env": "MYSQL_URL"},
		"gen": {"go": {"enabled": true}}
	}`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SchemaDir != "custom/schema" {
		t.Errorf("SchemaDir = %q, want custom/schema", cfg.SchemaDir)
	}
	if cfg.Database.Provider != "mysql" {
		t.Errorf("Provider = %q, want mysql", cfg.Database.Provider)
	}
	if cfg.Database.URLEnv != "MYSQL_URL" {
		t.Errorf("URLEnv = %q, want MYSQL_URL", cfg.Database.URLEnv)
	}
}

func TestLoad_PythonAsyncDefaultsTrue(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	writeConfig(t, dir, `{"gen": {"python": {"enabled": true}}}`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Gen.Python.Async {
		t.Error("python.async should default to true when not explicitly set")
	}
}

func TestLoad_PythonAsyncExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	writeConfig(t, dir, `{"gen": {"python": {"enabled": true, "async": false}}}`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Gen.Python.Async {
		t.Error("python.async should be false when explicitly set to false")
	}
}

func TestLoad_LegacySchemaPath(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	writeConfig(t, dir, `{"schema_path": "db/schema/schema.sql"}`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Legacy schema_path (file) → SchemaDir should be its directory
	if cfg.SchemaDir != "db/schema" {
		t.Errorf("SchemaDir = %q, want db/schema (derived from legacy schema_path)", cfg.SchemaDir)
	}
}

func TestValidate_UnsupportedProvider(t *testing.T) {
	cfg := &Config{
		Database:       Database{Provider: "oracle"},
		MigrationsPath: "db/migrations",
		ExportPath:     "db/export",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for unsupported provider, got nil")
	}
}

func TestValidate_SupportedProviders(t *testing.T) {
	for _, p := range []string{"postgresql", "postgres", "mysql", "sqlite", "sqlite3"} {
		cfg := &Config{
			Database:       Database{Provider: p},
			MigrationsPath: "db/migrations",
			ExportPath:     "db/export",
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("provider %q: unexpected error: %v", p, err)
		}
	}
}

func TestGetDatabaseURL_FromEnv(t *testing.T) {
	cfg := &Config{Database: Database{URLEnv: "TEST_DB_URL"}}
	os.Setenv("TEST_DB_URL", "postgres://user:pass@localhost/db")
	defer os.Unsetenv("TEST_DB_URL")

	url, err := cfg.GetDatabaseURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "postgres://user:pass@localhost/db" {
		t.Errorf("url = %q", url)
	}
}

func TestGetDatabaseURL_Missing(t *testing.T) {
	cfg := &Config{Database: Database{URLEnv: "NONEXISTENT_DB_URL_XYZ"}}
	os.Unsetenv("NONEXISTENT_DB_URL_XYZ")

	_, err := cfg.GetDatabaseURL()
	if err == nil {
		t.Error("expected error for missing env var, got nil")
	}
}

func TestGetSchemaFiles_ReturnsSQL(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "users.sql"), []byte("CREATE TABLE users (id INT);"), 0644)
	os.WriteFile(filepath.Join(dir, "posts.sql"), []byte("CREATE TABLE posts (id INT);"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("docs"), 0644)

	cfg := &Config{SchemaDir: dir}
	files, err := cfg.GetSchemaFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("files = %d, want 2 (only .sql files)", len(files))
	}
}

func TestGetSqlcEngine(t *testing.T) {
	cases := []struct{ provider, want string }{
		{"postgresql", "postgresql"},
		{"postgres", "postgresql"},
		{"mysql", "mysql"},
		{"sqlite", "sqlite"},
		{"sqlite3", "sqlite"},
	}
	for _, c := range cases {
		cfg := &Config{Database: Database{Provider: c.provider}}
		if got := cfg.GetSqlcEngine(); got != c.want {
			t.Errorf("provider %q: engine = %q, want %q", c.provider, got, c.want)
		}
	}
}
