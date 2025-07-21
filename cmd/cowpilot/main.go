package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Version information
const (
	serverName    = "cowpilot-everything"
	serverVersion = "1.0.0"
)

// Sample data for demonstrating capabilities
var (
	sampleResources = []mcp.Resource{
		{
			URI:         "example://text/hello",
			Name:        "Hello World Text",
			Description: "A simple text resource",
			MimeType:    "text/plain",
		},
		{
			URI:         "example://text/readme",
			Name:        "README",
			Description: "Project documentation",
			MimeType:    "text/markdown",
		},
		{
			URI:         "example://image/logo",
			Name:        "Logo Image",
			Description: "A small example image",
			MimeType:    "image/png",
		},
	}

	samplePrompts = []mcp.Prompt{
		{
			Name:        "simple_greeting",
			Description: "A simple greeting prompt",
		},
		{
			Name:        "code_review",
			Description: "Review code for improvements",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "language",
					Description: "Programming language",
					Required:    true,
				},
				{
					Name:        "code",
					Description: "Code to review",
					Required:    true,
				},
			},
		},
	}

	// Subscriptions tracking
	subscriptions = make(map[string]bool)
)

// Tiny example image (1x1 transparent PNG)
const tinyImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	// Add all tools
	setupTools(s)

	// Check if we're running on Fly.io or locally
	if os.Getenv("FLY_APP_NAME") != "" {
		// Run HTTP server for Fly.io
		runHTTPServer(s)
	} else {
		// Run stdio server for local development
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}
}

func setupTools(s *server.MCPServer) {
	// Hello tool (existing)
	helloTool := mcp.NewTool("hello",
		mcp.WithDescription("Says hello to the world"),
	)
	s.AddTool(helloTool, helloHandler)

	// Echo tool
	echoTool := mcp.NewTool("echo",
		mcp.WithDescription("Echoes back the input message"),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo")),
	)
	s.AddTool(echoTool, echoHandler)

	// Add numbers tool
	addTool := mcp.NewTool("add",
		mcp.WithDescription("Adds two numbers together"),
		mcp.WithNumber("a", mcp.Required(), mcp.Description("First number")),
		mcp.WithNumber("b", mcp.Required(), mcp.Description("Second number")),
	)
	s.AddTool(addTool, addHandler)

	// Get current time tool
	timeTool := mcp.NewTool("get_time",
		mcp.WithDescription("Gets the current time in various formats"),
		mcp.WithString("format", mcp.Description("Time format: 'unix', 'iso', or 'human'")),
	)
	s.AddTool(timeTool, timeHandler)

	// Base64 encode/decode tools
	encodeTool := mcp.NewTool("base64_encode",
		mcp.WithDescription("Encodes text to base64"),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to encode")),
	)
	s.AddTool(encodeTool, base64EncodeHandler)

	decodeTool := mcp.NewTool("base64_decode",
		mcp.WithDescription("Decodes base64 to text"),
		mcp.WithString("data", mcp.Required(), mcp.Description("Base64 data to decode")),
	)
	s.AddTool(decodeTool, base64DecodeHandler)

	// String manipulation tool
	stringTool := mcp.NewTool("string_operation",
		mcp.WithDescription("Performs various string operations"),
		mcp.WithString("text", mcp.Required(), mcp.Description("Input text")),
		mcp.WithString("operation", mcp.Required(), mcp.Description("Operation: 'upper', 'lower', 'reverse', 'length'")),
	)
	s.AddTool(stringTool, stringOperationHandler)

	// JSON formatter tool
	jsonTool := mcp.NewTool("format_json",
		mcp.WithDescription("Formats or minifies JSON"),
		mcp.WithString("json", mcp.Required(), mcp.Description("JSON string to format")),
		mcp.WithBoolean("minify", mcp.Description("Minify instead of prettify")),
	)
	s.AddTool(jsonTool, jsonFormatterHandler)

	// Long running operation tool
	longRunningTool := mcp.NewTool("long_running_operation",
		mcp.WithDescription("Simulates a long-running operation with progress"),
		mcp.WithNumber("duration", mcp.Description("Duration in seconds (default: 5)")),
		mcp.WithNumber("steps", mcp.Description("Number of progress steps (default: 5)")),
	)
	s.AddTool(longRunningTool, longRunningHandler)

	// List resources tool
	listResourcesTool := mcp.NewTool("list_resources",
		mcp.WithDescription("Lists available resources"),
	)
	s.AddTool(listResourcesTool, listResourcesToolHandler)

	// Read resource tool
	readResourceTool := mcp.NewTool("read_resource",
		mcp.WithDescription("Reads a specific resource"),
		mcp.WithString("uri", mcp.Required(), mcp.Description("Resource URI")),
	)
	s.AddTool(readResourceTool, readResourceToolHandler)

	// List prompts tool
	listPromptsTool := mcp.NewTool("list_prompts",
		mcp.WithDescription("Lists available prompts"),
	)
	s.AddTool(listPromptsTool, listPromptsToolHandler)

	// Get prompt tool
	getPromptTool := mcp.NewTool("get_prompt",
		mcp.WithDescription("Gets a specific prompt"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Prompt name")),
		mcp.WithObject("arguments", mcp.Description("Arguments for the prompt")),
	)
	s.AddTool(getPromptTool, getPromptToolHandler)

	// Test image tool
	getImageTool := mcp.NewTool("get_test_image",
		mcp.WithDescription("Returns a test image"),
	)
	s.AddTool(getImageTool, getTestImageHandler)

	// Get resource with embedded content
	getResourceContentTool := mcp.NewTool("get_resource_content",
		mcp.WithDescription("Gets a resource and returns it as embedded content"),
		mcp.WithString("uri", mcp.Required(), mcp.Description("Resource URI")),
	)
	s.AddTool(getResourceContentTool, getResourceContentHandler)
}

// Tool handlers
func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("Hello, World! This is the everything server demonstrating all MCP capabilities."), nil
}

func echoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, ok := request.Params.Arguments["message"].(string)
	if !ok {
		return mcp.NewToolResultError("message parameter is required"), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
}

func addHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a, ok := getNumber(request.Params.Arguments, "a")
	if !ok {
		return mcp.NewToolResultError("parameter 'a' is required and must be a number"), nil
	}
	b, ok := getNumber(request.Params.Arguments, "b")
	if !ok {
		return mcp.NewToolResultError("parameter 'b' is required and must be a number"), nil
	}
	
	result := a + b
	return mcp.NewToolResultText(fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result)), nil
}

func timeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	format, _ := request.Params.Arguments["format"].(string)
	if format == "" {
		format = "iso"
	}

	now := time.Now()
	var result string

	switch format {
	case "unix":
		result = fmt.Sprintf("%d", now.Unix())
	case "human":
		result = now.Format("Monday, January 2, 2006 3:04:05 PM MST")
	case "iso":
		fallthrough
	default:
		result = now.Format(time.RFC3339)
	}

	return mcp.NewToolResultText(result), nil
}

func base64EncodeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, ok := request.Params.Arguments["text"].(string)
	if !ok {
		return mcp.NewToolResultError("text parameter is required"), nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return mcp.NewToolResultText(encoded), nil
}

func base64DecodeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	data, ok := request.Params.Arguments["data"].(string)
	if !ok {
		return mcp.NewToolResultError("data parameter is required"), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode base64: %v", err)), nil
	}
	return mcp.NewToolResultText(string(decoded)), nil
}

func stringOperationHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, ok := request.Params.Arguments["text"].(string)
	if !ok {
		return mcp.NewToolResultError("text parameter is required"), nil
	}
	operation, ok := request.Params.Arguments["operation"].(string)
	if !ok {
		return mcp.NewToolResultError("operation parameter is required"), nil
	}

	var result string
	switch operation {
	case "upper":
		result = strings.ToUpper(text)
	case "lower":
		result = strings.ToLower(text)
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		result = string(runes)
	case "length":
		result = fmt.Sprintf("Length: %d characters, %d bytes", len([]rune(text)), len(text))
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Unknown operation: %s", operation)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func jsonFormatterHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jsonStr, ok := request.Params.Arguments["json"].(string)
	if !ok {
		return mcp.NewToolResultError("json parameter is required"), nil
	}
	minify, _ := request.Params.Arguments["minify"].(bool)

	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON: %v", err)), nil
	}

	var result []byte
	var err error
	if minify {
		result, err = json.Marshal(data)
	} else {
		result, err = json.MarshalIndent(data, "", "  ")
	}
	
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format JSON: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func longRunningHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	duration, _ := getNumber(request.Params.Arguments, "duration")
	if duration <= 0 {
		duration = 5
	}
	steps, _ := getNumber(request.Params.Arguments, "steps")
	if steps <= 0 {
		steps = 5
	}

	stepDuration := time.Duration(duration*1000/steps) * time.Millisecond
	
	// Note: Progress notifications would need to be implemented through the server API
	// For now, we just simulate the delay
	log.Printf("Starting long-running operation: %.0f seconds, %.0f steps", duration, steps)
	
	for i := 1; i <= int(steps); i++ {
		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("Operation cancelled"), nil
		case <-time.After(stepDuration):
			log.Printf("Progress: %d/%d", i, int(steps))
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf("Completed long-running operation: %.0f seconds, %.0f steps", duration, steps)), nil
}

// Resource-related tool handlers
func listResourcesToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var resourceList []string
	for _, resource := range sampleResources {
		resourceList = append(resourceList, fmt.Sprintf("- %s (%s): %s", resource.Name, resource.URI, resource.Description))
	}
	
	result := "Available resources:\n" + strings.Join(resourceList, "\n")
	return mcp.NewToolResultText(result), nil
}

func readResourceToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, ok := request.Params.Arguments["uri"].(string)
	if !ok {
		return mcp.NewToolResultError("uri parameter is required"), nil
	}
	
	switch uri {
	case "example://text/hello":
		return mcp.NewToolResultText("Hello, World! This is a simple text resource from the everything server."), nil
		
	case "example://text/readme":
		content := `# Everything Server

This is an example MCP server that implements all basic capabilities:

- **Tools**: Various utility functions
- **Resources**: Text and binary content  
- **Prompts**: Template-based interactions
- **Logging**: Server-side logging
- **Completions**: Argument suggestions

## Usage

Connect to this server using any MCP client to explore its capabilities.`
		return mcp.NewToolResultText(content), nil
		
	case "example://image/logo":
		// Return base64 image data with description
		return mcp.NewToolResultText(fmt.Sprintf("Image data (base64): %s", tinyImageBase64)), nil
		
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Resource not found: %s", uri)), nil
	}
}

// Prompt-related tool handlers
func listPromptsToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var promptList []string
	for _, prompt := range samplePrompts {
		args := ""
		if len(prompt.Arguments) > 0 {
			var argList []string
			for _, arg := range prompt.Arguments {
				req := ""
				if arg.Required {
					req = " (required)"
				}
				argList = append(argList, fmt.Sprintf("%s%s", arg.Name, req))
			}
			args = fmt.Sprintf(" [args: %s]", strings.Join(argList, ", "))
		}
		promptList = append(promptList, fmt.Sprintf("- %s: %s%s", prompt.Name, prompt.Description, args))
	}
	
	result := "Available prompts:\n" + strings.Join(promptList, "\n")
	return mcp.NewToolResultText(result), nil
}

func getPromptToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		return mcp.NewToolResultError("name parameter is required"), nil
	}
	
	arguments, _ := request.Params.Arguments["arguments"].(map[string]interface{})
	
	switch name {
	case "simple_greeting":
		result := "Prompt: simple_greeting\n" +
			"Description: A friendly greeting\n" +
			"Message: Please provide a friendly greeting for a new user joining our community."
		return mcp.NewToolResultText(result), nil
		
	case "code_review":
		language, _ := arguments["language"].(string)
		code, _ := arguments["code"].(string)
		
		if language == "" || code == "" {
			return mcp.NewToolResultError("language and code arguments are required"), nil
		}
		
		result := fmt.Sprintf("Prompt: code_review\n"+
			"Description: Code review for %s\n"+
			"Message: Please review the following %s code for improvements, potential bugs, and best practices:\n\n```%s\n%s\n```", 
			language, language, language, code)
		return mcp.NewToolResultText(result), nil
		
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Prompt not found: %s", name)), nil
	}
}

// Image handler
func getTestImageHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Decode the base64 image
	imageData, err := base64.StdEncoding.DecodeString(tinyImageBase64)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode image: %v", err)), nil
	}
	
	// Return as image content
	return &mcp.CallToolResult{
		Content: []interface{}{
			mcp.TextContent{
				Type: "text", 
				Text: "Here's a tiny test image (1x1 transparent PNG):",
			},
			mcp.ImageContent{
				Type:     "image",
				Data:     base64.StdEncoding.EncodeToString(imageData),
				MimeType: "image/png",
			},
		},
	}, nil
}

// Resource content handler - demonstrates embedded resources
func getResourceContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, ok := request.Params.Arguments["uri"].(string)
	if !ok {
		return mcp.NewToolResultError("uri parameter is required"), nil
	}
	
	// Find the resource
	var foundResource *mcp.Resource
	for _, resource := range sampleResources {
		if resource.URI == uri {
			foundResource = &resource
			break
		}
	}
	
	if foundResource == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Resource not found: %s", uri)), nil
	}
	
	// Return embedded resource content
	content := []interface{}{
		mcp.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Returning embedded resource: %s", foundResource.Name),
		},
	}
	
	// Add the actual resource content based on URI
	switch uri {
	case "example://text/hello":
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.TextResourceContents{
				URI:      uri,
				MimeType: "text/plain",
				Text:     "Hello, World! This is a simple text resource from the everything server.",
			},
		})
		
	case "example://text/readme":
		readmeContent := `# Everything Server

This is an example MCP server that implements all basic capabilities.`
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.TextResourceContents{
				URI:      uri,
				MimeType: "text/markdown",
				Text:     readmeContent,
			},
		})
		
	case "example://image/logo":
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.BlobResourceContents{
				URI:      uri,
				MimeType: "image/png",
				Blob:     tinyImageBase64,
			},
		})
	}
	
	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

// Helper functions
func getNumber(args map[string]interface{}, key string) (float64, bool) {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		}
	}
	return 0, false
}

func runHTTPServer(mcpServer *server.MCPServer) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)

	// Create SSE server and get its handler
	sseServer := server.NewSSEServer(mcpServer)
	mux.Handle("/", sseServer)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting HTTP server on port %s", port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	_ = srv.Close()
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
