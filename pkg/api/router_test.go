package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterHealth(t *testing.T) {
	router := NewRouter("1.0.0-test")
	router.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestRouterVersion(t *testing.T) {
	router := NewRouter("1.0.0-test")
	router.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	router := NewRouter("1.0.0-test")

	// Add middleware
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(RecoveryMiddleware())

	router.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check that request ID was added
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestCORSMiddleware(t *testing.T) {
	router := NewRouter("1.0.0-test")
	router.Use(CORSMiddleware(DefaultCORSConfig()))
	router.RegisterRoutes()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check CORS headers
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin == "" {
		t.Error("expected Access-Control-Allow-Origin header to be set")
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS, got %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	router := NewRouter("1.0.0-test")
	router.Use(RecoveryMiddleware())

	// Add a handler that panics
	router.Handle("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	// Should not panic, should return 500
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic recovery, got %d", w.Code)
	}
}
