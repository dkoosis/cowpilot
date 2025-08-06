package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeOauthCompliance verifies ALL requirements for Claude.ai MCP integration.
// This test starts its own server WITH OAuth enabled to test auth requirements.
// DO NOT MODIFY WITHOUT UNDERSTANDING IMPACT ON CLAUDE REGISTRATION
func TestClaudeOauthCompliance(t *testing.T) {
	// Start a separate server instance with OAuth ENABLED for these tests
	authServerURL := startAuthEnabledServer(t)

	t.Run("returns 401 with correct WWW-Authenticate header when unauthorized", func(t *testing.T) {
		// CRITICAL: Without proper 401 response, Claude won't show Connect button
		resp, err := http.Post(authServerURL+"/mcp", "application/json",
			strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"CRITICAL: /mcp must return 401 when unauthorized")

		wwwAuth := resp.Header.Get("WWW-Authenticate")
		assert.NotEmpty(t, wwwAuth, "CRITICAL: Missing WWW-Authenticate header - Claude Connect button won't appear!")

		expectedRealm := fmt.Sprintf("Bearer realm=\"%s/.well-known/oauth-protected-resource\"", authServerURL)
		assert.Equal(t, expectedRealm, wwwAuth,
			"CRITICAL: WWW-Authenticate must have exact format for Claude")
	})

	t.Run("provides correct protected resource metadata", func(t *testing.T) {
		// RFC 9728 - Required for Claude discovery
		resp, err := http.Get(authServerURL + "/.well-known/oauth-protected-resource")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"CRITICAL: Protected resource metadata must be accessible")

		var metadata map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		require.NoError(t, err)

		assert.NotEmpty(t, metadata["authorization_servers"], "Must specify authorization_servers array")
		assert.NotEmpty(t, metadata["resource"], "Must specify resource URI")

		expectedResource := authServerURL + "/mcp"
		assert.Equal(t, expectedResource, metadata["resource"], "Resource must point to /mcp endpoint")
	})

	t.Run("provides correct authorization server metadata", func(t *testing.T) {
		// RFC 8414 - Required for Claude to know OAuth endpoints
		resp, err := http.Get(authServerURL + "/.well-known/oauth-authorization-server")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"CRITICAL: Auth server metadata must be accessible")

		var metadata map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		require.NoError(t, err)

		assert.NotEmpty(t, metadata["authorization_endpoint"], "Must specify authorization_endpoint")
		assert.NotEmpty(t, metadata["token_endpoint"], "Must specify token_endpoint")
		assert.Contains(t, metadata["grant_types_supported"], "authorization_code", "Must support authorization_code grant")
		assert.Contains(t, metadata["response_types_supported"], "code", "Must support code response type")
	})

	t.Run("ensures critical OAuth endpoints are accessible", func(t *testing.T) {
		resp, err := http.Get(authServerURL + "/authorize")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "/authorize endpoint must exist")

		resp, err = http.Get(authServerURL + "/oauth/authorize")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "/oauth/authorize endpoint must exist")

		resp, err = http.Post(authServerURL+"/token", "application/x-www-form-urlencoded",
			strings.NewReader("grant_type=authorization_code&code=test"))
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "/token endpoint must exist")

		resp, err = http.Post(authServerURL+"/oauth/token", "application/x-www-form-urlencoded",
			strings.NewReader("grant_type=authorization_code&code=test"))
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "/oauth/token endpoint must exist")
	})

	t.Run("accepts content-type header with charset", func(t *testing.T) {
		// Claude sends "application/json; charset=utf-8"
		req, err := http.NewRequest("POST", authServerURL+"/mcp",
			strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Must accept Claude's Content-Type with charset")
	})
}

// startAuthEnabledServer starts a separate server instance with OAuth enabled
func startAuthEnabledServer(t *testing.T) string {
	// Find project root
	projectRoot := findProjectRoot()
	require.NotEmpty(t, projectRoot, "Could not find project root")

	// Build the server binary
	binaryPath := filepath.Join(projectRoot, "bin", "core-server-auth-test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, filepath.Join(projectRoot, "cmd", "core"))
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build server: %v\n%s", err, output)
	}

	// Start server WITH OAuth enabled (no --disable-auth flag)
	serverCmd := exec.Command(binaryPath)
	serverCmd.Env = append(os.Environ(),
		"FLY_APP_NAME=local-test-auth",
		"PORT=8081", // Different port to avoid conflicts
		"MCP_LOG_LEVEL=WARN",
	)

	// Capture output for debugging
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	require.NoError(t, serverCmd.Start())

	// Clean up server when test completes
	t.Cleanup(func() {
		_ = serverCmd.Process.Signal(syscall.SIGTERM)
		_ = serverCmd.Wait()
	})

	// Wait for server to be ready
	authServerURL := "http://localhost:8081"
	require.True(t, waitForAuthServer(authServerURL, 15*time.Second),
		"Auth-enabled server did not become ready in time")

	return authServerURL
}

// waitForAuthServer waits for the auth-enabled server to be ready
func waitForAuthServer(baseURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		// Check health endpoint
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()

			// Verify MCP endpoint returns 401 (auth enabled)
			mcpReq := []byte(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
			mcpResp, err := client.Post(baseURL+"/mcp", "application/json", strings.NewReader(string(mcpReq)))
			if err == nil {
				defer func() { _ = mcpResp.Body.Close() }()
				if mcpResp.StatusCode == http.StatusUnauthorized {
					return true
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
