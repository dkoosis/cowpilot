package main

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/mcp-adapters/internal/testutil"
)

func TestUtilityTools(t *testing.T) {
	t.Logf("Importance: This suite validates the core, non-RTM-specific utility tools of the 'everything' server. These tests ensure the fundamental building blocks of the server's functionality are stable and reliable.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s) // Assumes setupTools registers all the tools being tested.

	t.Run("echo tool correctly prefixes and preserves content", func(t *testing.T) {
		t.Logf("  > Why it's important: Validates the most basic tool functionality, ensuring the server can receive arguments and return formatted output.")
		req := testutil.NewCallToolRequest("echo", map[string]interface{}{"message": "Hello, World!"})
		result, err := echoHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Echo tool should execute without errors")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "Echo: Hello, World!", "Echo tool should correctly prefix the input message")
	})

	t.Run("add tool correctly calculates sums", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies that numeric inputs are parsed, handled, and calculated correctly.")
		req := testutil.NewCallToolRequest("add", map[string]interface{}{"a": 5.0, "b": 3.0})
		result, err := addHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Add tool should execute without errors")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "8.00", "Add tool should correctly calculate 5 + 3 = 8")
	})

	t.Run("time tool returns time in various formats", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the server can provide and format timestamps, a critical function for logging and data synchronization.")
		// Test ISO format
		reqISO := testutil.NewCallToolRequest("get_time", map[string]interface{}{"format": "iso"})
		resultISO, errISO := timeHandler(context.Background(), reqISO)
		testutil.AssertNoError(t, errISO, "Time tool (ISO) should execute without errors")
		textISO := resultISO.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, textISO, "T", "ISO format should include T separator")
		testutil.AssertContains(t, textISO, "Z", "ISO format should be in UTC (Z)")

		// Test Unix format
		reqUnix := testutil.NewCallToolRequest("get_time", map[string]interface{}{"format": "unix"})
		resultUnix, errUnix := timeHandler(context.Background(), reqUnix)
		testutil.AssertNoError(t, errUnix, "Time tool (Unix) should execute without errors")
		textUnix := resultUnix.Content[0].(mcp.TextContent).Text
		testutil.Assert(t, len(textUnix) >= 10, "Unix timestamp should have expected length (10+ digits)")
	})
}

func TestDataHandlingTools(t *testing.T) {
	t.Logf("Importance: This suite tests tools related to data encoding, decoding, and manipulation, which are common requirements for handling various data formats and structures.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("base64 tools encode and decode data correctly", func(t *testing.T) {
		t.Logf("  > Why it's important: Tests data encoding and decoding, a common requirement for handling binary data or secrets.")
		originalText := "Hello, World!"
		reqEncode := testutil.NewCallToolRequest("base64_encode", map[string]interface{}{"text": originalText})
		resultEncode, errEncode := base64EncodeHandler(context.Background(), reqEncode)
		testutil.AssertNoError(t, errEncode, "Base64 encode should execute without errors")

		encodedText := resultEncode.Content[0].(mcp.TextContent).Text
		expectedEncoded := base64.StdEncoding.EncodeToString([]byte(originalText))
		testutil.AssertEqual(t, expectedEncoded, encodedText, "Base64 encode should produce standard base64 output")
	})

	t.Run("base64 decode tool reports errors for invalid data", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the decoder handles corrupt data gracefully without crashing the server.")
		req := testutil.NewCallToolRequest("base64_decode", map[string]interface{}{"data": "not-valid-base64!!!"})
		result, err := base64DecodeHandler(context.Background(), req)
		testutil.AssertNoError(t, err, "Handler should return a result, not a protocol error")
		testutil.AssertContains(t, result.Content[0].(mcp.TextContent).Text, "Failed to decode", "The tool's result should contain a user-friendly error message")
	})

	t.Run("string operation tool performs various transformations", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies text manipulation capabilities, essential for data transformation tasks.")
		// Test uppercase
		reqUpper := testutil.NewCallToolRequest("string_operation", map[string]interface{}{"text": "Hello World", "operation": "upper"})
		resultUpper, errUpper := stringOperationHandler(context.Background(), reqUpper)
		testutil.AssertNoError(t, errUpper, "String operation (upper) should execute without errors")
		testutil.AssertEqual(t, "HELLO WORLD", resultUpper.Content[0].(mcp.TextContent).Text, "Uppercase transformation should convert all letters to capitals")

		// Test length
		reqLength := testutil.NewCallToolRequest("string_operation", map[string]interface{}{"text": "Hello", "operation": "length"})
		resultLength, errLength := stringOperationHandler(context.Background(), reqLength)
		testutil.AssertNoError(t, errLength, "String length operation should execute")
		testutil.AssertContains(t, resultLength.Content[0].(mcp.TextContent).Text, "5 characters", "Length operation should count characters correctly")
	})

	t.Run("json formatter tool handles valid and invalid JSON", func(t *testing.T) {
		t.Logf("  > Why it's important: Validates the server's ability to process and manipulate structured data like JSON.")
		// Test prettify
		reqPretty := testutil.NewCallToolRequest("format_json", map[string]interface{}{"json": `{"name":"test","value":123}`, "minify": false})
		resultPretty, errPretty := jsonFormatterHandler(context.Background(), reqPretty)
		testutil.AssertNoError(t, errPretty, "JSON formatter (pretty) should execute without errors")
		textPretty := resultPretty.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, textPretty, "  ", "Prettified JSON should include indentation")
		testutil.AssertContains(t, textPretty, "\n", "Prettified JSON should include line breaks")

		// Test error handling
		reqMalformed := testutil.NewCallToolRequest("format_json", map[string]interface{}{"json": `{"broken": json}`})
		resultMalformed, errMalformed := jsonFormatterHandler(context.Background(), reqMalformed)
		testutil.AssertNoError(t, errMalformed, "Handler should return a result, not a protocol error")
		textMalformed := resultMalformed.Content[0].(mcp.TextContent).Text
		testutil.AssertContains(t, textMalformed, "Invalid JSON", "Error message should indicate JSON parsing failure")
	})
}

func TestAdvancedContentTools(t *testing.T) {
	t.Logf("Importance: This suite tests advanced MCP features, such as returning multiple content types and embedding structured resources, which are key differentiators of the protocol.")
	s := server.NewMCPServer("test", "1.0.0")
	setupTools(s)

	t.Run("get_test_image tool returns mixed content types", func(t *testing.T) {
		t.Logf("  > Why it's important: Tests the server's ability to return multiple, mixed-media content blocks in a single response.")
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
	})

	t.Run("get_resource_content tool embeds a resource correctly", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies a more advanced MCP feature where a tool's output can embed a structured resource, not just plain text.")
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
	})
}
