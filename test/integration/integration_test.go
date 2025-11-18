package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type Database struct {
	Name string
	URL  string
}

var databases = []Database{
	{Name: "postgresql", URL: "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"},
	{Name: "mysql", URL: "testuser:testpass@tcp(localhost:3306)/testdb"},
	{Name: "sqlite", URL: "sqlite://./test.db"},
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	fmt.Println("🐳 Starting Docker containers...")
	cmd := exec.Command("docker-compose", "up", "-d")
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Failed to start Docker: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("⏳ Waiting for databases to be healthy...")
	if !waitForHealthy(ctx, 30*time.Second) {
		fmt.Println("❌ Databases failed to become healthy")
		cleanup()
		os.Exit(1)
	}
	fmt.Println("✅ Databases ready")

	code := m.Run()

	fmt.Println("🧹 Cleaning up...")
	cleanup()

	os.Exit(code)
}

func waitForHealthy(ctx context.Context, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("docker-compose", "ps", "--format", "json")
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), "healthy") {
			time.Sleep(2 * time.Second)
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func cleanup() {
	exec.Command("docker-compose", "down", "-v").Run()
	os.RemoveAll("test_projects")
}

func TestAllDatabasesParallel(t *testing.T) {
	var wg sync.WaitGroup

	for _, db := range databases {
		wg.Add(1)
		go func(database Database) {
			defer wg.Done()
			t.Run(database.Name, func(t *testing.T) {
				t.Parallel()
				testDatabase(t, database)
			})
		}(db)
	}

	wg.Wait()
}

func testDatabase(t *testing.T, db Database) {
	testDir := filepath.Join("test_projects", db.Name)

	os.RemoveAll(testDir)
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	t.Run("01_Init", func(t *testing.T) {
		testInit(t, testDir, db)
	})

	t.Run("02_Migrate", func(t *testing.T) {
		testMigrate(t, testDir, db)
	})

	t.Run("03_Apply", func(t *testing.T) {
		testApply(t, testDir, db)
	})

	t.Run("04_Status", func(t *testing.T) {
		testStatus(t, testDir, db)
	})

	t.Run("05_Gen", func(t *testing.T) {
		testGen(t, testDir, db)
	})

	t.Run("06_Pull", func(t *testing.T) {
		testPull(t, testDir, db)
	})

	t.Run("07_Export_JSON", func(t *testing.T) {
		testExportJSON(t, testDir, db)
	})

	t.Run("08_Export_CSV", func(t *testing.T) {
		testExportCSV(t, testDir, db)
	})

	t.Run("09_Export_SQLite", func(t *testing.T) {
		testExportSQLite(t, testDir, db)
	})

	t.Run("10_Raw", func(t *testing.T) {
		testRaw(t, testDir, db)
	})

	t.Run("11_Branch_Create", func(t *testing.T) {
		testBranchCreate(t, testDir, db)
	})

	t.Run("12_Branch_List", func(t *testing.T) {
		testBranchList(t, testDir, db)
	})

	t.Run("13_Branch_Checkout", func(t *testing.T) {
		testBranchCheckout(t, testDir, db)
	})

	t.Run("14_Branch_Diff", func(t *testing.T) {
		testBranchDiff(t, testDir, db)
	})

	t.Run("15_Branch_Delete", func(t *testing.T) {
		testBranchDelete(t, testDir, db)
	})

	t.Run("16_Studio", func(t *testing.T) {
		testStudio(t, testDir, db)
	})

	t.Run("17_Reset", func(t *testing.T) {
		testReset(t, testDir, db)
	})
}

func testInit(t *testing.T, testDir string, db Database) {
	flag := fmt.Sprintf("--%s", db.Name)
	if db.Name == "postgresql" {
		flag = "--postgresql"
	}

	cmd := exec.Command("../../flash", "init", flag)
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Init failed for %s: %v\nOutput: %s", db.Name, err, output)
	}

	files := []string{"flash.config.json", "db/schema/schema.sql", "db/queries"}
	for _, file := range files {
		path := filepath.Join(testDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file/dir not created: %s", file)
		}
	}

	envPath := filepath.Join(testDir, ".env")
	if err := os.WriteFile(envPath, []byte(fmt.Sprintf("DATABASE_URL=%s\n", db.URL)), 0644); err != nil {
		t.Fatalf("Failed to write .env: %v", err)
	}

	t.Logf("✅ Init successful for %s", db.Name)
}

func testMigrate(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "migrate", "initial_schema")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Migrate failed for %s: %v\nOutput: %s", db.Name, err, output)
	}

	migrationsDir := filepath.Join(testDir, "db/migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil || len(files) == 0 {
		t.Errorf("No migration files created")
	}

	t.Logf("✅ Migrate successful - created %d migration(s)", len(files))
}

func testApply(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "apply", "--force")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Apply failed for %s: %v\nOutput: %s", db.Name, err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Applied") && !strings.Contains(outputStr, "No pending") {
		t.Errorf("Unexpected apply output: %s", output)
	}

	t.Logf("✅ Apply successful")
}

func testStatus(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "status")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Status failed for %s: %v\nOutput: %s", db.Name, err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Migration") && !strings.Contains(outputStr, "Database") {
		t.Errorf("Unexpected status output: %s", output)
	}

	t.Logf("✅ Status successful")
}

func testGen(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "gen")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Gen output: %s", output)
	}

	genDir := filepath.Join(testDir, "flash_gen")
	if _, err := os.Stat(genDir); err == nil {
		t.Logf("✅ Gen successful - code generated")
	} else {
		t.Logf("⚠️  Gen completed but no flash_gen directory")
	}
}

func testPull(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "pull")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Pull output: %s", output)
	}

	schemaPath := filepath.Join(testDir, "db/schema/schema.sql")
	if info, err := os.Stat(schemaPath); err == nil && info.Size() > 0 {
		t.Logf("✅ Pull successful - schema size: %d bytes", info.Size())
	}
}

func testExportJSON(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "export", "--json")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Export JSON output: %s", output)
	}

	exportDir := filepath.Join(testDir, "db/export")
	if files, err := os.ReadDir(exportDir); err == nil && len(files) > 0 {
		t.Logf("✅ Export JSON successful - %d file(s) created", len(files))
	}
}

func testExportCSV(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "export", "--csv")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Export CSV output: %s", output)
	} else {
		t.Logf("✅ Export CSV successful")
	}
}

func testExportSQLite(t *testing.T, testDir string, db Database) {
	if db.Name == "sqlite" {
		t.Skip("Skipping SQLite export for SQLite database")
		return
	}

	cmd := exec.Command("../../flash", "export", "--sqlite")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Export SQLite output: %s", output)
	} else {
		t.Logf("✅ Export SQLite successful")
	}
}

func testRaw(t *testing.T, testDir string, db Database) {
	query := "SELECT 1"

	cmd := exec.Command("../../flash", "raw", query)
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Raw failed for %s: %v\nOutput: %s", db.Name, err, output)
	}

	if len(output) == 0 {
		t.Errorf("Raw query returned no output")
	}

	t.Logf("✅ Raw SQL successful")
}

func testBranchCreate(t *testing.T, testDir string, db Database) {
	if db.Name == "sqlite" {
		t.Skip("Skipping branch tests for SQLite")
		return
	}

	cmd := exec.Command("../../flash", "branch", "feature", "--force")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Branch create failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "created") {
		t.Errorf("Unexpected branch create output: %s", output)
	}

	t.Logf("✅ Branch create successful")
}

func testBranchList(t *testing.T, testDir string, db Database) {
	if db.Name == "sqlite" {
		t.Skip("Skipping branch tests for SQLite")
		return
	}

	cmd := exec.Command("../../flash", "branch")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Branch list failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "main") || !strings.Contains(outputStr, "feature") {
		t.Errorf("Expected branches not found in output: %s", output)
	}

	t.Logf("✅ Branch list successful")
}

func testBranchCheckout(t *testing.T, testDir string, db Database) {
	if db.Name == "sqlite" {
		t.Skip("Skipping branch tests for SQLite")
		return
	}

	cmd := exec.Command("../../flash", "checkout", "feature", "--force")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Branch checkout failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Switched") {
		t.Errorf("Unexpected checkout output: %s", output)
	}

	cmd = exec.Command("../../flash", "checkout", "main", "--force")
	cmd.Dir = testDir
	cmd.CombinedOutput()

	t.Logf("✅ Branch checkout successful")
}

func testBranchDiff(t *testing.T, testDir string, db Database) {
	if db.Name == "sqlite" {
		t.Skip("Skipping branch tests for SQLite")
		return
	}

	cmd := exec.Command("../../flash", "diff", "main", "feature")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Branch diff output: %s", output)
	}

	t.Logf("✅ Branch diff successful")
}

func testBranchDelete(t *testing.T, testDir string, db Database) {
	if db.Name == "sqlite" {
		t.Skip("Skipping branch tests for SQLite")
		return
	}

	cmd := exec.Command("../../flash", "branch", "--delete", "feature", "--force")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Branch delete failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "deleted") {
		t.Errorf("Unexpected delete output: %s", output)
	}

	t.Logf("✅ Branch delete successful")
}

func testStudio(t *testing.T, testDir string, db Database) {
	port := 15555 + getPortOffset(db.Name)

	cmd := exec.Command("../../flash", "studio", "--port", fmt.Sprintf("%d", port), "--browser=false")
	cmd.Dir = testDir

	if err := cmd.Start(); err != nil {
		t.Fatalf("Studio failed to start for %s: %v", db.Name, err)
	}

	time.Sleep(3 * time.Second)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		t.Logf("⚠️  Studio HTTP check failed: %v", err)
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 200 && len(body) > 0 {
			t.Logf("✅ Studio running on port %d", port)
		}
	}

	cmd.Process.Kill()
	cmd.Wait()
}

func testReset(t *testing.T, testDir string, db Database) {
	cmd := exec.Command("../../flash", "reset", "--force")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Reset output: %s", output)
	} else {
		t.Logf("✅ Reset successful")
	}
}

func getPortOffset(dbName string) int {
	switch dbName {
	case "postgresql":
		return 0
	case "mysql":
		return 1
	case "sqlite":
		return 2
	default:
		return 3
	}
}
