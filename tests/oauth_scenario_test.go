// tests/oauth_scenario_test.go

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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/cowpilot/internal/auth"
)

// TestOAuthEndToEndScenario covers the full user journey for OAuth
func TestOAuthEndToEndScenario(t *testing.T) {
	// ... (server setup code remains the same) ...
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()
	adapter := auth.NewOAuthAdapter(testServer.URL, 9090)
	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	mcpServer.AddTool(mcp.NewTool("test-tool", mcp.WithDescription("Test tool")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("Tool executed"), nil
		})
	streamableServer := server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))
	handler := auth.Middleware(adapter)(streamableServer)
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", adapter.HandleAuthServerMetadata)
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)
	mux.Handle("/mcp", handler)
	testServer.Config.Handler = mux

	t.Run("the OAuth flow completes successfully when all steps are executed correctly", func(t *testing.T) {
		// ... (test logic from OAuthFlow_CompletesSuccessfully_When_AllStepsExecuted) ...
		resp, err := http.Get(testServer.URL + "/.well-known/oauth-authorization-server")
		if err != nil {
			t.Fatalf("Failed to get OAuth metadata: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		var metadata map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&metadata)
		if metadata["authorization_endpoint"] == "" {
			t.Error("Missing authorization endpoint")
		}
		authURL := fmt.Sprintf("%s/oauth/authorize?client_id=test&redirect_uri=%s&state=test-state", testServer.URL, url.QueryEscape("http://localhost:9090/callback"))
		resp, err = http.Get(authURL)
		if err != nil {
			t.Fatalf("Failed to get auth page: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		csrfStart := strings.Index(bodyStr, `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(bodyStr[csrfStart:], `"`)
		csrfToken := bodyStr[csrfStart : csrfStart+csrfEnd]
		form := url.Values{}
		form.Add("client_id", "test")
		form.Add("redirect_uri", "http://localhost:9090/callback")
		form.Add("client_state", "test-state")
		form.Add("csrf_state", csrfToken)
		form.Add("api_key", "test-rtm-api-key")
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit auth: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusFound {
			t.Fatalf("Expected redirect, got %d", resp.StatusCode)
		}
		location, _ := url.Parse(resp.Header.Get("Location"))
		authCode := location.Query().Get("code")
		returnedState := location.Query().Get("state")
		if authCode == "" {
			t.Error("No auth code in redirect")
		}
		if returnedState != "test-state" {
			t.Error("State mismatch")
		}
		tokenForm := url.Values{}
		tokenForm.Add("grant_type", "authorization_code")
		tokenForm.Add("code", authCode)
		req, _ = http.NewRequest("POST", testServer.URL+"/oauth/token", strings.NewReader(tokenForm.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err = http.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Failed to exchange token: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		var tokenResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&tokenResp)
		accessToken := tokenResp["access_token"].(string)
		if accessToken == "" {
			t.Error("No access token received")
		}
		mcpReq := map[string]interface{}{"jsonrpc": "2.0", "method": "tools/list", "id": 1}
		jsonData, _ := json.Marshal(mcpReq)
		req, _ = http.NewRequest("POST", testServer.URL+"/mcp", strings.NewReader(string(jsonData)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to call MCP: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("MCP call failed with status %d", resp.StatusCode)
		}
		var mcpResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&mcpResp)
		if mcpResp["result"] == nil {
			t.Error("No result in MCP response")
		}
	})
}

// TestOAuthErrorScenarios checks for graceful failure on bad inputs
func TestOAuthErrorScenarios(t *testing.T) {
	adapter := auth.NewOAuthAdapter("http://localhost:8080", 9090)

	t.Run("the authorization endpoint returns a 400 Bad Request when the API key is missing", func(t *testing.T) {
		// ... (test logic from AuthorizeEndpoint_Returns400_When_APIKeyMissing) ...
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback", nil)
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)
		body := w.Body.String()
		csrfStart := strings.Index(body, `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(body[csrfStart:], `"`)
		csrfToken := body[csrfStart : csrfStart+csrfEnd]
		form := url.Values{}
		form.Add("client_id", "test")
		form.Add("redirect_uri", "http://localhost/callback")
		form.Add("csrf_state", csrfToken)
		req = httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("the token endpoint returns a 400 Bad Request for an invalid grant type", func(t *testing.T) {
		// ... (test logic from TokenEndpoint_Returns400_When_InvalidGrantType) ...
		form := url.Values{}
		form.Add("grant_type", "password")
		form.Add("code", "test-code")
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		adapter.HandleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("the token endpoint rejects an authorization code that has already been used", func(t *testing.T) {
		// ... (test logic from TokenEndpoint_RejectsCode_When_CodeReused) ...
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback", nil)
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)
		body := w.Body.String()
		csrfStart := strings.Index(body, `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(body[csrfStart:], `"`)
		csrfToken := body[csrfStart : csrfStart+csrfEnd]
		form := url.Values{}
		form.Add("client_id", "test")
		form.Add("redirect_uri", "http://localhost/callback")
		form.Add("csrf_state", csrfToken)
		form.Add("api_key", "test-key")
		req = httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)
		location := w.Header().Get("Location")
		u, _ := url.Parse(location)
		authCode := u.Query().Get("code")
		form = url.Values{}
		form.Add("grant_type", "authorization_code")
		form.Add("code", authCode)
		req = httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		adapter.HandleToken(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("First use failed with status %d, expected 200 OK", w.Code)
		}
		req = httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		adapter.HandleToken(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Code reuse should fail with 400 Bad Request, got %d", w.Code)
		}
	})
}
