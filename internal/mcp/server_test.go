package mcp

import (
	"encoding/json"
	"testing"

	"github.com/vcto/cowpilot/internal/testutil"
)

func TestServerCreation(t *testing.T) {
	testutil.Section(t, "MCP Server Creation")

	testutil.Given(t, "a new MCP server is created")
	server := NewServer()

	testutil.Then(t, "server initialization and tool registration")
	testutil.Assert(t, server != nil,
		"Server initializes successfully without errors")
	testutil.AssertEqual(t, 1, len(server.tools),
		"Server registers exactly one default tool on creation")
	_, hasHello := server.tools["hello"]
	testutil.Assert(t, hasHello,
		"Server includes the 'hello' tool in its registry")

	testutil.Summary(t, "MCP server creation and default tool registration")
}

func TestToolsListEndpoint(t *testing.T) {
	testutil.Section(t, "Tools List Endpoint")

	testutil.Given(t, "an MCP server with registered tools")
	server := NewServer()

	testutil.When(t, "client requests the list of available tools")
	req := Request{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}
	resp := server.HandleRequest(req)

	testutil.Then(t, "server returns complete tool inventory")
	testutil.Assert(t, resp.Error == nil,
		"Server handles tools/list request without errors")

	result, ok := resp.Result.(map[string]interface{})
	testutil.Assert(t, ok,
		"Server returns result as a properly structured map")

	tools, ok := result["tools"].([]Tool)
	testutil.Assert(t, ok,
		"Tools list is returned in the expected []Tool format")
	testutil.AssertEqual(t, 1, len(tools),
		"Server reports correct number of available tools")
	testutil.AssertEqual(t, "hello", tools[0].Name,
		"Tool metadata includes correct tool name")

	testutil.Summary(t, "Tool discovery via tools/list endpoint")
}

func TestHelloToolExecution(t *testing.T) {
	testutil.Section(t, "Hello Tool Execution")

	testutil.Given(t, "an MCP server with the hello tool")
	server := NewServer()

	testutil.When(t, "client calls the hello tool with no arguments")
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

	testutil.Then(t, "tool executes and returns greeting")
	testutil.Assert(t, resp.Error == nil,
		"Hello tool executes without errors")

	result, ok := resp.Result.(map[string]interface{})
	testutil.Assert(t, ok,
		"Tool result is properly structured")

	content, ok := result["content"].([]map[string]string)
	testutil.Assert(t, ok && len(content) == 1,
		"Tool returns single content item in expected format")
	testutil.AssertEqual(t, "text", content[0]["type"],
		"Content type is correctly set to 'text'")
	testutil.AssertEqual(t, "Hello, World!", content[0]["text"],
		"Hello tool returns the expected greeting message")

	testutil.Summary(t, "Basic tool execution and response formatting")
}

func TestErrorHandling(t *testing.T) {
	server := NewServer()

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "UnknownMethod",
			Behavior: "Server rejects unknown JSON-RPC methods with proper error code",
			Test: func(t *testing.T) {
				req := Request{
					JSONRPC: "2.0",
					Method:  "unknown/method",
					ID:      1,
				}
				resp := server.HandleRequest(req)

				testutil.Assert(t, resp.Error != nil,
					"Server returns error for unknown method")
				testutil.AssertErrorCode(t, resp.Error.Code, -32601,
					"Error code -32601 indicates 'Method not found'")
				testutil.AssertEqual(t, "Method not found", resp.Error.Message,
					"Error message clearly states the method was not found")
			},
		},
		{
			Name:     "UnknownTool",
			Behavior: "Server rejects calls to non-existent tools",
			Test: func(t *testing.T) {
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

				testutil.Assert(t, resp.Error != nil,
					"Server returns error for non-existent tool")
				testutil.AssertErrorCode(t, resp.Error.Code, -32602,
					"Error code -32602 indicates 'Invalid params'")
			},
		},
		{
			Name:     "MalformedJSON",
			Behavior: "Server handles malformed JSON parameters gracefully",
			Test: func(t *testing.T) {
				req := Request{
					JSONRPC: "2.0",
					Method:  "tools/call",
					Params:  json.RawMessage(`{"invalid": json}`),
					ID:      1,
				}
				resp := server.HandleRequest(req)

				testutil.Assert(t, resp.Error != nil,
					"Server returns error for malformed JSON")
				testutil.AssertErrorCode(t, resp.Error.Code, -32602,
					"Error code -32602 indicates parameter parsing failure")
			},
		},
	})

	testutil.Summary(t, "JSON-RPC error handling and proper error codes")
}
