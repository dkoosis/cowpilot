package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/cowpilot/internal/auth"
)

func TestOAuthFlow_CompleteScenario(t *testing.T) {
	// Create test MCP server
	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	mcpServer.AddTool(mcp.NewTool("test-tool", mcp.WithDescription("Test tool")), 
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("Tool executed"), nil
		})
	
	// Create OAuth adapter
	adapter := auth.NewOAuthAdapter("http://localhost:8080", 9090)
	
	// Create test server with OAuth middleware
	streamableServer := server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))
	handler := auth.Middleware(adapter)(streamableServer)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", adapter.HandleAuthServerMetadata)
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)
	mux.Handle("/mcp", handler)
	
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	
	// Update adapter URL to test server
	adapter.serverURL = testServer.URL
	
	t.Run("OAuthFlow_CompletesSuccessfully_When_AllStepsExecuted", func(t *testing.T) {
		// Step 1: Discover OAuth metadata
		resp, err := http.Get(testServer.URL + "/.well-known/oauth-authorization-server")
		if err != nil {
			t.Fatalf("Failed to get OAuth metadata: %v", err)
		}
		defer resp.Body.Close()
		
		var metadata map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&metadata)
		
		if metadata["authorization_endpoint"] == "" {
			t.Error("Missing authorization endpoint")
		}
		
		// Step 2: Start authorization flow
		authURL := fmt.Sprintf("%s/oauth/authorize?client_id=test&redirect_uri=%s&state=test-state",
			testServer.URL, url.QueryEscape("http://localhost:9090/callback"))
		
		resp, err = http.Get(authURL)
		if err != nil {
			t.Fatalf("Failed to get auth page: %v", err)
		}
		defer resp.Body.Close()
		
		// Extract CSRF token from form
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		
		csrfStart := strings.Index(bodyStr, `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(bodyStr[csrfStart:], `"`)
		csrfToken := bodyStr[csrfStart : csrfStart+csrfEnd]
		
		// Step 3: Submit authorization with API key
		form := url.Values{}
		form.Add("client_id", "test")
		form.Add("redirect_uri", "http://localhost:9090/callback")
		form.Add("client_state", "test-state")
		form.Add("csrf_state", csrfToken)
		form.Add("api_key", "test-rtm-api-key")
		
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		}
		
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit auth: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("Expected redirect, got %d", resp.StatusCode)
		}
		
		// Extract auth code from redirect
		location, _ := url.Parse(resp.Header.Get("Location"))
		authCode := location.Query().Get("code")
		returnedState := location.Query().Get("state")
		
		if authCode == "" {
			t.Error("No auth code in redirect")
		}
		if returnedState != "test-state" {
			t.Error("State mismatch")
		}
		
		// Step 4: Exchange code for token
		tokenForm := url.Values{}
		tokenForm.Add("grant_type", "authorization_code")
		tokenForm.Add("code", authCode)
		
		req, _ = http.NewRequest("POST", testServer.URL+"/oauth/token", strings.NewReader(tokenForm.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		
		resp, err = http.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Failed to exchange token: %v", err)
		}
		defer resp.Body.Close()
		
		var tokenResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&tokenResp)
		
		accessToken := tokenResp["access_token"].(string)
		if accessToken == "" {
			t.Error("No access token received")
		}
		
		// Step 5: Use token to call MCP endpoint
		mcpReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"id":      1,
		}
		
		jsonData, _ := json.Marshal(mcpReq)
		req, _ = http.NewRequest("POST", testServer.URL+"/mcp", strings.NewReader(string(jsonData)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to call MCP: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("MCP call failed with status %d", resp.StatusCode)
		}
		
		// Verify we got tools list
		var mcpResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&mcpResp)
		
		if mcpResp["result"] == nil {
			t.Error("No result in MCP response")
		}
	})
}

func TestOAuthFlow_ErrorScenarios(t *testing.T) {
	adapter := auth.NewOAuthAdapter("http://localhost:8080", 9090)
	
	tests := []struct {
		name     string
		scenario func(t *testing.T, adapter *auth.OAuthAdapter)
	}{
		{
			name: "AuthorizeEndpoint_Returns400_When_APIKeyMissing",
			scenario: func(t *testing.T, adapter *auth.OAuthAdapter) {
				csrfToken := adapter.callbackServer.GenerateStateToken("test")
				
				form := url.Values{}
				form.Add("client_id", "test")
				form.Add("redirect_uri", "http://localhost/callback")
				form.Add("csrf_state", csrfToken)
				// Missing api_key
				
				req := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				
				adapter.HandleAuthorize(w, req)
				
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400, got %d", w.Code)
				}
			},
		},
		{
			name: "TokenEndpoint_Returns400_When_InvalidGrantType",
			scenario: func(t *testing.T, adapter *auth.OAuthAdapter) {
				form := url.Values{}
				form.Add("grant_type", "password") // Invalid
				form.Add("code", "test-code")
				
				req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				
				adapter.HandleToken(w, req)
				
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400, got %d", w.Code)
				}
			},
		},
		{
			name: "TokenEndpoint_RejectsCode_When_CodeReused",
			scenario: func(t *testing.T, adapter *auth.OAuthAdapter) {
				// Add auth code
				adapter.authCodes["reuse-code"] = &auth.AuthCode{
					Code:      "reuse-code",
					RTMAPIKey: "test-key",
					ExpiresAt: time.Now().Add(5 * time.Minute),
				}
				
				form := url.Values{}
				form.Add("grant_type", "authorization_code")
				form.Add("code", "reuse-code")
				
				// First use - should succeed
				req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				
				adapter.HandleToken(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("First use failed: %d", w.Code)
				}
				
				// Second use - should fail
				req = httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w = httptest.NewRecorder()
				
				adapter.HandleToken(w, req)
				if w.Code != http.StatusBadRequest {
					t.Errorf("Code reuse should fail, got %d", w.Code)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.scenario(t, adapter)
		})
	}
}
