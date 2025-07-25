package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vcto/cowpilot/internal/mcp"
	"github.com/vcto/cowpilot/internal/transport"
)

func TestMCP_Integration_HandlesFullLifecycle_When_ClientMakesSequentialRequests(t *testing.T) {
	// Create server and transport
	mcpServer := mcp.NewServer()
	httpTransport := transport.NewHTTPTransport(mcpServer)

	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(httpTransport.HandleMCP))
	defer ts.Close()

	t.Run("ListTools", func(t *testing.T) {
		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/list",
			ID:      1,
		}

		resp := makeRequest(t, ts.URL, req)

		if resp.Error != nil {
			t.Fatalf("Unexpected error: %v", resp.Error)
		}

		result := resp.Result.(map[string]interface{})
		tools := result["tools"].([]interface{})

		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("CallHelloTool", func(t *testing.T) {
		params, _ := json.Marshal(map[string]interface{}{
			"name":      "hello",
			"arguments": map[string]interface{}{},
		})

		req := mcp.Request{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  params,
			ID:      2,
		}

		resp := makeRequest(t, ts.URL, req)

		if resp.Error != nil {
			t.Fatalf("Unexpected error: %v", resp.Error)
		}

		result := resp.Result.(map[string]interface{})
		content := result["content"].([]interface{})

		if len(content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(content))
		}

		item := content[0].(map[string]interface{})
		if item["text"] != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got '%s'", item["text"])
		}
	})

	t.Run("HandleMultipleRequests", func(t *testing.T) {
		// Test that server can handle multiple sequential requests
		for i := 0; i < 5; i++ {
			req := mcp.Request{
				JSONRPC: "2.0",
				Method:  "tools/list",
				ID:      i + 10,
			}

			resp := makeRequest(t, ts.URL, req)

			if resp.Error != nil {
				t.Fatalf("Request %d failed: %v", i, resp.Error)
			}

			// Check response ID matches request ID
			respID, ok := resp.ID.(float64)
			if !ok {
				t.Fatalf("Response ID is not a number: %v", resp.ID)
			}

			if int(respID) != i+10 {
				t.Errorf("Expected ID %d, got %d", i+10, int(respID))
			}
		}
	})
}

func makeRequest(t *testing.T, url string, req mcp.Request) mcp.Response {
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var mcpResp mcp.Response
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return mcpResp
}
