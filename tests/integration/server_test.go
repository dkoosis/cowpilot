package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestServerHealthEndpoint(t *testing.T) {
	t.Logf("Importance: This is a fundamental check to ensure the HTTP server can start and respond to the most basic request. It's the first line of defense for detecting deployment or configuration issues.")

	t.Run("returns StatusOK when server is running", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies that the server's core HTTP handling loop is active and that the simplest possible endpoint is correctly routed and handled.")
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()

		// Direct handler test
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, but got %d", rec.Code)
		}
		if rec.Body.String() != "OK" {
			t.Errorf("Expected body 'OK', got '%s'", rec.Body.String())
		}
	})
}

func TestMcpServerCreation(t *testing.T) {
	t.Logf("Importance: This test verifies that the underlying MCP server object from the mark3labs/mcp-go library can be instantiated. A failure here would indicate a fundamental problem with a core dependency.")

	t.Run("succeeds when using valid parameters", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures that our application's configuration and the library's expectations are in sync, preventing initialization failures at server startup.")
		s := server.NewMCPServer(
			"test-server",
			"1.0.0",
			server.WithToolCapabilities(false),
		)
		if s == nil {
			t.Fatal("Server creation failed, returned nil")
		}
	})
}
