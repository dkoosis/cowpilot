package debug

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MessageInterceptor intercepts and logs MCP messages
type MessageInterceptor struct {
	storage   Storage
	sessionID string
	config    *DebugConfig
}

// NewMessageInterceptor creates a new message interceptor
func NewMessageInterceptor(storage Storage, config *DebugConfig) *MessageInterceptor {
	return &MessageInterceptor{
		storage:   storage,
		sessionID: generateSessionID(),
		config:    config,
	}
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("session_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
}

// LogRequest logs an incoming MCP request
func (mi *MessageInterceptor) LogRequest(method string, params interface{}) {
	if !mi.storage.IsEnabled() {
		return
	}

	start := time.Now()
	err := mi.storage.LogMessage(mi.sessionID, "inbound", method, params, nil, nil, 0)
	if err != nil && mi.config.Level == "DEBUG" {
		log.Printf("Failed to log request: %v", err)
	}

	if mi.config.Level == "DEBUG" {
		log.Printf("[DEBUG] Inbound: %s | Session: %s | Duration: %v", method, mi.sessionID, time.Since(start))
	}
}

// LogResponse logs an MCP response
func (mi *MessageInterceptor) LogResponse(method string, result interface{}, errorMsg interface{}, performanceMS int64) {
	if !mi.storage.IsEnabled() {
		return
	}

	start := time.Now()
	err := mi.storage.LogMessage(mi.sessionID, "outbound", method, nil, result, errorMsg, performanceMS)
	if err != nil && mi.config.Level == "DEBUG" {
		log.Printf("Failed to log response: %v", err)
	}

	if mi.config.Level == "DEBUG" {
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
	interceptor *MessageInterceptor
	storage     Storage
	config      *DebugConfig
}

// NewMCPDebugProxy creates a new debug proxy
func NewMCPDebugProxy(storage Storage, config *DebugConfig) *MCPDebugProxy {
	interceptor := NewMessageInterceptor(storage, config)
	
	return &MCPDebugProxy{
		interceptor: interceptor,
		storage:     storage,
		config:      config,
	}
}

// DebugMiddleware creates HTTP middleware for debugging MCP communications
func DebugMiddleware(storage Storage, config *DebugConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !storage.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			interceptor := NewMessageInterceptor(storage, config)
			
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
