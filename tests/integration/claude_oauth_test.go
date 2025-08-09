//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/mcp-adapters/internal/core"
	"github.com/vcto/mcp-adapters/internal/debug"
	"github.com/vcto/mcp-adapters/internal/rtm"
)

// TestClaudeOAuthCompliance validates CRITICAL requirements for Claude.ai registration
func TestClaudeOAuthCompliance(t *testing.T) {
	// Setup test server
	mcpServer := server.NewMCPServer("test-rtm", "1.0.0")
	rtmHandler := &rtm.Handler{}

	config := core.InfrastructureConfig{
		ServerURL:    "http://localhost:8081",
		Port:         "8081",
		AuthDisabled: false,
		RTMHandler:   rtmHandler,
		DebugStorage: &debug.NoOpStorage{},
		DebugConfig:  &debug.DebugConfig{Enabled: false},
		ServerName:   "test-rtm",
	}

	result := core.SetupInfrastructure(mcpServer, config)
	ts := httptest.NewServer(result.Server.Handler)
	defer ts.Close()

	t.Run("CRITICAL: WWW-Authenticate header on 401", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/mcp", "application/json",
			strings.NewReader(`{"jsonrpc":"2.0","method":"initialize","id":1}`))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("Expected 401, got %d", resp.StatusCode)
		}

		authHeader := resp.Header.Get("WWW-Authenticate")
		if authHeader == "" {
			t.Fatal("MISSING WWW-Authenticate header - Claude.ai won't show Connect button!")
		}
		if !strings.Contains(authHeader, "Bearer realm=") {
			t.Fatalf("Invalid WWW-Authenticate format: %s", authHeader)
		}
	})

	t.Run("OAuth discovery endpoints have correct paths", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/.well-known/oauth-authorization-server")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Discovery endpoint returned %d", resp.StatusCode)
		}

		var metadata map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
			t.Fatal(err)
		}

		// CRITICAL: Endpoints MUST have /oauth prefix
		authEndpoint := metadata["authorization_endpoint"].(string)
		tokenEndpoint := metadata["token_endpoint"].(string)

		if !strings.HasSuffix(authEndpoint, "/oauth/authorize") {
			t.Fatalf("Authorization endpoint missing /oauth prefix: %s", authEndpoint)
		}
		if !strings.HasSuffix(tokenEndpoint, "/oauth/token") {
			t.Fatalf("Token endpoint missing /oauth prefix: %s", tokenEndpoint)
		}
	})

	t.Run("Authorization endpoint exists", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/oauth/authorize?client_id=test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			t.Fatal("Authorization endpoint not found - OAuth flow will fail")
		}
	})

	t.Run("Token endpoint exists", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/oauth/token",
			"application/x-www-form-urlencoded",
			strings.NewReader("grant_type=authorization_code&code=test"))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			t.Fatal("Token endpoint not found - OAuth flow will fail")
		}
	})
}

// TestRTMOAuthFlow tests the complete OAuth flow
func TestRTMOAuthFlow(t *testing.T) {
	// This test validates the complete flow that Claude.ai uses
	ts := setupTestServer(t)
	defer ts.Close()

	// Step 1: Discovery
	t.Run("Step 1: Discovery", func(t *testing.T) {
		checkDiscovery(t, ts.URL)
	})

	// Step 2: Authorization request
	t.Run("Step 2: Authorization", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/oauth/authorize?client_id=test&redirect_uri=http://localhost:3000/callback&state=xyz", ts.URL))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "<form") && resp.StatusCode != 302 {
			t.Fatal("Authorization endpoint should return form or redirect")
		}
	})

	// Step 3: Token exchange
	t.Run("Step 3: Token exchange", func(t *testing.T) {
		// Simulate authorization code exchange
		resp, err := http.Post(ts.URL+"/oauth/token",
			"application/x-www-form-urlencoded",
			strings.NewReader("grant_type=authorization_code&code=test-code&client_id=test"))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			t.Fatal("Token endpoint not found")
		}
	})
}

func setupTestServer(t *testing.T) *httptest.Server {
	mcpServer := server.NewMCPServer("test-rtm", "1.0.0")
	rtmHandler := &rtm.Handler{}

	config := core.InfrastructureConfig{
		ServerURL:    "http://localhost:8081",
		Port:         "8081",
		AuthDisabled: false,
		RTMHandler:   rtmHandler,
		DebugStorage: &debug.NoOpStorage{},
		DebugConfig:  &debug.DebugConfig{Enabled: false},
		ServerName:   "test-rtm",
	}

	result := core.SetupInfrastructure(mcpServer, config)
	return httptest.NewServer(result.Server.Handler)
}

func checkDiscovery(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/.well-known/oauth-authorization-server")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var metadata map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&metadata)

	required := []string{
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"response_types_supported",
		"grant_types_supported",
	}

	for _, field := range required {
		if _, ok := metadata[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}
}
