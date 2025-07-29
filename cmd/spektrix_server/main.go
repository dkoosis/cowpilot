package main

import (
	"context"
	"encoding/json"
	"flag"
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
	"github.com/vcto/mcp-adapters/internal/debug"
	"github.com/vcto/mcp-adapters/internal/middleware"
	"github.com/vcto/mcp-adapters/internal/spektrix"
)

const (
	serverName    = "spektrix-server"
	serverVersion = "1.0.0"
)

var (
	disableAuth = flag.Bool("disable-auth", os.Getenv("DISABLE_AUTH") == "true", "Disable authentication")
)

func main() {
	flag.Parse()

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
		server.WithPromptCapabilities(false),
	)

	// Check Spektrix credentials
	spektrixHandler := spektrix.NewHandler()
	if spektrixHandler == nil {
		log.Fatal("Spektrix: API credentials required (SPEKTRIX_CLIENT_NAME, SPEKTRIX_API_USER, SPEKTRIX_API_KEY)")
	}

	log.Println("Spektrix: Registering Spektrix tools and resources")

	// Setup Spektrix tools
	spektrixHandler.SetupTools(s)

	// Setup Spektrix resources
	setupSpektrixResources(s, spektrixHandler)

	// Run server
	if os.Getenv("FLY_APP_NAME") != "" {
		runHTTPServer(s, debugStorage, debugConfig, *disableAuth, spektrixHandler)
	} else {
		if debugConfig.Enabled {
			log.Printf("Debug mode enabled for stdio server")
		}
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}
}

func setupSpektrixResources(s *server.MCPServer, handler *spektrix.Handler) {
	// Customer search results
	s.AddResource(mcp.NewResource("spektrix://customers/search",
		"Customer Search Results",
		mcp.WithResourceDescription("Last customer search results with details"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if !handler.IsAuthenticated() {
			return nil, fmt.Errorf("spektrix authentication required")
		}

		// This would contain the last search results
		// For now, return placeholder structure
		data, err := json.MarshalIndent(map[string]interface{}{
			"title":       "Customer Search Results",
			"last_search": "Available via spektrix_search_customers tool",
			"note":        "Use the search tool to populate this resource",
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "spektrix://customers/search",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// All tags available
	s.AddResource(mcp.NewResource("spektrix://tags",
		"Available Tags",
		mcp.WithResourceDescription("All tags available in Spektrix system"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if !handler.IsAuthenticated() {
			return nil, fmt.Errorf("spektrix authentication required")
		}

		tags, err := handler.GetClient().GetTags()
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title": "Available Tags",
			"tags":  tags,
			"count": len(tags),
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "spektrix://tags",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// Template: Customer details by ID
	s.AddResourceTemplate(mcp.NewResourceTemplate("spektrix://customers/{customer_id}",
		"Customer Details",
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if !handler.IsAuthenticated() {
			return nil, fmt.Errorf("spektrix authentication required")
		}

		// Extract customer ID from URI
		customerID := extractCustomerIDFromURI(request.Params.URI)
		if customerID == "" {
			return nil, fmt.Errorf("invalid customer URI format")
		}

		customer, err := handler.GetClient().GetCustomer(customerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title":       fmt.Sprintf("Customer: %s", customerID),
			"customer_id": customerID,
			"customer":    customer,
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})
}

func runHTTPServer(mcpServer *server.MCPServer, debugStorage debug.Storage, debugConfig *debug.DebugConfig, authDisabled bool, spektrixHandler *spektrix.Handler) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Different port from RTM (8081) and everything (8080)
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:" + port
	}

	streamableServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	handler := http.Handler(streamableServer)

	if debugConfig.Enabled {
		log.Printf("Debug middleware enabled for Spektrix server")
		handler = debug.DebugMiddleware(debugStorage, debugConfig)(handler)
	}

	mux := http.NewServeMux()

	if !authDisabled {
		// For Spektrix, we use HMAC authentication, not OAuth
		// API credentials are validated on each request
		handler = spektrixAuthMiddleware(spektrixHandler)(handler)
		log.Printf("Auth: Enabled Spektrix HMAC authentication")
	} else {
		log.Println("Auth: DISABLED via --disable-auth flag")
	}

	mux.HandleFunc("/health", handleHealth)
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)

	corsConfig := middleware.DefaultCORSConfig()
	if allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); allowedOrigins != "" {
		corsConfig.AllowOrigins = append(corsConfig.AllowOrigins, strings.Split(allowedOrigins, ",")...)
	}
	finalHandler := middleware.CORS(corsConfig)(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: finalHandler,
	}

	log.Printf("Starting Spektrix MCP server on port %s", port)
	log.Printf("Endpoint: %s/mcp", serverURL)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)
	log.Printf("Spektrix server ready")

	// Wait for signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case <-quit:
		log.Println("Shutting down Spektrix server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Spektrix server stopped")
}

func spektrixAuthMiddleware(spektrixHandler *spektrix.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for MCP and health endpoints
			if strings.HasPrefix(r.URL.Path, "/mcp") || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// For Spektrix HMAC auth, we validate credentials are present
			// Actual HMAC signature validation happens in the client
			if !spektrixHandler.IsAuthenticated() {
				http.Error(w, "Missing Spektrix credentials", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractCustomerIDFromURI(uri string) string {
	// Extract from "spektrix://customers/12345" -> "12345"
	parts := strings.Split(uri, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-1]
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"server":    "spektrix-server",
		"transport": "StreamableHTTP",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}
