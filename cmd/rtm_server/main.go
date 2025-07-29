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
	"github.com/vcto/mcp-adapters/internal/rtm"
)

const (
	serverName    = "rtm-server"
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

	// Check RTM credentials
	rtmHandler := rtm.NewHandler()
	if rtmHandler == nil {
		log.Fatal("RTM: API credentials required (RTM_API_KEY and RTM_API_SECRET)")
	}

	log.Println("RTM: Registering RTM tools and resources")

	// Setup RTM tools
	rtmHandler.SetupTools(s)

	// Setup RTM resources
	setupRTMResources(s, rtmHandler)

	// Run server
	if os.Getenv("FLY_APP_NAME") != "" {
		runHTTPServer(s, debugStorage, debugConfig, *disableAuth, rtmHandler)
	} else {
		if debugConfig.Enabled {
			log.Printf("Debug mode enabled for stdio server")
		}
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}
}

func setupRTMResources(s *server.MCPServer, handler *rtm.Handler) {
	// Today's tasks
	s.AddResource(mcp.NewResource("rtm://today",
		"Today's Tasks",
		mcp.WithResourceDescription("Tasks due today, sorted by priority"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		// Get today's tasks
		tasks, err := handler.GetClient().GetTasks("due:today", "")
		if err != nil {
			return nil, fmt.Errorf("failed to get today's tasks: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title": "Today's Tasks",
			"date":  time.Now().Format("2006-01-02"),
			"tasks": tasks,
			"count": len(tasks),
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "rtm://today",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// Inbox tasks
	s.AddResource(mcp.NewResource("rtm://inbox",
		"Inbox",
		mcp.WithResourceDescription("Tasks in the default inbox"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		tasks, err := handler.GetClient().GetTasks("list:Inbox", "")
		if err != nil {
			return nil, fmt.Errorf("failed to get inbox tasks: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title": "Inbox Tasks",
			"tasks": tasks,
			"count": len(tasks),
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "rtm://inbox",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// Overdue tasks
	s.AddResource(mcp.NewResource("rtm://overdue",
		"Overdue Tasks",
		mcp.WithResourceDescription("Tasks past their due date"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		tasks, err := handler.GetClient().GetTasks("dueBefore:today", "")
		if err != nil {
			return nil, fmt.Errorf("failed to get overdue tasks: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title": "Overdue Tasks",
			"tasks": tasks,
			"count": len(tasks),
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "rtm://overdue",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// This week's tasks
	s.AddResource(mcp.NewResource("rtm://week",
		"This Week",
		mcp.WithResourceDescription("Tasks due in the next 7 days"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		tasks, err := handler.GetClient().GetTasks("due:within 1 week", "")
		if err != nil {
			return nil, fmt.Errorf("failed to get week's tasks: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title": "This Week's Tasks",
			"tasks": tasks,
			"count": len(tasks),
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "rtm://week",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// All lists
	s.AddResource(mcp.NewResource("rtm://lists",
		"All Lists",
		mcp.WithResourceDescription("All lists with task counts"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		lists, err := handler.GetClient().GetLists()
		if err != nil {
			return nil, fmt.Errorf("failed to get lists: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title": "All Lists",
			"lists": lists,
			"count": len(lists),
		}, "", "  ")
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "rtm://lists",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// Template: Tasks in specific list
	s.AddResourceTemplate(mcp.NewResourceTemplate("rtm://lists/{list_name}",
		"List Tasks",
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		// Extract list name from URI
		listName := extractListNameFromURI(request.Params.URI)
		if listName == "" {
			return nil, fmt.Errorf("invalid list URI format")
		}

		// Search for tasks in this list
		tasks, err := handler.GetClient().GetTasks("list:"+listName, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get list tasks: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title":     fmt.Sprintf("Tasks in '%s'", listName),
			"list_name": listName,
			"tasks":     tasks,
			"count":     len(tasks),
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

	// Template: Smart lists
	s.AddResourceTemplate(mcp.NewResourceTemplate("rtm://smart/{list_name}",
		"Smart List",
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if handler.GetClient().AuthToken == "" {
			return nil, fmt.Errorf("RTM authentication required")
		}

		// Extract smart list name from URI
		smartListName := extractListNameFromURI(request.Params.URI)
		if smartListName == "" {
			return nil, fmt.Errorf("invalid smart list URI format")
		}

		// Get all lists to find the smart list
		lists, err := handler.GetClient().GetLists()
		if err != nil {
			return nil, fmt.Errorf("failed to get lists: %v", err)
		}

		var smartListID string
		for _, list := range lists {
			if list.Name == smartListName && list.Smart == "1" {
				smartListID = list.ID
				break
			}
		}

		if smartListID == "" {
			return nil, fmt.Errorf("smart list '%s' not found", smartListName)
		}

		// Get tasks from smart list
		tasks, err := handler.GetClient().GetTasks("", smartListID)
		if err != nil {
			return nil, fmt.Errorf("failed to get smart list tasks: %v", err)
		}

		data, err := json.MarshalIndent(map[string]interface{}{
			"title":           fmt.Sprintf("Smart List: '%s'", smartListName),
			"smart_list_name": smartListName,
			"smart_list_id":   smartListID,
			"tasks":           tasks,
			"count":           len(tasks),
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

func runHTTPServer(mcpServer *server.MCPServer, debugStorage debug.Storage, debugConfig *debug.DebugConfig, authDisabled bool, rtmHandler *rtm.Handler) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Different port from everything server
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
		log.Printf("Debug middleware enabled for RTM server")
		handler = debug.DebugMiddleware(debugStorage, debugConfig)(handler)
	}

	mux := http.NewServeMux()

	if !authDisabled {
		rtmAPIKey := os.Getenv("RTM_API_KEY")
		rtmSecret := os.Getenv("RTM_API_SECRET")

		if rtmAPIKey != "" && rtmSecret != "" {
			rtmAdapter := rtm.NewOAuthAdapter(rtmAPIKey, rtmSecret, serverURL)
			rtmSetup := rtm.NewSetupHandler()

			// OAuth endpoints
			mux.HandleFunc("/authorize", rtmAdapter.HandleAuthorize)
			mux.HandleFunc("/token", rtmAdapter.HandleToken)
			mux.HandleFunc("/oauth/authorize", rtmAdapter.HandleAuthorize)
			mux.HandleFunc("/oauth/token", rtmAdapter.HandleToken)
			mux.HandleFunc("/rtm/callback", rtmAdapter.HandleCallback)
			mux.HandleFunc("/rtm/check-auth", rtmAdapter.HandleCheckAuth)
			mux.HandleFunc("/rtm/setup", rtmSetup.HandleSetup)

			// OAuth discovery endpoints (RFC 9728 + Claude compatibility)
			mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
				metadata := map[string]interface{}{
					"authorization_servers": []string{serverURL},
					"resource":               serverURL + "/mcp",
					"scopes_supported":       []string{"rtm:read", "rtm:write"},
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(metadata); err != nil {
					log.Printf("Failed to encode OAuth metadata: %v", err)
				}
			})

			// Authorization server metadata (Claude expects /authorize not /oauth/authorize)
			mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
				metadata := map[string]interface{}{
					"issuer":                       serverURL,
					"authorization_endpoint":       serverURL + "/authorize",
					"token_endpoint":               serverURL + "/token", 
					"scopes_supported":             []string{"rtm:read", "rtm:write"},
					"response_types_supported":     []string{"code"},
					"grant_types_supported":        []string{"authorization_code"},
					"resource_indicators_supported": true,
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(metadata); err != nil {
					log.Printf("Failed to encode auth server metadata: %v", err)
				}
			})

			// Auth middleware
			handler = rtmAuthMiddleware(rtmAdapter, rtmHandler)(handler)
			log.Printf("OAuth: Enabled RTM OAuth adapter")
		} else {
			log.Fatal("RTM_API_KEY and RTM_API_SECRET required when auth enabled")
		}
	} else {
		log.Println("OAuth: DISABLED via --disable-auth flag")
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

	log.Printf("Starting RTM MCP server on port %s", port)
	log.Printf("Endpoint: %s/mcp", serverURL)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)
	log.Printf("RTM server ready")

	// Wait for signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case <-quit:
		log.Println("Shutting down RTM server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("RTM server stopped")
}

func rtmAuthMiddleware(adapter *rtm.OAuthAdapter, rtmHandler *rtm.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for OAuth endpoints
			if strings.HasPrefix(r.URL.Path, "/oauth/") ||
				strings.HasPrefix(r.URL.Path, "/rtm/") ||
				r.URL.Path == "/health" ||
				r.URL.Path == "/authorize" ||
				r.URL.Path == "/token" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

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

			if rtmHandler != nil {
				rtmHandler.SetAuthToken(token)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractListNameFromURI(uri string) string {
	// Extract from "rtm://lists/Shopping" -> "Shopping"
	// or "rtm://smart/Work" -> "Work"
	parts := strings.Split(uri, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-1]
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"server":    "rtm-server",
		"transport": "StreamableHTTP",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}
