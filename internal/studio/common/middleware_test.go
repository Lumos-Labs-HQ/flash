package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	token := "test-secret"
	handler := AuthMiddleware(token)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No token → 401
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: got %d, want 401", rr.Code)
	}

	// Wrong token → 401
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", rr.Code)
	}

	// Valid token → 200
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("valid token: got %d, want 200", rr.Code)
	}
}

func TestAuthMiddlewareEmptyToken(t *testing.T) {
	handler := AuthMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("empty token (auth disabled): got %d, want 200", rr.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Localhost origin → CORS header set
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("localhost: got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}

	// Non-localhost origin → no CORS header
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("evil origin: got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}

	// OPTIONS preflight → 204
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS: got %d, want 204", rr.Code)
	}
}

func TestMaxBytesMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 2048)
		_, _ = r.Body.Read(buf)
		w.WriteHeader(http.StatusOK)
	})
	handler := MaxBytesMiddleware(1024)(inner)

	req := httptest.NewRequest("POST", "/test", strings.NewReader(strings.Repeat("x", 2048)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized: got %d, want 413", rr.Code)
	}
}
