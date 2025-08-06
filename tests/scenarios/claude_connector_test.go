package scenarios

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/mcp-adapters/internal/auth"
	"github.com/vcto/mcp-adapters/internal/middleware"
)

// Helper function to create a client with a cookie jar.
func clientWithCookieJar() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err) // Should not happen in a test
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Do not follow redirects automatically
		},
	}
}

func TestClaudeConnectorIntegration(t *testing.T) {
	t.Logf("Importance: This suite performs end-to-end tests that simulate a full Claude connector lifecycle, from transport negotiation to a complete, multi-step OAuth authentication flow. Failures here indicate a critical breakdown in the user-facing integration.")

	// --- Server Setup ---
	mcpServer := server.NewMCPServer("cowpilot-test", "1.0.0", server.WithToolCapabilities(true))
	mcpServer.AddTool(mcp.NewTool("echo", mcp.WithDescription("Echo the message")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("mock echo"), nil
		})
	streamableServer := server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true), server.WithEndpointPath("/mcp"))
	corsConfig := middleware.DefaultCORSConfig()
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
		t.Logf("  > Why it's important: Verifies that the server correctly handles the Server-Sent Events (SSE) transport, which is a primary communication method for real-time updates in MCP.")
		req, _ := http.NewRequest("GET", testServer.URL+"/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		// Using a client with a timeout to simulate a client that connects and waits for events
		client := &http.Client{Timeout: 200 * time.Millisecond}
		resp, err := client.Do(req)

		// Expect a timeout because a successful SSE connection stays open.
		if err == nil {
			defer resp.Body.Close()
			t.Fatalf("Expected a timeout error for an open SSE stream, but the request completed.")
		}
		if ue, ok := err.(*url.Error); !ok || !ue.Timeout() {
			t.Fatalf("Expected a timeout error, but got a different error: %v", err)
		}

		// A successful connection will be indicated by the timeout error, but we can still check the response if one was partially received.
		if resp != nil && resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK for SSE connection, got %d", resp.StatusCode)
		}
	})

	t.Run("completes the full oauth flow when authentication is required", func(t *testing.T) {
		t.Logf("  > Why it's important: This is a full, end-to-end simulation of the user authentication and authorization process. It ensures CSRF protection, code generation, and token exchange all work together seamlessly.")
		client := clientWithCookieJar()

		// Step 1: GET /oauth/authorize to get the form and CSRF cookie
		resp, err := client.Get(testServer.URL + "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=test-state")
		if err != nil {
			t.Fatalf("Step 1 Failed: Could not GET auth form: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Extract CSRF token from the HTML form
		re := regexp.MustCompile(`name="csrf_state" value="([^"]+)"`)
		matches := re.FindStringSubmatch(string(body))
		if len(matches) < 2 {
			t.Fatalf("Step 1 Failed: Could not find CSRF token in form response")
		}
		csrfToken := matches[1]

		// Step 2: POST to /oauth/authorize with API key and CSRF token
		form := url.Values{"client_state": {"test-state"}, "csrf_state": {csrfToken}, "api_key": {"test-api-key"}, "redirect_uri": {"http://localhost/callback"}}
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Step 2 Failed: Could not POST to auth form: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusFound {
			t.Fatalf("Step 2 Failed: Expected redirect after form submission, got %d", resp.StatusCode)
		}

		// Step 3: Extract auth code from redirect
		location, _ := resp.Location()
		authCode := location.Query().Get("code")
		if authCode == "" {
			t.Fatal("Step 3 Failed: Authorization code was missing from redirect URL")
		}

		// Step 4: Exchange authorization code for an access token
		tokenForm := url.Values{"grant_type": {"authorization_code"}, "code": {authCode}}
		resp, err = client.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Step 4 Failed: Could not exchange token: %v", err)
		}
		defer resp.Body.Close()

		var tokenResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			t.Fatalf("Step 4 Failed: Could not decode token response: %v", err)
		}

		if _, ok := tokenResp["access_token"]; !ok {
			t.Fatal("Step 4 Failed: Final response did not include an access_token")
		}
	})
}
