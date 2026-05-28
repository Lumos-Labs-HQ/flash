package integration

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"testing"
	"time"
)

// startStudio launches the studio and waits until it's ready, returning a cleanup func.
func startStudio(t *testing.T, dir string, port int, extraArgs ...string) (url string, stop func()) {
	t.Helper()
	args := append([]string{"studio", "--port", fmt.Sprintf("%d", port), "--browser=false"}, extraArgs...)
	cmd := exec.Command(flashBinary, args...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("studio failed to start: %v", err)
	}
	stop = func() { cmd.Process.Kill(); cmd.Wait() }

	url = fmt.Sprintf("http://localhost:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return url, stop
		}
		time.Sleep(200 * time.Millisecond)
	}
	return url, stop
}

// TestStudioSecurity verifies the security hardening behaviours added in the
// "feat(Security hardening)" PR: auth token enforcement and the 0.0.0.0 guard.
func TestStudioSecurity(t *testing.T) {
	// Use SQLite so no external DB is needed.
	db := Database{Name: "sqlite", URL: getEnv("SQLITE_URL", "sqlite://./test.db")}

	dir := "test_projects/studio_security"
	setupProject(t, dir, db)
	t.Cleanup(func() { flash(t, dir, "reset", "--force") })

	// ── 1. Auth token: unauthenticated request returns 401 ────────────────────
	t.Run("AuthToken_Unauthenticated", func(t *testing.T) {
		port := 15600
		url, stop := startStudio(t, dir, port, "--auth-token", "supersecret")
		defer stop()

		resp, err := http.Get(url + "/api/tables")
		if err != nil {
			t.Fatalf("GET /api/tables: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("unauthenticated: got %d, want 401", resp.StatusCode)
		}
	})

	// ── 2. Auth token: valid Bearer token returns 200 ─────────────────────────
	t.Run("AuthToken_ValidBearer", func(t *testing.T) {
		port := 15601
		token := "supersecret"
		url, stop := startStudio(t, dir, port, "--auth-token", token)
		defer stop()

		req, _ := http.NewRequest("GET", url+"/api/tables", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /api/tables: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("valid token: got %d, want 200", resp.StatusCode)
		}
	})

	// ── 3. Auth token: wrong token returns 401 ────────────────────────────────
	t.Run("AuthToken_WrongToken", func(t *testing.T) {
		port := 15602
		url, stop := startStudio(t, dir, port, "--auth-token", "supersecret")
		defer stop()

		req, _ := http.NewRequest("GET", url+"/api/tables", nil)
		req.Header.Set("Authorization", "Bearer wrongtoken")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /api/tables: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("wrong token: got %d, want 401", resp.StatusCode)
		}
	})

	// ── 4. No auth token: all requests pass through (local-only mode) ─────────
	t.Run("NoAuthToken_LocalMode", func(t *testing.T) {
		port := 15603
		url, stop := startStudio(t, dir, port) // no --auth-token
		defer stop()

		resp, err := http.Get(url + "/api/tables")
		if err != nil {
			t.Fatalf("GET /api/tables: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("no-auth local mode: got %d, want 200", resp.StatusCode)
		}
	})

	// ── 5. 0.0.0.0 without --auth-token must be refused at startup ────────────
	t.Run("BindAllInterfaces_NoToken_Refused", func(t *testing.T) {
		out, err := flash(t, dir, "studio", "--host", "0.0.0.0", "--browser=false")
		if err == nil {
			t.Errorf("expected error when binding 0.0.0.0 without token, got none\nOutput: %s", out)
		}
	})
}
