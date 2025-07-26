// tests/claude_connector_test.go

package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/cowpilot/internal/auth"
	"github.com/vcto/cowpilot/internal/middleware"
)

// Helper function to create a client with a cookie jar.
func clientWithCookieJar() (*http.Client, http.CookieJar) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err) // Should not happen in a test
	}
	return &http.Client{Jar: jar}, jar
}

func TestClaudeConnectorIntegration(t *testing.T) {
	// --- Server Setup ---
	mcpServer := server.NewMCPServer("cowpilot", "1.0.0", server.WithToolCapabilities(true))
	mcpServer.AddTool(mcp.NewTool("echo", mcp.WithDescription("Echo the message")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("mock echo"), nil
		})
	streamableServer := server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true), server.WithEndpointPath("/mcp"))
	corsConfig := middleware.CORSConfig{
		AllowOrigins: []string{"https://claude.ai"}, AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Request-Id", "Accept"}, MaxAge: 3600,
	}
	mcpHandlerNoAuth := middleware.CORS(corsConfig)(streamableServer)
	mux := http.NewServeMux()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	adapter := auth.NewOAuthAdapter(testServer.URL, 9090)
	mcpHandlerWithAuth := middleware.CORS(corsConfig)(auth.Middleware(adapter)(streamableServer))
	mux.HandleFunc("/.well-known/oauth-authorization-server", adapter.HandleAuthServerMetadata)
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)
	mux.Handle("/mcp", mcpHandlerNoAuth)
	mux.Handle("/mcp-auth", mcpHandlerWithAuth)

	// --- Tests ---
	t.Run("handles SSE transport when a client connects with an event stream", func(t *testing.T) {
		req, _ := http.NewRequest("GET", testServer.URL+"/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		client := &http.Client{Timeout: 1 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			if ue, ok := err.(*url.Error); !ok || !ue.Timeout() {
				t.Fatalf("Expected a timeout error, but got: %v", err)
			}
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %d", resp.StatusCode)
		}
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
		}
	})

	t.Run("completes the full o-auth flow when authentication is required", func(t *testing.T) {
		client, _ := clientWithCookieJar()
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		resp, err := client.Get(testServer.URL + "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=test-state")
		if err != nil {
			t.Fatalf("Failed to get auth form: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		csrfStart := strings.Index(string(body), `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(string(body)[csrfStart:], `"`)
		if csrfStart < 26 || csrfEnd < 0 {
			t.Fatalf("Could not find CSRF token in form response")
		}
		csrfToken := string(body)[csrfStart : csrfStart+csrfEnd]

		form := url.Values{"client_id": {"test"}, "redirect_uri": {"http://localhost/callback"}, "client_state": {"test-state"}, "csrf_state": {csrfToken}, "api_key": {"test-api-key"}}
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit auth: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusFound {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected redirect, got %d: %s", resp.StatusCode, string(bodyBytes))
		}

		location, _ := url.Parse(resp.Header.Get("Location"))
		authCode := location.Query().Get("code")

		tokenForm := url.Values{"grant_type": {"authorization_code"}, "code": {authCode}}
		resp, err = client.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Failed to exchange token: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var tokenResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			t.Fatalf("Failed to decode token response: %v", err)
		}
	})
}
