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
	"github.com/vcto/cowpilot/internal/debug"
	"github.com/vcto/cowpilot/internal/middleware"
)

const (
	serverName    = "test-server"
	serverVersion = "1.0.0"
)

// Tiny example image (1x1 transparent PNG)
const tinyImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func main() {

	// Initialize debug system
	debugStorage, debugConfig, err := debug.StartDebugSystem()
	if err != nil {
		log.Printf("Warning: Failed to initialize debug system: %v", err)
		debugStorage = &debug.NoOpStorage{}
	}
	defer func() {
		if err := debugStorage.Close(); err != nil {
			log.Printf("Failed to close debug storage: %v", err)
		}
	}()

	// Create MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Setup test tools, resources, and prompts
	setupTestTools(s)
	setupTestResources(s)
	setupTestPrompts(s)

	// Run server
	if os.Getenv("FLY_APP_NAME") != "" {
		runHTTPServer(s, debugStorage, debugConfig)
	} else {
		if debugConfig.Enabled {
			log.Printf("Debug mode enabled for stdio server")
		}
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}
}

func setupTestTools(s *server.MCPServer) {
	// Hello tool
	s.AddTool(mcp.NewTool("hello",
		mcp.WithDescription("Says hello to the world"),
	), helloHandler)

	// Echo tool
	s.AddTool(mcp.NewTool("echo",
		mcp.WithDescription("Echoes back the input message"),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo")),
	), echoHandler)

	// Add numbers tool
	s.AddTool(mcp.NewTool("add",
		mcp.WithDescription("Adds two numbers together"),
		mcp.WithNumber("a", mcp.Required(), mcp.Description("First number")),
		mcp.WithNumber("b", mcp.Required(), mcp.Description("Second number")),
	), addHandler)

	// Get current time tool
	s.AddTool(mcp.NewTool("get_time",
		mcp.WithDescription("Gets the current time in various formats"),
		mcp.WithString("format", mcp.Description("Time format: 'unix', 'iso', or 'human'")),
	), timeHandler)

	// Base64 encode/decode tools
	s.AddTool(mcp.NewTool("base64_encode",
		mcp.WithDescription("Encodes text to base64"),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to encode")),
	), base64EncodeHandler)

	s.AddTool(mcp.NewTool("base64_decode",
		mcp.WithDescription("Decodes base64 to text"),
		mcp.WithString("data", mcp.Required(), mcp.Description("Base64 data to decode")),
	), base64DecodeHandler)

	// String manipulation tool
	s.AddTool(mcp.NewTool("string_operation",
		mcp.WithDescription("Performs various string operations"),
		mcp.WithString("text", mcp.Required(), mcp.Description("Input text")),
		mcp.WithString("operation", mcp.Required(), mcp.Description("Operation: 'upper', 'lower', 'reverse', 'length'")),
	), stringOperationHandler)

	// JSON formatter tool
	s.AddTool(mcp.NewTool("format_json",
		mcp.WithDescription("Formats or minifies JSON"),
		mcp.WithString("json", mcp.Required(), mcp.Description("JSON string to format")),
		mcp.WithBoolean("minify", mcp.Description("Minify instead of prettify")),
	), jsonFormatterHandler)

	// Long running operation tool
	s.AddTool(mcp.NewTool("long_running_operation",
		mcp.WithDescription("Simulates a long-running operation with progress"),
		mcp.WithNumber("duration", mcp.Description("Duration in seconds (default: 5)")),
		mcp.WithNumber("steps", mcp.Description("Number of progress steps (default: 5)")),
	), longRunningHandler)

	// Test image tool
	s.AddTool(mcp.NewTool("get_test_image",
		mcp.WithDescription("Returns a test image"),
	), getTestImageHandler)

	// Get resource with embedded content
	s.AddTool(mcp.NewTool("get_resource_content",
		mcp.WithDescription("Gets a resource and returns it as embedded content"),
		mcp.WithString("uri", mcp.Required(), mcp.Description("Resource URI")),
	), getResourceContentHandler)
}

func setupTestResources(s *server.MCPServer) {
	// Add static text resource
	s.AddResource(mcp.NewResource("example://text/hello",
		"Hello World Text",
		mcp.WithResourceDescription("A simple text resource"),
		mcp.WithMIMEType("text/plain"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "example://text/hello",
				MIMEType: "text/plain",
				Text:     "Hello, World! This is a simple text resource from the test server.",
			},
		}, nil
	})

	// Add README resource
	s.AddResource(mcp.NewResource("example://text/readme",
		"README",
		mcp.WithResourceDescription("Project documentation"),
		mcp.WithMIMEType("text/markdown"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		content := `# Test Server

This is a test/demo MCP server with example tools and resources:

- **Tools**: Various utility functions for testing
- **Resources**: Example text and binary content  
- **Prompts**: Template-based interactions

## Usage

Connect to this server using any MCP client to test basic functionality.`
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "example://text/readme",
				MIMEType: "text/markdown",
				Text:     content,
			},
		}, nil
	})

	// Add image resource
	s.AddResource(mcp.NewResource("example://image/logo",
		"Logo Image",
		mcp.WithResourceDescription("A small example image"),
		mcp.WithMIMEType("image/png"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.BlobResourceContents{
				URI:      "example://image/logo",
				MIMEType: "image/png",
				Blob:     tinyImageBase64,
			},
		}, nil
	})

	// Add a dynamic resource template
	s.AddResourceTemplate(mcp.NewResourceTemplate(
		"example://dynamic/{id}",
		"Dynamic Resource",
		mcp.WithTemplateDescription("A dynamic resource that accepts an ID"),
		mcp.WithTemplateMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := strings.TrimPrefix(request.Params.URI, "example://dynamic/")
		data := map[string]interface{}{
			"id":        id,
			"timestamp": time.Now().Format(time.RFC3339),
			"message":   fmt.Sprintf("This is dynamic content for ID: %s", id),
		}

		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
}

func setupTestPrompts(s *server.MCPServer) {
	// Simple greeting prompt
	simplePrompt := mcp.Prompt{
		Name:        "simple_greeting",
		Description: "A simple greeting prompt",
	}
	s.AddPrompt(simplePrompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: "Please provide a friendly greeting for a new user joining our community.",
					},
				},
			},
		}, nil
	})

	// Code review prompt with arguments
	codeReviewPrompt := mcp.Prompt{
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
	}
	s.AddPrompt(codeReviewPrompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		language := request.Params.Arguments["language"]
		code := request.Params.Arguments["code"]

		if language == "" || code == "" {
			return nil, fmt.Errorf("language and code arguments are required")
		}

		message := fmt.Sprintf("Please review the following %s code for improvements, potential bugs, and best practices:\n\n```%s\n%s\n```",
			language, language, code)

		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: message,
					},
				},
			},
		}, nil
	})
}

// Tool handlers
func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("Hello, World! This is the test server demonstrating MCP capabilities."), nil
}

func echoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	message, ok := args["message"].(string)
	if !ok {
		return mcp.NewToolResultError("message parameter is required and must be a string"), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
}

func addHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	a, ok := getNumber(args, "a")
	if !ok {
		return mcp.NewToolResultError("parameter 'a' is required and must be a number"), nil
	}
	b, ok := getNumber(args, "b")
	if !ok {
		return mcp.NewToolResultError("parameter 'b' is required and must be a number"), nil
	}

	result := a + b
	return mcp.NewToolResultText(fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result)), nil
}

func timeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}
	format, _ := args["format"].(string)
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
		result = now.UTC().Format(time.RFC3339)
	}

	return mcp.NewToolResultText(result), nil
}

func base64EncodeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	text, ok := args["text"].(string)
	if !ok {
		return mcp.NewToolResultError("text parameter is required and must be a string"), nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return mcp.NewToolResultText(encoded), nil
}

func base64DecodeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	data, ok := args["data"].(string)
	if !ok {
		return mcp.NewToolResultError("data parameter is required and must be a string"), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode base64: %v", err)), nil
	}
	return mcp.NewToolResultText(string(decoded)), nil
}

func stringOperationHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	text, ok := args["text"].(string)
	if !ok {
		return mcp.NewToolResultError("text parameter is required and must be a string"), nil
	}
	operation, ok := args["operation"].(string)
	if !ok {
		return mcp.NewToolResultError("operation parameter is required and must be a string"), nil
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
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	jsonStr, ok := args["json"].(string)
	if !ok {
		return mcp.NewToolResultError("json parameter is required and must be a string"), nil
	}
	minify, _ := args["minify"].(bool)

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
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}
	duration, _ := getNumber(args, "duration")
	if duration <= 0 {
		duration = 5
	}
	steps, _ := getNumber(args, "steps")
	if steps <= 0 {
		steps = 5
	}

	stepDuration := time.Duration(duration*1000/steps) * time.Millisecond
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

func getTestImageHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageData, err := base64.StdEncoding.DecodeString(tinyImageBase64)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode image: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "Here's a tiny test image (1x1 transparent PNG):",
			},
			mcp.ImageContent{
				Type:     "image",
				Data:     base64.StdEncoding.EncodeToString(imageData),
				MIMEType: "image/png",
			},
		},
	}, nil
}

func getResourceContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	uri, ok := args["uri"].(string)
	if !ok {
		return mcp.NewToolResultError("uri parameter is required and must be a string"), nil
	}

	content := []mcp.Content{
		mcp.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Returning embedded resource: %s", uri),
		},
	}

	switch uri {
	case "example://text/hello":
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     "Hello, World! This is a simple text resource from the test server.",
			},
		})

	case "example://text/readme":
		readmeContent := `# Test Server

This is a test/demo MCP server with example capabilities.`
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/markdown",
				Text:     readmeContent,
			},
		})

	case "example://image/logo":
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.BlobResourceContents{
				URI:      uri,
				MIMEType: "image/png",
				Blob:     tinyImageBase64,
			},
		})

	default:
		return mcp.NewToolResultError(fmt.Sprintf("Resource not found: %s", uri)), nil
	}

	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

func getNumber(args map[string]any, key string) (float64, bool) {
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

func runHTTPServer(mcpServer *server.MCPServer, debugStorage debug.Storage, debugConfig *debug.DebugConfig) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Different port from everything and RTM servers
	}

	streamableServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	handler := http.Handler(streamableServer)

	if debugConfig.Enabled {
		log.Printf("Debug middleware enabled for test server")
		handler = debug.DebugMiddleware(debugStorage, debugConfig)(handler)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)

	corsConfig := middleware.DefaultCORSConfig()
	finalHandler := middleware.CORS(corsConfig)(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: finalHandler,
	}

	log.Printf("Starting Test MCP server on port %s", port)
	log.Printf("Endpoint: http://localhost:%s/mcp", port)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)
	log.Printf("Test server ready")

	// Wait for signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case <-quit:
		log.Println("Shutting down test server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Test server stopped")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"server":    "test-server",
		"transport": "StreamableHTTP",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}
