package scenarios

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverURL string

func init() {
	serverURL = os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
}

func TestProtocolConformance(t *testing.T) {
	t.Logf("Importance: This suite acts as a low-level contract test, verifying that the server correctly implements the JSON-RPC 2.0 specification, which is the foundational transport protocol for all MCP communication. Failures here mean no client can reliably talk to our server.")

	t.Run("returns Invalid Request error for wrong JSON-RPC version", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the server correctly rejects requests using outdated protocol versions, preventing compatibility issues.")
		rawRequest := `{"jsonrpc":"1.0","id":1,"method":"tools/list"}`

		resp, err := http.Post(serverURL+"/mcp", "application/json", bytes.NewBufferString(rawRequest))
		require.NoError(t, err, "HTTP request should succeed")
		defer func() { _ = resp.Body.Close() }()

		var jsonResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err, "Response should be valid JSON")

		errorObj, ok := jsonResp["error"].(map[string]interface{})
		require.True(t, ok, "Response should contain an error object")

		assert.Equal(t, -32600.0, errorObj["code"], "Error code should be Invalid Request (-32600)")
	})

	t.Run("returns Method Not Found for an invalid method", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies the server correctly identifies and rejects calls to unsupported methods, which is crucial for security and stability.")
		rawRequest := `{"jsonrpc":"2.0","id":3,"method":"nonexistent/method","params":{}}`

		resp, err := http.Post(serverURL+"/mcp", "application/json", bytes.NewBufferString(rawRequest))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var jsonResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		errorObj, ok := jsonResp["error"].(map[string]interface{})
		require.True(t, ok, "Should return an error for an invalid method")
		assert.Equal(t, -32601.0, errorObj["code"], "Should be Method Not Found (-32601)")
	})

	t.Run("returns Invalid Params error for missing required tool parameters", func(t *testing.T) {
		t.Logf("  > Why it's important: Confirms that the server's validation layer is working correctly for tool calls, preventing malformed requests from reaching business logic.")
		// This request is missing the required 'name' parameter for tools/call
		rawRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{}}}`

		resp, err := http.Post(serverURL+"/mcp", "application/json", bytes.NewBufferString(rawRequest))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var jsonResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)

		errorObj, ok := jsonResp["error"].(map[string]interface{})
		require.True(t, ok, "Should return an error for invalid parameters")
		assert.Equal(t, -32602.0, errorObj["code"], "Should be Invalid Params (-32602)")
	})

	t.Run("handles large payloads without crashing", func(t *testing.T) {
		t.Logf("  > Why it's important: A basic stress test to ensure the server doesn't fall over when receiving a large, but valid, request, which is key for denial-of-service protection.")
		largeString := make([]byte, 1024*100) // 100KB string
		for i := range largeString {
			largeString[i] = 'a'
		}
		// Using the 'echo' tool which is expected to exist in the test server
		reqBody := map[string]interface{}{
			"jsonrpc": "2.0", "id": "large-1", "method": "tools/call",
			"params": map[string]interface{}{
				"name":      "echo",
				"arguments": map[string]interface{}{"message": string(largeString)},
			},
		}
		jsonData, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(serverURL+"/mcp", "application/json", bytes.NewReader(jsonData))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var jsonResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err, "Server should handle large payloads")
		assert.Equal(t, "large-1", jsonResp["id"])
	})
}
