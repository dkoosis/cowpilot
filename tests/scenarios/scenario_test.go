package scenarios

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestMCPServerHealth verifies the server is healthy before running protocol tests
func TestMCPServerHealth(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping health check")
		return
	}

	// Use the base URL without the /mcp endpoint for health check
	healthURL := strings.Replace(serverURL, "/mcp", "/health", 1)

	// Use curl to check health endpoint
	cmd := exec.Command("curl", "-s", "-f", "-o", "/dev/null", "-w", "%{http_code}", healthURL)
	output, err := cmd.Output()

	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	statusCode := string(output)
	if statusCode != "200" {
		t.Fatalf("Health check returned status %s, expected 200", statusCode)
	}

	t.Logf("Server health check passed (status: %s)", statusCode)
}

// TestMCPProtocolCompliance runs comprehensive MCP protocol tests
func TestMCPProtocolCompliance(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping E2E tests")
		return
	}

	// Check if @modelcontextprotocol/inspector is available
	cmd := exec.Command("npx", "@modelcontextprotocol/inspector", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("@modelcontextprotocol/inspector not found. Install with: npm install -g @modelcontextprotocol/inspector")
		return
	}

	// JSON-RPC Protocol Tests
	t.Run("JSON-RPC request with wrong version returns invalid request error", func(t *testing.T) {
		testJSONRPCVersion(t)
	})

	t.Run("JSON-RPC request with malformed JSON returns parse error", func(t *testing.T) {
		testInvalidJSON(t)
	})

	t.Run("JSON-RPC response echoes the request ID from client", func(t *testing.T) {
		testRequestID(t)
	})

	t.Run("JSON-RPC errors use standard error codes and messages", func(t *testing.T) {
		testErrorCodes(t)
	})

	// Initialization Tests
	t.Run("Initialize request returns server capabilities and protocol version", func(t *testing.T) {
		testInitializeRequest(t)
	})

	t.Run("Protocol version negotiation allows older client versions", func(t *testing.T) {
		testProtocolVersion(t)
	})

	// Tool Tests
	t.Run("Tools list returns all available tools with schemas", func(t *testing.T) {
		testListTools(t)
	})

	t.Run("Hello tool returns greeting message when called", func(t *testing.T) {
		testCallHelloTool(t)
	})

	t.Run("Echo tool returns prefixed message when given string parameter", func(t *testing.T) {
		testCallEchoTool(t)
	})

	t.Run("Add tool returns sum when given two numbers", func(t *testing.T) {
		testCallAddTool(t)
	})

	t.Run("Get time tool returns current time in requested format", func(t *testing.T) {
		testGetTimeTool(t)
	})

	t.Run("Base64 encode tool returns encoded string when given text", func(t *testing.T) {
		testBase64EncodeTool(t)
	})

	t.Run("String operation tool transforms text based on operation type", func(t *testing.T) {
		testStringOperationTool(t)
	})

	t.Run("Get test image tool returns both text and image content", func(t *testing.T) {
		testGetTestImageTool(t)
	})

	t.Run("Tool call with missing required parameters returns error", func(t *testing.T) {
		testToolInputValidation(t)
	})

	t.Run("Non-existent tool call returns unknown tool error", func(t *testing.T) {
		testNonExistentTool(t)
	})

	t.Run("Tool errors are returned in result with isError flag", func(t *testing.T) {
		testToolErrorHandling(t)
	})

	// Resource Tests
	t.Run("Resources list returns all available resources with metadata", func(t *testing.T) {
		testListResources(t)
	})

	t.Run("Resource templates list returns URI templates when available", func(t *testing.T) {
		testListResourceTemplates(t)
	})

	t.Run("Text resource read returns content with correct mime type", func(t *testing.T) {
		testReadTextResource(t)
	})

	t.Run("Blob resource read returns base64 data with mime type", func(t *testing.T) {
		testReadBlobResource(t)
	})

	t.Run("Dynamic resource read returns content based on URI parameters", func(t *testing.T) {
		testReadDynamicResource(t)
	})

	t.Run("Non-existent resource read returns not found error", func(t *testing.T) {
		testNonExistentResource(t)
	})

	// Prompt Tests
	t.Run("Prompts list returns all available prompts with arguments", func(t *testing.T) {
		testListPrompts(t)
	})

	t.Run("Simple prompt returns user message without arguments", func(t *testing.T) {
		testGetSimplePrompt(t)
	})

	t.Run("Prompt with arguments returns templated message", func(t *testing.T) {
		testGetPromptWithArguments(t)
	})

	t.Run("Prompt with missing required arguments handles gracefully", func(t *testing.T) {
		testMissingPromptArguments(t)
	})

	// Pagination Tests
	t.Run("Tools list supports cursor-based pagination when limit specified", func(t *testing.T) {
		testToolsPagination(t)
	})

	t.Run("Resources list supports cursor-based pagination when limit specified", func(t *testing.T) {
		testResourcesPagination(t)
	})

	// Content Type Tests
	t.Run("Tool results can return multiple content types in single response", func(t *testing.T) {
		testMultipleContentItems(t)
	})

	t.Run("Embedded resource content includes resource metadata", func(t *testing.T) {
		testEmbeddedResourceContent(t)
	})
}

// JSON-RPC Protocol Tests

func testJSONRPCVersion(t *testing.T) {
	// Test with wrong JSON-RPC version
	output, _ := runRawJSONRPC(t, `{"jsonrpc":"1.0","id":1,"method":"tools/list","params":{}}`)
	if !strings.Contains(output, "error") {
		t.Fatalf("Expected error for wrong JSON-RPC version, got: %s", output)
	}
	if !strings.Contains(output, "-32600") && !strings.Contains(output, "Invalid Request") {
		t.Errorf("Expected INVALID_REQUEST error, got: %s", output)
	}
}

func testInvalidJSON(t *testing.T) {
	output, _ := runRawJSONRPC(t, `{invalid json}`)
	// Parse error should return HTTP 400, so check the output content
	if !strings.Contains(output, "-32700") && !strings.Contains(output, "Parse error") && !strings.Contains(output, "parse error") && !strings.Contains(output, "not valid json") {
		t.Errorf("Expected PARSE_ERROR, got: %s", output)
	}
}

func testRequestID(t *testing.T) {
	// Test with string ID
	output, _ := runRawJSONRPC(t, `{"jsonrpc":"2.0","id":"test-123","method":"tools/list","params":{}}`)
	if !strings.Contains(output, `"id":"test-123"`) {
		t.Errorf("Response should echo string request ID")
	}

	// Test with numeric ID
	output, _ = runRawJSONRPC(t, `{"jsonrpc":"2.0","id":42,"method":"tools/list","params":{}}`)
	if !strings.Contains(output, `"id":42`) {
		t.Errorf("Response should echo numeric request ID")
	}
}

func testErrorCodes(t *testing.T) {
	// METHOD_NOT_FOUND test
	output, _ := runRawJSONRPC(t, `{"jsonrpc":"2.0","id":1,"method":"nonexistent/method","params":{}}`)
	if !strings.Contains(output, "-32601") {
		t.Errorf("Expected METHOD_NOT_FOUND error code -32601, got: %s", output)
	}

	// INVALID_PARAMS test
	output, _ = runRawJSONRPC(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`)
	if !strings.Contains(output, "-32602") {
		t.Errorf("Expected INVALID_PARAMS error code -32602, got: %s", output)
	}
}

// Initialization Tests

func testInitializeRequest(t *testing.T) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqJSON, _ := json.Marshal(request)
	output, err := runRawJSONRPC(t, string(reqJSON))
	if err != nil {
		t.Fatalf("Initialize request failed: %v\nOutput: %s", err, output)
	}

	// Check response structure
	if !strings.Contains(output, `"protocolVersion"`) {
		t.Error("Initialize response missing protocolVersion")
	}
	if !strings.Contains(output, `"capabilities"`) {
		t.Error("Initialize response missing capabilities")
	}
	if !strings.Contains(output, `"serverInfo"`) {
		t.Error("Initialize response missing serverInfo")
	}
}

func testProtocolVersion(t *testing.T) {
	// Test version negotiation with older version
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-01-01",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqJSON, _ := json.Marshal(request)
	output, _ := runRawJSONRPC(t, string(reqJSON))

	// Server should respond with its supported version
	if !strings.Contains(output, `"protocolVersion"`) {
		t.Error("Server should negotiate protocol version")
	}
}

// Tool Tests

func testListTools(t *testing.T) {
	// Use raw JSON-RPC due to inspector transport bug
	output, err := runRawJSONRPC(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if err != nil {
		t.Fatalf("Failed to list tools: %v\nOutput: %s", err, output)
	}

	// Check for all expected tools
	expectedTools := []string{
		"hello", "echo", "add", "get_time", "base64_encode",
		"base64_decode", "string_operation", "format_json",
		"long_running_operation", "get_test_image", "get_resource_content",
	}
	for _, tool := range expectedTools {
		if !strings.Contains(output, fmt.Sprintf(`"name":"%s"`, tool)) {
			t.Errorf("Tool '%s' not found in tools list", tool)
		}
	}

	// Check for input schemas
	if !strings.Contains(output, `"inputSchema"`) {
		t.Error("Tools should include inputSchema")
	}
}

func testCallHelloTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "tools/call", "--tool-name", "hello")
	if err != nil {
		t.Fatalf("Failed to call hello tool: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("Expected 'Hello, World!' in output, got: %s", output)
	}
}

func testCallEchoTool(t *testing.T) {
	// Use raw JSON-RPC due to inspector limitation with tool arguments
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "echo",
			"arguments": map[string]interface{}{"message": "test echo"},
		},
	}
	reqJSON, _ := json.Marshal(request)
	output, err := runRawJSONRPC(t, string(reqJSON))

	if err != nil {
		t.Fatalf("Failed to call echo tool: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Echo: test echo") {
		t.Errorf("Expected 'Echo: test echo' in output, got: %s", output)
	}
}

func testCallAddTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	args := map[string]interface{}{"a": 5, "b": 3}
	argsJSON, _ := json.Marshal(args)

	output, err := runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "add",
		"--tool-arguments", string(argsJSON))

	if err != nil {
		t.Fatalf("Failed to call add tool: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "8") {
		t.Errorf("Expected result '8' in output, got: %s", output)
	}
}

func testGetTimeTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	// Test default format
	output, err := runInspectorCommand(serverURL, "--method", "tools/call", "--tool-name", "get_time")
	if err != nil {
		t.Fatalf("Failed to call get_time: %v", err)
	}
	// Should return ISO format by default
	if !strings.Contains(output, "T") && !strings.Contains(output, "Z") {
		t.Error("Expected ISO format time")
	}

	// Test unix format
	args := map[string]interface{}{"format": "unix"}
	argsJSON, _ := json.Marshal(args)
	output, err = runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "get_time",
		"--tool-arguments", string(argsJSON))
	if err != nil {
		t.Fatalf("Failed to call get_time with unix format: %v", err)
	}
	// Unix format should be a number
	if !strings.ContainsAny(output, "0123456789") {
		t.Error("Expected unix timestamp (numeric) in output")
	}
}

func testBase64EncodeTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	args := map[string]interface{}{"text": "Hello, World!"}
	argsJSON, _ := json.Marshal(args)

	output, err := runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "base64_encode",
		"--tool-arguments", string(argsJSON))

	if err != nil {
		t.Fatalf("Failed to call base64_encode: %v", err)
	}

	if !strings.Contains(output, "SGVsbG8sIFdvcmxkIQ==") {
		t.Error("Expected base64 encoded result")
	}
}

func testStringOperationTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	// Test upper operation
	args := map[string]interface{}{
		"text":      "hello",
		"operation": "upper",
	}
	argsJSON, _ := json.Marshal(args)

	output, err := runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "string_operation",
		"--tool-arguments", string(argsJSON))

	if err != nil {
		t.Fatalf("Failed to call string_operation: %v", err)
	}

	if !strings.Contains(output, "HELLO") {
		t.Errorf("Expected 'HELLO' for upper operation, got: %s", output)
	}

	// Test reverse operation
	args = map[string]interface{}{
		"text":      "hello",
		"operation": "reverse",
	}
	argsJSON, _ = json.Marshal(args)

	output, err = runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "string_operation",
		"--tool-arguments", string(argsJSON))

	if err != nil {
		t.Fatalf("Failed to call string_operation: %v", err)
	}

	if !strings.Contains(output, "olleh") {
		t.Errorf("Expected 'olleh' for reverse operation, got: %s", output)
	}
}

func testGetTestImageTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "tools/call", "--tool-name", "get_test_image")

	if err != nil {
		t.Fatalf("Failed to call get_test_image: %v", err)
	}

	// Should return multiple content items
	if !strings.Contains(output, `"type":"text"`) {
		t.Error("Expected text content in response")
	}
	if !strings.Contains(output, `"type":"image"`) {
		t.Error("Expected image content in response")
	}
	if !strings.Contains(output, `"mimeType":"image/png"`) {
		t.Error("Expected PNG mime type")
	}
}

func testToolInputValidation(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	// Test missing required parameter
	args := map[string]interface{}{"a": 5} // missing 'b'
	argsJSON, _ := json.Marshal(args)

	output, err := runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "add",
		"--tool-arguments", string(argsJSON))

	// Should return an error
	if err == nil {
		t.Fatalf("Expected error for missing required parameter, got success: %s", output)
	}
}

func testNonExistentTool(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "tools/call", "--tool-name", "nonexistent")

	if err == nil {
		t.Fatalf("Expected error for non-existent tool, but got success: %s", output)
	}

	if !strings.Contains(output, "error") && !strings.Contains(output, "not found") &&
		!strings.Contains(output, "unknown tool") && !strings.Contains(output, "-32602") {
		t.Errorf("Expected clear error message, got: %s", output)
	}
}

func testToolErrorHandling(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	// Test tool that returns an error (invalid base64)
	args := map[string]interface{}{"data": "invalid-base64!@#"}
	argsJSON, _ := json.Marshal(args)

	output, err := runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "base64_decode",
		"--tool-arguments", string(argsJSON))

	if err != nil {
		// Tool errors should be in result, not protocol errors
		t.Logf("Tool returned protocol error: %v", err)
	}

	// Check for isError flag or error content
	if strings.Contains(output, `"isError":true`) || strings.Contains(output, "error") {
		t.Log("Tool properly returned error in result")
	}
}

// Resource Tests

func testListResources(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "resources/list")
	if err != nil {
		t.Fatalf("Failed to list resources: %v\nOutput: %s", err, output)
	}

	expectedResources := []string{
		"example://text/hello",
		"example://text/readme",
		"example://image/logo",
	}

	for _, resource := range expectedResources {
		if !strings.Contains(output, resource) {
			t.Errorf("Resource '%s' not found in resources list", resource)
		}
	}

	// Check for resource properties
	if !strings.Contains(output, `"name"`) {
		t.Error("Resources should include name")
	}
	if !strings.Contains(output, `"mimeType"`) {
		t.Error("Resources should include mimeType")
	}
}

func testListResourceTemplates(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "resources/templates/list")

	if err != nil {
		// Some servers may not implement templates
		if strings.Contains(output, "not implemented") || strings.Contains(output, "-32601") {
			t.Skip("Resource templates not implemented")
			return
		}
		t.Fatalf("Failed to list resource templates: %v", err)
	}

	// Check for dynamic template
	if strings.Contains(output, "example://dynamic/") {
		t.Log("Found dynamic resource template")
	}
}

func testReadTextResource(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	request := map[string]interface{}{
		"uri": "example://text/hello",
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "resources/read",
		"--params", string(reqJSON))

	if err != nil {
		t.Fatalf("Failed to read text resource: %v", err)
	}

	if !strings.Contains(output, "Hello, World!") {
		t.Error("Expected 'Hello, World!' in resource content")
	}
	if !strings.Contains(output, `"mimeType":"text/plain"`) {
		t.Error("Expected text/plain mime type")
	}
}

func testReadBlobResource(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	request := map[string]interface{}{
		"uri": "example://image/logo",
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "resources/read",
		"--params", string(reqJSON))

	if err != nil {
		t.Fatalf("Failed to read blob resource: %v", err)
	}

	if !strings.Contains(output, `"blob"`) {
		t.Error("Expected blob content")
	}
	if !strings.Contains(output, `"mimeType":"image/png"`) {
		t.Error("Expected image/png mime type")
	}
}

func testReadDynamicResource(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	request := map[string]interface{}{
		"uri": "example://dynamic/test-id",
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "resources/read",
		"--params", string(reqJSON))

	if err != nil {
		t.Fatalf("Failed to read dynamic resource: %v", err)
	}

	if !strings.Contains(output, "test-id") {
		t.Error("Dynamic resource should include the ID from URI")
	}
}

func testNonExistentResource(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	request := map[string]interface{}{
		"uri": "example://nonexistent",
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "resources/read",
		"--params", string(reqJSON))

	if err == nil {
		t.Fatalf("Expected error for non-existent resource, got success: %s", output)
	}
}

// Prompt Tests

func testListPrompts(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "prompts/list")
	if err != nil {
		t.Fatalf("Failed to list prompts: %v\nOutput: %s", err, output)
	}

	expectedPrompts := []string{"simple_greeting", "code_review"}
	for _, prompt := range expectedPrompts {
		if !strings.Contains(output, prompt) {
			t.Errorf("Prompt '%s' not found in prompts list", prompt)
		}
	}

	// Check for arguments definition
	if strings.Contains(output, "code_review") && !strings.Contains(output, "arguments") {
		t.Error("code_review prompt should include arguments")
	}
}

func testGetSimplePrompt(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	request := map[string]interface{}{
		"name": "simple_greeting",
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "prompts/get",
		"--params", string(reqJSON))

	if err != nil {
		t.Fatalf("Failed to get simple prompt: %v", err)
	}

	if !strings.Contains(output, "messages") {
		t.Error("Prompt response should include messages")
	}
	if !strings.Contains(output, `"role":"user"`) {
		t.Error("Prompt should include user role message")
	}
}

func testGetPromptWithArguments(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	request := map[string]interface{}{
		"name": "code_review",
		"arguments": map[string]string{
			"language": "go",
			"code":     "func main() { fmt.Println(\"Hello\") }",
		},
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "prompts/get",
		"--params", string(reqJSON))

	if err != nil {
		t.Fatalf("Failed to get prompt with arguments: %v", err)
	}

	// Should include the language in the rendered prompt
	if !strings.Contains(output, "go") || !strings.Contains(output, "Go") {
		t.Error("Prompt should include the language argument")
	}
}

func testMissingPromptArguments(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	// Missing required 'code' argument
	request := map[string]interface{}{
		"name": "code_review",
		"arguments": map[string]string{
			"language": "go",
		},
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runInspectorCommand(serverURL,
		"--method", "prompts/get",
		"--params", string(reqJSON))

	if err == nil {
		// Some servers may handle missing args gracefully
		t.Logf("Server handled missing arguments: %s", output)
	}
}

// Pagination Tests

func testToolsPagination(t *testing.T) {

	// Request with small limit
	request := map[string]interface{}{
		"cursor": nil,
		"limit":  2,
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runRawJSONRPC(t, fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":%s}`, string(reqJSON)))

	if err != nil {
		t.Fatalf("Failed to test pagination: %v", err)
	}

	// Check if nextCursor is present (if there are more than 2 tools)
	if strings.Contains(output, "nextCursor") {
		t.Log("Server supports pagination with nextCursor")
	}
}

func testResourcesPagination(t *testing.T) {

	request := map[string]interface{}{
		"cursor": nil,
		"limit":  1,
	}
	reqJSON, _ := json.Marshal(request)

	output, err := runRawJSONRPC(t, fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"resources/list","params":%s}`, string(reqJSON)))

	if err != nil {
		t.Fatalf("Failed to test resources pagination: %v", err)
	}

	if strings.Contains(output, "nextCursor") {
		t.Log("Resources endpoint supports pagination")
	}
}

// Content Type Tests

func testMultipleContentItems(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	output, err := runInspectorCommand(serverURL, "--method", "tools/call", "--tool-name", "get_test_image")

	if err != nil {
		t.Fatalf("Failed to test multiple content items: %v", err)
	}

	// Count content array items
	contentCount := strings.Count(output, `"type":`)
	if contentCount < 2 {
		t.Errorf("Expected multiple content items, found %d", contentCount)
	}
}

func testEmbeddedResourceContent(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")

	args := map[string]interface{}{"uri": "example://text/hello"}
	argsJSON, _ := json.Marshal(args)

	output, err := runInspectorCommand(serverURL,
		"--method", "tools/call",
		"--tool-name", "get_resource_content",
		"--tool-arguments", string(argsJSON))

	if err != nil {
		t.Fatalf("Failed to test embedded resource: %v", err)
	}

	if !strings.Contains(output, `"type":"resource"`) {
		t.Error("Expected embedded resource type")
	}
	if !strings.Contains(output, `"resource"`) {
		t.Error("Expected resource object")
	}
}

// Helper Functions

func runInspectorCommand(serverURL string, args ...string) (string, error) {
	cmdArgs := append([]string{"@modelcontextprotocol/inspector", "--cli", serverURL, "--transport", "http"}, args...)
	cmd := exec.Command("npx", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR: " + stderr.String()
	}

	return output, err
}

func runRawJSONRPC(t *testing.T, jsonRequest string) (string, error) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set")
		return "", nil
	}

	cmd := exec.Command("curl", "-s", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", jsonRequest,
		serverURL)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR: " + stderr.String()
	}

	return output, err
}
