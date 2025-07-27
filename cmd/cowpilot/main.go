package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag" // Import the flag package
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/cowpilot/internal/auth"
	"github.com/vcto/cowpilot/internal/debug"
	"github.com/vcto/cowpilot/internal/middleware"
	"github.com/vcto/cowpilot/internal/rtm"
)

// Version information
const (
	serverName    = "cowpilot-everything"
	serverVersion = "1.0.0"
)

// Tiny example image (1x1 transparent PNG)
const tinyImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

// Define the command-line flag
var (
	disableAuth = flag.Bool("disable-auth", os.Getenv("DISABLE_AUTH") == "true", "Disable authentication for testing or insecure environments")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// Initialize debug system (zero cost when disabled)
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

	// Add all tools
	setupTools(s)

	// Add RTM tools if credentials available
	if rtmHandler := rtm.NewHandler(); rtmHandler != nil {
		log.Println("RTM: Registering RTM tools (API credentials found)")
		rtmHandler.SetupTools(s)
	} else {
		log.Println("RTM: Skipping RTM tools (no API credentials)")
	}

	// Add native resources
	setupResources(s)

	// Add native prompts
	setupPrompts(s)

	// Check if we're running on Fly.io or locally
	if os.Getenv("FLY_APP_NAME") != "" {
		// Run HTTP server for Fly.io, passing the auth flag
		runHTTPServer(s, debugStorage, debugConfig, *disableAuth)
	} else {
		// Run stdio server for local development
		if debugConfig.Enabled {
			log.Printf("Debug mode enabled for stdio server (session logging active)")
		}
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}
}

func runHTTPServer(mcpServer *server.MCPServer, debugStorage debug.Storage, debugConfig *debug.DebugConfig, authDisabled bool) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:" + port
	}

	// StreamableHTTP transport with stateless mode for testing compatibility
	// Using /mcp endpoint to force HTTP transport detection in MCP Inspector
	streamableServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	// Base handler
	handler := http.Handler(streamableServer)

	// Apply protocol detection middleware first
	handler = protocolDetectionMiddleware(handler)

	// Conditionally add debug middleware
	if debugConfig.Enabled {
		log.Printf("Debug middleware enabled for StreamableHTTP server")
		handler = debug.DebugMiddleware(debugStorage, debugConfig)(handler)
	}

	mux := http.NewServeMux()

	// Conditionally apply OAuth middleware and endpoints
	if !authDisabled {
		// Check if RTM credentials are available
		rtmAPIKey := os.Getenv("RTM_API_KEY")
		rtmSecret := os.Getenv("RTM_API_SECRET")

		if rtmAPIKey != "" && rtmSecret != "" {
			// Use RTM OAuth adapter
			rtmAdapter := rtm.NewOAuthAdapter(rtmAPIKey, rtmSecret, serverURL)

			// OAuth endpoints for RTM
			mux.HandleFunc("/oauth/authorize", rtmAdapter.HandleAuthorize)
			mux.HandleFunc("/oauth/token", rtmAdapter.HandleToken)
			mux.HandleFunc("/rtm/callback", rtmAdapter.HandleCallback)
			mux.HandleFunc("/rtm/check-auth", rtmAdapter.HandleCheckAuth)

			// Add auth middleware that accepts RTM tokens
			handler = rtmAuthMiddleware(rtmAdapter)(handler)

			log.Printf("OAuth: Enabled RTM OAuth adapter")
		} else {
			// Use generic OAuth adapter
			callbackPort := 9090 // Default callback port
			if cbPort := os.Getenv("OAUTH_CALLBACK_PORT"); cbPort != "" {
				if p, err := strconv.Atoi(cbPort); err == nil {
					callbackPort = p
				}
			}
			oauthAdapter := auth.NewOAuthAdapter(serverURL, callbackPort)

			// Add auth middleware to the MCP handler
			handler = auth.Middleware(oauthAdapter)(handler)

			// OAuth endpoints
			mux.HandleFunc("/.well-known/oauth-protected-resource", oauthAdapter.HandleProtectedResourceMetadata)
			mux.HandleFunc("/.well-known/oauth-authorization-server", oauthAdapter.HandleAuthServerMetadata)
			mux.HandleFunc("/oauth/authorize", oauthAdapter.HandleAuthorize)
			mux.HandleFunc("/oauth/token", oauthAdapter.HandleToken)
			mux.HandleFunc("/oauth/register", oauthAdapter.HandleRegister)
			log.Printf("OAuth: Enabled generic OAuth adapter")
		}
	} else {
		log.Println("OAuth: DISABLED via --disable-auth flag")
	}

	// Health check
	mux.HandleFunc("/health", handleHealth)

	// MCP server handles requests at /mcp endpoint
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)

	// Apply CORS as the outermost middleware
	corsConfig := middleware.DefaultCORSConfig()
	if allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); allowedOrigins != "" {
		corsConfig.AllowOrigins = append(corsConfig.AllowOrigins, strings.Split(allowedOrigins, ",")...)
	}
	finalHandler := middleware.CORS(corsConfig)(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: finalHandler,
	}

	log.Printf("Starting MCP server with StreamableHTTP transport on port %s", port)
	log.Printf("Protocol: StreamableHTTP (VERIFIED: Works with MCP Inspector CLI)")
	log.Printf("CORS: Enabled for %v", corsConfig.AllowOrigins)

	if !authDisabled {
		log.Printf("Endpoint: %s/mcp (protected)", serverURL)
		log.Printf("Auth flow: %s/oauth/authorize", serverURL)
	} else {
		log.Printf("Endpoint: %s/mcp (unprotected)", serverURL)
	}

	log.Printf("Test with: npx @modelcontextprotocol/inspector --cli %s/mcp --method tools/list", serverURL)
	log.Printf("Raw test: curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1}' %s/mcp", serverURL)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait a moment to ensure server is listening before health checks start
	time.Sleep(100 * time.Millisecond)
	log.Printf("Server ready to accept connections")

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case <-quit:
		log.Println("Shutdown signal received, starting graceful shutdown...")
	}

	// Create a context with a 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt the graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

func setupResources(s *server.MCPServer) {
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
				Text:     "Hello, World! This is a simple text resource from the everything server.",
			},
		}, nil
	})

	// Add README resource
	s.AddResource(mcp.NewResource("example://text/readme",
		"README",
		mcp.WithResourceDescription("Project documentation"),
		mcp.WithMIMEType("text/markdown"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		content := `# Everything Server

This is an example MCP server that implements all basic capabilities:

- **Tools**: Various utility functions
- **Resources**: Text and binary content  
- **Prompts**: Template-based interactions
- **Logging**: Server-side logging
- **Completions**: Argument suggestions

## Usage

Connect to this server using any MCP client to explore its capabilities.`
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
		// Extract ID from URI
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

func setupPrompts(s *server.MCPServer) {
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
		args = make(map[string]any) // No arguments is valid for this handler
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
		args = make(map[string]any) // No arguments is valid, will use defaults
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

// Image handler
func getTestImageHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Decode the base64 image
	imageData, err := base64.StdEncoding.DecodeString(tinyImageBase64)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode image: %v", err)), nil
	}

	// Return as image content
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

// Resource content handler - demonstrates embedded resources
func getResourceContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	uri, ok := args["uri"].(string)
	if !ok {
		return mcp.NewToolResultError("uri parameter is required and must be a string"), nil
	}

	// Return embedded resource content
	content := []mcp.Content{
		mcp.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Returning embedded resource: %s", uri),
		},
	}

	// Add the actual resource content based on URI
	switch uri {
	case "example://text/hello":
		content = append(content, mcp.EmbeddedResource{
			Type: "resource",
			Resource: mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
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

// Helper functions
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

// protocolDetectionMiddleware logs client protocol detection to prevent confusion
func protocolDetectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect client type by headers and method
		clientType := "UNKNOWN"
		accept := r.Header.Get("Accept")
		contentType := r.Header.Get("Content-Type")
		userAgent := r.Header.Get("User-Agent")

		if strings.Contains(accept, "text/event-stream") {
			clientType = "SSE_BROWSER"
		} else if r.Method == "POST" && strings.Contains(contentType, "application/json") {
			if strings.Contains(userAgent, "node") || strings.Contains(userAgent, "inspector") {
				clientType = "MCP_INSPECTOR_CLI"
			} else if strings.Contains(userAgent, "curl") {
				clientType = "CURL_TEST"
			} else {
				clientType = "HTTP_POST_CLIENT"
			}
		}

		log.Printf("[PROTOCOL] Client: %s | Method: %s | Accept: %s | Content-Type: %s",
			clientType, r.Method, accept, contentType)

		next.ServeHTTP(w, r)
	})
}

// rtmAuthMiddleware validates RTM bearer tokens
func rtmAuthMiddleware(adapter *rtm.OAuthAdapter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for OAuth endpoints
			if strings.HasPrefix(r.URL.Path, "/oauth/") ||
				strings.HasPrefix(r.URL.Path, "/rtm/") ||
				strings.HasPrefix(r.URL.Path, "/.well-known/") ||
				r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Check Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Extract bearer token
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				http.Error(w, "Invalid Authorization format", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, bearerPrefix)
			if !adapter.ValidateBearer(token) {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Store token for RTM handler to use
			if rtmHandler := rtm.NewHandler(); rtmHandler != nil {
				rtmHandler.SetAuthToken(token)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Log health check requests for debugging
	log.Printf("[HEALTH] Health check from %s", r.RemoteAddr)

	// Protocol diagnostic endpoint
	if r.URL.Query().Get("protocol") == "true" {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":                       "healthy",
			"transport":                    "StreamableHTTP",
			"supports":                     []string{"HTTP_POST_JSON_RPC", "SSE_EVENT_STREAM"},
			"mcp_inspector_cli_compatible": true,
			"client_type_detection":        "automatic",
			"test_commands": map[string]string{
				"cli":  "npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list",
				"curl": "curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1}' http://localhost:8080/",
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode protocol response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	// Simple health check response
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
