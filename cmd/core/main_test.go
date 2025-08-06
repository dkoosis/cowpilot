package main

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	testutil "github.com/vcto/mcp-adapters/internal/testutil"
	"github.comcom/mark3labs/mcp-go/mcp"
)

// Importance: This test suite validates the core, non-RTM-specific tools of the "everything" server.
// These tests ensure the fundamental building blocks of your server's functionality are stable and reliable.
// A failure here would indicate a problem in the most basic features of the server.

func TestEchoTool(t *testing.T) {
	t.Logf("Importance: Validates the most basic tool functionality, ensuring the server can receive arguments and return formatted output.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("prefixes a message and preserves content", func(t *testing.T) {
		req := testutil.NewCallToolRequest("echo", map[string]interface{}{"message": "Hello, World!"})
		result, err := echoHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Echo tool should execute without errors")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "Echo: Hello, World!", "Echo tool should correctly prefix the input message")
	})

	t.Run("handles empty messages gracefully", func(t *testing.T) {
		req := testutil.NewCallToolRequest("echo", map[string]interface{}{"message": ""})
		result, err := echoHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Echo tool should handle empty input without errors")
		testutil.AssertEqual(t, "Echo: ", result.Content[0].(mcp.TextContent).Text, "Echo tool should return only the prefix for empty input")
	})

	t.Run("preserves special characters and formatting", func(t *testing.T) {
		req := testutil.NewCallToolRequest("echo", map[string]interface{}{"message": "Test with 特殊文字 and emojis 🎉"})
		result, err := echoHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Echo tool should handle special characters")
		text := result.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, text, "特殊文字", "Echo tool should preserve Unicode characters")
		testutil.AssertContains(t, text, "🎉", "Echo tool should preserve emoji characters")
	})
}

func TestMathTool(t *testing.T) {
	t.Logf("Importance: Verifies that numeric inputs are parsed, handled, and calculated correctly.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("adds two positive numbers correctly", func(t *testing.T) {
		req := testutil.NewCallToolRequest("add", map[string]interface{}{"a": 5.0, "b": 3.0})
		result, err := addHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Add tool should execute without errors")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "8.00", "Add tool should correctly calculate 5 + 3 = 8")
	})

	t.Run("handles negative numbers correctly", func(t *testing.T) {
		req := testutil.NewCallToolRequest("add", map[string]interface{}{"a": -10.0, "b": 5.0})
		result, err := addHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Add tool should handle negative numbers")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "-5.00", "Add tool should correctly handle -10 + 5 = -5")
	})
}

func TestBase64Tools(t *testing.T) {
	t.Logf("Importance: Tests data encoding and decoding, a common requirement for handling binary data or secrets.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("encodes a string to valid Base64", func(t *testing.T) {
		req := testutil.NewCallToolRequest("base64_encode", map[string]interface{}{"text": "Hello, World!"})
		result, err := base64EncodeHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Base64 encode should execute without errors")
		expected := base64.StdEncoding.EncodeToString([]byte("Hello, World!"))
		testutil.AssertEqual(t, expected, result.Content[0].(mcp.TextContent).Text, "Base64 encode should produce standard base64 output")
	})

	t.Run("returns an error when decoding invalid Base64 data", func(t *testing.T) {
		req := testutil.NewCallToolRequest("base64_decode", map[string]interface{}{"data": "not-valid-base64!!!"})
		result, err := base64DecodeHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Handler should return a result, not a protocol error")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "Failed to decode", "The tool's result should contain a user-friendly error message")
	})
}

func TestStringOperationTool(t *testing.T) {
	t.Logf("Importance: Verifies text manipulation capabilities, essential for data transformation tasks.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("converts text to uppercase", func(t *testing.T) {
		req := testutil.NewCallToolRequest("string_operation", map[string]interface{}{"text": "Hello World", "operation": "upper"})
		result, err := stringOperationHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "String operation should execute without errors")
		testutil.AssertEqual(t, "HELLO WORLD", result.Content[0].(mcp.TextContent).Text, "Uppercase transformation should convert all letters to capitals")
	})

	t.Run("counts characters accurately", func(t *testing.T) {
		req := testutil.NewCallToolRequest("string_operation", map[string]interface{}{"text": "Hello", "operation": "length"})
		result, err := stringOperationHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "String length operation should execute")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "5 characters", "Length operation should count characters correctly")
	})
}

func TestTimeTool(t *testing.T) {
	t.Logf("Importance: Ensures the server can provide and format timestamps, a critical function for logging and data synchronization.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("returns time in Unix format", func(t *testing.T) {
		req := testutil.NewCallToolRequest("get_time", map[string]interface{}{"format": "unix"})
		result, err := timeHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Time tool should execute without errors")
		text := result.Content[0].(mcp.TextContent).Text
		testutil.Assert(t, len(text) >= 10, "Unix timestamp should have expected length (10+ digits)")
	})

	t.Run("returns time in ISO format", func(t *testing.T) {
		req := testutil.NewCallToolRequest("get_time", map[string]interface{}{"format": "iso"})
		result, err := timeHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Time tool should execute without errors")
		text := result.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, text, "T", "ISO format should include T separator")
		testutil.AssertContains(t, text, "Z", "ISO format should be in UTC (Z)")
	})
}

func TestMultiContentTool(t *testing.T) {
	t.Logf("Importance: Tests the server's ability to return multiple, mixed-media content blocks in a single response, a key MCP feature.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)
	req := testutil.NewCallToolRequest("get_test_image", map[string]interface{}{})

	result, err := getTestImageHandler(context.Background(), req)

	testutil.AssertNoError(t, err, "Image tool should execute without errors")
	testutil.AssertEqual(t, 2, len(result.Content), "Tool should return exactly 2 content items (text + image)")

	textContent, ok := result.Content[0].(mcp.TextContent)
	testutil.Assert(t, ok, "First content item should be text")
	testutil.AssertContains(t, textContent.Text, "test image", "Text content should describe the image")

	imageContent, ok := result.Content[1].(mcp.ImageContent)
	testutil.Assert(t, ok, "Second content item should be an image")
	testutil.AssertEqual(t, "image/png", imageContent.MIMEType, "Image should specify PNG MIME type")
	testutil.Assert(t, len(imageContent.Data) > 0, "Image content should include base64 data")
}

func TestJSONFormatterTool(t *testing.T) {
	t.Logf("Importance: Validates the server's ability to process and manipulate structured data like JSON.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("prettifies a JSON string", func(t *testing.T) {
		req := testutil.NewCallToolRequest("format_json", map[string]interface{}{"json": `{"name":"test","value":123}`, "minify": false})
		result, err := jsonFormatterHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "JSON formatter should execute without errors")
		text := result.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, text, "  ", "Prettified JSON should include indentation")
		testutil.AssertContains(t, text, "\n", "Prettified JSON should include line breaks")
	})

	t.Run("reports an error for malformed JSON", func(t *testing.T) {
		req := testutil.NewCallToolRequest("format_json", map[string]interface{}{"json": `{"broken": json}`})
		result, err := jsonFormatterHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Handler should return a result, not a protocol error")
		text := result.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, text, "Invalid JSON", "Error message should indicate JSON parsing failure")
	})
}

func TestResourceEmbeddingTool(t *testing.T) {
	t.Logf("Importance: Verifies a more advanced MCP feature where a tool's output can embed a structured resource, not just plain text.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s) // setupTools also calls setupResources now
	req := testutil.NewCallToolRequest("get_resource_content", map[string]interface{}{"uri": "example://text/hello"})

	result, err := getResourceContentHandler(context.Background(), req)

	testutil.AssertNoError(t, err, "Resource tool should execute without errors")
	testutil.Assert(t, len(result.Content) >= 2, "Response should include description and embedded resource")

	var foundResource bool
	for _, content := range result.Content {
		if embedded, ok := content.(mcp.EmbeddedResource); ok {
			foundResource = true
			testutil.AssertEqual(t, "resource", embedded.Type, "Embedded content should have 'resource' type")
			if textResource, ok := embedded.Resource.(mcp.TextResourceContents); ok {
				testutil.AssertEqual(t, "example://text/hello", textResource.URI, "Embedded resource should maintain correct URI")
				testutil.Assert(t, len(textResource.Text) > 0, "Embedded text resource should include content")
			}
		}
	}
	testutil.Assert(t, foundResource, "Response must include an embedded resource content block")
}
