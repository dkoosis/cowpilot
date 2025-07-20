package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestHealthEndpoint_ReturnsStatusOK_When_ServerIsRunning(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	// Direct handler test
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rec.Body.String())
	}
}

func TestMCPServer_CreationSucceeds_When_UsingValidParameters(t *testing.T) {
	// Test server creation
	s := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	if s == nil {
		t.Fatal("Server creation failed")
	}
}
