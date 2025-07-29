package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// MCP server URL from environment or default to deployed instance
func getServerURL() string {
	if url := os.Getenv("MCP_SERVER_URL"); url != "" {
		return url
	}
	return "https://mcp-adapters.fly.dev/mcp"
}

// JSON-RPC request structure
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id"`
}

// JSON-RPC response structure
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// Helper to make JSON-RPC calls
func callMCP(t *testing.T, method string, params interface{}) json.RawMessage {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", getServerURL(), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Add auth header if testing against deployed server
	if testToken := os.Getenv("MCP_TEST_TOKEN"); testToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+testToken)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log but don't fail on close errors
			t.Logf("Warning: failed to close response body: %v", err)
		}
	}()

	// Handle auth errors gracefully
	if resp.StatusCode == 401 {
		t.Skip("Server requires authentication - set MCP_TEST_TOKEN env var or deploy with DISABLE_AUTH=true")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if rpcResp.Error != nil {
		t.Fatalf("RPC error: %v", rpcResp.Error)
	}

	return rpcResp.Result
}

func TestMCP_ToolsList(t *testing.T) {
	result := callMCP(t, "tools/list", nil)

	var response struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Failed to unmarshal tools list: %v", err)
	}

	expectedTools := []string{
		"hello", "echo", "add", "get_time", "base64_encode",
		"base64_decode", "string_operation", "format_json",
		"long_running_operation", "get_test_image", "get_resource_content",
	}

	// Check all expected tools exist
	toolMap := make(map[string]bool)
	for _, tool := range response.Tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("Missing expected tool: %s", expected)
		}
	}
}

func TestMCP_ToolCall_Hello(t *testing.T) {
	params := map[string]interface{}{
		"name":      "hello",
		"arguments": map[string]interface{}{},
	}

	result := callMCP(t, "tools/call", params)

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Failed to unmarshal tool response: %v", err)
	}

	if len(response.Content) == 0 {
		t.Fatal("No content in response")
	}

	expected := "Hello, World! This is the everything server demonstrating all MCP capabilities."
	if response.Content[0].Text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response.Content[0].Text)
	}
}

func TestMCP_ToolCall_Echo(t *testing.T) {
	params := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "Test message",
		},
	}

	result := callMCP(t, "tools/call", params)

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Failed to unmarshal tool response: %v", err)
	}

	expected := "Echo: Test message"
	if response.Content[0].Text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response.Content[0].Text)
	}
}

func TestMCP_ResourcesList(t *testing.T) {
	result := callMCP(t, "resources/list", nil)

	var response struct {
		Resources []struct {
			URI         string `json:"uri"`
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			MIMEType    string `json:"mimeType"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Failed to unmarshal resources list: %v", err)
	}

	expectedResources := []string{
		"example://text/hello",
		"example://text/readme",
		"example://image/logo",
	}

	// Check expected resources exist
	resourceMap := make(map[string]bool)
	for _, resource := range response.Resources {
		resourceMap[resource.URI] = true
	}

	for _, expected := range expectedResources {
		if !resourceMap[expected] {
			t.Errorf("Missing expected resource: %s", expected)
		}
	}
}

func TestMCP_ResourceRead(t *testing.T) {
	params := map[string]interface{}{
		"uri": "example://text/hello",
	}

	result := callMCP(t, "resources/read", params)

	var response struct {
		Contents []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
			Text     string `json:"text,omitempty"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}

	if len(response.Contents) == 0 {
		t.Fatal("No contents in response")
	}

	content := response.Contents[0]
	if content.URI != "example://text/hello" {
		t.Errorf("Wrong URI: %s", content.URI)
	}
	if content.MIMEType != "text/plain" {
		t.Errorf("Wrong MIME type: %s", content.MIMEType)
	}
	if !bytes.Contains([]byte(content.Text), []byte("Hello, World!")) {
		t.Errorf("Content doesn't contain expected text")
	}
}

func TestMCP_PromptsList(t *testing.T) {
	result := callMCP(t, "prompts/list", nil)

	var response struct {
		Prompts []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"prompts"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("Failed to unmarshal prompts list: %v", err)
	}

	expectedPrompts := []string{"simple_greeting", "code_review"}

	// Check all expected prompts exist (don't count total)
	promptMap := make(map[string]bool)
	for _, prompt := range response.Prompts {
		promptMap[prompt.Name] = true
	}

	for _, expected := range expectedPrompts {
		if !promptMap[expected] {
			t.Errorf("Missing expected prompt: %s", expected)
		}
	}
}

// TestWithInspectorCLI uses the MCP Inspector CLI if available
func TestWithInspectorCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Inspector CLI test in short mode")
	}

	// Check if inspector is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found, skipping Inspector CLI test")
	}

	serverURL := getServerURL()

	// Skip if auth required (Inspector CLI doesn't handle auth easily)
	if os.Getenv("MCP_TEST_TOKEN") == "" {
		// Test if server requires auth
		resp, err := http.Post(serverURL, "application/json", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)))
		if err == nil {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: failed to close response body: %v", err)
				}
			}()
			if resp.StatusCode == 401 {
				t.Skip("Server requires authentication - Inspector CLI test skipped")
			}
		}
	}

	// Test tools/list with Inspector
	cmd := exec.Command("npx", "@modelcontextprotocol/inspector",
		"--cli", serverURL,
		"--method", "tools/list",
		"--transport", "http")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Inspector CLI failed: %v\nOutput: %s", err, output)
	}

	// Verify output contains expected tools
	if !bytes.Contains(output, []byte("hello")) {
		t.Errorf("Output doesn't contain 'hello' tool")
	}
}

// Test runner helper
func TestMain(m *testing.M) {
	// Check if server is running - don't start our own if one is already running
	serverURL := getServerURL()

	// For local testing, assume external script handles server lifecycle
	if serverURL == "http://localhost:8080/mcp" {
		// Wait for server to be ready (started by external script)
		fmt.Println("⏳ Waiting for server to be ready...")
		for i := 0; i < 60; i++ { // Increased wait time
			// Try both IPv4 and health endpoint
			resp, err := http.Get("http://127.0.0.1:8080/health")
			if err == nil {
				if err := resp.Body.Close(); err != nil {
					fmt.Printf("Warning: failed to close response body: %v\n", err)
				}
				// Also verify MCP endpoint is responding
				client := &http.Client{Timeout: 5 * time.Second}
				testReq, _ := http.NewRequest("POST", "http://127.0.0.1:8080/mcp",
					bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)))
				testReq.Header.Set("Content-Type", "application/json")
				if testResp, testErr := client.Do(testReq); testErr == nil {
					if err := testResp.Body.Close(); err != nil {
						fmt.Printf("Warning: failed to close test response body: %v\n", err)
					}
					fmt.Println("✅ Server ready for testing (health + MCP responding)")
					break
				}
			}
			if i%10 == 0 && i > 0 {
				fmt.Printf("  Still waiting... (%d/60)\n", i)
			}
			time.Sleep(500 * time.Millisecond)
			if i == 59 {
				fmt.Println("❌ Server not ready after 30 seconds")
				os.Exit(1)
			}
		}
		os.Exit(m.Run())
	}

	// For deployed servers, just run tests directly
	os.Exit(m.Run())
}
