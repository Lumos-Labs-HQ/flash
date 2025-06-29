package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.SchemaPath != "db/schema.sql" {
		t.Errorf("Expected schema_path to be 'db/schema.sql', got '%s'", config.SchemaPath)
	}
	
	if config.MigrationsPath != "migrations" {
		t.Errorf("Expected migrations_path to be 'migrations', got '%s'", config.MigrationsPath)
	}
	
	if config.BackupPath != "db_backup" {
		t.Errorf("Expected backup_path to be 'db_backup', got '%s'", config.BackupPath)
	}
	
	if config.Database.Provider != "postgresql" {
		t.Errorf("Expected database provider to be 'postgresql', got '%s'", config.Database.Provider)
	}
	
	if config.Database.URLEnv != "DATABASE_URL" {
		t.Errorf("Expected database url_env to be 'DATABASE_URL', got '%s'", config.Database.URLEnv)
	}
}

func TestInitializeProject(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "graft-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// Create a go.mod file to make it look like a Go project
	if err := os.WriteFile("go.mod", []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	// Test initialization
	if err := InitializeProject(); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}
	
	// Check if config file was created
	configPath := filepath.Join(tempDir, "graft.config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", configPath)
	}
	
	// Check if directories were created
	dirs := []string{"migrations", "db_backup", "db"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
	
	// Test that second initialization fails
	if err := InitializeProject(); err == nil {
		t.Error("Expected second initialization to fail, but it succeeded")
	}
}

func TestIsInitialized(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "graft-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// Should not be initialized initially
	if IsInitialized() {
		t.Error("Expected project to not be initialized, but it was")
	}
	
	// Create config file
	configPath := filepath.Join(tempDir, "graft.config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Should be initialized now
	if !IsInitialized() {
		t.Error("Expected project to be initialized, but it wasn't")
	}
}
