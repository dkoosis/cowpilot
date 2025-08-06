package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeOAuthCompliance verifies ALL requirements for Claude.ai MCP integration
// DO NOT MODIFY WITHOUT UNDERSTANDING IMPACT ON CLAUDE REGISTRATION
func TestClaudeOAuthCompliance(t *testing.T) {
	t.Run("Critical_401_Response", func(t *testing.T) {
		// CRITICAL: Without proper 401 response, Claude won't show Connect button
		resp, err := http.Post(serverURL+"/mcp", "application/json",
			strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"CRITICAL: /mcp must return 401 when unauthorized")

		// CRITICAL: WWW-Authenticate header MUST be present and exact
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		assert.NotEmpty(t, wwwAuth, "CRITICAL: Missing WWW-Authenticate header - Claude Connect button won't appear!")

		expectedRealm := fmt.Sprintf("Bearer realm=\"%s/.well-known/oauth-protected-resource\"", serverURL)
		assert.Equal(t, expectedRealm, wwwAuth,
			"CRITICAL: WWW-Authenticate must have exact format for Claude")
	})

	t.Run("Protected_Resource_Metadata", func(t *testing.T) {
		// RFC 9728 - Required for Claude discovery
		resp, err := http.Get(serverURL + "/.well-known/oauth-protected-resource")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"CRITICAL: Protected resource metadata must be accessible")

		var metadata map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		require.NoError(t, err)

		// Required fields
		assert.NotEmpty(t, metadata["authorization_servers"],
			"Must specify authorization_servers array")
		assert.NotEmpty(t, metadata["resource"],
			"Must specify resource URI")

		// Resource should be /mcp endpoint
		expectedResource := serverURL + "/mcp"
		assert.Equal(t, expectedResource, metadata["resource"],
			"Resource must point to /mcp endpoint")
	})

	t.Run("Auth_Server_Metadata", func(t *testing.T) {
		// RFC 8414 - Required for Claude to know OAuth endpoints
		resp, err := http.Get(serverURL + "/.well-known/oauth-authorization-server")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"CRITICAL: Auth server metadata must be accessible")

		var metadata map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		require.NoError(t, err)

		// Critical endpoints Claude needs
		assert.NotEmpty(t, metadata["authorization_endpoint"],
			"Must specify authorization_endpoint")
		assert.NotEmpty(t, metadata["token_endpoint"],
			"Must specify token_endpoint")
		assert.Contains(t, metadata["grant_types_supported"], "authorization_code",
			"Must support authorization_code grant")
		assert.Contains(t, metadata["response_types_supported"], "code",
			"Must support code response type")
	})

	t.Run("OAuth_Endpoints_Accessible", func(t *testing.T) {
		// Test /authorize endpoint exists
		resp, err := http.Get(serverURL + "/authorize")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"/authorize endpoint must exist")

		// Test /oauth/authorize endpoint exists (Claude may use either)
		resp, err = http.Get(serverURL + "/oauth/authorize")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"/oauth/authorize endpoint must exist")

		// Test /token endpoint exists
		resp, err = http.Post(serverURL+"/token", "application/x-www-form-urlencoded",
			strings.NewReader("grant_type=authorization_code&code=test"))
		require.NoError(t, err)
		_ = resp.Body.Close()
		// Should get 400 Bad Request (invalid code) not 404
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"/token endpoint must exist")

		// Test /oauth/token endpoint exists
		resp, err = http.Post(serverURL+"/oauth/token", "application/x-www-form-urlencoded",
			strings.NewReader("grant_type=authorization_code&code=test"))
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"/oauth/token endpoint must exist")
	})

	t.Run("Content_Type_With_Charset", func(t *testing.T) {
		// Claude sends "application/json; charset=utf-8"
		req, err := http.NewRequest("POST", serverURL+"/mcp",
			strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
		require.NoError(t, err)

		// Simulate Claude's exact Content-Type
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should get 401 (needs auth) not 400 (bad content-type)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"Must accept Claude's Content-Type with charset")
	})
}
