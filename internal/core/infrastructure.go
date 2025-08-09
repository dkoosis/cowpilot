// Package core provides shared infrastructure for MCP servers including middleware,
// authentication setup, server configuration, and graceful shutdown handling.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/vcto/mcp-adapters/internal/auth"
	"github.com/vcto/mcp-adapters/internal/debug"
	"github.com/vcto/mcp-adapters/internal/middleware"
	"github.com/vcto/mcp-adapters/internal/rtm"
)

// InfrastructureConfig configures shared MCP server infrastructure
type InfrastructureConfig struct {
	ServerURL      string
	Port           string
	AuthDisabled   bool
	RTMHandler     *rtm.Handler
	DebugStorage   debug.Storage
	DebugConfig    *debug.DebugConfig
	ServerName     string
	AllowedOrigins []string
}

// MCPServerResult contains the configured server and shutdown function
type MCPServerResult struct {
	Server       *http.Server
	ShutdownFunc func() error
}

// SetupInfrastructure creates a complete MCP server with all infrastructure components.
// It configures middleware, authentication, OAuth endpoints, and standard health/logo endpoints.
// Returns an MCPServerResult containing the configured HTTP server and shutdown function.
func SetupInfrastructure(mcpServer *server.MCPServer, config InfrastructureConfig) *MCPServerResult {
	// Create StreamableHTTP transport
	streamableServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	// Build middleware stack
	handler := buildMiddlewareStack(streamableServer, config)

	// Create HTTP mux
	mux := http.NewServeMux()

	// Setup OAuth if enabled
	if !config.AuthDisabled {
		setupOAuthEndpoints(mux, config, &handler)
	} else {
		log.Println("OAuth: DISABLED via configuration")
	}

	// Setup standard endpoints
	setupStandardEndpoints(mux)

	// Mount MCP handler
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)

	// Apply CORS as outermost middleware
	corsConfig := middleware.DefaultCORSConfig()
	if len(config.AllowedOrigins) > 0 {
		corsConfig.AllowOrigins = append(corsConfig.AllowOrigins, config.AllowedOrigins...)
	}
	finalHandler := middleware.CORS(corsConfig)(mux)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: finalHandler,
	}

	// Setup graceful shutdown
	shutdownFunc := func() error {
		return gracefulShutdown(srv)
	}

	return &MCPServerResult{
		Server:       srv,
		ShutdownFunc: shutdownFunc,
	}
}

// StartServer starts the HTTP server and handles graceful shutdown on interrupt signals.
// It logs server startup information, waits for SIGINT/SIGTERM, then performs graceful
// shutdown with a 5-second timeout. This function blocks until the server stops.
func StartServer(result *MCPServerResult, config InfrastructureConfig) {
	log.Printf("Starting MCP server with StreamableHTTP transport on port %s", config.Port)
	log.Printf("Protocol: StreamableHTTP (VERIFIED: Works with MCP Inspector CLI)")

	if !config.AuthDisabled {
		log.Printf("Endpoint: %s/mcp (protected)", config.ServerURL)
		log.Printf("Auth flow: %s/oauth/authorize", config.ServerURL)
	} else {
		log.Printf("Endpoint: %s/mcp (unprotected)", config.ServerURL)
	}

	log.Printf("Test with: npx @modelcontextprotocol/inspector --cli %s/mcp --method tools/list", config.ServerURL)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on :%s", config.Port)
		if err := result.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)
	log.Printf("Server ready to accept connections")

	// Wait for interrupt or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case <-quit:
		log.Println("Shutdown signal received, starting graceful shutdown...")
	}

	if err := result.ShutdownFunc(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

// buildMiddlewareStack creates the middleware chain
func buildMiddlewareStack(streamableServer *server.StreamableHTTPServer, config InfrastructureConfig) http.Handler {
	handler := http.Handler(streamableServer)

	// Apply protocol detection middleware first
	handler = protocolDetectionMiddleware(handler)

	// Conditionally add debug middleware
	if config.DebugConfig.Enabled {
		log.Printf("Debug middleware enabled for StreamableHTTP server")
		handler = debug.DebugMiddleware(config.DebugStorage, config.DebugConfig)(handler)
	}

	return handler
}

// setupOAuthEndpoints configures OAuth authentication
func setupOAuthEndpoints(mux *http.ServeMux, config InfrastructureConfig, handler *http.Handler) {
	rtmAPIKey := os.Getenv("RTM_API_KEY")
	rtmSecret := os.Getenv("RTM_API_SECRET")

	if rtmAPIKey != "" && rtmSecret != "" {
		// Use RTM OAuth adapter
		rtmAdapter := rtm.NewOAuthAdapter(rtmAPIKey, rtmSecret, config.ServerURL)
		rtmSetup := rtm.NewSetupHandler()

		// OAuth endpoints for RTM (claude.ai compatibility)
		mux.HandleFunc("/authorize", rtmAdapter.HandleAuthorize)
		mux.HandleFunc("/token", rtmAdapter.HandleToken)
		mux.HandleFunc("/oauth/authorize", rtmAdapter.HandleAuthorize)
		mux.HandleFunc("/oauth/token", rtmAdapter.HandleToken)
		mux.HandleFunc("/oauth/register", rtmAdapter.HandleRegister)
		mux.HandleFunc("/rtm/callback", rtmAdapter.HandleCallback)
		mux.HandleFunc("/rtm/check-auth", rtmAdapter.HandleCheckAuth)
		mux.HandleFunc("/rtm/setup", rtmSetup.HandleSetup)

		// OAuth discovery endpoints (RFC 9728 + Claude compatibility)
		setupRTMWellKnownEndpoints(mux, config.ServerURL)

		// Add auth middleware to the MCP handler
		*handler = rtmAuthMiddleware(rtmAdapter, config.RTMHandler, config)(*handler)

		log.Printf("OAuth: Enabled RTM OAuth adapter")
	} else {
		// Use generic OAuth adapter
		callbackPort := 9090 // Default callback port
		if cbPort := os.Getenv("OAUTH_CALLBACK_PORT"); cbPort != "" {
			if p, err := strconv.Atoi(cbPort); err == nil {
				callbackPort = p
			}
		}
		oauthAdapter := auth.NewOAuthAdapter(config.ServerURL, callbackPort)

		// Add auth middleware to the MCP handler
		*handler = auth.Middleware(oauthAdapter)(*handler)

		// OAuth endpoints
		mux.HandleFunc("/.well-known/oauth-protected-resource", oauthAdapter.HandleProtectedResourceMetadata)
		mux.HandleFunc("/.well-known/oauth-authorization-server", oauthAdapter.HandleAuthServerMetadata)
		mux.HandleFunc("/oauth/authorize", oauthAdapter.HandleAuthorize)
		mux.HandleFunc("/oauth/token", oauthAdapter.HandleToken)
		mux.HandleFunc("/oauth/register", oauthAdapter.HandleRegister)
		log.Printf("OAuth: Enabled generic OAuth adapter")
	}
}

// setupRTMWellKnownEndpoints adds RTM-specific discovery endpoints
func setupRTMWellKnownEndpoints(mux *http.ServeMux, serverURL string) {
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		metadata := map[string]interface{}{
			"authorization_servers": []string{serverURL},
			"resource":              serverURL + "/mcp",
			"scopes_supported":      []string{"rtm:read", "rtm:write"},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("Failed to encode OAuth metadata: %v", err)
		}
	})

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		metadata := map[string]interface{}{
			"issuer":                           serverURL,
			"authorization_endpoint":           serverURL + "/oauth/authorize", // FIX: Added /oauth prefix
			"token_endpoint":                   serverURL + "/oauth/token",     // FIX: Added /oauth prefix
			"registration_endpoint":            serverURL + "/oauth/register",
			"scopes_supported":                 []string{"rtm:read", "rtm:write"},
			"response_types_supported":         []string{"code"},
			"grant_types_supported":            []string{"authorization_code"},
			"code_challenge_methods_supported": []string{"S256"},
			"resource_indicators_supported":    true,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("Failed to encode auth server metadata: %v", err)
		}
	})
}

// setupStandardEndpoints adds health check and logo endpoints
func setupStandardEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/logo", handleLogo)
}

// protocolDetectionMiddleware logs client protocol detection and fixes content-type
func protocolDetectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect client type by headers and method
		clientType := "UNKNOWN"
		accept := r.Header.Get("Accept")
		contentType := r.Header.Get("Content-Type")
		userAgent := r.Header.Get("User-Agent")

		// Fix content-type for mcp-go v0.32.0 compatibility
		if strings.HasPrefix(contentType, "application/json") {
			// Strip charset parameter that v0.32.0 rejects
			r.Header.Set("Content-Type", "application/json")
			contentType = "application/json"
		}

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
func rtmAuthMiddleware(adapter *rtm.OAuthAdapter, rtmHandler *rtm.Handler, config InfrastructureConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for OAuth and standard endpoints
			if strings.HasPrefix(r.URL.Path, "/oauth/") ||
				strings.HasPrefix(r.URL.Path, "/rtm/") ||
				strings.HasPrefix(r.URL.Path, "/.well-known/") ||
				r.URL.Path == "/health" ||
				r.URL.Path == "/logo" ||
				r.URL.Path == "/authorize" ||
				r.URL.Path == "/token" {
				log.Printf("[AUTH] Skipping auth for: %s", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			// Check Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// CRITICAL: WWW-Authenticate header required by MCP OAuth spec (RFC 9728)
				// Claude.ai needs this to show Connect button - DO NOT REMOVE
				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s/.well-known/oauth-protected-resource\"", config.ServerURL))
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
				// CRITICAL: WWW-Authenticate header required for ALL 401 responses
				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s/.well-known/oauth-protected-resource\"", config.ServerURL))
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Set token on the registered RTM handler instance
			if rtmHandler != nil {
				rtmHandler.SetAuthToken(token)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// handleHealth provides health check endpoint
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

// handleLogo provides logo endpoint for Claude.ai
func handleLogo(w http.ResponseWriter, r *http.Request) {
	// mcp adapters logo - blue circle with white cow
	logo := `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
		<circle cx="32" cy="32" r="32" fill="#1976d2"/>
		<!-- Cow head -->
		<ellipse cx="32" cy="35" rx="15" ry="12" fill="white"/>
		<!-- Horns -->
		<path d="M20 28 Q18 24 16 26 Q17 29 20 28" fill="white"/>
		<path d="M44 28 Q46 24 48 26 Q47 29 44 28" fill="white"/>
		<!-- Ears -->
		<ellipse cx="22" cy="30" rx="3" ry="5" fill="white"/>
		<ellipse cx="42" cy="30" rx="3" ry="5" fill="white"/>
		<!-- Eyes -->
		<circle cx="28" cy="32" r="2.5" fill="black"/>
		<circle cx="36" cy="32" r="2.5" fill="black"/>
		<circle cx="28.5" cy="31.5" r="0.8" fill="white"/>
		<circle cx="36.5" cy="31.5" r="0.8" fill="white"/>
		<!-- Nostrils -->
		<ellipse cx="30" cy="38" rx="1" ry="1.5" fill="black"/>
		<ellipse cx="34" cy="38" rx="1" ry="1.5" fill="black"/>
		<!-- Hair spikes -->
		<path d="M32 22 L30 18 L32 20 L34 18 L32 22" fill="#333"/>
		<path d="M28 23 L26 19 L28 21" fill="#333"/>
		<path d="M36 23 L38 19 L36 21" fill="#333"/>
	</svg>`

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(logo))
}

// gracefulShutdown handles server shutdown
func gracefulShutdown(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
