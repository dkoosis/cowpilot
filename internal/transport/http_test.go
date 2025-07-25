package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vcto/cowpilot/internal/mcp"
)

func TestHTTP_Transport_ReturnsSuccess_When_RequestIsValid(t *testing.T) {
	server := mcp.NewServer()
	transport := NewHTTPTransport(server)

	// Create test request
	reqBody := mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	transport.HandleMCP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp mcp.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}

func TestHTTP_Transport_ReturnsMethodNotAllowed_When_RequestMethodIsGET(t *testing.T) {
	server := mcp.NewServer()
	transport := NewHTTPTransport(server)

	req := httptest.NewRequest("GET", "/mcp", nil)
	rec := httptest.NewRecorder()

	transport.HandleMCP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}
}

func TestHTTP_Transport_ReturnsParseError_When_RequestBodyIsInvalidJSON(t *testing.T) {
	server := mcp.NewServer()
	transport := NewHTTPTransport(server)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	transport.HandleMCP(rec, req)

	var resp mcp.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp.Error.Code != -32700 {
		t.Errorf("Expected error code -32700, got %d", resp.Error.Code)
	}
}

func TestHTTP_Transport_ReturnsInvalidRequestError_When_JSON_RPC_VersionIsWrong(t *testing.T) {
	server := mcp.NewServer()
	transport := NewHTTPTransport(server)

	reqBody := map[string]interface{}{
		"jsonrpc": "1.0", // Invalid version
		"method":  "tools/list",
		"id":      1,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	transport.HandleMCP(rec, req)

	var resp mcp.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp.Error.Code != -32600 {
		t.Errorf("Expected error code -32600, got %d", resp.Error.Code)
	}
}

func TestStream_Reader_ReadsMultipleMessages_When_StreamContainsMultipleJSON_Objects(t *testing.T) {
	messages := []mcp.Request{
		{JSONRPC: "2.0", Method: "tools/list", ID: 1},
		{JSONRPC: "2.0", Method: "tools/call", ID: 2},
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, msg := range messages {
		_ = encoder.Encode(msg)
	}

	reader := NewStreamReader(&buf)

	for i, expected := range messages {
		msg, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read message %d: %v", i, err)
		}

		if msg.Method != expected.Method {
			t.Errorf("Message %d: expected method %s, got %s", i, expected.Method, msg.Method)
		}
	}
}
