package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MessageInterceptor intercepts and logs MCP messages
type MessageInterceptor struct {
	storage   *ConversationStorage
	sessionID string
	enabled   bool
}

// NewMessageInterceptor creates a new message interceptor
func NewMessageInterceptor(storage *ConversationStorage) *MessageInterceptor {
	sessionID := generateSessionID()
	
	return &MessageInterceptor{
		storage:   storage,
		sessionID: sessionID,
		enabled:   isDebugEnabled(),
	}
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("session_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
}

// isDebugEnabled checks environment variables to determine if debugging is enabled
func isDebugEnabled() bool {
	enabled := os.Getenv("MCP_DEBUG_ENABLED")
	return enabled == "true" || enabled == "1"
}

// LogRequest logs an incoming MCP request
func (mi *MessageInterceptor) LogRequest(method string, params interface{}) {
	if !mi.enabled || mi.storage == nil {
		return
	}

	start := time.Now()
	err := mi.storage.LogMessage(mi.sessionID, "inbound", method, params, nil, nil, 0)
	if err != nil {
		log.Printf("Failed to log request: %v", err)
	}

	logLevel := os.Getenv("MCP_DEBUG_LEVEL")
	if logLevel == "DEBUG" || logLevel == "TRACE" {
		log.Printf("[DEBUG] Inbound: %s | Session: %s | Duration: %v", method, mi.sessionID, time.Since(start))
	}
}

// LogResponse logs an MCP response
func (mi *MessageInterceptor) LogResponse(method string, result interface{}, errorMsg interface{}, performanceMS int64) {
	if !mi.enabled || mi.storage == nil {
		return
	}

	start := time.Now()
	err := mi.storage.LogMessage(mi.sessionID, "outbound", method, nil, result, errorMsg, performanceMS)
	if err != nil {
		log.Printf("Failed to log response: %v", err)
	}

	logLevel := os.Getenv("MCP_DEBUG_LEVEL")
	if logLevel == "DEBUG" || logLevel == "TRACE" {
		log.Printf("[DEBUG] Outbound: %s | Session: %s | Performance: %dms | Duration: %v", 
			method, mi.sessionID, performanceMS, time.Since(start))
	}
}

// GetSessionID returns the current session ID
func (mi *MessageInterceptor) GetSessionID() string {
	return mi.sessionID
}

// SetSessionID sets a custom session ID
func (mi *MessageInterceptor) SetSessionID(sessionID string) {
	mi.sessionID = sessionID
}

// MCPDebugProxy wraps an MCP server with debugging capabilities
type MCPDebugProxy struct {
	target      *server.MCPServer
	interceptor *MessageInterceptor
	storage     *ConversationStorage
}

// NewMCPDebugProxy creates a new debug proxy for an MCP server
func NewMCPDebugProxy(target *server.MCPServer, storage *ConversationStorage) *MCPDebugProxy {
	interceptor := NewMessageInterceptor(storage)
	
	return &MCPDebugProxy{
		target:      target,
		interceptor: interceptor,
		storage:     storage,
	}
}

// ProxySSEHandler wraps an SSE handler with debugging
func (proxy *MCPDebugProxy) ProxySSEHandler(originalHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !proxy.interceptor.enabled {
			// If debugging is disabled, pass through directly
			originalHandler.ServeHTTP(w, r)
			return
		}

		// Log the SSE connection establishment
		proxy.interceptor.LogRequest("sse_connection", map[string]interface{}{
			"method":     r.Method,
			"url":        r.URL.String(),
			"user_agent": r.Header.Get("User-Agent"),
			"remote_addr": r.RemoteAddr,
		})

		// Create a response wrapper to intercept responses
		wrapper := &responseWrapper{
			ResponseWriter: w,
			proxy:         proxy,
		}

		start := time.Now()
		originalHandler.ServeHTTP(wrapper, r)
		duration := time.Since(start)

		proxy.interceptor.LogResponse("sse_connection", map[string]interface{}{
			"status":   wrapper.status,
			"duration": duration.Milliseconds(),
		}, wrapper.error, duration.Milliseconds())
	})
}

// responseWrapper intercepts HTTP responses for logging
type responseWrapper struct {
	http.ResponseWriter
	proxy  *MCPDebugProxy
	status int
	error  interface{}
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.status = code
	if code >= 400 {
		rw.error = fmt.Sprintf("HTTP %d", code)
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapper) Write(data []byte) (int, error) {
	// For debugging, we could log the response data here
	// but we need to be careful about large responses
	if rw.status == 0 {
		rw.status = 200
	}
	return rw.ResponseWriter.Write(data)
}

// JSONRPCMessageInterceptor intercepts JSON-RPC messages at the transport level
type JSONRPCMessageInterceptor struct {
	interceptor *MessageInterceptor
}

// NewJSONRPCMessageInterceptor creates a new JSON-RPC message interceptor
func NewJSONRPCMessageInterceptor(storage *ConversationStorage) *JSONRPCMessageInterceptor {
	return &JSONRPCMessageInterceptor{
		interceptor: NewMessageInterceptor(storage),
	}
}

// InterceptRequest intercepts and logs JSON-RPC requests
func (ji *JSONRPCMessageInterceptor) InterceptRequest(data []byte) error {
	if !ji.interceptor.enabled {
		return nil
	}

	// Parse JSON-RPC request
	var request map[string]interface{}
	if err := json.Unmarshal(data, &request); err != nil {
		log.Printf("Failed to parse JSON-RPC request: %v", err)
		return err
	}

	method, _ := request["method"].(string)
	params := request["params"]

	ji.interceptor.LogRequest(method, params)
	return nil
}

// InterceptResponse intercepts and logs JSON-RPC responses
func (ji *JSONRPCMessageInterceptor) InterceptResponse(data []byte, performanceMS int64) error {
	if !ji.interceptor.enabled {
		return nil
	}

	// Parse JSON-RPC response
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		log.Printf("Failed to parse JSON-RPC response: %v", err)
		return err
	}

	result := response["result"]
	errorMsg := response["error"]

	// Try to determine the method from context
	// This is challenging without request/response correlation
	method := "unknown"

	ji.interceptor.LogResponse(method, result, errorMsg, performanceMS)
	return nil
}

// DebugConfig holds configuration for the debug system
type DebugConfig struct {
	Enabled              bool
	Level                string // DEBUG, INFO, WARN, ERROR
	StoragePath          string
	MaxRetentionDays     int
	SecurityMonitoring   bool
	PerformanceThreshold int64 // milliseconds
}

// LoadDebugConfig loads debug configuration from environment variables
func LoadDebugConfig() *DebugConfig {
	config := &DebugConfig{
		Enabled:              isDebugEnabled(),
		Level:                getEnvDefault("MCP_DEBUG_LEVEL", "INFO"),
		StoragePath:          getEnvDefault("MCP_DEBUG_STORAGE_PATH", "./debug_conversations.db"),
		MaxRetentionDays:     getEnvInt("MCP_DEBUG_RETENTION_DAYS", 30),
		SecurityMonitoring:   getEnvDefault("MCP_SECURITY_MONITORING", "false") == "true",
		PerformanceThreshold: int64(getEnvInt("MCP_PERFORMANCE_THRESHOLD_MS", 1000)),
	}

	return config
}

// getEnvDefault gets environment variable with default value
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as integer with default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// DebugMiddleware creates HTTP middleware for debugging MCP communications
func DebugMiddleware(storage *ConversationStorage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isDebugEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			interceptor := NewMessageInterceptor(storage)
			
			// Log the HTTP request
			interceptor.LogRequest("http_request", map[string]interface{}{
				"method":      r.Method,
				"url":         r.URL.String(),
				"headers":     sanitizeHeaders(r.Header),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.Header.Get("User-Agent"),
			})

			// Wrap the response writer
			wrapper := &debugResponseWriter{
				ResponseWriter: w,
				interceptor:    interceptor,
				start:          time.Now(),
			}

			next.ServeHTTP(wrapper, r)
		})
	}
}

// debugResponseWriter wraps http.ResponseWriter for debug logging
type debugResponseWriter struct {
	http.ResponseWriter
	interceptor *MessageInterceptor
	start       time.Time
	status      int
}

func (w *debugResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *debugResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}

	duration := time.Since(w.start)
	
	// Log the response
	w.interceptor.LogResponse("http_response", map[string]interface{}{
		"status":        w.status,
		"duration_ms":   duration.Milliseconds(),
		"response_size": len(data),
	}, nil, duration.Milliseconds())

	return w.ResponseWriter.Write(data)
}

// sanitizeHeaders removes sensitive headers for logging
func sanitizeHeaders(headers http.Header) http.Header {
	sanitized := make(http.Header)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"x-api-key":     true,
		"x-auth-token":  true,
	}

	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if sensitiveHeaders[lowerKey] {
			sanitized[key] = []string{"[REDACTED]"}
		} else {
			sanitized[key] = values
		}
	}

	return sanitized
}

// StartDebugSystem initializes and starts the debug system
func StartDebugSystem(config *DebugConfig) (*ConversationStorage, error) {
	if !config.Enabled {
		log.Println("Debug system is disabled")
		return nil, nil
	}

	log.Printf("Starting MCP Debug System...")
	log.Printf("  Storage Path: %s", config.StoragePath)
	log.Printf("  Debug Level: %s", config.Level)
	log.Printf("  Retention Days: %d", config.MaxRetentionDays)
	log.Printf("  Security Monitoring: %v", config.SecurityMonitoring)

	storage, err := NewConversationStorage(config.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize conversation storage: %w", err)
	}

	// Set up cleanup routine if retention is configured
	if config.MaxRetentionDays > 0 {
		go func() {
			ticker := time.NewTicker(24 * time.Hour) // Run cleanup daily
			defer ticker.Stop()

			for range ticker.C {
				maxAge := time.Duration(config.MaxRetentionDays) * 24 * time.Hour
				if err := storage.CleanupOldRecords(maxAge); err != nil {
					log.Printf("Error during cleanup: %v", err)
				}
			}
		}()
	}

	log.Println("MCP Debug System started successfully")
	return storage, nil
}

// StopDebugSystem gracefully shuts down the debug system
func StopDebugSystem(storage *ConversationStorage) {
	if storage != nil {
		log.Println("Shutting down MCP Debug System...")
		storage.Close()
		log.Println("MCP Debug System stopped")
	}
}
