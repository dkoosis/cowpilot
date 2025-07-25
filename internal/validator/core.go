package validator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ValidationLevel represents the severity of validation issues
type ValidationLevel int

const (
	LevelInfo ValidationLevel = iota
	LevelWarning
	LevelError
	LevelCritical
)

func (l ValidationLevel) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	case LevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	ID         string            `json:"id"`
	Level      ValidationLevel   `json:"level"`
	Message    string            `json:"message"`
	Field      string            `json:"field,omitempty"`
	Expected   interface{}       `json:"expected,omitempty"`
	Actual     interface{}       `json:"actual,omitempty"`
	Suggestion string            `json:"suggestion,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Context    map[string]string `json:"context,omitempty"`
}

// ValidationReport contains all validation results for a message
type ValidationReport struct {
	SessionID    string             `json:"session_id"`
	MessageID    string             `json:"message_id"`
	MessageType  string             `json:"message_type"`
	Method       string             `json:"method,omitempty"`
	Results      []ValidationResult `json:"results"`
	Score        float64            `json:"score"` // 0.0-100.0
	IsValid      bool               `json:"is_valid"`
	ProcessingMS int64              `json:"processing_ms"`
	Timestamp    time.Time          `json:"timestamp"`
}

// AddResult adds a validation result to the report
func (r *ValidationReport) AddResult(result ValidationResult) {
	result.Timestamp = time.Now()
	r.Results = append(r.Results, result)
	r.updateScore()
}

// AddError adds an error-level validation result
func (r *ValidationReport) AddError(id, message string, context map[string]string) {
	r.AddResult(ValidationResult{
		ID:      id,
		Level:   LevelError,
		Message: message,
		Context: context,
	})
}

// AddWarning adds a warning-level validation result
func (r *ValidationReport) AddWarning(id, message string, context map[string]string) {
	r.AddResult(ValidationResult{
		ID:      id,
		Level:   LevelWarning,
		Message: message,
		Context: context,
	})
}

// AddCritical adds a critical-level validation result
func (r *ValidationReport) AddCritical(id, message string, context map[string]string) {
	r.AddResult(ValidationResult{
		ID:      id,
		Level:   LevelCritical,
		Message: message,
		Context: context,
	})
}

// updateScore calculates the validation score based on results
func (r *ValidationReport) updateScore() {
	if len(r.Results) == 0 {
		r.Score = 100.0
		r.IsValid = true
		return
	}

	totalDeductions := 0.0
	for _, result := range r.Results {
		switch result.Level {
		case LevelCritical:
			totalDeductions += 25.0
		case LevelError:
			totalDeductions += 10.0
		case LevelWarning:
			totalDeductions += 2.0
		case LevelInfo:
			totalDeductions += 0.5
		}
	}

	r.Score = 100.0 - totalDeductions
	if r.Score < 0 {
		r.Score = 0
	}

	// Consider valid if score > 80 and no critical/error issues
	r.IsValid = r.Score > 80.0 && !r.HasCriticalOrErrors()
}

// HasCriticalOrErrors checks if the report contains critical or error level issues
func (r *ValidationReport) HasCriticalOrErrors() bool {
	for _, result := range r.Results {
		if result.Level == LevelCritical || result.Level == LevelError {
			return true
		}
	}
	return false
}

// GetByLevel returns all results of a specific level
func (r *ValidationReport) GetByLevel(level ValidationLevel) []ValidationResult {
	var results []ValidationResult
	for _, result := range r.Results {
		if result.Level == level {
			results = append(results, result)
		}
	}
	return results
}

// MCPMessage represents a generic MCP protocol message
type MCPMessage struct {
	JSONRPCVersion string                 `json:"jsonrpc"`
	ID             interface{}            `json:"id,omitempty"`
	Method         string                 `json:"method,omitempty"`
	Params         map[string]interface{} `json:"params,omitempty"`
	Result         interface{}            `json:"result,omitempty"`
	Error          *MCPError              `json:"error,omitempty"`
}

// MCPError represents an MCP error object
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Validator interface for all validation implementations
type Validator interface {
	Validate(message *MCPMessage, context map[string]string) *ValidationReport
	GetName() string
	GetDescription() string
}

// ValidatorConfig holds configuration for validators
type ValidatorConfig struct {
	Enabled        bool                   `json:"enabled"`
	StrictMode     bool                   `json:"strict_mode"`     // Fail on warnings
	SkipValidators []string               `json:"skip_validators"` // Validator names to skip
	CustomRules    map[string]interface{} `json:"custom_rules"`
}

// ValidationEngine coordinates multiple validators
type ValidationEngine struct {
	validators []Validator
	config     *ValidatorConfig
}

// NewValidationEngine creates a new validation engine
func NewValidationEngine(config *ValidatorConfig) *ValidationEngine {
	if config == nil {
		config = &ValidatorConfig{
			Enabled:    true,
			StrictMode: false,
		}
	}

	return &ValidationEngine{
		validators: make([]Validator, 0),
		config:     config,
	}
}

// RegisterValidator adds a validator to the engine
func (e *ValidationEngine) RegisterValidator(validator Validator) {
	// Check if validator should be skipped
	for _, skip := range e.config.SkipValidators {
		if skip == validator.GetName() {
			return
		}
	}

	e.validators = append(e.validators, validator)
}

// ValidateMessage runs all validators against a message
func (e *ValidationEngine) ValidateMessage(sessionID, messageID string, messageData []byte, context map[string]string) (*ValidationReport, error) {
	if !e.config.Enabled {
		return &ValidationReport{
			SessionID: sessionID,
			MessageID: messageID,
			Score:     100.0,
			IsValid:   true,
			Timestamp: time.Now(),
		}, nil
	}

	start := time.Now()

	// Parse the message
	var message MCPMessage
	if err := json.Unmarshal(messageData, &message); err != nil {
		return &ValidationReport{
			SessionID:   sessionID,
			MessageID:   messageID,
			MessageType: "invalid",
			Results: []ValidationResult{{
				ID:        "json_parse_error",
				Level:     LevelCritical,
				Message:   fmt.Sprintf("Failed to parse JSON: %v", err),
				Timestamp: time.Now(),
			}},
			Score:        0.0,
			IsValid:      false,
			ProcessingMS: time.Since(start).Milliseconds(),
			Timestamp:    time.Now(),
		}, nil
	}

	// Determine message type
	messageType := "unknown"
	if message.Method != "" {
		if message.ID != nil {
			messageType = "request"
		} else {
			messageType = "notification"
		}
	} else if message.Result != nil || message.Error != nil {
		messageType = "response"
	}

	// Create base report
	report := &ValidationReport{
		SessionID:   sessionID,
		MessageID:   messageID,
		MessageType: messageType,
		Method:      message.Method,
		Results:     make([]ValidationResult, 0),
		Timestamp:   time.Now(),
	}

	// Run all validators
	for _, validator := range e.validators {
		validatorReport := validator.Validate(&message, context)
		if validatorReport != nil {
			report.Results = append(report.Results, validatorReport.Results...)
		}
	}

	// Update final score and validity
	report.updateScore()

	// Apply strict mode
	if e.config.StrictMode && len(report.GetByLevel(LevelWarning)) > 0 {
		report.IsValid = false
	}

	report.ProcessingMS = time.Since(start).Milliseconds()
	return report, nil
}

// GetValidatorNames returns names of all registered validators
func (e *ValidationEngine) GetValidatorNames() []string {
	names := make([]string, len(e.validators))
	for i, validator := range e.validators {
		names[i] = validator.GetName()
	}
	return names
}

// GetStats returns validation engine statistics
func (e *ValidationEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":         e.config.Enabled,
		"strict_mode":     e.config.StrictMode,
		"validator_count": len(e.validators),
		"validators":      e.GetValidatorNames(),
		"skip_list":       e.config.SkipValidators,
	}
}

// Helper functions for validation

// IsValidJSONRPCVersion checks if the JSON-RPC version is valid
func IsValidJSONRPCVersion(version string) bool {
	return version == "2.0"
}

// IsValidMCPMethod checks if the method name follows MCP conventions
func IsValidMCPMethod(method string) bool {
	if method == "" {
		return false
	}

	// MCP method names should not contain spaces or special characters
	if strings.Contains(method, " ") || strings.ContainsAny(method, "!@#$%^&*()") {
		return false
	}

	return true
}

// IsValidID checks if the ID is valid (string, number, or null)
func IsValidID(id interface{}) bool {
	if id == nil {
		return true // null is valid
	}

	switch id.(type) {
	case string, int, int64, float64:
		return true
	default:
		return false
	}
}
