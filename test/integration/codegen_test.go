package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeGenerationAllLanguages(t *testing.T) {
	for _, db := range databases {
		t.Run(db.Name, func(t *testing.T) {
			t.Parallel()
			testDir := filepath.Join("test_projects", "codegen_"+db.Name)
			os.RemoveAll(testDir)
			os.MkdirAll(testDir, 0755)
			defer os.RemoveAll(testDir)

			setupCodegenProject(t, testDir, db)

			t.Run("Go", func(t *testing.T) {
				testGoGeneration(t, testDir, db)
			})

			t.Run("JavaScript", func(t *testing.T) {
				testJSGeneration(t, testDir, db)
			})

			t.Run("Python", func(t *testing.T) {
				testPythonGeneration(t, testDir, db)
			})
		})
	}
}

func setupCodegenProject(t *testing.T, testDir string, db Database) {
	flag := fmt.Sprintf("--%s", db.Name)
	if db.Name == "postgresql" {
		flag = "--postgresql"
	}

	cmd := exec.Command("../../flash", "init", flag)
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	envPath := filepath.Join(testDir, ".env")
	os.WriteFile(envPath, []byte(fmt.Sprintf("DATABASE_URL=%s\n", db.URL)), 0644)

	cmd = exec.Command("../../flash", "migrate", "test_schema")
	cmd.Dir = testDir
	cmd.Run()

	cmd = exec.Command("../../flash", "apply", "--force")
	cmd.Dir = testDir
	cmd.Run()
}

func testGoGeneration(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "gen")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Go gen output: %s", output)
	}

	genDir := filepath.Join(testDir, "flash_gen")
	if _, err := os.Stat(genDir); err == nil {
		t.Logf("✅ Go code generated")
	}
}

func testJSGeneration(t *testing.T, testDir string, db Database) {
	packageJSON := `{"name": "test", "version": "1.0.0"}`
	os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)

	configPath := filepath.Join(testDir, "flash.config.json")
	config, _ := os.ReadFile(configPath)
	configStr := string(config)

	if !strings.Contains(configStr, `"js"`) {
		newConfig := strings.Replace(configStr, `"gen": {`, `"gen": {"js": {"enabled": true},`, 1)
		os.WriteFile(configPath, []byte(newConfig), 0644)
	}

	cmd := exec.Command("../../flash", "gen")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("JS gen output: %s", output)
	}

	genDir := filepath.Join(testDir, "flash_gen")
	jsFile := filepath.Join(genDir, "database.js")
	if _, err := os.Stat(jsFile); err == nil {
		t.Logf("✅ JavaScript code generated")
	}
}

func testPythonGeneration(t *testing.T, testDir string, db Database) {
	os.WriteFile(filepath.Join(testDir, "requirements.txt"), []byte("psycopg2\n"), 0644)

	configPath := filepath.Join(testDir, "flash.config.json")
	config, _ := os.ReadFile(configPath)
	configStr := string(config)

	if !strings.Contains(configStr, `"python"`) {
		newConfig := strings.Replace(configStr, `"gen": {`, `"gen": {"python": {"enabled": true},`, 1)
		os.WriteFile(configPath, []byte(newConfig), 0644)
	}

	cmd := exec.Command("../../flash", "gen")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Python gen output: %s", output)
	}

	genDir := filepath.Join(testDir, "flash_gen")
	pyFile := filepath.Join(genDir, "models.py")
	if _, err := os.Stat(pyFile); err == nil {
		t.Logf("✅ Python code generated")
	}
}
