package main

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	testutil "github.com/vcto/mcp-adapters/internal/testing"
)

func TestEchoTool(t *testing.T) {
	testutil.Section(t, "Echo Tool Functionality")

	testutil.Given(t, "an MCP server with echo tool")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "BasicEcho",
			Behavior: "Echo tool prefixes message with 'Echo: ' and preserves content",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("echo", map[string]interface{}{
					"message": "Hello, World!",
				})
				result, err := echoHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Echo tool executes without errors")
				testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "Echo: Hello, World!",
					"Echo tool correctly prefixes and preserves the input message")
			},
		},
		{
			Name:     "EmptyMessage",
			Behavior: "Echo tool handles empty messages gracefully",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("echo", map[string]interface{}{
					"message": "",
				})
				result, err := echoHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Echo tool handles empty input without errors")
				testutil.AssertEqual(t, "Echo: ", result.Content[0].(mcp.TextContent).Text,
					"Echo tool returns 'Echo: ' for empty input")
			},
		},
		{
			Name:     "SpecialCharacters",
			Behavior: "Echo tool preserves special characters and formatting",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("echo", map[string]interface{}{
					"message": "Test with 特殊文字 and emojis 🎉",
				})
				result, err := echoHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Echo tool handles special characters")
				text := result.Content[0].(mcp.TextContent).Text
				testutil.AssertContains(t, text, "特殊文字",
					"Echo tool preserves Unicode characters correctly")
				testutil.AssertContains(t, text, "🎉",
					"Echo tool preserves emoji characters correctly")
			},
		},
	})

	testutil.Summary(t, "Echo tool message handling and character preservation")
}

func TestMathTools(t *testing.T) {
	testutil.Section(t, "Mathematical Operations")

	testutil.Given(t, "an MCP server with add tool")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "Addition",
			Behavior: "Add tool performs accurate addition",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("add", map[string]interface{}{
					"a": 5.0,
					"b": 3.0,
				})
				result, err := addHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Add tool executes without errors")
				testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "8.00",
					"Add tool correctly calculates 5 + 3 = 8")
			},
		},
		{
			Name:     "NegativeNumbers",
			Behavior: "Add tool handles negative numbers correctly",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("add", map[string]interface{}{
					"a": -10.0,
					"b": 5.0,
				})
				result, err := addHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Add tool handles negative numbers")
				testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "-5.00",
					"Add tool correctly handles negative numbers: -10 + 5 = -5")
			},
		},
	})

	testutil.Summary(t, "Mathematical operations with various number types")
}

func TestBase64Tools(t *testing.T) {
	testutil.Section(t, "Base64 Encoding/Decoding")

	testutil.Given(t, "an MCP server with base64 tools")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "Encoding",
			Behavior: "Base64 encode produces valid base64 output",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("base64_encode", map[string]interface{}{
					"text": "Hello, World!",
				})
				result, err := base64EncodeHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Base64 encode executes without errors")
				expected := base64.StdEncoding.EncodeToString([]byte("Hello, World!"))
				testutil.AssertEqual(t, expected, result.Content[0].(mcp.TextContent).Text,
					"Base64 encode produces standard base64 encoding")
			},
		},
		{
			Name:     "InvalidDecode",
			Behavior: "Base64 decode handles invalid input gracefully",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("base64_decode", map[string]interface{}{
					"data": "not-valid-base64!!!",
				})
				result, err := base64DecodeHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Handler returns result even with invalid input")
				testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "Failed to decode",
					"Base64 decode returns error message for invalid base64 input")
			},
		},
	})

	testutil.Summary(t, "Base64 encoding/decoding with edge cases")
}

func TestStringOperations(t *testing.T) {
	testutil.Section(t, "String Transformation Operations")

	testutil.Given(t, "an MCP server with string operation tool")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "Uppercase",
			Behavior: "String operation converts to uppercase correctly",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("string_operation", map[string]interface{}{
					"text":      "Hello World",
					"operation": "upper",
				})
				result, err := stringOperationHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "String operation executes without errors")
				testutil.AssertEqual(t, "HELLO WORLD", result.Content[0].(mcp.TextContent).Text,
					"Uppercase transformation converts all letters to capitals")
			},
		},
		{
			Name:     "Length",
			Behavior: "String operation counts characters accurately",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("string_operation", map[string]interface{}{
					"text":      "Hello",
					"operation": "length",
				})
				result, err := stringOperationHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "String length operation executes")
				testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "5 characters",
					"Length operation counts all characters correctly")
			},
		},
	})

	testutil.Summary(t, "String transformation operations and validation")
}

func TestTimeTool(t *testing.T) {
	testutil.Section(t, "Time Tool Formatting")

	testutil.Given(t, "an MCP server with time tool")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "UnixFormat",
			Behavior: "Time tool returns valid Unix timestamp",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("get_time", map[string]interface{}{
					"format": "unix",
				})
				result, err := timeHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Time tool executes without errors")
				// Unix timestamp should be a number
				text := result.Content[0].(mcp.TextContent).Text
				testutil.Assert(t, len(text) >= 10,
					"Unix timestamp has expected length (10+ digits)")
			},
		},
		{
			Name:     "ISOFormat",
			Behavior: "Time tool returns valid ISO 8601 formatted date",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("get_time", map[string]interface{}{
					"format": "iso",
				})
				result, err := timeHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Time tool executes without errors")
				text := result.Content[0].(mcp.TextContent).Text
				testutil.AssertContains(t, text, "T",
					"ISO format includes T separator between date and time")
				testutil.AssertContains(t, text, "Z",
					"ISO format includes UTC timezone indicator")
			},
		},
	})

	testutil.Summary(t, "Time formatting options and default behavior")
}

func TestMultiContentResponse(t *testing.T) {
	testutil.Section(t, "Multi-Content Response Handling")

	testutil.Given(t, "an MCP server with get_test_image tool")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	req := testutil.NewCallToolRequest("get_test_image", map[string]interface{}{})

	testutil.When(t, "calling get_test_image tool")
	result, err := getTestImageHandler(context.Background(), req)

	testutil.Then(t, "tool returns multiple content items")
	testutil.AssertNoError(t, err, "Image tool executes without errors")
	testutil.AssertEqual(t, 2, len(result.Content),
		"Image tool returns exactly 2 content items (text + image)")

	// Check text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	testutil.Assert(t, ok,
		"First content item is text description")
	testutil.AssertContains(t, textContent.Text, "test image",
		"Text content describes the image")

	// Check image content
	imageContent, ok := result.Content[1].(mcp.ImageContent)
	testutil.Assert(t, ok,
		"Second content item is image data")
	testutil.AssertEqual(t, "image", imageContent.Type,
		"Image content has correct type")
	testutil.AssertEqual(t, "image/png", imageContent.MIMEType,
		"Image content specifies PNG MIME type")
	testutil.Assert(t, len(imageContent.Data) > 0,
		"Image content includes base64 data")

	testutil.Summary(t, "Multi-content responses with mixed media types")
}

func TestJSONFormatter(t *testing.T) {
	testutil.Section(t, "JSON Formatting Tool")

	testutil.Given(t, "an MCP server with JSON formatter")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	testutil.RunScenarios(t, []testutil.TestScenario{
		{
			Name:     "PrettifyJSON",
			Behavior: "JSON formatter adds proper indentation",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("format_json", map[string]interface{}{
					"json":   `{"name":"test","value":123}`,
					"minify": false,
				})
				result, err := jsonFormatterHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "JSON formatter executes without errors")
				text := result.Content[0].(mcp.TextContent).Text
				testutil.AssertContains(t, text, "  ",
					"Prettified JSON includes indentation")
				testutil.AssertContains(t, text, "\n",
					"Prettified JSON includes line breaks")
			},
		},
		{
			Name:     "InvalidJSON",
			Behavior: "JSON formatter reports errors for malformed JSON",
			Test: func(t *testing.T) {
				req := testutil.NewCallToolRequest("format_json", map[string]interface{}{
					"json": `{"broken": json}`,
				})
				result, err := jsonFormatterHandler(context.Background(), req)
				testutil.AssertNoError(t, err, "Handler returns result for invalid JSON")
				text := result.Content[0].(mcp.TextContent).Text
				testutil.AssertContains(t, text, "Invalid JSON",
					"Error message indicates JSON parsing failure")
			},
		},
	})

	testutil.Summary(t, "JSON formatting and validation")
}

func TestResourceEmbedding(t *testing.T) {
	testutil.Section(t, "Resource Embedding in Tool Responses")

	testutil.Given(t, "an MCP server with get_resource_content tool")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	req := testutil.NewCallToolRequest("get_resource_content", map[string]interface{}{
		"uri": "example://text/hello",
	})

	testutil.When(t, "requesting embedded text resource")
	result, err := getResourceContentHandler(context.Background(), req)

	testutil.Then(t, "tool returns embedded resource")
	testutil.AssertNoError(t, err, "Resource tool executes without errors")
	testutil.Assert(t, len(result.Content) >= 2,
		"Resource tool returns description and embedded resource")

	// Find embedded resource
	var foundResource bool
	for _, content := range result.Content {
		if embedded, ok := content.(mcp.EmbeddedResource); ok {
			foundResource = true
			testutil.AssertEqual(t, "resource", embedded.Type,
				"Embedded content has 'resource' type")

			if textResource, ok := embedded.Resource.(mcp.TextResourceContents); ok {
				testutil.AssertEqual(t, "example://text/hello", textResource.URI,
					"Embedded resource maintains correct URI")
				testutil.AssertEqual(t, "text/plain", textResource.MIMEType,
					"Embedded resource specifies correct MIME type")
				testutil.Assert(t, len(textResource.Text) > 0,
					"Text resource includes content")
			}
		}
	}

	testutil.Assert(t, foundResource,
		"Response includes embedded resource content")

	testutil.Summary(t, "Resource embedding in tool responses")
}
