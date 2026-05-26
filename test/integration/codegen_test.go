package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodegen verifies that all three code generators produce the expected files
// for every supported database provider.
func TestCodegen(t *testing.T) {
	for _, db := range getDatabases() {
		db := db
		t.Run(db.Name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join("test_projects", "codegen_"+db.Name)
			os.RemoveAll(dir)
			os.MkdirAll(dir, 0755)
			t.Cleanup(func() {
				// Reset the shared database BEFORE removing the project directory,
				// because flash needs to chdir into it to read config.
				if out, err := flash(t, dir, "reset", "--force"); err != nil {
					t.Logf("cleanup reset error: %v\n%s", err, out)
				}
				os.RemoveAll(dir)
			})

			setupProject(t, dir, db)

			t.Run("Go", func(t *testing.T) {
				out, err := flash(t, dir, "gen")
				t.Logf("gen go: %s", out)
				if err != nil {
					t.Logf("gen go error (non-fatal): %v", err)
				}
				for _, f := range []string{"models.go", "db.go"} {
					if _, err := os.Stat(filepath.Join(dir, "flash_gen", f)); os.IsNotExist(err) {
						t.Errorf("missing %s", f)
					}
				}
			})

			t.Run("JavaScript", func(t *testing.T) {
				// Enable JS gen
				cfgPath := filepath.Join(dir, "flash.toml")
				raw, _ := os.ReadFile(cfgPath)
				cfg := string(raw)
				if !strings.Contains(cfg, `[gen.js]`) {
					cfg += "\n[gen.js]\nenabled = true\nout = \"flash_gen\"\n"
					os.WriteFile(cfgPath, []byte(cfg), 0644)
				}
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"t","version":"1.0.0"}`), 0644)

				out, err := flash(t, dir, "gen")
				t.Logf("gen js: %s", out)
				if err != nil {
					t.Logf("gen js error (non-fatal): %v", err)
				}
				for _, f := range []string{"index.js", "index.d.ts"} {
					if _, err := os.Stat(filepath.Join(dir, "flash_gen", f)); os.IsNotExist(err) {
						t.Errorf("missing %s", f)
					}
				}
			})

			t.Run("Python", func(t *testing.T) {
				cfgPath := filepath.Join(dir, "flash.toml")
				raw, _ := os.ReadFile(cfgPath)
				cfg := string(raw)
				if !strings.Contains(cfg, `[gen.python]`) {
					cfg += "\n[gen.python]\nenabled = true\nout = \"flash_gen\"\nasync = true\n"
					os.WriteFile(cfgPath, []byte(cfg), 0644)
				}
				os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("psycopg2\n"), 0644)

				out, err := flash(t, dir, "gen")
				t.Logf("gen python: %s", out)
				if err != nil {
					t.Logf("gen python error (non-fatal): %v", err)
				}
				for _, f := range []string{"models.py", "database.py", "database.pyi", "__init__.py"} {
					if _, err := os.Stat(filepath.Join(dir, "flash_gen", f)); os.IsNotExist(err) {
						t.Errorf("missing %s", f)
					}
				}
			})
		})
	}
}
