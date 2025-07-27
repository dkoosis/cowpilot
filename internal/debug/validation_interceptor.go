package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/vcto/cowpilot/internal/validator"
)

// ValidatingInterceptor adds protocol validation to debug interceptor
type ValidatingInterceptor struct {
	storage       Storage
	config        *DebugConfig
	validator     *validator.MCPValidator
	stats         *ValidationStats
	statsLock     sync.RWMutex
}

// ValidationStats tracks validation metrics
type ValidationStats struct {
	TotalRequests   int64
	TotalViolations int64
	ViolationsByMethod map[string]int64
	ViolationsByType   map[string]int64
	LastReset       time.Time
}

// NewValidatingInterceptor creates interceptor with validation
func NewValidatingInterceptor(storage Storage, config *DebugConfig) *ValidatingInterceptor {
	return &ValidatingInterceptor{
		storage:   storage,
		config:    config,
		validator: validator.NewMCPValidator(),
		stats: &ValidationStats{
			ViolationsByMethod: make(map[string]int64),
			ViolationsByType:   make(map[string]int64),
			LastReset:          time.Now(),
		},
	}
}

// InterceptRequest validates and logs incoming requests
func (vi *ValidatingInterceptor) InterceptRequest(r *http.Request, sessionID string) (*http.Request, error) {
	if !vi.config.ValidateProto {
		return r, nil
	}

	// Read body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return r, err
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Parse JSON-RPC
	var req map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		return r, nil // Not JSON, skip validation
	}

	method, _ := req["method"].(string)
	if method == "" {
		return r, nil
	}

	// Validate
	violations := vi.validateRequest(method, req)
	if len(violations) > 0 {
		vi.recordViolations(sessionID, method, violations)
		
		if vi.config.ValidateMode == "enforce" {
			return nil, fmt.Errorf("protocol violations: %v", violations)
		}
	}

	vi.recordRequest(method)
	return r, nil
}

func (vi *ValidatingInterceptor) validateRequest(method string, req map[string]interface{}) []string {
	var violations []string

	// Validate JSON-RPC structure
	if errs := vi.validator.ValidateJSONRPC(req); len(errs) > 0 {
		violations = append(violations, errs...)
	}

	// Validate MCP method
	switch method {
	case "tools/list", "resources/list", "prompts/list":
		// No params validation needed
	case "tools/call":
		if errs := vi.validator.ValidateToolCall(req); len(errs) > 0 {
			violations = append(violations, errs...)
		}
	case "resources/read":
		if errs := vi.validator.ValidateResourceRead(req); len(errs) > 0 {
			violations = append(violations, errs...)
		}
	case "prompts/get":
		if errs := vi.validator.ValidatePromptGet(req); len(errs) > 0 {
			violations = append(violations, errs...)
		}
	}

	return violations
}

func (vi *ValidatingInterceptor) recordViolations(sessionID, method string, violations []string) {
	vi.statsLock.Lock()
	vi.stats.TotalViolations += int64(len(violations))
	vi.stats.ViolationsByMethod[method]++
	for _, v := range violations {
		vi.stats.ViolationsByType[v]++
	}
	vi.statsLock.Unlock()

	// Log to storage
	severity := "WARN"
	if vi.config.ValidateMode == "enforce" {
		severity = "ERROR"
	}
	
	if err := vi.storage.LogValidation(sessionID, method, violations, severity); err != nil {
		log.Printf("Failed to log validation: %v", err)
	}
}

func (vi *ValidatingInterceptor) recordRequest(method string) {
	vi.statsLock.Lock()
	vi.stats.TotalRequests++
	vi.statsLock.Unlock()
}

// GetStats returns validation statistics
func (vi *ValidatingInterceptor) GetStats() map[string]interface{} {
	vi.statsLock.RLock()
	defer vi.statsLock.RUnlock()

	return map[string]interface{}{
		"validation_enabled": vi.config.ValidateProto,
		"validation_mode":   vi.config.ValidateMode,
		"total_requests":    vi.stats.TotalRequests,
		"total_violations":  vi.stats.TotalViolations,
		"violations_by_method": vi.stats.ViolationsByMethod,
		"violations_by_type":   vi.stats.ViolationsByType,
		"stats_since":       vi.stats.LastReset,
	}
}
