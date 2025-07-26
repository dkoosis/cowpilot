package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// TestClaudeConnector_Integration tests the complete Claude.ai integration flow
func TestClaudeConnector_Integration(t *testing.T) {
	// Create MCP server with all features
	mcpServer := server.NewMCPServer("cowpilot", "1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Add tools that Claude.ai will use
	mcpServer.AddTool(mcp.NewTool("echo",
		mcp.WithDescription("Echo the message"),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo"))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := req.Params.Arguments.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid arguments")
			}
			message, _ := args["message"].(string)
			return mcp.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
		})

	mcpServer.AddTool(mcp.NewTool("get_time", mcp.WithDescription("Get current time")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(time.Now().UTC().Format(time.RFC3339)), nil
		})

	// Add resources
	mcpServer.AddResource(mcp.NewResource("example://text/hello",
		"Hello Resource",
		mcp.WithResourceDescription("Hello resource"),
		mcp.WithMIMEType("text/plain")),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "example://text/hello",
					MIMEType: "text/plain",
					Text:     "Hello from Cowpilot!",
				},
			}, nil
		})

	// Add prompts
	simplePrompt := mcp.Prompt{
		Name:        "simple_greeting",
		Description: "Simple greeting prompt",
	}
	mcpServer.AddPrompt(simplePrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: "Please provide a friendly greeting"},
				},
			},
		}, nil
	})

	// Create streamable server with SSE support - use stateless mode for testing
	streamableServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"))

	// Setup CORS config
	corsConfig := middleware.CORSConfig{
		AllowOrigins: []string{"https://claude.ai"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Request-Id", "Accept"},
		MaxAge:       3600,
	}

	// For testing, we'll create handlers with and without auth
	mcpHandlerNoAuth := middleware.CORS(corsConfig)(streamableServer)

	// Create test server
	mux := http.NewServeMux()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Create OAuth adapter with the test server's URL
	adapter := auth.NewOAuthAdapter(testServer.URL, 9090)

	// Create a handler with auth using the new adapter
	mcpHandlerWithAuth := middleware.CORS(corsConfig)(auth.Middleware(adapter)(streamableServer))

	// Apply CORS to OAuth endpoints
	mux.Handle("/.well-known/oauth-authorization-server",
		middleware.CORS(corsConfig)(http.HandlerFunc(adapter.HandleAuthServerMetadata)))
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)

	// MCP endpoints
	mux.Handle("/mcp", mcpHandlerNoAuth)
	mux.Handle("/mcp/", mcpHandlerNoAuth)
	mux.Handle("/mcp-auth", mcpHandlerWithAuth)
	mux.Handle("/mcp-auth/", mcpHandlerWithAuth)

	t.Run("ClaudeConnector_SupportsOAuthFlow_When_ClaudeInitiatesConnection", func(t *testing.T) {
		// Step 1: Claude.ai discovers OAuth metadata
		req, _ := http.NewRequest("GET", testServer.URL+"/.well-known/oauth-authorization-server", nil)
		req.Header.Set("Origin", "https://claude.ai")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get OAuth metadata: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Verify CORS headers
		if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "https://claude.ai" {
			t.Errorf("Expected CORS origin https://claude.ai, got %s", origin)
		}

		var metadata map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&metadata)

		if metadata["authorization_endpoint"] == "" {
			t.Error("Missing authorization endpoint")
		}
		if metadata["token_endpoint"] == "" {
			t.Error("Missing token endpoint")
		}
		if metadata["grant_types_supported"] == nil {
			t.Error("Missing grant types")
		}
	})

	t.Run("ClaudeConnector_HandlesSSETransport_When_ClaudeConnectsWithEventStream", func(t *testing.T) {
		// Simulate Claude.ai using SSE transport
		req, _ := http.NewRequest("GET", testServer.URL+"/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Origin", "https://claude.ai")

		// Add timeout to prevent hanging
		client := &http.Client{
			Timeout: 2 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to connect SSE: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			t.Errorf("Expected SSE content type, got %s", contentType)
		}

		// Read initial SSE response (limited to prevent blocking)
		buf := make([]byte, 1024)
		n, _ := resp.Body.Read(buf) // Read from the body
		sseData := string(buf[:n])

		// Check if we received an SSE-like response. A valid SSE stream will not be empty.
		if !strings.Contains(sseData, "data:") {
			t.Errorf("Expected SSE data containing 'data:', but got: %s", sseData)
		} else {
			t.Logf("SSE endpoint responded correctly with: %s", sseData)
		}
	})

	t.Run("ClaudeConnector_ExecutesTools_When_NoAuth", func(t *testing.T) {
		// Test tool listing without auth
		toolReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"id":      1,
		}

		jsonData, _ := json.Marshal(toolReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/mcp", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://claude.ai")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Tool list failed with status %d: %s", resp.StatusCode, body)
		}

		var result map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&result)

		if result["result"] == nil {
			t.Error("No result in response")
		}
	})

	t.Run("ClaudeConnector_RequiresAuth_When_UsingAuthEndpoint", func(t *testing.T) {
		// Test that auth endpoint requires authentication
		toolReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"id":      1,
		}

		jsonData, _ := json.Marshal(toolReq)
		req, _ := http.NewRequest("POST", testServer.URL+"/mcp-auth", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://claude.ai")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("ClaudeConnector_ListsCapabilities_When_ClaudeRequestsInfo", func(t *testing.T) {
		// Test that Claude can discover available tools, resources, and prompts
		methods := []string{"tools/list", "resources/list", "prompts/list"}

		for _, method := range methods {
			req := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  method,
				"id":      1,
			}

			jsonData, _ := json.Marshal(req)
			httpReq, _ := http.NewRequest("POST", testServer.URL+"/mcp", bytes.NewReader(jsonData))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Origin", "https://claude.ai")

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Errorf("Failed to call %s: %v", method, err)
				continue
			}
			defer func() { _ = resp.Body.Close() }()

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Errorf("Failed to parse response for %s: %v, body: %s", method, err, body)
				continue
			}

			if result["result"] == nil {
				t.Errorf("No result for %s, response: %s", method, body)
			}
		}
	})

	t.Run("ClaudeConnector_HandlesErrors_When_InvalidRequests", func(t *testing.T) {
		// Test various error scenarios Claude might encounter
		errorTests := []struct {
			name     string
			request  map[string]interface{}
			expected int
		}{
			{
				name: "invalid_method",
				request: map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "invalid/method",
					"id":      1,
				},
				expected: -32601, // Method not found
			},
			{
				name: "missing_tool_name",
				request: map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "tools/call",
					"params": map[string]interface{}{
						// Missing name parameter
						"arguments": map[string]interface{}{},
					},
					"id": 1,
				},
				expected: -32602, // Invalid params
			},
		}

		for _, tt := range errorTests {
			t.Run(tt.name, func(t *testing.T) {
				jsonData, _ := json.Marshal(tt.request)
				req, _ := http.NewRequest("POST", testServer.URL+"/mcp", bytes.NewReader(jsonData))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", "https://claude.ai")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("Request failed: %v", err)
				}
				defer func() { _ = resp.Body.Close() }()

				body, _ := io.ReadAll(resp.Body)
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Errorf("Failed to parse error response: %v, body: %s", err, body)
					return
				}

				if errObj, ok := result["error"].(map[string]interface{}); ok {
					if code := int(errObj["code"].(float64)); code != tt.expected {
						t.Errorf("Expected error code %d, got %d", tt.expected, code)
					}
				} else {
					t.Errorf("Expected error response, got: %s", body)
				}
			})
		}
	})

	t.Run("ClaudeConnector_FullOAuthFlow_When_AuthRequired", func(t *testing.T) {
		// Test full OAuth flow
		// Step 1: Get authorization form
		resp, err := http.Get(testServer.URL + "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=test-state")
		if err != nil {
			t.Fatalf("Failed to get auth form: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Extract CSRF token from form
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		csrfStart := strings.Index(bodyStr, `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(bodyStr[csrfStart:], `"`)
		csrfToken := bodyStr[csrfStart : csrfStart+csrfEnd]

		// Step 2: Submit form with API key
		form := url.Values{}
		form.Add("client_id", "test")
		form.Add("redirect_uri", "http://localhost/callback")
		form.Add("client_state", "test-state")
		form.Add("csrf_state", csrfToken)
		form.Add("api_key", "test-api-key")

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit auth: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusFound {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected redirect, got %d: %s", resp.StatusCode, body)
		}

		// Extract auth code from redirect
		location, _ := url.Parse(resp.Header.Get("Location"))
		authCode := location.Query().Get("code")

		// Step 3: Exchange code for token
		tokenForm := url.Values{}
		tokenForm.Add("grant_type", "authorization_code")
		tokenForm.Add("code", authCode)

		resp, err = http.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Failed to exchange token: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var tokenResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&tokenResp)
		accessToken := tokenResp["access_token"].(string)

		// Step 4: Use token to call protected endpoint
		toolReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"id":      1,
		}

		jsonData, _ := json.Marshal(toolReq)
		req, _ = http.NewRequest("POST", testServer.URL+"/mcp-auth", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to call with auth: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Authenticated call failed with status %d: %s", resp.StatusCode, body)
		}
	})
}

// TestClaudeConnector_CORSCompliance verifies all CORS requirements for Claude.ai
func TestClaudeConnector_CORSCompliance(t *testing.T) {
	// Setup minimal server with CORS
	mcpServer := server.NewMCPServer("test", "1.0.0")
	streamableServer := server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))

	handler := middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://claude.ai"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Request-Id"},
		MaxAge:       3600,
	})(streamableServer)

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	t.Run("CORS_AllowsClaudeOrigin_When_PreflightRequest", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", testServer.URL+"/", nil)
		req.Header.Set("Origin", "https://claude.ai")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Preflight failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Check CORS headers
		if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "https://claude.ai" {
			t.Errorf("Expected CORS origin https://claude.ai, got %s", origin)
		}

		if maxAge := resp.Header.Get("Access-Control-Max-Age"); maxAge != "3600" {
			t.Errorf("Expected max age 3600, got %s", maxAge)
		}
	})

	t.Run("CORS_RejectsOtherOrigins_When_NotClaude", func(t *testing.T) {
		req, _ := http.NewRequest("POST", testServer.URL+"/", strings.NewReader("{}"))
		req.Header.Set("Origin", "https://evil.com")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin == "https://evil.com" {
			t.Error("Should not allow evil.com origin")
		}
	})
}

// TestClaudeConnector_NameAndDescriptionCompliance ensures naming follows Claude's requirements
func TestClaudeConnector_NameAndDescriptionCompliance(t *testing.T) {
	// Claude.ai requirements:
	// - No punctuation in connector name (spaces OK)
	// - Description must be >30 characters
	// - Config values must be strings only

	connectorName := "Cowpilot MCP Server"
	connectorDescription := "A comprehensive MCP server implementation with tools, resources, and prompts for AI assistance"

	t.Run("ConnectorName_MeetsRequirements_When_Validated", func(t *testing.T) {
		// Check no punctuation except spaces
		for _, char := range connectorName {
			if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') &&
				(char < '0' || char > '9') && char != ' ' {
				t.Errorf("Invalid character in name: %c", char)
			}
		}
	})

	t.Run("ConnectorDescription_MeetsRequirements_When_Validated", func(t *testing.T) {
		if len(connectorDescription) <= 30 {
			t.Errorf("Description too short: %d chars (need >30)", len(connectorDescription))
		}
	})
}
