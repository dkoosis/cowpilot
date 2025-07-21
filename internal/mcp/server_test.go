package mcp

import (
	"encoding/json"
	"testing"
)

func TestServerCreation_SucceedsAndRegistersTools_When_NewServerIsCalled(t *testing.T) {
	server := NewServer()

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}

	if _, ok := server.tools["hello"]; !ok {
		t.Error("hello tool not registered")
	}
}

func TestServer_ReturnsToolList_When_HandlingToolsListRequest(t *testing.T) {
	server := NewServer()

	req := Request{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	tools, ok := result["tools"].([]Tool)
	if !ok {
		t.Fatal("tools is not a []Tool")
	}

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "hello" {
		t.Errorf("Expected tool name 'hello', got '%s'", tools[0].Name)
	}
}

func TestServer_ReturnsCorrectResult_When_HandlingValidToolCallRequest(t *testing.T) {
	server := NewServer()

	params, _ := json.Marshal(map[string]interface{}{
		"name":      "hello",
		"arguments": map[string]interface{}{},
	})

	req := Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  params,
		ID:      1,
	}

	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	content, ok := result["content"].([]map[string]string)
	if !ok {
		t.Fatal("content is not []map[string]string")
	}

	if len(content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(content))
	}

	if content[0]["type"] != "text" {
		t.Errorf("Expected type 'text', got '%s'", content[0]["type"])
	}

	if content[0]["text"] != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", content[0]["text"])
	}
}

func TestServer_ReturnsMethodNotFoundError_When_MethodIsUnknown(t *testing.T) {
	server := NewServer()

	req := Request{
		JSONRPC: "2.0",
		Method:  "unknown/method",
		ID:      1,
	}

	resp := server.HandleRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}

	if resp.Error.Message != "Method not found" {
		t.Errorf("Expected error message 'Method not found', got '%s'", resp.Error.Message)
	}
}

func TestServer_ReturnsInvalidParamsError_When_ToolNameIsUnknown(t *testing.T) {
	server := NewServer()

	params, _ := json.Marshal(map[string]interface{}{
		"name":      "unknown",
		"arguments": map[string]interface{}{},
	})

	req := Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  params,
		ID:      1,
	}

	resp := server.HandleRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}
}

func TestServer_ReturnsInvalidParamsError_When_ParamsAreMalformedJSON(t *testing.T) {
	server := NewServer()

	req := Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"invalid": json}`),
		ID:      1,
	}

	resp := server.HandleRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}
}
