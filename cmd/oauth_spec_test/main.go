package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Custom context key type to avoid collisions
type contextKey string

const userContextKey contextKey = "user"

// OAuth 2.0 Protected Resource Metadata (RFC 9728)
type ResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	Resource             string   `json:"resource"`
	Scopes               []string `json:"scopes_supported,omitempty"`
}

// Simple token store for testing
var validTokens = map[string]TokenInfo{
	"test-token-123": {
		Resource: "https://test-mcp-resource-server.local/mcp",
		Subject:  "test-user",
		Scopes:   []string{"mcp:read", "mcp:tools"},
		IssuedAt: time.Now(),
	},
}

type TokenInfo struct {
	Resource string
	Subject  string
	Scopes   []string
	IssuedAt time.Time
}

func main() {
	// Resource server port (MCP server)
	resourcePort := getEnv("RESOURCE_PORT", "8090")
	// Authorization server port (separate service)
	authPort := getEnv("AUTH_PORT", "8091")

	// Expected resource identifier for this server
	resourceURI := "https://test-mcp-resource-server.local/mcp"
	authServerURI := fmt.Sprintf("http://localhost:%s", authPort)

	log.Printf("🧪 MCP OAuth Spec Compliance Test")
	log.Printf("📋 Resource Server: http://localhost:%s/mcp", resourcePort)
	log.Printf("🔐 Auth Server: %s", authServerURI)
	log.Printf("🎯 Resource URI: %s", resourceURI)

	// Start authorization server in background
	go startAuthServer(authPort, resourceURI)

	// Start resource server (MCP server)
	startResourceServer(resourcePort, authServerURI, resourceURI)
}

func startResourceServer(port, authServerURI, resourceURI string) {
	// Create MCP server with token validation
	s := server.NewMCPServer(
		"oauth-test-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	// Add test tool that requires authentication
	s.AddTool(mcp.Tool{
		Name:        "test_auth_tool",
		Description: "Test tool that requires authentication",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Test message",
				},
			},
		},
	}, handleTestTool)

	// Add test resource
	s.AddResource(mcp.NewResource("test://auth-required",
		"Test Resource",
		mcp.WithResourceDescription("Test resource requiring authentication"),
		mcp.WithMIMEType("text/plain"),
	), handleTestResource)

	// Create streamable HTTP server
	streamableServer := server.NewStreamableHTTPServer(
		s,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	// Custom HTTP handler for OAuth metadata and token validation
	mux := http.NewServeMux()

	// OAuth 2.0 Protected Resource Metadata endpoint (RFC 9728)
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		metadata := ResourceMetadata{
			AuthorizationServers: []string{authServerURI},
			Resource:             resourceURI,
			Scopes:               []string{"mcp:read", "mcp:tools"},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("Failed to encode resource metadata: %v", err)
		}
		log.Printf("📋 Served resource metadata to %s", r.RemoteAddr)
	})

	// MCP endpoint with token validation middleware
	mux.Handle("/mcp", tokenValidationMiddleware(resourceURI, streamableServer))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintf(w, "OAuth Test Resource Server OK"); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	log.Printf("🚀 Resource Server starting on port %s", port)
	log.Printf("💡 Test with: curl -H 'Authorization: Bearer test-token-123' http://localhost:%s/mcp", port)
	log.Printf("📋 Metadata: curl http://localhost:%s/.well-known/oauth-protected-resource", port)

	// Graceful shutdown
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Resource server failed: %v", err)
		}
	}()

	// Wait for interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	log.Println("🛑 Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

func startAuthServer(port, resourceURI string) {
	mux := http.NewServeMux()

	// OAuth 2.0 Authorization Server Metadata
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		metadata := map[string]interface{}{
			"issuer":                        fmt.Sprintf("http://localhost:%s", port),
			"authorization_endpoint":        fmt.Sprintf("http://localhost:%s/authorize", port),
			"token_endpoint":                fmt.Sprintf("http://localhost:%s/token", port),
			"resource_indicators_supported": true,
			"scopes_supported":              []string{"mcp:read", "mcp:tools"},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			log.Printf("Failed to encode auth server metadata: %v", err)
		}
		log.Printf("🔐 Served auth server metadata to %s", r.RemoteAddr)
	})

	// Simple authorize endpoint (for testing)
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		resource := r.URL.Query().Get("resource")
		log.Printf("🔐 Authorization request for resource: %s", resource)

		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>OAuth Test Authorization</title></head>
<body>
<h1>OAuth Spec Compliance Test</h1>
<p><strong>Resource:</strong> %s</p>
<p><strong>Expected Resource:</strong> %s</p>
<p><strong>Resource Indicators (RFC 8707):</strong> %s</p>
<form method="post" action="/token">
<input type="hidden" name="resource" value="%s">
<button type="submit">Issue Test Token</button>
</form>
</body>
</html>`, resource, resourceURI,
			map[bool]string{true: "✅ Present", false: "❌ Missing"}[resource != ""],
			resource)

		w.Header().Set("Content-Type", "text/html")
		if _, err := fmt.Fprint(w, html); err != nil {
			log.Printf("Failed to write authorize page: %v", err)
		}
	})

	// Simple token endpoint (for testing)
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
				return
			}
			resource := r.FormValue("resource")

			if resource != resourceURI {
				http.Error(w, fmt.Sprintf("Invalid resource: got %s, expected %s", resource, resourceURI), http.StatusBadRequest)
				return
			}

			response := map[string]interface{}{
				"access_token": "test-token-123",
				"token_type":   "Bearer",
				"resource":     resource,
				"scope":        "mcp:read mcp:tools",
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Failed to encode token response: %v", err)
			}
			log.Printf("🎫 Issued token for resource: %s", resource)
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	log.Printf("🔐 Auth Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Auth server error: %v", err)
	}
}

// Token validation middleware (RFC 8707 Resource Indicators)
func tokenValidationMiddleware(expectedResource string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			log.Printf("❌ Missing bearer token from %s", r.RemoteAddr)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		tokenInfo, valid := validTokens[token]

		if !valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Printf("❌ Invalid token from %s", r.RemoteAddr)
			return
		}

		// RFC 8707: Validate resource indicator
		if tokenInfo.Resource != expectedResource {
			http.Error(w, fmt.Sprintf("Token not valid for this resource. Expected: %s, Got: %s",
				expectedResource, tokenInfo.Resource), http.StatusForbidden)
			log.Printf("❌ Resource mismatch: expected %s, got %s", expectedResource, tokenInfo.Resource)
			return
		}

		log.Printf("✅ Valid token for user %s, resource %s", tokenInfo.Subject, tokenInfo.Resource)

		// Add user context to request
		ctx := context.WithValue(r.Context(), userContextKey, tokenInfo.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handleTestTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user, _ := ctx.Value(userContextKey).(string)
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		args = make(map[string]any)
	}
	message, _ := args["message"].(string)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("✅ OAuth Test Success! User: %s, Message: %s", user, message),
			},
		},
	}, nil
}

func handleTestResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	user, _ := ctx.Value(userContextKey).(string)
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "test://auth-required",
			MIMEType: "text/plain",
			Text: fmt.Sprintf("✅ OAuth Resource Access Success! User: %s, Time: %s",
				user, time.Now().Format(time.RFC3339)),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
