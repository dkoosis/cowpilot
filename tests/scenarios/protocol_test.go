//go:build scenario

package scenarios

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Protocol conformance tests - replaces raw_sse_test.sh
// These tests verify wire-protocol correctness independent of Go SDK

func TestProtocolConformance_InvalidJSONRPCVersion(t *testing.T) {
	// This replaces the curl/jq tests from raw_sse_test.sh
	rawRequest := `{"jsonrpc":"1.0","id":1,"method":"tools/list"}`

	resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewBufferString(rawRequest))
	require.NoError(t, err, "HTTP request should succeed")
	defer func() { _ = resp.Body.Close() }()

	var jsonResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	require.NoError(t, err, "Response should be valid JSON")

	// Assert proper JSON-RPC error
	errorObj, ok := jsonResp["error"].(map[string]interface{})
	require.True(t, ok, "Response should contain an error object")

	assert.Equal(t, -32600.0, errorObj["code"], "Error code should be Invalid Request")
	assert.Contains(t, errorObj["message"], "Invalid Request", "Error message should indicate invalid request")
}

func TestProtocolConformance_MissingMethod(t *testing.T) {
	rawRequest := `{"jsonrpc":"2.0","id":2}`

	resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewBufferString(rawRequest))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var jsonResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	require.NoError(t, err)

	errorObj, ok := jsonResp["error"].(map[string]interface{})
	require.True(t, ok, "Should return error for missing method")
	assert.Equal(t, -32600.0, errorObj["code"], "Should be Invalid Request error")
}

func TestProtocolConformance_InvalidMethod(t *testing.T) {
	rawRequest := `{"jsonrpc":"2.0","id":3,"method":"nonexistent/method","params":{}}`

	resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewBufferString(rawRequest))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var jsonResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	require.NoError(t, err)

	errorObj, ok := jsonResp["error"].(map[string]interface{})
	require.True(t, ok, "Should return error for invalid method")
	assert.Equal(t, -32601.0, errorObj["code"], "Should be Method Not Found error")
}

func TestProtocolConformance_BatchRequest(t *testing.T) {
	// Test batch JSON-RPC requests
	rawRequest := `[
		{"jsonrpc":"2.0","id":"batch-1","method":"tools/list","params":{}},
		{"jsonrpc":"2.0","id":"batch-2","method":"resources/list","params":{}}
	]`

	resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewBufferString(rawRequest))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var jsonResp []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	require.NoError(t, err, "Should decode batch response")

	assert.Len(t, jsonResp, 2, "Should return 2 responses for batch request")

	// Verify each response
	for i, response := range jsonResp {
		assert.Equal(t, "2.0", response["jsonrpc"], "Response %d should have correct JSON-RPC version", i)
		expectedID := fmt.Sprintf("batch-%d", i+1)
		assert.Equal(t, expectedID, response["id"], "Response %d should have matching ID", i)

		_, hasResult := response["result"]
		_, hasError := response["error"]
		assert.True(t, hasResult || hasError, "Response %d should have either result or error", i)
	}
}

func TestProtocolConformance_NotificationRequest(t *testing.T) {
	// Notification requests have no ID and should not return a response
	rawRequest := `{"jsonrpc":"2.0","method":"notifications/progress","params":{"progress":50}}`

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post("http://localhost:8080/mcp", "application/json", bytes.NewBufferString(rawRequest))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// For notifications, the server may return 204 No Content or empty 200 OK
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 204,
		"Notification should return 200 or 204, got %d", resp.StatusCode)
}

func TestProtocolConformance_LargePayload(t *testing.T) {
	// Test handling of large payloads
	largeString := make([]byte, 1024*100) // 100KB string
	for i := range largeString {
		largeString[i] = 'a'
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "large-1",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "echo",
			"arguments": map[string]interface{}{
				"data": string(largeString),
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var jsonResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	require.NoError(t, err, "Should handle large payload")

	assert.Equal(t, "2.0", jsonResp["jsonrpc"])
	assert.Equal(t, "large-1", jsonResp["id"])
}

func TestProtocolConformance_ContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expectError bool
	}{
		{"Valid JSON", "application/json", false},
		{"Valid JSON with charset", "application/json; charset=utf-8", false},
		{"Invalid content type", "text/plain", true},
		{"Missing content type", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://localhost:8080/mcp",
				bytes.NewBufferString(`{"jsonrpc":"2.0","id":"ct-1","method":"tools/list"}`))
			require.NoError(t, err)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			if tt.expectError {
				assert.NotEqual(t, http.StatusOK, resp.StatusCode,
					"Should reject request with content-type: %s", tt.contentType)
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode,
					"Should accept request with content-type: %s", tt.contentType)
			}
		})
	}
}

func TestProtocolConformance_Timeouts(t *testing.T) {
	// Test that server properly handles client timeouts
	client := &http.Client{Timeout: 100 * time.Millisecond}

	// Create a request that would take longer than timeout
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "timeout-1",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "slow_operation",
			"arguments": map[string]interface{}{
				"delay_ms": 500,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	_, err = client.Post("http://localhost:8080/mcp", "application/json", bytes.NewReader(jsonData))

	// Should timeout on client side
	assert.Error(t, err, "Request should timeout")
	assert.Contains(t, err.Error(), "timeout", "Error should be timeout-related")
}

func TestProtocolConformance_SSEStream(t *testing.T) {
	// Test Server-Sent Events streaming if supported
	t.Skip("SSE streaming tests require special client - implement if SSE is enabled")
}

func TestProtocolConformance_ErrorCodes(t *testing.T) {
	tests := []struct {
		name         string
		request      string
		expectedCode float64
		description  string
	}{
		{
			"Parse Error",
			`{invalid json}`,
			-32700,
			"Malformed JSON should return Parse Error",
		},
		{
			"Invalid Request",
			`{"jsonrpc":"1.0","method":"test"}`,
			-32600,
			"Wrong JSON-RPC version should return Invalid Request",
		},
		{
			"Method Not Found",
			`{"jsonrpc":"2.0","id":1,"method":"invalid/method"}`,
			-32601,
			"Unknown method should return Method Not Found",
		},
		{
			"Invalid Params",
			`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":"not_an_object"}`,
			-32602,
			"Invalid params type should return Invalid Params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post("http://localhost:8080/mcp", "application/json",
				bytes.NewBufferString(tt.request))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			var jsonResp map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&jsonResp)

			if errorObj, ok := jsonResp["error"].(map[string]interface{}); ok {
				assert.Equal(t, tt.expectedCode, errorObj["code"], tt.description)
			} else {
				t.Errorf("Expected error response for %s", tt.name)
			}
		})
	}
}
