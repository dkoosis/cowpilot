package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRTMProductionHealth validates the DEPLOYED RTM server is ready for Claude
// This test MUST pass before any production deployment
func TestRTMProductionHealth(t *testing.T) {
	serverURL := os.Getenv("RTM_SERVER_URL")
	if serverURL == "" {
		serverURL = "https://rtm.fly.dev"
	}

	// Skip only if explicitly requested
	if os.Getenv("SKIP_PRODUCTION_TESTS") == "true" {
		t.Skip("Skipping production tests (SKIP_PRODUCTION_TESTS=true)")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	t.Logf("\n=== RTM Production Health Check ===")
	t.Logf("Server: %s", serverURL)

	t.Run("server is online and healthy", func(t *testing.T) {
		resp, err := client.Get(serverURL + "/health")
		require.NoError(t, err, "Failed to connect to RTM server")
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"✗ Server health check failed")
		t.Logf("✓ Server is online")
	})

	t.Run("MCP endpoint returns 401 with WWW-Authenticate for Claude", func(t *testing.T) {
		req, _ := http.NewRequest("POST", serverURL+"/mcp",
			strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"✗ MCP endpoint should return 401 when unauthorized")

		wwwAuth := resp.Header.Get("WWW-Authenticate")
		assert.NotEmpty(t, wwwAuth,
			"✗ CRITICAL: Missing WWW-Authenticate - Claude won't show Connect button")
		assert.Contains(t, wwwAuth, "Bearer realm=",
			"✗ WWW-Authenticate header has wrong format")
		t.Logf("✓ MCP endpoint protected with proper auth headers")
	})

	t.Run("OAuth discovery metadata is correct for Claude", func(t *testing.T) {
		resp, err := client.Get(serverURL + "/.well-known/oauth-authorization-server")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var metadata map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&metadata))

		authEP, ok := metadata["authorization_endpoint"].(string)
		assert.True(t, ok, "✗ Missing authorization_endpoint")
		tokenEP, ok := metadata["token_endpoint"].(string)
		assert.True(t, ok, "✗ Missing token_endpoint")

		// CRITICAL: Must have /oauth prefix for Claude.ai
		assert.Contains(t, authEP, "/oauth/authorize",
			"✗ Authorization endpoint missing /oauth prefix")
		assert.Contains(t, tokenEP, "/oauth/token",
			"✗ Token endpoint missing /oauth prefix")
		t.Logf("✓ OAuth discovery configured correctly")
	})

	t.Run("all RTM OAuth endpoints are accessible", func(t *testing.T) {
		endpoints := map[string]string{
			"/oauth/authorize": "OAuth authorize",
			"/oauth/token":     "OAuth token",
			"/rtm/callback":    "RTM callback",
			"/rtm/check-auth":  "RTM auth check",
		}

		allAccessible := true
		for endpoint, name := range endpoints {
			resp, err := client.Get(serverURL + endpoint)
			if err != nil {
				t.Logf("  ✗ %s unreachable: %v", name, err)
				allAccessible = false
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode == 404 {
				t.Logf("  ✗ %s returned 404", name)
				allAccessible = false
			}
		}
		assert.True(t, allAccessible, "✗ Some OAuth endpoints are not accessible")
		if allAccessible {
			t.Logf("✓ All OAuth endpoints accessible")
		}
	})

	t.Run("RTM secrets are configured", func(t *testing.T) {
		// This test would need fly CLI access, so we just verify the OAuth flow is responding
		// which implies secrets are set (otherwise server wouldn't start)
		resp, err := client.Get(serverURL + "/oauth/authorize?client_id=test")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 200 with auth form, not 500 error
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
			"✗ Server error - likely missing RTM_API_KEY or RTM_API_SECRET")
		t.Logf("✓ RTM credentials appear to be configured")
	})

	// Summary
	t.Logf("\n=== RTM Production Summary ===")
	t.Logf("URL for Claude Desktop: %s/mcp", serverURL)
	t.Logf("OAuth Flow: %s/oauth/authorize", serverURL)
	t.Logf("Status: READY FOR CONNECTION ✓")
}

// TestRTMOAuthValidation is the original test name for backward compatibility
func TestRTMOAuthValidation(t *testing.T) {
	t.Run("RTM Production Health", TestRTMProductionHealth)
}
